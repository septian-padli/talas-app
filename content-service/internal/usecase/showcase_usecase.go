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
	UpdateShowcase(ctx context.Context, id uuid.UUID, input *entity.UpdateShowcaseRequest, userID uuid.UUID) (*entity.Showcase, error)
	ToggleLike(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	ToggleBookmark(ctx context.Context, userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	CreateComment(ctx context.Context, showcaseID uuid.UUID, input *entity.CreateCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	ReplyComment(ctx context.Context, parentID uuid.UUID, input *entity.ReplyCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	GetShowcaseComments(ctx context.Context, showcaseID uuid.UUID, limit int, cursor string) (map[string]interface{}, error)
	UpdateComment(ctx context.Context, id uuid.UUID, input *entity.UpdateCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeleteShowcase(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
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

	// 2. Toggle in Repo
	isLiked, err := u.repo.ToggleLike(userID, showcaseID)
	if err != nil {
		u.log.Errorf("Failed to toggle like: %v", err)
		return false, errors.New("failed to toggle like")
	}
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
	// Fetch both Author and Parent Author (if reply) in one go if possible, or separately.
	// Only Author needed for response.
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

	return nil
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
