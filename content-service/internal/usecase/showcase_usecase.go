package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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
	GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error)
	GetShowcasesByUser(ctx context.Context, userIDStr string, limit int, cursor string) (map[string]interface{}, error)
	GetMyShowcases(ctx context.Context, userID uuid.UUID, limit int, cursor string) (map[string]interface{}, error)
	UpdateShowcase(ctx context.Context, id uuid.UUID, input *entity.UpdateShowcaseRequest, userID uuid.UUID) (*entity.Showcase, error)
	ToggleLike(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	ToggleBookmark(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	CreateComment(ctx context.Context, showcaseID uuid.UUID, input *entity.CreateCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	ReplyComment(ctx context.Context, parentID uuid.UUID, input *entity.ReplyCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	GetShowcaseComments(ctx context.Context, showcaseID uuid.UUID, limit int, cursor string) (map[string]interface{}, error)
	UpdateComment(ctx context.Context, id uuid.UUID, input *entity.UpdateCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ToggleCommentLike(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) (bool, int, error)
	DeleteShowcase(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	RemoveCollaborator(ctx context.Context, showcaseID uuid.UUID, targetUserID uuid.UUID, actorUserID uuid.UUID) error
	GetCollaborators(ctx context.Context, showcaseID uuid.UUID, userID uuid.UUID) ([]entity.Collaborator, error)
	DeleteInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID) error
	GetPendingInvitations(ctx context.Context, userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *repository.PaginationMeta, error)
	InviteCollaborators(ctx context.Context, showcaseID uuid.UUID, usernames []string, actorUserID uuid.UUID) ([]string, error)
	RespondInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID, response string) error
	
	// Search
	SearchShowcases(ctx context.Context, query string, page int, limit int) (map[string]interface{}, error)
}

type showcaseUsecase struct {
	repo           repository.ShowcaseRepository
	searchRepo     repository.SearchRepository
	userClient     clients.UserClient
	mediaUploader  media.MediaUploader
	eventPublisher rabbitmq.EventPublisher
	cfg            *config.Config
	log            *logrus.Logger
	validate       *validator.Validate
}

func NewShowcaseUsecase(repo repository.ShowcaseRepository, searchRepo repository.SearchRepository, userClient clients.UserClient, mediaUploader media.MediaUploader, eventPublisher rabbitmq.EventPublisher, cfg *config.Config, log *logrus.Logger) ShowcaseUsecase {
	return &showcaseUsecase{
		repo:           repo,
		searchRepo:     searchRepo,
		userClient:     userClient,
		mediaUploader:  mediaUploader,
		eventPublisher: eventPublisher,
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
		Title:       input.Title,
		Slug:        slug,
		Content:     input.Content,
		CategoryID:  categoryID,
		Tags:        strings.Split(input.Tags, ","),
		Media:       mediaList,
		Collaborators: []entity.Collaborator{creatorCollaborator},
		LikesCount: 0,
		ViewsCount: 0,
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
	if creatorUsername == "" { creatorUsername = "unknown" }
	if creatorName == "" { creatorName = "Unknown User" }
	creatorAvatar := ""
	if creatorData, ok := usersMap[userID]; ok {
		creatorAvatar = creatorData.AvatarURL
	}

	err = u.repo.Create(showcase)
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
				"id":        userID,
				"username":  creatorUsername,
				"full_name": creatorName,
				"avatar_url": creatorAvatar,
			},
			"collaborators": []map[string]interface{}{}, // Owner is separate, initially empty collaborators
			"like_count":  showcase.LikesCount,
			"view_count":  showcase.ViewsCount,
			"created_at":  showcase.CreatedAt,
			"updated_at":  showcase.UpdatedAt,
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
				ID:        userID,
				Name:      creatorName,
				Username:  creatorUsername,
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


func (u *showcaseUsecase) UpdateShowcase(ctx context.Context, id uuid.UUID, input *entity.UpdateShowcaseRequest, userID uuid.UUID) (*entity.Showcase, error) {
	// 1. Get Existing Showcase
	showcase, err := u.repo.GetByID(id)
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
		// Ideally we should update slug too, but for simplicity let's keep slug stable or regenerate
		// For now: Keep slug or update? Let's keep slug stable to avoid broken links.
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
		showcase.CategoryID = catID
		updated = true
	}

	if input.Tags != nil {
		showcase.Tags = input.Tags
		updated = true
	}

	if updated {
		showcase.IsEdited = true
		// 4. Save
		err = u.repo.Update(showcase)
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
	showcase, err := u.repo.GetByID(showcaseID)
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
	isLiked, err := u.repo.ToggleLike(userID, showcaseID)
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
	showcase, err := u.repo.GetByID(showcaseID)
	if err != nil {
		return false, err
	}
	if showcase == nil {
		return false, errors.New("showcase not found")
	}

	// 2. Toggle in Repo
	isBookmarked, err := u.repo.ToggleBookmark(userID, showcaseID)
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

func (u *showcaseUsecase) CreateComment(ctx context.Context, showcaseID uuid.UUID, input *entity.CreateCommentRequest, userID uuid.UUID) (*entity.Comment, error) {
	// 1. Validate Input
	if len(input.Content) < 1 || len(input.Content) > 1000 {
		return nil, errors.New("content must be between 1 and 1000 characters")
	}

	// 2. Check Showcase
	showcase, err := u.repo.GetByID(showcaseID)
	if err != nil {
		return nil, err
	}
	if showcase == nil {
		return nil, errors.New("showcase not found")
	}

	comment := &entity.Comment{
		ShowcaseID: showcaseID,
		UserID:     userID,
		Body:       input.Content,
		LikesCount: 0,
	}

	// 3. Handle Reply
	if input.ParentID != nil && *input.ParentID != "" {
		parentUUID, err := uuid.Parse(*input.ParentID)
		if err != nil {
			return nil, errors.New("invalid parent_id format")
		}

		parentComment, err := u.repo.GetCommentByID(parentUUID)
		if err != nil {
			return nil, err
		}
		if parentComment == nil {
			return nil, errors.New("parent comment not found")
		}

		// Ensure parent comment belongs to same showcase (optional consistency check)
		if parentComment.ShowcaseID != showcaseID {
			return nil, errors.New("parent comment does not belong to this showcase")
		}

		comment.ParentID = &parentUUID
		
		// Fetch Parent Author Username for ReplyTo (Snapshot)
		usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{parentComment.UserID})
		if err == nil {
			if user, ok := usersMap[parentComment.UserID]; ok {
				comment.ReplyTo = user.Username
			}
		} else {
			u.log.Warnf("Failed to fetch parent comment author info: %v", err)
		}
	}

	// 4. Create in Repo
	if err := u.repo.CreateComment(comment); err != nil {
		u.log.Errorf("Failed to create comment: %v", err)
		return nil, errors.New("failed to create comment")
	}

	// 5. Populate Author Info
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err == nil {
		if userDetail, ok := usersMap[userID]; ok {
			comment.Author = &entity.User{
				ID:        userDetail.ID,
				Name:      userDetail.Name,
				Username:  userDetail.Username,
				AvatarURL: userDetail.AvatarURL,
			}
		}
	} else {
		u.log.Warnf("Failed to fetch comment author info: %v", err)
	}

	// 6. Publish Event (Async, Fail-safe)
	// Need Showcase Owner ID
	var showcaseOwnerID uuid.UUID
	for _, col := range showcase.Collaborators {
		if col.Role == entity.CollaborationRoleOwner {
			showcaseOwnerID = col.UserID
			break
		}
	}

	go func() {
		// Determine Event Type and Payload
		var routingKey string
		var eventData map[string]interface{}
		
		if comment.ParentID != nil {
			// It is a REPLY
			routingKey = "comment.replied"
			
			// We need Parent Author ID. We fetched ParentComment in Step 3.
			// But variables in Step 3 are scoped. We need to fetch Parent again or restructure.
			// Re-fetching parent for event safety or use closure if refactoring.
			// Since Step 3 logic is inside `if`, we can't easily access `parentComment` here unless we declare it outside.
			// I will fetch parent again inside this goroutine or check if I can modify Step 3 scope.
			
			parent, err := u.repo.GetCommentByID(*comment.ParentID)
			if err == nil && parent != nil {
				eventData = map[string]interface{}{
					"reply_id":          comment.ID,
					"parent_id":         *comment.ParentID,
					"showcase_id":       showcaseID,
					"actor_id":          userID,
					"target_user_id":    parent.UserID, // Parent Author
					"showcase_owner_id": showcaseOwnerID,
					"content":           input.Content,
				}
			}
		} else {
			// It is a DIRECT COMMENT
			routingKey = "comment.created"
			eventData = map[string]interface{}{
				"comment_id":     comment.ID,
				"showcase_id":    showcaseID,
				"content":        input.Content,
				"actor_id":       userID,
				"target_user_id": showcaseOwnerID, // Showcase Author
			}
		}

		if eventData != nil {
			if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
				u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
			} else {
				u.log.Debugf("Published %s event for comment %s", routingKey, comment.ID)
			}
		}
	}()

	return comment, nil
}

func (u *showcaseUsecase) ReplyComment(ctx context.Context, parentID uuid.UUID, input *entity.ReplyCommentRequest, userID uuid.UUID) (*entity.Comment, error) {
	// 1. Get Parent Comment to find ShowcaseID
	parentComment, err := u.repo.GetCommentByID(parentID)
	if err != nil {
		return nil, err
	}
	if parentComment == nil {
		return nil, errors.New("parent comment not found")
	}

	// 2. Reuse CreateComment Logic
	// Transform ReplyCommentRequest to CreateCommentRequest
	parentIDStr := parentID.String()
	createInput := &entity.CreateCommentRequest{
		Content:  input.Content,
		ParentID: &parentIDStr,
	}

	return u.CreateComment(ctx, parentComment.ShowcaseID, createInput, userID)
}

func (u *showcaseUsecase) GetShowcaseComments(ctx context.Context, showcaseID uuid.UUID, limit int, cursor string) (map[string]interface{}, error) {
	// 1. Get Parents and Replies from Repo
	parents, replies, meta, err := u.repo.GetCommentsByShowcaseID(showcaseID, limit, cursor)
	if err != nil {
		u.log.Errorf("Failed to fetch comments for showcase %s: %v", showcaseID, err)
		return nil, errors.New("failed to fetch comments")
	}

	// 2. Collect User IDs for Bulk Fetch
	userIDsMap := make(map[uuid.UUID]bool)
	for _, p := range parents {
		userIDsMap[p.UserID] = true
	}
	for _, r := range replies {
		userIDsMap[r.UserID] = true
	}

	var userIDs []uuid.UUID
	for uid := range userIDsMap {
		userIDs = append(userIDs, uid)
	}

	// 3. Bulk Fetch Users
	usersMap := make(map[uuid.UUID]clients.UserDetail)
	if len(userIDs) > 0 {
		fetchedUsers, err := u.userClient.GetUsersBulk(userIDs)
		if err == nil {
			usersMap = fetchedUsers
		} else {
			u.log.Warnf("Failed to fetch users bulk: %v", err)
		}
	}

	// 4. Construct Nested Structure & Populate Author
	// Helper to populate user
	populateUser := func(c *entity.Comment) {
		// If deleted and we are here (meaning it has children), mask it
		if c.DeletedAt.Valid {
			c.Body = "[Comment deleted]"
			c.Author = &entity.User{
				Name:     "[Unknown]",
				Username: "unknown",
			}
			return
		}

		if detail, ok := usersMap[c.UserID]; ok {
			c.Author = &entity.User{
				ID:        detail.ID,
				Name:      detail.Name,
				Username:  detail.Username,
				AvatarURL: detail.AvatarURL,
			}
		}
	}

	// Group replies by ParentID
	repliesMap := make(map[uuid.UUID][]entity.Comment)
	for i := range replies {
		// Populate user first, masking will be handled during pruning/traversal if needed
		// But wait, pruning happens after tree construction.
		if replies[i].ParentID != nil {
			pid := *replies[i].ParentID
			repliesMap[pid] = append(repliesMap[pid], replies[i])
		}
	}

	// Attach replies to parents (Build Full Tree)
	for i := range parents {
		if r, ok := repliesMap[parents[i].ID]; ok {
			parents[i].Replies = r
		} else {
			parents[i].Replies = []entity.Comment{}
		}
	}

	// 5. Recursive Pruning (Post-Order Traversal)
	// Logic:
	// - Active: Keep.
	// - Deleted: Keep ONLY IF visible children exist. Else prune.
	var prune func(comments []entity.Comment) []entity.Comment
	prune = func(comments []entity.Comment) []entity.Comment {
		var filtered []entity.Comment
		for i := range comments {
			// Recurse first (Post-order)
			if len(comments[i].Replies) > 0 {
				comments[i].Replies = prune(comments[i].Replies)
			}

			// Check visibility
			if !comments[i].DeletedAt.Valid {
				// Active -> Visible
				populateUser(&comments[i])
				filtered = append(filtered, comments[i])
			} else {
				// Deleted
				if len(comments[i].Replies) > 0 {
					// Has visible children -> Tombstone
					populateUser(&comments[i])
					filtered = append(filtered, comments[i])
				} else {
					// No visible children -> Hide (Prune)
				}
			}
		}
		return filtered
	}

	finalComments := prune(parents)

	return map[string]interface{}{
		"comments":   finalComments,
		"pagination": meta,
	}, nil
}

func (u *showcaseUsecase) UpdateComment(ctx context.Context, id uuid.UUID, input *entity.UpdateCommentRequest, userID uuid.UUID) (*entity.Comment, error) {
	// 1. Get Comment
	comment, err := u.repo.GetCommentByID(id)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, errors.New("comment not found")
	}

	// 2. Check Ownership
	if comment.UserID != userID {
		return nil, errors.New("forbidden: only owner can update comment")
	}

	// 3. Update Content
	comment.Body = input.Content
	// You might want to update UpdatedAt explicitly if GORM doesn't handle it automatically on Save with struct
	// but GORM usually handles time.Time fields update on Save if they are not zero.
	// But let's rely on GORM's auto update for now or repo implementation.

	// 3. Update Content
	comment.Body = input.Content
	comment.IsEdited = true

	if err := u.repo.UpdateComment(comment); err != nil {
		u.log.Errorf("Failed to update comment %s: %v", id, err)
		return nil, errors.New("failed to update comment")
	}

	// 4. Populate Author (Consistency)
	usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{userID})
	if err == nil {
		if userDetail, ok := usersMap[userID]; ok {
			comment.Author = &entity.User{
				ID:        userDetail.ID,
				Name:      userDetail.Name,
				Username:  userDetail.Username,
				AvatarURL: userDetail.AvatarURL,
			}
		}
	} else {
		u.log.Warnf("Failed to fetch author info for updated comment: %v", err)
	}

	return comment, nil
}

func (u *showcaseUsecase) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// 1. Get Comment
	comment, err := u.repo.GetCommentByID(id)
	if err != nil {
		return err
	}
	if comment == nil {
		return errors.New("comment not found")
	}

	// 2. Check Ownership
	if comment.UserID != userID {
		return errors.New("forbidden: only owner can delete comment")
	}

	// 3. Delete
	if err := u.repo.DeleteComment(id); err != nil {
		u.log.Errorf("Failed to delete comment %s: %v", id, err)
		return errors.New("failed to delete comment")
	}

	// 4. Publish Event (Async, Fail-safe)
	go func() {
		routingKey := "comment.deleted"
		eventData := map[string]interface{}{
			"comment_id":     id,
			"showcase_id":    comment.ShowcaseID,
			"user_id":        comment.UserID,
		}

		if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
			u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
		} else {
			u.log.Debugf("Published %s event for comment %s", routingKey, id)
		}
	}()

	return nil
}

func (u *showcaseUsecase) ToggleCommentLike(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) (bool, int, error) {
	// 1. Verify comment exists
	comment, err := u.repo.GetCommentByID(commentID)
	if err != nil {
		return false, 0, err
	}
	if comment == nil {
		return false, 0, errors.New("comment not found")
	}

	// 2. Toggle like
	isLiked, likesCount, err := u.repo.ToggleCommentLike(userID, commentID)
	if err != nil {
		u.log.Errorf("Failed to toggle like on comment %s: %v", commentID, err)
		return false, 0, errors.New("failed to toggle like")
	}

	// 3. Publish Event (Async, Fail-safe)
	go func() {
		routingKey := "comment.unliked"
		if isLiked {
			routingKey = "comment.liked"
		}

		eventData := map[string]interface{}{
			"comment_id":     commentID,
			"showcase_id":    comment.ShowcaseID,
			"actor_id":       userID,
			"target_user_id": comment.UserID,
		}

		if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
			u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
		} else {
			u.log.Debugf("Published %s event for comment %s", routingKey, commentID)
		}
	}()

	return isLiked, likesCount, nil
}

func (u *showcaseUsecase) DeleteShowcase(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// 1. Get Showcase
	showcase, err := u.repo.GetByID(id)
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
	if err := u.repo.DeleteShowcase(id); err != nil {
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

func (u *showcaseUsecase) RemoveCollaborator(ctx context.Context, showcaseID uuid.UUID, targetUserID uuid.UUID, actorUserID uuid.UUID) error {
	// 1. Get Showcase with Collaborators
	showcase, err := u.repo.GetByID(showcaseID)
	if err != nil {
		return err
	}
	if showcase == nil {
		return errors.New("showcase not found")
	}

	// 2. Find Logic Actors
	var actorCol *entity.Collaborator
	var targetCol *entity.Collaborator
	
	for i := range showcase.Collaborators {
		if showcase.Collaborators[i].UserID == actorUserID {
			actorCol = &showcase.Collaborators[i]
		}
		if showcase.Collaborators[i].UserID == targetUserID {
			targetCol = &showcase.Collaborators[i]
		}
	}

	if actorCol == nil {
		return errors.New("forbidden: actor is not a collaborator")
	}
	// Case: Kick (target must be part of project to be kicked)
	// Or Leave (target == actor, found)
	// BUT invalidation: if deleting non-existing member? Repo delete handles "not found" silently usuallly, but business rule should return "user not in project" if strict.
	// For idempotency or UX, let's verify target existence.
	if targetCol == nil {
		return errors.New("target user is not a collaborator")
	}

	// 3. Logic Branching
	isSelfAction := (targetUserID == actorUserID)
	var finalOwnerID uuid.UUID
	
	// Identify current owner
	for _, c := range showcase.Collaborators {
		if c.Role == entity.CollaborationRoleOwner {
			finalOwnerID = c.UserID
			break
		}
	}

	// Scenario A: LEAVE (Self Action)
	if isSelfAction {
		// If NOT owner -> Just leave
		if actorCol.Role != entity.CollaborationRoleOwner {
			if err := u.repo.DeleteCollaborator(showcaseID, actorUserID); err != nil {
				return err
			}
		} else {
			// If OWNER -> Check successors
			var successors []entity.Collaborator
			for _, c := range showcase.Collaborators {
				if c.UserID != actorUserID && c.Status == entity.CollaborationStatusAccepted {
					successors = append(successors, c)
				}
			}

			if len(successors) == 0 {
				return errors.New("forbidden: sole owner cannot leave. please delete the showcase instead")
			}

			// Find Oldest Successor
			oldest := successors[0]
			for _, s := range successors {
				if s.CreatedAt.Before(oldest.CreatedAt) {
					oldest = s
				}
			}

			// Transfer Ownership
			if err := u.repo.UpdateCollaboratorRole(showcaseID, oldest.UserID, entity.CollaborationRoleOwner); err != nil {
				return err
			}
			finalOwnerID = oldest.UserID // Owner changed

			// Delete Self
			if err := u.repo.DeleteCollaborator(showcaseID, actorUserID); err != nil {
				return err
			}
		}
	} else {
		// Scenario B: KICK (Actor != Target)
		if actorCol.Role != entity.CollaborationRoleOwner {
			return errors.New("forbidden: only owner can remove members")
		}
		if targetCol.Role == entity.CollaborationRoleOwner {
			return errors.New("forbidden: cannot kick another owner")
		}

		if err := u.repo.DeleteCollaborator(showcaseID, targetUserID); err != nil {
			return err
		}
	}

	// 4. Publish Event (Async, Fail-safe)
	go func() {
		routingKey := "collaborator.removed"
		eventData := map[string]interface{}{
			"showcase_id":       showcaseID,
			"showcase_title":    showcase.Title,
			"actor_id":          actorUserID,
			"target_user_id":    targetUserID,
			"is_self_removal":   isSelfAction,
			"showcase_owner_id": finalOwnerID,
		}

		if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
			u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
		} else {
			u.log.Debugf("Published %s event for user %s", routingKey, targetUserID)
		}
	}()

	return nil
}

func (u *showcaseUsecase) GetCollaborators(ctx context.Context, showcaseID uuid.UUID, userID uuid.UUID) ([]entity.Collaborator, error) {
	// 1. Fetch Collaborators
	collaborators, err := u.repo.GetCollaboratorsByShowcaseID(showcaseID)
	if err != nil {
		return nil, err
	}

	// 2. Access Control: Check if requester is a member (Accepted or Pending? User said "Only member", implies accepted/pending is fine as long as they are related to project. Usually Pending users shouldn't see details, but maybe they should see who invited them? Safer to assume ANY record in collaborators table = access)
	isMember := false
	for _, c := range collaborators {
		if c.UserID == userID {
			isMember = true
			break
		}
	}

	if !isMember {
		return nil, errors.New("forbidden: you are not a member of this project")
	}

	return collaborators, nil
}

func (u *showcaseUsecase) DeleteInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID) error {
	// 1. Get Collaborator (Invitation)
	invitation, err := u.repo.GetCollaboratorByID(id)
	if err != nil {
		return err // Returns error if not found
	}
	// Note: If repo returns error on not found, we handle it. If Gorm returns ErrRecordNotFound, we should probably wrap it or handler checks it.

	// 2. Validate Status
	if invitation.Status != entity.CollaborationStatusPending {
		return errors.New("cannot delete processed invitation. use remove collaborator instead")
	}

	// 3. Get Showcase for Owner Check
	showcase, err := u.repo.GetByID(invitation.ShowcaseID)
	if err != nil {
		return err
	}
	if showcase == nil {
		return errors.New("showcase not found")
	}

	// 4. Check if Actor is Owner
	isOwner := false
	for _, c := range showcase.Collaborators {
		if c.UserID == actorUserID && c.Role == entity.CollaborationRoleOwner {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return errors.New("forbidden: only owner can cancel invitations")
	}

	// 5. Delete
	return u.repo.DeleteCollaboratorByID(id)
}

func (u *showcaseUsecase) GetPendingInvitations(ctx context.Context, userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *repository.PaginationMeta, error) {
	// Call repo
	invitations, meta, err := u.repo.GetPendingInvitations(userID, limit, cursor)
	if err != nil {
		return nil, nil, err
	}
	
	// Since repo returns *repository.PaginationMeta, check if casting or direct return works.
	// Logic above signature changed to *repository.PaginationMeta.
	// So direct return should work.
	
	return invitations, meta, nil
}

func (u *showcaseUsecase) InviteCollaborators(ctx context.Context, showcaseID uuid.UUID, usernames []string, actorUserID uuid.UUID) ([]string, error) {
	// 1. Get Showcase & Verify Owner
	showcase, err := u.repo.GetByID(showcaseID)
	if err != nil {
		return nil, err
	}
	if showcase == nil {
		return nil, errors.New("showcase not found")
	}

	// Verify Owner
	// Need to check collaborators. Repo preloaded "Collaborators" inside GetByID?
	// Let's check GetByID implementation. Assuming yes, or separate fetch.
	// Step 4571 view showed GetByID Preloads.
	isOwner := false
	for _, c := range showcase.Collaborators {
		if c.UserID == actorUserID && c.Role == entity.CollaborationRoleOwner {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return nil, errors.New("forbidden: only owner can invite")
	}

	// 2. Resolve Usernames
	usersMap, err := u.userClient.GetUsersByUsernames(usernames)
	if err != nil {
		return nil, fmt.Errorf("failed to resolving users: %w", err)
	}

	// 3. Prepare List & Check Duplicates
	// Need existing collaborators to avoid re-inviting or inviting existing members
	// Using showcase.Collaborators is enough if it contains all members.
	existingMembers := make(map[uuid.UUID]bool)
	for _, c := range showcase.Collaborators {
		existingMembers[c.UserID] = true // Includes pending etc? Yes.
	}

	var newCollaborators []entity.Collaborator
	var invitedUsernames []string

	for _, username := range usernames {
		user, found := usersMap[username]
		if !found {
			return nil, fmt.Errorf("user not found: %s", username)
		}

		if existingMembers[user.ID] {
			// Skip or Error? Docs: "Validasi gagal (misal ... sudah diundang)".
			// Assuming Error for strictness, or Skip for idempotency.
			// Let's return Error if any user is duplicate? Or just skip?
			// Docs 400 suggested "Validasi gagal".
			return nil, fmt.Errorf("user %s is already a collaborator or invited", username)
		}

		// Prepare Struct
		newCol := entity.Collaborator{
			ShowcaseID: showcaseID,
			UserID:     user.ID,
			Role:       entity.CollaborationRoleCollaborator,
			Status:     entity.CollaborationStatusPending,
			ExpiredAt:  time.Now().Add(7 * 24 * time.Hour), // 7 days
		}
		// Explicitly generate ID for event usage
		newCol.ID = uuid.New()
		
		newCollaborators = append(newCollaborators, newCol)
		invitedUsernames = append(invitedUsernames, username)
	}

	if len(newCollaborators) == 0 {
		return []string{}, nil // Nothing to add
	}

	// 4. Save
	if err := u.repo.AddCollaborators(newCollaborators); err != nil {
		return nil, err
	}

	// 5. Publish Events (Async, Loop)
	go func() {
		// Fetch Inviter Info
		inviterUsername := ""
		inviterMap, err := u.userClient.GetUsersBulk([]uuid.UUID{actorUserID})
		if err == nil {
			if inviter, ok := inviterMap[actorUserID]; ok {
				inviterUsername = inviter.Username
			}
		} else {
			u.log.Warnf("Failed to fetch inviter info for event: %v", err)
		}

		for _, col := range newCollaborators {
			routingKey := "collaborator.invited"
			eventData := map[string]interface{}{
				"invitation_id":    col.ID,
				"showcase_id":      showcaseID,
				"showcase_title":   showcase.Title,
				"inviter_id":       actorUserID,
				"inviter_username": inviterUsername,
				"target_user_id":   col.UserID,
			}

			if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
				u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
			} else {
				u.log.Debugf("Published %s event for user %s", routingKey, col.UserID)
			}
		}
	}()

	return invitedUsernames, nil
}

func (u *showcaseUsecase) RespondInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID, response string) error {
	// 1. Validate Input
	if response != entity.CollaborationStatusAccepted && response != entity.CollaborationStatusRejected {
		return errors.New("invalid response value. must be ACCEPTED or REJECTED")
	}

	// 2. Get Collaborator
	invitation, err := u.repo.GetCollaboratorByID(id)
	if err != nil {
		return err
	}
	// Gorm returns error if not found? No, my GetCollaboratorByID returns err. 
	// But previously I said "returns error if not found" in comment.
	// Actually `db.First` returns ErrRecordNotFound.
	// So err implies not found or db error.
	
	// 3. Validate Logic
	if invitation.UserID != actorUserID {
		return errors.New("forbidden: this invitation is not for you")
	}

	if invitation.Status != entity.CollaborationStatusPending {
		return errors.New("invitation is no longer pending")
	}

	// 4. Update
	if err := u.repo.UpdateCollaboratorStatus(id, response); err != nil {
		return err
	}

	// 5. Publish Event (Async, Fail-safe)
	// ONLY if Accepted
	if response == entity.CollaborationStatusAccepted {
		// Need Showcase Title, Owner ID, Responder Username
		go func() {
			// Fetch Showcase for Title & Owner
			showcase, err := u.repo.GetByID(invitation.ShowcaseID)
			if err != nil || showcase == nil {
				u.log.Warnf("Failed to fetch showcase for RespondInvitation event: %v", err)
				return
			}

			var ownerID uuid.UUID
			for _, c := range showcase.Collaborators {
				if c.Role == entity.CollaborationRoleOwner {
					ownerID = c.UserID
					break
				}
			}

			// Fetch Responder Username & Full Name
			responderUsername := ""
			responderFullName := ""
			responderAvatar := ""
			usersMap, err := u.userClient.GetUsersBulk([]uuid.UUID{actorUserID})
			if err == nil {
				if user, ok := usersMap[actorUserID]; ok {
					responderUsername = user.Username
					responderFullName = user.Name
					responderAvatar = user.AvatarURL
				}
			} else {
				u.log.Warnf("Failed to fetch responder info: %v", err)
			}

			routingKey := "collaborator.responded"
			eventData := map[string]interface{}{
				"invitation_id":       id,
				"showcase_id":         invitation.ShowcaseID,
				"showcase_title":      showcase.Title,
				"response_status":     response,
				"responder_id":        actorUserID,
				"responder_username":  responderUsername,
				"responder_full_name": responderFullName,
				"responder_avatar_url": responderAvatar,
				"target_user_id":      ownerID,
			}

			if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
				u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
			} else {
				u.log.Debugf("Published %s event for invitation %s", routingKey, id)
			}
		}()
	}

	return nil
}

func (u *showcaseUsecase) GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error) {
	// 1. Get from Repo
	var showcase *entity.Showcase
	var err error

	// Check if input is UUID
	if id, parseErr := uuid.Parse(slug); parseErr == nil {
		showcase, err = u.repo.GetByID(id)
	} else {
		showcase, err = u.repo.GetBySlug(slug)
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

func (u *showcaseUsecase) SearchShowcases(ctx context.Context, query string, page int, limit int) (map[string]interface{}, error) {
	showcases, total, err := u.searchRepo.SearchShowcases(ctx, query, page, limit)
	if err != nil {
		u.log.Errorf("Failed to search showcases: %v", err)
		return nil, err
	}

	// Transform Response: Flatten Collaborators to list of Users
	var transformedData []map[string]interface{}
	for _, s := range showcases {
		// Convert to map to preserve all fields
		var item map[string]interface{}
		// Efficient enough for paginated view
		if b, err := json.Marshal(s); err == nil {
			_ = json.Unmarshal(b, &item)
		}

		// Extract Users from Collaborators
		users := make([]*entity.User, 0, len(s.EnrichedCollaborators))
		for _, ec := range s.EnrichedCollaborators {
			if ec.User != nil {
				users = append(users, ec.User)
			}
		}

		// Overwrite "collaborators" with list of Users
		item["collaborators"] = users
		transformedData = append(transformedData, item)
	}

	return map[string]interface{}{
		"data": transformedData,
		"meta": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}, nil
}
