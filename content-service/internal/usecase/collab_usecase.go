package usecase

import (
	"context"
	"errors"
	"fmt"
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

type CollabUsecase interface {
	RemoveCollaborator(ctx context.Context, showcaseID uuid.UUID, targetUserID uuid.UUID, actorUserID uuid.UUID) error
	GetCollaborators(ctx context.Context, showcaseID uuid.UUID, userID uuid.UUID) ([]entity.Collaborator, error)
	DeleteInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID) error
	GetPendingInvitations(ctx context.Context, userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *entity.PaginationMeta, error)
	InviteCollaborators(ctx context.Context, showcaseID uuid.UUID, usernames []string, actorUserID uuid.UUID) ([]string, error)
	RespondInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID, response string) error
}

type collabUsecase struct {
	showcaseRepo   repository.ShowcaseRepository
	collabRepo     repository.CollabRepository
	searchRepo     repository.SearchRepository
	userClient     clients.UserClient
	mediaUploader  media.MediaUploader
	eventPublisher rabbitmq.EventPublisher
	redisClient    *redis.Client
	cfg            *config.Config
	log            *logrus.Logger
	validate       *validator.Validate
}

func NewCollabUsecase(
	showcaseRepo repository.ShowcaseRepository,
	collabRepo repository.CollabRepository,
	searchRepo repository.SearchRepository,
	userClient clients.UserClient,
	mediaUploader media.MediaUploader,
	eventPublisher rabbitmq.EventPublisher,
	redisClient *redis.Client,
	cfg *config.Config,
	log *logrus.Logger,
) CollabUsecase {
	return &collabUsecase{
		showcaseRepo:   showcaseRepo,
		collabRepo:     collabRepo,
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

func (u *collabUsecase) RemoveCollaborator(ctx context.Context, showcaseID uuid.UUID, targetUserID uuid.UUID, actorUserID uuid.UUID) error {
	// 1. Get Showcase with Collaborators
	showcase, err := u.showcaseRepo.GetByID(showcaseID)
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
			if err := u.collabRepo.DeleteCollaborator(showcaseID, actorUserID); err != nil {
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
			if err := u.collabRepo.UpdateCollaboratorRole(showcaseID, oldest.UserID, entity.CollaborationRoleOwner); err != nil {
				return err
			}
			finalOwnerID = oldest.UserID // Owner changed

			// Delete Self
			if err := u.collabRepo.DeleteCollaborator(showcaseID, actorUserID); err != nil {
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

		if err := u.collabRepo.DeleteCollaborator(showcaseID, targetUserID); err != nil {
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

func (u *collabUsecase) GetCollaborators(ctx context.Context, showcaseID uuid.UUID, userID uuid.UUID) ([]entity.Collaborator, error) {
	// 1. Fetch Collaborators
	collaborators, err := u.collabRepo.GetCollaboratorsByShowcaseID(showcaseID)
	if err != nil {
		return nil, err
	}

	// 2. Access Control & collect userIDs in one loop
	isMember := false
	userIDs := make([]uuid.UUID, 0, len(collaborators))
	for _, c := range collaborators {
		if c.UserID == userID {
			isMember = true
		}
		userIDs = append(userIDs, c.UserID)
	}
	if !isMember {
		return nil, errors.New("forbidden: you are not a member of this project")
	}

	usersMap, err := u.userClient.GetUsersBulk(userIDs)
	if err != nil {
		u.log.Warnf("Failed to fetch user data for collaborators of showcase %s: %v", showcaseID, err)
		return nil, errors.New("failed to fetch collaborator user data")
	}

	// Map user details to collaborators
	for i, c := range collaborators {
		if userData, found := usersMap[c.UserID]; found {
			if collaborators[i].User == nil {
				collaborators[i].User = &entity.User{}
			}
			collaborators[i].User.ID = userData.ID
			collaborators[i].User.Username = userData.Username
			collaborators[i].User.Name = userData.Name
			collaborators[i].User.AvatarURL = userData.AvatarURL
		}
	}

	return collaborators, nil
}

func (u *collabUsecase) DeleteInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID) error {
	// 1. Get Collaborator (Invitation)
	invitation, err := u.collabRepo.GetCollaboratorByID(id)
	if err != nil {
		return err // Returns error if not found
	}
	// Note: If collabRepo returns error on not found, we handle it. If Gorm returns ErrRecordNotFound, we should probably wrap it or handler checks it.

	// 2. Validate Status
	if invitation.Status != entity.CollaborationStatusPending {
		return errors.New("cannot delete processed invitation. use remove collaborator instead")
	}

	// 3. Get Showcase for Owner Check
	showcase, err := u.showcaseRepo.GetByID(invitation.ShowcaseID)
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
	return u.collabRepo.DeleteCollaboratorByID(id)
}

func (u *collabUsecase) GetPendingInvitations(ctx context.Context, userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *entity.PaginationMeta, error) {
	// Call collabRepo
	invitations, meta, err := u.collabRepo.GetPendingInvitations(userID, limit, cursor)
	if err != nil {
		return nil, nil, err
	}

	// Since collabRepo returns *repository.PaginationMeta, check if casting or direct return works.
	// Logic above signature changed to *repository.PaginationMeta.
	// So direct return should work.

	return invitations, meta, nil
}

func (u *collabUsecase) InviteCollaborators(ctx context.Context, showcaseID uuid.UUID, usernames []string, actorUserID uuid.UUID) ([]string, error) {
	// 1. Get Showcase & Verify Owner
	showcase, err := u.showcaseRepo.GetByID(showcaseID)
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
	if err := u.collabRepo.AddCollaborators(newCollaborators); err != nil {
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

func (u *collabUsecase) RespondInvitation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID, response string) error {
	// 1. Validate Input
	if response != entity.CollaborationStatusAccepted && response != entity.CollaborationStatusRejected {
		return errors.New("invalid response value. must be ACCEPTED or REJECTED")
	}

	// 2. Get Collaborator
	invitation, err := u.collabRepo.GetCollaboratorByID(id)
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
	if err := u.collabRepo.UpdateCollaboratorStatus(id, response); err != nil {
		return err
	}

	// 5. Publish Event (Async, Fail-safe)
	// ONLY if Accepted
	if response == entity.CollaborationStatusAccepted {
		// Need Showcase Title, Owner ID, Responder Username
		go func() {
			// Fetch Showcase for Title & Owner
			showcase, err := u.showcaseRepo.GetByID(invitation.ShowcaseID)
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
				"invitation_id":        id,
				"showcase_id":          invitation.ShowcaseID,
				"showcase_title":       showcase.Title,
				"response_status":      response,
				"responder_id":         actorUserID,
				"responder_username":   responderUsername,
				"responder_full_name":  responderFullName,
				"responder_avatar_url": responderAvatar,
				"target_user_id":       ownerID,
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

// GetShowcaseByID fetches showcase by ID only (lightweight, for internal API)
func (u *collabUsecase) GetShowcaseByID(ctx context.Context, id uuid.UUID) (*entity.Showcase, error) {
	showcase, err := u.showcaseRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return showcase, nil
}

func (u *collabUsecase) GetShowcaseBySlug(ctx context.Context, slug string) (*entity.Showcase, error) {
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
		if err := u.collabRepo.IncrementViewCount(showcase.ID); err != nil {
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

func (u *collabUsecase) GetShowcasesByUser(ctx context.Context, userIDStr string, limit int, cursor string) (map[string]interface{}, error) {
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

func (u *collabUsecase) GetMyShowcases(ctx context.Context, userID uuid.UUID, limit int, cursor string) (map[string]interface{}, error) {
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
