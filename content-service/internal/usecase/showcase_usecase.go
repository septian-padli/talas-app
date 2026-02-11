package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/repository"
	"github.com/septianpadli/talas/content-service/pkg/clients"
	"github.com/septianpadli/talas/content-service/pkg/media"
	"github.com/septianpadli/talas/content-service/pkg/rabbitmq"
	"github.com/sirupsen/logrus"
)

type ShowcaseUsecase interface {
	CreateShowcase(ctx context.Context, input *entity.CreateShowcaseRequest, files []*multipart.FileHeader, userID uuid.UUID) (*entity.Showcase, error)
	GetShowcaseByID(ctx context.Context, id uuid.UUID) (*entity.Showcase, error)
	GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error)
	GetShowcasesByUser(ctx context.Context, userIDStr string, limit int, cursor string) (map[string]interface{}, error)
	GetMyShowcases(ctx context.Context, userID uuid.UUID, limit int, cursor string) (map[string]interface{}, error)
	UpdateShowcase(ctx context.Context, id uuid.UUID, input *entity.UpdateShowcaseRequest, userID uuid.UUID, files []*multipart.FileHeader) (*entity.Showcase, error)
	ToggleLike(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	ToggleBookmark(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	DeleteShowcase(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// Search
	SearchShowcases(ctx context.Context, query string, limit int, cursor string, categorySlugs string, sortBy string) (map[string]interface{}, error)
	GetTrendingFeeds(ctx context.Context, limit int, cursor string) (map[string]interface{}, error)
}

type showcaseUsecase struct {
	showcaseRepo   repository.ShowcaseRepository
	categoryRepo   repository.CategoryRepository
	searchRepo     repository.SearchRepository
	userClient     clients.UserClient
	mediaUploader  media.MediaUploader
	eventPublisher rabbitmq.EventPublisher
	redisClient    *redis.Client
	cfg            *config.Config
	log            *logrus.Logger
	validate       *validator.Validate
}

func NewShowcaseUsecase(
	showcaseRepo repository.ShowcaseRepository,
	categoryRepo repository.CategoryRepository,
	searchRepo repository.SearchRepository,
	userClient clients.UserClient,
	mediaUploader media.MediaUploader,
	eventPublisher rabbitmq.EventPublisher,
	redisClient *redis.Client,
	cfg *config.Config,
	log *logrus.Logger,
) ShowcaseUsecase {
	return &showcaseUsecase{
		showcaseRepo:   showcaseRepo,
		categoryRepo:   categoryRepo,
		searchRepo:     searchRepo,
		userClient:     userClient,
		mediaUploader:  mediaUploader,
		eventPublisher: eventPublisher,
		redisClient:    redisClient,
		cfg:            cfg,
		log:            log,
		validate:       validator.New(),
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
	var mediaList []entity.ShowcaseMedia

	for i, file := range files {
		fileContent, err := file.Open()
		if err != nil {
			return nil, err
		}

		// Use u.mediaUploader
		secureURL, err := u.mediaUploader.Upload(ctx, fileContent, file.Filename, "talas/showcases")
		fileContent.Close() // Close immediately after upload

		if err != nil {
			u.log.Errorf("Failed to upload file %s: %v", file.Filename, err)
			return nil, errors.New("failed to upload image: " + file.Filename)
		}

		mediaList = append(mediaList, entity.ShowcaseMedia{
			URL:      secureURL,
			Type:     "IMAGE",
			Position: i + 1,
		})
	}

	// 4. Create Entity
	categoryID, _ := uuid.Parse(input.CategoryID)
	slug := strings.ReplaceAll(strings.ToLower(input.Title), " ", "-") + "-" + uuid.New().String()[:8]

	// Create Showcase without UserID, but with Creator as Collaborator (OWNER)
	creatorCollaborator := entity.Collaborator{
		Base:   entity.Base{ID: uuid.New()}, // Generate ID manually to ensure it's in response
		Role:   entity.CollaborationRoleOwner,
		Status: entity.CollaborationStatusAccepted,
		UserID: userID,
	}

	showcase := &entity.Showcase{
		Title:         input.Title,
		Slug:          slug,
		Content:       input.Content,
		CategoryID:    categoryID,
		Tags:          strings.Split(input.Tags, ","),
		Media:         mediaList,
		Collaborators: []entity.Collaborator{creatorCollaborator},
		LikesCount:    0,
		ViewsCount:    0,
	}
	// showcase := &entity.Showcase{
	// 	Title:       input.Title,
	// 	Slug:        slug,
	// 	Content:     input.Content,
	// 	CategoryID:  categoryID,
	// 	Tags:        strings.Split(input.Tags, ","),
	// 	Media:       mediaList,
	// 	Collaborators: []entity.Collaborator{creatorCollaborator},
	// 	LikesCount: 0,
	// 	ViewsCount: 0,
	// }

	// 5. Enrich with Author Data (Fetch before saving/publishing to ensure data availability)
	// We need this for the event payload to be rich
	var creatorName, creatorUsername string
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err == nil {
		if creatorData, found := usersMap[userID]; found {
			creatorName = creatorData.Name
			creatorUsername = creatorData.Username
		}
	} else {
		u.log.Warnf("Failed to fetch author data for showcase creation: %v", err)
	}

	// Fallback if user service fails (should ideally retry or fail, but for now fallback)
	if creatorUsername == "" {
		creatorUsername = "unknown"
	}
	if creatorName == "" {
		creatorName = "Unknown User"
	}
	creatorAvatar := ""
	if creatorData, ok := usersMap[userID]; ok {
		creatorAvatar = creatorData.AvatarURL
	}

	err = u.showcaseRepo.Create(showcase)
	if err != nil {
		u.log.Errorf("Failed to create showcase: %v", err)
		return nil, errors.New("failed to save showcase")
	}

	// 6. Publish Event (Async, Fail-safe) - Send full data for ES indexing
	go func() {
		eventData := map[string]interface{}{
			"id":          showcase.ID,
			"title":       showcase.Title,
			"slug":        showcase.Slug,
			"tags":        showcase.Tags,
			"content":     showcase.Content,
			"category_id": showcase.CategoryID,
			"owner": map[string]interface{}{
				"id":         userID,
				"username":   creatorUsername,
				"full_name":  creatorName,
				"avatar_url": creatorAvatar,
			},
			"collaborators": []map[string]interface{}{}, // Owner is separate, initially empty collaborators
			"like_count":    showcase.LikesCount,
			"view_count":    showcase.ViewsCount,
			"created_at":    showcase.CreatedAt,
			"updated_at":    showcase.UpdatedAt,
		}
		if err := u.eventPublisher.Publish(context.Background(), "showcase.created", eventData); err != nil {
			u.log.Errorf("Failed to publish showcase.created event: %v", err)
		} else {
			u.log.Debugf("Published showcase.created event for %s", showcase.ID)
		}
	}()

	// Enrich with Author Data (Already fetched above)
	showcase.EnrichedCollaborators = []entity.EnrichedCollaborator{
		{
			ID:     creatorCollaborator.ID,
			Role:   entity.CollaborationRoleOwner,
			Status: entity.CollaborationStatusAccepted,
			User: &entity.User{
				ID:       userID,
				Name:     creatorName,
				Username: creatorUsername,
			},
		},
	}

	// For now, let's keep it simple. The previous code re-fetched.
	// We can reuse the fetched data if we extract variable scope.
	// But to avoid large diff, we can just use the variables we have.
	// Only AvatarURL is missing from local vars.
	if creatorData, found := usersMap[userID]; found {
		showcase.EnrichedCollaborators[0].User.AvatarURL = creatorData.AvatarURL
	}

	return showcase, nil
}

func (u *showcaseUsecase) UpdateShowcase(ctx context.Context, id uuid.UUID, input *entity.UpdateShowcaseRequest, userID uuid.UUID, files []*multipart.FileHeader) (*entity.Showcase, error) {
	// 1. Get Existing Showcase
	showcase, err := u.showcaseRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	// Note: First() returns error if not found, so err check covers it.

	// 2. Check Ownership (Must be OWNER)
	isOwner := false
	for _, col := range showcase.Collaborators {
		if col.UserID == userID && col.Role == entity.CollaborationRoleOwner {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return nil, errors.New("forbidden: only owner can update showcase")
	}

	// 3. Apply Updates
	updated := false

	if input.Title != nil {
		if len(*input.Title) < 3 { // Validation constraint
			return nil, errors.New("title must be at least 3 characters")
		}
		showcase.Title = *input.Title
		updated = true
	}

	if input.Content != nil {
		if len(*input.Content) < 10 {
			return nil, errors.New("content must be at least 10 characters")
		}
		showcase.Content = *input.Content
		updated = true
	}

	if input.CategoryID != nil {
		catID, err := uuid.Parse(*input.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category id format")
		}
		showcase.CategoryID = catID
		updated = true
	}

	if input.Tags != nil {
		showcase.Tags = input.Tags
		updated = true
	}

	// Tambahkan validasi dan update media jika ada file baru
	if len(files) > 0 {
		if len(files) > 10 {
			return nil, errors.New("maximum 10 image files allowed")
		}
		var mediaList []entity.ShowcaseMedia
		for i, file := range files {
			if file.Size > 2*1024*1024 { // 2MB
				return nil, errors.New("file " + file.Filename + " is too large (max 2MB)")
			}
			ext := strings.ToLower(file.Filename[strings.LastIndex(file.Filename, ".")+1:])
			if ext != "jpg" && ext != "jpeg" && ext != "png" {
				return nil, errors.New("file " + file.Filename + " has invalid type (only jpg, jpeg, png allowed)")
			}
			fileContent, err := file.Open()
			if err != nil {
				return nil, err
			}
			secureURL, err := u.mediaUploader.Upload(ctx, fileContent, file.Filename, "talas/showcases")
			fileContent.Close()
			if err != nil {
				u.log.Errorf("Failed to upload file %s: %v", file.Filename, err)
				return nil, errors.New("failed to upload image: " + file.Filename)
			}
			mediaList = append(mediaList, entity.ShowcaseMedia{
				URL:      secureURL,
				Type:     "IMAGE",
				Position: i + 1,
			})
		}
		showcase.Media = mediaList
		updated = true
	}

	if updated {
		showcase.IsEdited = true
		// 4. Save
		err = u.showcaseRepo.Update(showcase)
		if err != nil {
			u.log.Errorf("Failed to update showcase %s: %v", id, err)
			return nil, errors.New("failed to update showcase")
		}

		// 5. Publish Event (Async, Fail-safe)
		go func() {
			eventData := map[string]interface{}{
				"id":          showcase.ID,
				"title":       showcase.Title,
				"slug":        showcase.Slug,
				"content":     showcase.Content,
				"tags":        showcase.Tags,
				"category_id": showcase.CategoryID,
				"owner_id":    userID,
				"like_count":  showcase.LikesCount,
				"view_count":  showcase.ViewsCount,
				"created_at":  showcase.CreatedAt,
				"updated_at":  showcase.UpdatedAt,
			}
			if err := u.eventPublisher.Publish(context.Background(), "showcase.updated", eventData); err != nil {
				u.log.Errorf("Failed to publish showcase.updated event: %v", err)
			} else {
				u.log.Debugf("Published showcase.updated event for %s", showcase.ID)
			}
		}()
	}

	return showcase, nil
}

func (u *showcaseUsecase) ToggleLike(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error) {
	// 1. Check if Showcase exists
	showcase, err := u.showcaseRepo.GetByID(showcaseID)
	if err != nil {
		return false, err
	}
	if showcase == nil {
		return false, errors.New("showcase not found")
	}

	// Extract Owner ID for notification
	var ownerID uuid.UUID
	for _, col := range showcase.Collaborators {
		if col.Role == entity.CollaborationRoleOwner {
			ownerID = col.UserID
			break
		}
	}

	// 2. Toggle in Repo
	isLiked, err := u.showcaseRepo.ToggleLike(userID, showcaseID)
	if err != nil {
		u.log.Errorf("Failed to toggle like: %v", err)
		return false, errors.New("failed to toggle like")
	}

	// 3. Publish Event (Async, Fail-safe)
	go func() {
		routingKey := "showcase.unliked"
		if isLiked {
			routingKey = "showcase.liked"
		}

		eventData := map[string]interface{}{
			"showcase_id":    showcaseID,
			"actor_id":       userID,
			"target_user_id": ownerID,
		}

		if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
			u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
		} else {
			u.log.Debugf("Published %s event for showcase %s", routingKey, showcaseID)
		}
	}()

	return isLiked, nil
}

func (u *showcaseUsecase) ToggleBookmark(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error) {
	// 1. Check if Showcase exists
	showcase, err := u.showcaseRepo.GetByID(showcaseID)
	if err != nil {
		return false, err
	}
	if showcase == nil {
		return false, errors.New("showcase not found")
	}

	// 2. Toggle in Repo
	isBookmarked, err := u.showcaseRepo.ToggleBookmark(userID, showcaseID)
	if err != nil {
		u.log.Errorf("Failed to toggle bookmark: %v", err)
		return false, errors.New("failed to toggle bookmark")
	}

	// 3. Publish Event (Async, Fail-safe)
	go func() {
		routingKey := "showcase.unbookmarked"
		if isBookmarked {
			routingKey = "showcase.bookmarked"
		}

		eventData := map[string]interface{}{
			"showcase_id": showcaseID,
			"actor_id":    userID,
		}

		if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
			u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
		} else {
			u.log.Debugf("Published %s event for showcase %s", routingKey, showcaseID)
		}
	}()

	return isBookmarked, nil
}

func (u *showcaseUsecase) DeleteShowcase(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// 1. Get Showcase
	showcase, err := u.showcaseRepo.GetByID(id)
	if err != nil {
		return err
	}
	if showcase == nil {
		return errors.New("showcase not found")
	}

	// 2. Check Ownership
	isOwner := false
	for _, col := range showcase.Collaborators {
		if col.UserID == userID && col.Role == entity.CollaborationRoleOwner {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return errors.New("forbidden: only owner can delete showcase")
	}

	// 3. Delete (Soft)
	if err := u.showcaseRepo.DeleteShowcase(id); err != nil {
		u.log.Errorf("Failed to delete showcase %s: %v", id, err)
		return errors.New("failed to delete showcase")
	}

	// 4. Publish Event (Async, Fail-safe)
	go func() {
		eventData := map[string]interface{}{
			"id": id,
		}
		if err := u.eventPublisher.Publish(context.Background(), "showcase.deleted", eventData); err != nil {
			u.log.Errorf("Failed to publish showcase.deleted event: %v", err)
		} else {
			u.log.Debugf("Published showcase.deleted event for %s", id)
		}
	}()

	return nil
}

// GetShowcaseByID fetches showcase by ID only (lightweight, for internal API)
func (u *showcaseUsecase) GetShowcaseByID(ctx context.Context, id uuid.UUID) (*entity.Showcase, error) {
	showcase, err := u.showcaseRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return showcase, nil
}

func (u *showcaseUsecase) GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error) {
	// 1. Get from Repo
	var showcase *entity.Showcase
	var err error

	// Check if input is UUID
	if id, parseErr := uuid.Parse(slug); parseErr == nil {
		showcase, err = u.showcaseRepo.GetByID(id)
	} else {
		showcase, err = u.showcaseRepo.GetBySlug(slug)
	}

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

	// 3. Increment View Count (Fire & Forget)
	// Only if showcase is found
	go func() {
		// Increment DB
		if err := u.showcaseRepo.IncrementViewCount(showcase.ID); err != nil {
			u.log.Warnf("Failed to increment view count for showcase %s: %v", showcase.ID, err)
		}

		// Find Owner ID for notification/analytics
		var ownerID uuid.UUID
		for _, col := range showcase.Collaborators {
			if col.Role == entity.CollaborationRoleOwner {
				ownerID = col.UserID
				break
			}
		}

		// Publish Event for Search Sync / Analytics
		eventData := map[string]interface{}{
			"showcase_id": showcase.ID,
			"viewer_id":   nil, // Anonymous for now, or from context if auth available?
			// Context here is tricky if we user ID.
			// For now, let's just send showcase_id and owner.
			"target_user_id": ownerID,
		}

		if err := u.eventPublisher.Publish(context.Background(), "showcase.viewed", eventData); err != nil {
			u.log.Errorf("Failed to publish showcase.viewed event: %v", err)
		}
	}()

	return showcase, nil
}

func (u *showcaseUsecase) GetShowcasesByUser(ctx context.Context, userIDStr string, limit int, cursor string) (map[string]interface{}, error) {
	// Validate User ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id format")
	}

	// 1. Get from Repo (Results filtered by JOIN collaborators.user_id = userID)
	showcases, meta, err := u.showcaseRepo.GetByUserID(userID, limit, cursor)
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
	showcases, meta, err := u.showcaseRepo.GetByUserID(userID, limit, cursor)
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

type searchUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
}

type searchShowcaseResponse struct {
	ID            uuid.UUID            `json:"id"`
	Title         string               `json:"title"`
	Slug          string               `json:"slug"`
	Content       string               `json:"content"`
	Thumbnail     string               `json:"thumbnail,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	DeletedAt     *time.Time           `json:"deleted_at"`
	Tags          []string             `json:"tags"`
	LikesCount    int                  `json:"likes_count"`
	ViewsCount    int                  `json:"views_count"`
	CommentsCount int                  `json:"comments_count"`
	SharesCount   int                  `json:"shares_count"`
	CategoryID    uuid.UUID            `json:"category_id"`
	IsEdited      bool                 `json:"is_edited"`
	Collaborators []searchUserResponse `json:"collaborators"`
}

func (u *showcaseUsecase) SearchShowcases(ctx context.Context, query string, limit int, cursor string, categorySlugs string, sortBy string) (map[string]interface{}, error) {
	// 1. Resolve Category Slugs to IDs
	var categoryIDs []uuid.UUID
	if categorySlugs != "" {
		slugs := strings.Split(categorySlugs, ",")
		// Clean spacing just in case
		for i := range slugs {
			slugs[i] = strings.TrimSpace(slugs[i])
		}

		var err error
		categoryIDs, err = u.categoryRepo.GetCategoryIDsBySlugs(slugs)
		if err != nil {
			u.log.Warnf("Failed to resolve category slugs: %v", err)
			return nil, err
		}
	}

	// 2. Decode Cursor
	var cursorSlice []interface{}
	var err error
	if cursor != "" {
		cursorSlice, err = u.decodeCursor(cursor)
		if err != nil {
			return nil, errors.New("invalid cursor format")
		}
	}

	// 3. Construct Filter
	filter := repository.SearchFilter{
		CategoryIDs: categoryIDs,
		SortBy:      sortBy,
	}

	showcases, lastSortValues, err := u.searchRepo.SearchShowcases(ctx, query, limit, cursorSlice, filter)
	if err != nil {
		u.log.Errorf("Failed to search showcases: %v", err)
		return nil, err
	}

	// Transform Response: Direct Mapping (Zero Allocation Optimization)
	transformedData := make([]searchShowcaseResponse, 0, len(showcases))
	for _, s := range showcases {
		// Extract Users from Collaborators
		users := make([]searchUserResponse, 0, len(s.EnrichedCollaborators))
		for _, ec := range s.EnrichedCollaborators {
			if ec.User != nil {
				users = append(users, searchUserResponse{
					ID:        ec.User.ID,
					Name:      ec.User.Name,
					Username:  ec.User.Username,
					AvatarURL: ec.User.AvatarURL,
				})
			}
		}

		// Handle DeletedAt
		var deletedAt *time.Time
		if s.DeletedAt.Valid {
			deletedAt = &s.DeletedAt.Time
		}

		// Direct Mapping to DTO
		item := searchShowcaseResponse{
			ID:      s.ID,
			Title:   s.Title,
			Slug:    s.Slug,
			Content: s.Content,
			// Thumbnail is removed from entity, likely handled by Media relation or separate logic
			CreatedAt:     s.CreatedAt,
			UpdatedAt:     s.UpdatedAt,
			DeletedAt:     deletedAt,
			Tags:          s.Tags,
			LikesCount:    s.LikesCount,
			ViewsCount:    s.ViewsCount,
			CommentsCount: s.CommentsCount,
			SharesCount:   s.SharesCount,
			CategoryID:    s.CategoryID,
			IsEdited:      s.IsEdited,
			Collaborators: users,
		}
		transformedData = append(transformedData, item)
	}

	// 2. Encode Next Cursor
	var nextCursor string
	var hasMore bool = false

	if len(showcases) == limit {
		// If we got exactly limit items, potentially there are more.
		// Note: search_after doesn't guarantee has_more unless we fetch limit+1.
		// But for simple infinite scroll, returning next_cursor is enough.
		// If next fetch returns empty, then has_more is false.
		// Or we can assume has_more if len == limit.
		hasMore = true
		if len(lastSortValues) > 0 {
			nextCursor = u.encodeCursor(lastSortValues)
		}
	} else if len(showcases) > 0 {
		// Got last page partial
		hasMore = false
	}

	return map[string]interface{}{
		"data": transformedData,
		"meta": map[string]interface{}{
			"next_cursor": nextCursor,
			"has_more":    hasMore,
			"limit":       limit,
		},
	}, nil
}

func (u *showcaseUsecase) GetTrendingFeeds(ctx context.Context, limit int, cursor string) (map[string]interface{}, error) {
	// 1. Determine Page/Offset from Cursor
	page := 1
	if cursor != "" {
		decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			var p int
			if _, err := fmt.Sscanf(string(decodedBytes), "page:%d", &p); err == nil && p > 0 {
				page = p
			}
		}
	}

	offset := (page - 1) * limit
	cacheKey := fmt.Sprintf("feeds:trending:%d:%d", page, limit)

	// 2. Check Redis Cache
	val, err := u.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			u.log.Infof("Cache Hit for Trending Feeds: %s", cacheKey)
			return result, nil
		}
	} else if err != redis.Nil {
		u.log.Warnf("Redis Get Error: %v", err)
	}

	// 3. Get from Repository (Elasticsearch)
	showcases, err := u.searchRepo.GetTrendingShowcases(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// 4. Map to DTO
	var responseDTOs []map[string]interface{}
	for _, s := range showcases {
		var owner interface{}
		var collaborators []interface{}

		for _, col := range s.EnrichedCollaborators {
			var avatar string
			if col.User != nil {
				avatar = col.User.AvatarURL
			}

			u := map[string]interface{}{
				"id":         col.User.ID,
				"username":   col.User.Username,
				"full_name":  col.User.Name,
				"avatar_url": avatar,
			}

			if col.Role == "OWNER" {
				owner = u
			} else {
				collaborators = append(collaborators, u)
			}
		}

		if collaborators == nil {
			collaborators = []interface{}{}
		}

		dto := map[string]interface{}{
			"id":            s.ID,
			"slug":          s.Slug,
			"title":         s.Title,
			"content":       s.Content,
			"tags":          s.Tags,
			"like_count":    s.LikesCount,
			"view_count":    s.ViewsCount,
			"comment_count": s.CommentsCount,
			"owner":         owner,
			"collaborators": collaborators,
			"created_at":    s.CreatedAt,
		}
		responseDTOs = append(responseDTOs, dto)
	}

	if responseDTOs == nil {
		responseDTOs = []map[string]interface{}{}
	}

	// 5. Construct Result & Next Cursor
	nextPage := page + 1
	nextCursor := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("page:%d", nextPage)))

	hasMore := len(showcases) == limit
	if !hasMore {
		nextCursor = ""
	}

	result := map[string]interface{}{
		"data": responseDTOs,
		"meta": map[string]interface{}{
			"next_cursor": nextCursor,
			"has_more":    hasMore,
			"limit":       limit,
		},
	}

	// 6. Set Redis Cache (TTL 10m)
	cacheData, err := json.Marshal(result)
	if err == nil {
		if err := u.redisClient.Set(ctx, cacheKey, cacheData, 10*time.Minute).Err(); err != nil {
			u.log.Warnf("Redis Set Error: %v", err)
		}
	}

	return result, nil
}

// Helper: Decode Cursor (Base64 -> JSON)
func (u *showcaseUsecase) decodeCursor(cursorStr string) ([]interface{}, error) {
	data, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil, err
	}
	var cursor []interface{}
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	return cursor, nil
}

// Helper: Encode Cursor (JSON -> Base64)
func (u *showcaseUsecase) encodeCursor(sortValues []interface{}) string {
	data, _ := json.Marshal(sortValues)
	return base64.StdEncoding.EncodeToString(data)
}
