package usecase

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/repository"
	"github.com/septianpadli/talas/content-service/pkg/clients"
	"github.com/sirupsen/logrus"
)

type ShowcaseUsecase interface {
	CreateShowcase(ctx context.Context, input *entity.CreateShowcaseRequest, files []*multipart.FileHeader, userID uuid.UUID) (*entity.Showcase, error)
	GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error)
	GetShowcasesByUser(ctx context.Context, userIDStr string, limit int, cursor string) (map[string]interface{}, error)
	GetMyShowcases(ctx context.Context, userID uuid.UUID, statusFilter string, limit int, cursor string) (map[string]interface{}, error)
}

type showcaseUsecase struct {
	repo       repository.ShowcaseRepository
	userClient clients.UserClient
	cfg        *config.Config
	log        *logrus.Logger
	validate   *validator.Validate
}

func NewShowcaseUsecase(repo repository.ShowcaseRepository, userClient clients.UserClient, cfg *config.Config, log *logrus.Logger) ShowcaseUsecase {
	return &showcaseUsecase{
		repo:       repo,
		userClient: userClient,
		cfg:        cfg,
		log:        log,
		validate:   validator.New(),
	}
}

func (u *showcaseUsecase) CreateShowcase(ctx context.Context, input *entity.CreateShowcaseRequest, files []*multipart.FileHeader, userID uuid.UUID) (*entity.Showcase, error) {
	// 1. Validate Input
	if err := u.validate.Struct(input); err != nil {
		return nil, err
	}

	// 2. Validate Files (Count & Size)
	if len(files) == 0 {
		return nil, errors.New("at least 1 image file is required")
	}
	if len(files) > 10 {
		return nil, errors.New("maximum 10 image files allowed")
	}

	for _, file := range files {
		if file.Size > 2*1024*1024 { // 2MB
			return nil, errors.New("file " + file.Filename + " is too large (max 2MB)")
		}
		ext := strings.ToLower(file.Filename[strings.LastIndex(file.Filename, ".")+1:])
		if ext != "jpg" && ext != "jpeg" && ext != "png" {
			return nil, errors.New("file " + file.Filename + " has invalid type (only jpg, jpeg, png allowed)")
		}
	}

	// 3. Upload to Cloudinary & Prepare Media
	cld, err := cloudinary.NewFromParams(u.cfg.CloudinaryCloudName, u.cfg.CloudinaryAPIKey, u.cfg.CloudinaryAPISecret)
	if err != nil {
		u.log.Errorf("Failed to init cloudinary: %v", err)
		return nil, errors.New("internal server error")
	}

	var mediaList []entity.ShowcaseMedia

	for i, file := range files {
		fileContent, err := file.Open()
		if err != nil {
			return nil, err
		}
		
		uploadResult, err := cld.Upload.Upload(ctx, fileContent, uploader.UploadParams{
			Folder: "talas/showcases",
		})
		fileContent.Close() // Close immediately after upload

		if err != nil {
			u.log.Errorf("Failed to upload file %s: %v", file.Filename, err)
			return nil, errors.New("failed to upload image: " + file.Filename)
		}

		mediaList = append(mediaList, entity.ShowcaseMedia{
			URL:      uploadResult.SecureURL,
			Type:     "IMAGE",
			Position: i + 1,
		})
	}

	// 4. Create Entity
	categoryID, _ := uuid.Parse(input.CategoryID)
	slug := strings.ReplaceAll(strings.ToLower(input.Title), " ", "-") + "-" + uuid.New().String()[:8]

	showcase := &entity.Showcase{
		UserID:      userID,
		Title:       input.Title,
		Slug:        slug,
		Description: input.Description,
		CategoryID:  categoryID,
		Status:      entity.StatusPublished,
		Tags:        strings.Split(input.Tags, ","),
		Media:       mediaList,
	}

	// 5. Save to DB
	if err := u.repo.Create(showcase); err != nil {
		u.log.Errorf("Failed to create showcase in db: %v", err)
		return nil, err
	}

	return showcase, nil
}

func (u *showcaseUsecase) GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error) {
	// 1. Get from Repo
	showcase, err := u.repo.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	// 2. Fetch Author & Collaborators Data (Enrichment)
	userIDs := []uuid.UUID{showcase.UserID}
	
	// Collect Collaborator IDs
	for _, col := range showcase.Collaborators {
		userIDs = append(userIDs, col.UserID)
	}

	// Use UserClient to get bulk data
	usersMap, err := u.userClient.GetUsersBulk(userIDs)
	if err != nil {
		u.log.Warnf("Failed to fetch user data for showcase %s: %v", slug, err)
	} else {
		// Map Author
		if authorData, found := usersMap[showcase.UserID]; found {
			showcase.Author = &entity.User{
				ID:        showcase.UserID,
				Name:      authorData.Name,
				Username:  authorData.Username,
				AvatarURL: authorData.AvatarURL,
			}
		}

		// Map Collaborators
		var enrichedCols []*entity.User
		for _, col := range showcase.Collaborators {
			if userData, found := usersMap[col.UserID]; found {
				enrichedCols = append(enrichedCols, &entity.User{
					ID:        col.UserID,
					Name:      userData.Name,
					Username:  userData.Username,
					AvatarURL: userData.AvatarURL,
				})
			}
		}
		showcase.EnrichedCollaborators = enrichedCols
	}
	return showcase, nil
}

func (u *showcaseUsecase) GetShowcasesByUser(ctx context.Context, userIDStr string, limit int, cursor string) (map[string]interface{}, error) {
	// Validate User ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id format")
	}

	// 1. Get from Repo (Public only shows PUBLISHED)
	showcases, meta, err := u.repo.GetByUserID(userID, []string{entity.StatusPublished}, limit, cursor)
	if err != nil {
		return nil, err
	}

	// 2. Enrich with Author Data
	// Since all showcases belong to the SAME user, we only need to fetch ONE user from User Service.
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err != nil {
		u.log.Warnf("Failed to fetch author data for user feed %s: %v", userIDStr, err)
	}

	// Apply Author Data to all items
	// Note: We are returning []entity.Showcase, but we can return []FeedShowcaseItem DTO if preferred.
	// For simplicity, we assume entity.Showcase has JSON tags close enough to FeedShowcaseItem or we map it here.
	// Looking at API contract FeedShowcaseItem, it matches entity.Showcase JSON tags.

	if authorDetail, found := usersMap[userID]; found {
		authorEntity := &entity.User{
			ID:        userID,
			Name:      authorDetail.Name,
			Username:  authorDetail.Username,
			AvatarURL: authorDetail.AvatarURL,
		}
		for i := range showcases {
			showcases[i].Author = authorEntity
		}
	}

	// 3. Construct Response
	response := map[string]interface{}{
		"showcases": showcases,
		"pagination": map[string]interface{}{
			"next_cursor": meta.NextCursor,
			"has_next":    meta.HasNext,
		},
	}

	return response, nil
}

func (u *showcaseUsecase) GetMyShowcases(ctx context.Context, userID uuid.UUID, statusFilter string, limit int, cursor string) (map[string]interface{}, error) {
	// Determine Allowed Statuses
	var allowedStatuses []string
	switch strings.ToLower(statusFilter) {
	case "published":
		allowedStatuses = []string{entity.StatusPublished}
	case "archived":
		allowedStatuses = []string{entity.StatusArchived}
	case "draft": // Optional if needed
		allowedStatuses = []string{entity.StatusDraft}
	default: // "all" or empty
		allowedStatuses = []string{entity.StatusPublished, entity.StatusArchived, entity.StatusDraft}
	}

	// 1. Get FROM Repo
	showcases, meta, err := u.repo.GetByUserID(userID, allowedStatuses, limit, cursor)
	if err != nil {
		return nil, err
	}

	// 2. Enrich Author Data (Self)
	// Even though it's "Me", fetching from User Service ensures we get latest Avatar/Name
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err != nil {
		u.log.Warnf("Failed to fetch author data for my feed %s: %v", userID, err)
	}

	if authorDetail, found := usersMap[userID]; found {
		authorEntity := &entity.User{
			ID:        userID,
			Name:      authorDetail.Name,
			Username:  authorDetail.Username,
			AvatarURL: authorDetail.AvatarURL,
		}
		for i := range showcases {
			showcases[i].Author = authorEntity
		}
	}

	// 3. Construct Response
	response := map[string]interface{}{
		"showcases": showcases,
		"pagination": map[string]interface{}{
			"next_cursor": meta.NextCursor,
			"has_next":    meta.HasNext,
		},
	}

	return response, nil
}
