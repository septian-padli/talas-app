package usecase

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/repository"
	"github.com/septianpadli/talas/content-service/pkg/clients"
	"github.com/septianpadli/talas/content-service/pkg/rabbitmq"
	"github.com/sirupsen/logrus"
)

type CommentUsecase interface {
	CreateComment(ctx context.Context, commentID uuid.UUID, input *entity.CreateCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	ReplyComment(ctx context.Context, parentID uuid.UUID, input *entity.ReplyCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	GetShowcaseComments(ctx context.Context, commentID uuid.UUID, limit int, cursor string) (map[string]interface{}, error)
	UpdateComment(ctx context.Context, id uuid.UUID, input *entity.UpdateCommentRequest, userID uuid.UUID) (*entity.Comment, error)
	DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ToggleCommentLike(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) (bool, int, error)
}

type commentUsecase struct {
	repo           repository.CommentRepository
	searchRepo     repository.SearchRepository
	showcaseRepo   repository.ShowcaseRepository
	userClient     clients.UserClient
	eventPublisher rabbitmq.EventPublisher
	cfg            *config.Config
	log            *logrus.Logger
	validate       *validator.Validate
}

func NewCommentUsecase(
	repo repository.CommentRepository,
	searchRepo repository.SearchRepository,
	showcaseRepo repository.ShowcaseRepository,
	userClient clients.UserClient,
	eventPublisher rabbitmq.EventPublisher,
	cfg *config.Config,
	log *logrus.Logger,
) CommentUsecase {
	return &commentUsecase{
		repo:           repo,
		searchRepo:     searchRepo,
		showcaseRepo:   showcaseRepo,
		userClient:     userClient,
		eventPublisher: eventPublisher,
		cfg:            cfg,
		log:            log,
		validate:       validator.New(),
	}
}

func (u *commentUsecase) CreateComment(ctx context.Context, commentID uuid.UUID, input *entity.CreateCommentRequest, userID uuid.UUID) (*entity.Comment, error) {
	// 1. Validate Input
	if len(input.Content) < 1 || len(input.Content) > 1000 {
		return nil, errors.New("content must be between 1 and 1000 characters")
	}

	// 2. Check Showcase
	showcase, err := u.showcaseRepo.GetByID(commentID)
	if err != nil {
		return nil, err
	}
	if showcase == nil {
		return nil, errors.New("showcase not found")
	}

	comment := &entity.Comment{
		ShowcaseID: commentID,
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
		if parentComment.ShowcaseID != commentID {
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
					"showcase_id":       commentID,
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
				"showcase_id":    commentID,
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

func (u *commentUsecase) ReplyComment(ctx context.Context, parentID uuid.UUID, input *entity.ReplyCommentRequest, userID uuid.UUID) (*entity.Comment, error) {
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

func (u *commentUsecase) GetShowcaseComments(ctx context.Context, commentID uuid.UUID, limit int, cursor string) (map[string]interface{}, error) {
	// 1. Get Parents and Replies from Repo
	parents, replies, meta, err := u.repo.GetCommentsByShowcaseID(commentID, limit, cursor)
	if err != nil {
		u.log.Errorf("Failed to fetch comments for showcase %s: %v", commentID, err)
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

func (u *commentUsecase) UpdateComment(ctx context.Context, id uuid.UUID, input *entity.UpdateCommentRequest, userID uuid.UUID) (*entity.Comment, error) {
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

func (u *commentUsecase) DeleteComment(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
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
			"comment_id":  id,
			"showcase_id": comment.ShowcaseID,
			"user_id":     comment.UserID,
		}

		if err := u.eventPublisher.Publish(context.Background(), routingKey, eventData); err != nil {
			u.log.Errorf("Failed to publish %s event: %v", routingKey, err)
		} else {
			u.log.Debugf("Published %s event for comment %s", routingKey, id)
		}
	}()

	return nil
}

func (u *commentUsecase) ToggleCommentLike(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) (bool, int, error) {
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
