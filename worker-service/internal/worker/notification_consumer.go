package worker

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/septian/worker-service/internal/domain"
	"github.com/septian/worker-service/internal/repository"
	"github.com/septian/worker-service/pkg/gateway"
	"github.com/sirupsen/logrus"
)

const (
	NotificationQueueName = "notification_queue"
)

// NotificationConsumer processes events and creates notifications
type NotificationConsumer struct {
	channel *amqp.Channel
	repo    repository.NotificationRepository
	gateway *gateway.InternalGateway
	log     *logrus.Logger
}

// NewNotificationConsumer creates a new notification consumer
func NewNotificationConsumer(
	channel *amqp.Channel,
	repo repository.NotificationRepository,
	gateway *gateway.InternalGateway,
	log *logrus.Logger,
) *NotificationConsumer {
	return &NotificationConsumer{
		channel: channel,
		repo:    repo,
		gateway: gateway,
		log:     log,
	}
}

// Setup declares queue and bindings for notification events
func (c *NotificationConsumer) Setup() error {
	// Declare Queue (Durable)
	_, err := c.channel.QueueDeclare(
		NotificationQueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}
	c.log.Infof("Declared queue: %s", NotificationQueueName)

	// Bind to routing keys
	routingKeys := []string{
		"showcase.liked",
		"comment.created",
		"user.followed",
		"collaborator.responded",
	}

	for _, key := range routingKeys {
		err = c.channel.QueueBind(NotificationQueueName, key, ExchangeName, false, nil)
		if err != nil {
			return err
		}
		c.log.Infof("Bound queue %s to routing key: %s", NotificationQueueName, key)
	}

	return nil
}

// Start begins consuming notification events
func (c *NotificationConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		NotificationQueueName,
		"",    // consumer tag
		false, // auto-ack (manual for reliability)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	c.log.Info("Notification Consumer started")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Notification Consumer stopping...")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			if err := c.processMessage(ctx, msg); err != nil {
				c.log.Errorf("Failed to process message: %v", err)
				msg.Nack(false, true) // Requeue on error
			} else {
				msg.Ack(false)
			}
		}
	}
}

// processMessage handles incoming event messages
func (c *NotificationConsumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	// Parse envelope
	var envelope domain.EventEnvelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		c.log.Errorf("Failed to unmarshal envelope: %v", err)
		return err
	}

	c.log.Infof("Processing event: %s (ID: %s)", envelope.EventType, envelope.EventID)

	// Route to specific handler
	switch envelope.EventType {
	case "showcase.liked":
		return c.handleShowcaseLiked(ctx, envelope)
	case "comment.created":
		return c.handleCommentCreated(ctx, envelope)
	case "user.followed":
		return c.handleUserFollowed(ctx, envelope)
	case "collaborator.responded":
		return c.handleCollaboratorResponded(ctx, envelope)
	default:
		c.log.Warnf("Unknown event type: %s", envelope.EventType)
		return nil // Ack unknown events
	}
}

// handleShowcaseLiked processes showcase like events
func (c *NotificationConsumer) handleShowcaseLiked(ctx context.Context, envelope domain.EventEnvelope) error {
	// Parse payload
	var payload struct {
		ShowcaseID   string `json:"showcase_id"`
		ActorID      string `json:"actor_id"`
		TargetUserID string `json:"target_user_id"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Narcissist check
	if payload.ActorID == payload.TargetUserID {
		c.log.Debug("Skipping self-like notification")
		return nil
	}

	// Fetch actor info
	users, err := c.gateway.GetUsersByIDs(ctx, []string{payload.ActorID})
	if err != nil {
		c.log.Errorf("Failed to fetch user data: %v", err)
		return err
	}
	actor, ok := users[payload.ActorID]
	if !ok {
		c.log.Warnf("Actor not found: %s", payload.ActorID)
		return nil
	}

	// Fetch showcase info
	showcase, err := c.gateway.GetShowcaseByID(ctx, payload.ShowcaseID)
	if err != nil {
		c.log.Errorf("Failed to fetch showcase: %v", err)
		return err
	}
	if showcase == nil {
		c.log.Warnf("Showcase not found: %s", payload.ShowcaseID)
		return nil
	}

	// Build JSON snapshot for Data field
	notificationData := map[string]interface{}{
		"actor": map[string]interface{}{
			"id":         actor.ID,
			"username":   actor.Username,
			"avatar_url": actor.AvatarURL,
		},
		"entity": map[string]interface{}{
			"type":  "SHOWCASE",
			"id":    showcase.ID,
			"title": showcase.Title,
			"slug":  showcase.Slug,
		},
		"meta": map[string]interface{}{
			"message_key": "notification_like_showcase",
		},
	}

	jsonData, err := json.Marshal(notificationData)
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	notification := &domain.Notification{
		EventID:  envelope.EventID,
		UserID:   payload.TargetUserID,
		SenderID: &payload.ActorID,
		Type:     "LIKE",
		Data:     json.RawMessage(jsonData),
		IsRead:   false,
	}

	return c.repo.Create(ctx, notification)
}

// handleCommentCreated processes comment creation events
func (c *NotificationConsumer) handleCommentCreated(ctx context.Context, envelope domain.EventEnvelope) error {
	// Parse payload
	var payload struct {
		CommentID    string `json:"comment_id"`
		ShowcaseID   string `json:"showcase_id"`
		Content      string `json:"content"`
		ActorID      string `json:"actor_id"`
		TargetUserID string `json:"target_user_id"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Narcissist check
	if payload.ActorID == payload.TargetUserID {
		c.log.Debug("Skipping self-comment notification")
		return nil
	}

	// Fetch actor info
	users, err := c.gateway.GetUsersByIDs(ctx, []string{payload.ActorID})
	if err != nil {
		c.log.Errorf("Failed to fetch user data: %v", err)
		return err
	}
	actor, ok := users[payload.ActorID]
	if !ok {
		c.log.Warnf("Actor not found: %s", payload.ActorID)
		return nil
	}

	// Fetch showcase info
	showcase, err := c.gateway.GetShowcaseByID(ctx, payload.ShowcaseID)
	if err != nil {
		c.log.Errorf("Failed to fetch showcase: %v", err)
		return err
	}
	if showcase == nil {
		c.log.Warnf("Showcase not found: %s", payload.ShowcaseID)
		return nil
	}

	// Truncate content for preview
	preview := payload.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	notificationData := map[string]interface{}{
		"actor": map[string]interface{}{
			"id":         actor.ID,
			"username":   actor.Username,
			"avatar_url": actor.AvatarURL,
		},
		"entity": map[string]interface{}{
			"type":  "SHOWCASE",
			"id":    showcase.ID,
			"title": showcase.Title,
			"slug":  showcase.Slug,
		},
		"preview": map[string]interface{}{
			"text": preview,
			"id":   payload.CommentID,
		},
		"meta": map[string]interface{}{
			"message_key": "notification_comment_showcase",
		},
	}

	jsonData, err := json.Marshal(notificationData)
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	notification := &domain.Notification{
		EventID:  envelope.EventID,
		UserID:   payload.TargetUserID,
		SenderID: &payload.ActorID,
		Type:     "COMMENT",
		Data:     json.RawMessage(jsonData),
		IsRead:   false,
	}

	return c.repo.Create(ctx, notification)
}

// handleUserFollowed processes user follow events
func (c *NotificationConsumer) handleUserFollowed(ctx context.Context, envelope domain.EventEnvelope) error {
	// Parse payload
	var payload struct {
		FollowerID   string `json:"follower_id"`
		FollowedID   string `json:"followed_id"`
		FollowerInfo struct {
			Username  string `json:"username"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"follower_info"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Narcissist check
	if payload.FollowerID == payload.FollowedID {
		c.log.Debug("Skipping self-follow notification")
		return nil
	}

	notificationData := map[string]interface{}{
		"actor": map[string]interface{}{
			"id":         payload.FollowerID,
			"username":   payload.FollowerInfo.Username,
			"avatar_url": payload.FollowerInfo.AvatarURL,
		},
		"meta": map[string]interface{}{
			"message_key": "notification_follow_user",
		},
	}

	jsonData, err := json.Marshal(notificationData)
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	notification := &domain.Notification{
		EventID:  envelope.EventID,
		UserID:   payload.FollowedID,
		SenderID: &payload.FollowerID,
		Type:     "FOLLOW",
		Data:     json.RawMessage(jsonData),
		IsRead:   false,
	}

	return c.repo.Create(ctx, notification)
}

// handleCollaboratorResponded processes collaborator response events
func (c *NotificationConsumer) handleCollaboratorResponded(ctx context.Context, envelope domain.EventEnvelope) error {
	// Parse payload
	var payload struct {
		InvitationID       string `json:"invitation_id"`
		ShowcaseID         string `json:"showcase_id"`
		ShowcaseTitle      string `json:"showcase_title"`
		ResponseStatus     string `json:"response_status"`
		ResponderID        string `json:"responder_id"`
		ResponderUsername  string `json:"responder_username"`
		ResponderFullName  string `json:"responder_full_name"`
		ResponderAvatarURL string `json:"responder_avatar_url"`
		TargetUserID       string `json:"target_user_id"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Only process ACCEPTED responses
	if payload.ResponseStatus != "ACCEPTED" {
		c.log.Debug("Skipping non-accepted collaborator response")
		return nil
	}

	// Narcissist check
	if payload.ResponderID == payload.TargetUserID {
		c.log.Debug("Skipping self-acceptance notification")
		return nil
	}

	notificationData := map[string]interface{}{
		"actor": map[string]interface{}{
			"id":         payload.ResponderID,
			"username":   payload.ResponderUsername,
			"avatar_url": payload.ResponderAvatarURL,
		},
		"entity": map[string]interface{}{
			"type":  "SHOWCASE",
			"id":    payload.ShowcaseID,
			"title": payload.ShowcaseTitle,
		},
		"meta": map[string]interface{}{
			"message_key": "notification_collab_join",
		},
	}

	jsonData, err := json.Marshal(notificationData)
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	notification := &domain.Notification{
		EventID:  envelope.EventID,
		UserID:   payload.TargetUserID,
		SenderID: &payload.ResponderID,
		Type:     "COLLABORATOR_ACCEPTED",
		Data:     json.RawMessage(jsonData),
		IsRead:   false,
	}

	return c.repo.Create(ctx, notification)
}
