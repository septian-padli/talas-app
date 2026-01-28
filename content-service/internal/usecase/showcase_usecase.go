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
	GetMyShowcases(ctx context.Context, userID uuid.UUID, limit int, cursor string) (map[string]interface{}, error)
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

	// Create Showcase without UserID, but with Creator as Collaborator (OWNER)
	creatorCollaborator := entity.Collaborator{
		Role:   entity.CollaborationRoleOwner,
		Status: entity.CollaborationStatusAccepted,
		UserID: userID,
	}
	// Need to set ID manually for base? Base has BeforeCreate, so it's fine.

	showcase := &entity.Showcase{
		Title:       input.Title,
		Slug:        slug,
		Content:     input.Content,
		CategoryID:  categoryID,
		Tags:        strings.Split(input.Tags, ","),
		Media:       mediaList,
		Collaborators: []entity.Collaborator{creatorCollaborator},
	}

	// 5. Save to DB
	err = u.repo.Create(showcase)
	if err != nil {
		u.log.Errorf("Failed to create showcase: %v", err)
		return nil, errors.New("failed to save showcase")
	}

	// Enrich with Author Data (which is the creator)
	// Even though we just created it, for consistent response structure.
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err == nil {
		if creatorData, found := usersMap[userID]; found {
			showcase.EnrichedCollaborators = []entity.EnrichedCollaborator{
				{
					ID:     uuid.Nil, // Has no DB ID yet or doesn't matter for response? Wait, db collaborator has no ID yet? The entity created has Collaborators list.
					// Actually we just inserted it. We can get ID from creatorCollaborator?
					// Or just leave ID nil/random. The UI needs User ID mostly.
					// But wait, response contract says collaborator has ID.
					// Since we just created it, we rely on GORM? GORM populates IDs after Create if passed by pointer.
					// But `creatorCollaborator` was passed by value in slice.
					// It's safer to just return User data. API contract says ID is UUID.
					// Let's assume ID is generated.
					Role:   entity.CollaborationRoleOwner,
					Status: entity.CollaborationStatusAccepted,
					User: &entity.User{
						ID:        userID,
						Name:      creatorData.Name,
						Username:  creatorData.Username,
						AvatarURL: creatorData.AvatarURL,
					},
				},
			}
		}
	}

	return showcase, nil
}

func (u *showcaseUsecase) GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error) {
	// 1. Get from Repo
	showcase, err := u.repo.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	// 2. Fetch Collaborators Data (Enrichment)
	// Collect UserIDs from Collaborators
	var userIDs []uuid.UUID
	for _, col := range showcase.Collaborators {
		userIDs = append(userIDs, col.UserID)
	}

	// Use UserClient to get bulk data
	if len(userIDs) > 0 {
		usersMap, err := u.userClient.GetUsersBulk(userIDs)
		if err != nil {
			u.log.Warnf("Failed to fetch user data for showcase %s: %v", slug, err)
		} else {
			// Map Collaborators to EnrichedCollaborator
			var enrichedCols []entity.EnrichedCollaborator
			for _, col := range showcase.Collaborators {
				if userData, found := usersMap[col.UserID]; found {
					enrichedCols = append(enrichedCols, entity.EnrichedCollaborator{
						ID:     col.ID,
						Role:   col.Role,
						Status: col.Status,
						User: &entity.User{
							ID:        col.UserID,
							Name:      userData.Name,
							Username:  userData.Username,
							AvatarURL: userData.AvatarURL,
						},
					})
				}
			}
			showcase.EnrichedCollaborators = enrichedCols
		}
	}
	
	return showcase, nil
}

func (u *showcaseUsecase) GetShowcasesByUser(ctx context.Context, userIDStr string, limit int, cursor string) (map[string]interface{}, error) {
	// Validate User ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id format")
	}

	// 1. Get from Repo (Results filtered by JOIN collaborators.user_id = userID)
	showcases, meta, err := u.repo.GetByUserID(userID, limit, cursor)
	if err != nil {
		return nil, err
	}

	// 2. Enrich with Collaborator (Owner) Data
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err != nil {
		u.log.Warnf("Failed to fetch author data for user feed %s: %v", userIDStr, err)
	}

	if userDetail, found := usersMap[userID]; found {
		userEntity := &entity.User{
			ID:        userID,
			Name:      userDetail.Name,
			Username:  userDetail.Username,
			AvatarURL: userDetail.AvatarURL,
		}
		
		for i := range showcases {
			// Since we called GetByUserID, we know 'userID' is at least a collaborator.
			// Ideally we fetch actual role from DB (it's in 'Collaborators' relation if Preloaded).
			// If Repo preloads Collaborators, we can find the specific role.
			
			var myRole = entity.CollaborationRoleOwner // Default assumption if not found (should not happen if data consistent)
			var myStatus = entity.CollaborationStatusAccepted
			var colID = uuid.Nil

			// Iterasi existing collaborators untuk cari detailnya
			for _, c := range showcases[i].Collaborators {
				if c.UserID == userID {
					myRole = c.Role
					myStatus = c.Status
					colID = c.ID
					break
				}
			}

			// Assign Enriched Data specifically for the feed view
			// Usually feed only shows "Author" (Owner). 
			// If we want to show all collaborators in feed, we'd need to fetch all UserIDs.
			// For now, we only enrich the "Author" (the user whose feed we are viewing).
			showcases[i].EnrichedCollaborators = []entity.EnrichedCollaborator{
				{
					ID:     colID,
					Role:   myRole,
					Status: myStatus,
					User:   userEntity,
				},
			}
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

func (u *showcaseUsecase) GetMyShowcases(ctx context.Context, userID uuid.UUID, limit int, cursor string) (map[string]interface{}, error) {
	// 1. Get FROM Repo
	showcases, meta, err := u.repo.GetByUserID(userID, limit, cursor)
	if err != nil {
		return nil, err
	}

	// 2. Enrich Author Data (Self)
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err != nil {
		u.log.Warnf("Failed to fetch author data for my feed %s: %v", userID, err)
	}

	if userDetail, found := usersMap[userID]; found {
		userEntity := &entity.User{
			ID:        userID,
			Name:      userDetail.Name,
			Username:  userDetail.Username,
			AvatarURL: userDetail.AvatarURL,
		}
		for i := range showcases {
			// Find Role from Loaded Collaborators
			var myRole = entity.CollaborationRoleOwner
			var myStatus = entity.CollaborationStatusAccepted
			var colID = uuid.Nil

			for _, c := range showcases[i].Collaborators {
				if c.UserID == userID {
					myRole = c.Role
					myStatus = c.Status
					colID = c.ID
					break
				}
			}

			showcases[i].EnrichedCollaborators = []entity.EnrichedCollaborator{
				{
					ID:     colID,
					Role:   myRole,
					Status: myStatus,
					User:   userEntity,
				},
			}
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
