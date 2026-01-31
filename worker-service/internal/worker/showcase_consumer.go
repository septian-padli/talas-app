package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/septian/worker-service/internal/domain"
	"github.com/septian/worker-service/internal/repository"
	"github.com/sirupsen/logrus"
)

const (
	ExchangeName = "talas.events"
	QueueName    = "search_sync_queue"
)

// ShowcaseConsumer consumes showcase events from RabbitMQ
type ShowcaseConsumer struct {
	channel *amqp.Channel
	repo    repository.ElasticsearchRepository
	log     *logrus.Logger
}

// NewShowcaseConsumer creates a new consumer instance
func NewShowcaseConsumer(
	channel *amqp.Channel,
	repo repository.ElasticsearchRepository,
	log *logrus.Logger,
) *ShowcaseConsumer {
	return &ShowcaseConsumer{
		channel: channel,
		repo:    repo,
		log:     log,
	}
}

// Setup declares exchange, queue, and bindings
func (c *ShowcaseConsumer) Setup() error {
	// 1. Declare Exchange (Topic, Durable)
	err := c.channel.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}
	c.log.Infof("Declared exchange: %s", ExchangeName)

	// 2. Declare Queue (Durable)
	_, err = c.channel.QueueDeclare(
		QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}
	c.log.Infof("Declared queue: %s", QueueName)

	// 3. Bind Queue to Routing Keys
	routingKeys := []string{
		"showcase.created",
		"showcase.updated",
		"showcase.deleted",
		"collaborator.responded",
		"collaborator.removed",
		"showcase.liked",
		"showcase.unliked",
		"showcase.viewed",
		"comment.created",
		"comment.deleted",
	}
	for _, key := range routingKeys {
		err = c.channel.QueueBind(QueueName, key, ExchangeName, false, nil)
		if err != nil {
			return err
		}
		c.log.Infof("Bound queue %s to routing key: %s", QueueName, key)
	}

	return nil
}

// Start begins consuming messages (blocking)
func (c *ShowcaseConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		QueueName,
		"",    // consumer tag
		false, // auto-ack (MANUAL for reliability)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	c.log.Info("Consumer started. Waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Context cancelled, stopping consumer...")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				c.log.Error("RabbitMQ channel closed unexpectedly")
				return nil
			}
			c.handleMessage(msg)
		}
	}
}

// handleMessage processes a single message based on routing key
func (c *ShowcaseConsumer) handleMessage(msg amqp.Delivery) {
	c.log.Infof("Received message [%s]: %s", msg.RoutingKey, string(msg.Body))

	switch msg.RoutingKey {
	case "showcase.created", "showcase.updated":
		c.handleIndex(msg)
	case "showcase.deleted":
		c.handleDelete(msg)
	case "collaborator.responded":
		c.handleAddCollaborator(msg)
	case "collaborator.removed":
		c.handleRemoveCollaborator(msg)
	case "showcase.liked", "showcase.unliked", "showcase.viewed", "comment.created", "comment.deleted":
		c.handleCountersUpdate(msg)
	default:
		c.log.Warnf("Unknown routing key: %s", msg.RoutingKey)
		msg.Ack(false) // Ack unknown messages to prevent queue buildup
	}
}

// handleIndex processes create/update events
func (c *ShowcaseConsumer) handleIndex(msg amqp.Delivery) {
	// 1. Parse Event Envelope
	var envelope domain.EventEnvelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		c.log.Errorf("Failed to unmarshal event envelope: %v", err)
		msg.Ack(false)
		return
	}

	// 2. Extract showcase data from envelope
	dataBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		c.log.Errorf("Failed to re-marshal data field: %v", err)
		msg.Ack(false)
		return
	}

	var showcase domain.Showcase
	if err := json.Unmarshal(dataBytes, &showcase); err != nil {
		c.log.Errorf("Failed to unmarshal showcase data: %v", err)
		msg.Ack(false)
		return
	}

	// 3. Validate ID
	if showcase.ID == uuid.Nil {
		c.log.Errorf("Showcase ID is empty/nil")
		msg.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.repo.IndexShowcase(ctx, &showcase); err != nil {
		c.log.Errorf("Failed to index showcase %s: %v", showcase.ID, err)
		msg.Nack(false, true)
		return
	}

	c.log.Infof("Successfully indexed showcase: %s", showcase.ID)
	msg.Ack(false)
}

// handleDelete processes delete events
func (c *ShowcaseConsumer) handleDelete(msg amqp.Delivery) {
	// 1. Parse Event Envelope
	var envelope domain.EventEnvelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		c.log.Errorf("Failed to unmarshal delete event envelope: %v", err)
		msg.Ack(false)
		return
	}

	// 2. Extract ID from data
	idVal, ok := envelope.Data["id"]
	if !ok {
		c.log.Errorf("Delete payload missing 'id' field in data")
		msg.Ack(false)
		return
	}

	idStr, ok := idVal.(string)
	if !ok {
		c.log.Errorf("Delete payload 'id' is not a string")
		msg.Ack(false)
		return
	}

	// 3. Validate UUID
	if _, err := uuid.Parse(idStr); err != nil {
		c.log.Errorf("Invalid UUID in delete payload: %s", idStr)
		msg.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.repo.DeleteShowcase(ctx, idStr); err != nil {
		c.log.Errorf("Failed to delete showcase %s: %v", idStr, err)
		msg.Nack(false, true)
		return
	}

	c.log.Infof("Successfully deleted showcase: %s", idStr)
	msg.Ack(false)
}

// handleAddCollaborator processes collaborator.responded events
func (c *ShowcaseConsumer) handleAddCollaborator(msg amqp.Delivery) {
	// 1. Parse Event Envelope
	var envelope domain.EventEnvelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		c.log.Errorf("Failed to unmarshal event envelope: %v", err)
		msg.Ack(false)
		return
	}

	// 2. Extract Data
	dataBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		c.log.Errorf("Failed to re-marshal data field: %v", err)
		msg.Ack(false)
		return
	}

	var eventData domain.CollaboratorEventData
	if err := json.Unmarshal(dataBytes, &eventData); err != nil {
		c.log.Errorf("Failed to unmarshal collaborator event data: %v", err)
		msg.Ack(false)
		return
	}

	// 3. Check Status (Only ACCEPTED)
	if eventData.ResponseStatus != "ACCEPTED" {
		c.log.Infof("Ignoring collaborator response status: %s", eventData.ResponseStatus)
		msg.Ack(false)
		return
	}

	// 4. Validate IDs
	if eventData.ShowcaseID == "" || eventData.ResponderID == "" {
		c.log.Error("ShowcaseID or ResponderID is empty")
		msg.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collaborator := domain.Collaborator{
		ID:        eventData.InvitationID, // Use Invitation/Collaboration ID as primary ID
		UserID:    eventData.ResponderID,  // Store User ID separately
		Username:  eventData.ResponderUsername,
		FullName:  eventData.ResponderFullName,
		AvatarURL: eventData.ResponderAvatarURL,
	}

	if err := c.repo.AddCollaborator(ctx, eventData.ShowcaseID, collaborator); err != nil {
		c.log.Errorf("Failed to add collaborator to showcase %s: %v", eventData.ShowcaseID, err)
		msg.Nack(false, true)
		return
	}

	c.log.Infof("Successfully added collaborator %s to showcase %s", eventData.ResponderID, eventData.ShowcaseID)
	msg.Ack(false)
}

// handleRemoveCollaborator processes collaborator.removed events
func (c *ShowcaseConsumer) handleRemoveCollaborator(msg amqp.Delivery) {
	// 1. Parse Event Envelope
	var envelope domain.EventEnvelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		c.log.Errorf("Failed to unmarshal event envelope: %v", err)
		msg.Ack(false)
		return
	}

	// 2. Extract Data
	dataBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		c.log.Errorf("Failed to re-marshal data field: %v", err)
		msg.Ack(false)
		return
	}

	var eventData domain.CollaboratorRemoveData
	if err := json.Unmarshal(dataBytes, &eventData); err != nil {
		c.log.Errorf("Failed to unmarshal collaborator remove data: %v", err)
		msg.Ack(false)
		return
	}

	// 3. Validate IDs
	if eventData.ShowcaseID == "" || eventData.TargetUserID == "" {
		c.log.Error("ShowcaseID or TargetUserID is empty")
		msg.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 4. Call Repository
	if err := c.repo.RemoveCollaborator(ctx, eventData.ShowcaseID, eventData.TargetUserID); err != nil {
		c.log.Errorf("Failed to remove collaborator from showcase %s: %v", eventData.ShowcaseID, err)
		msg.Nack(false, true)
		return
	}

	c.log.Infof("Successfully removed collaborator %s from showcase %s", eventData.TargetUserID, eventData.ShowcaseID)
	msg.Ack(false)
}

// handleCountersUpdate processes like/unlike/view events
func (c *ShowcaseConsumer) handleCountersUpdate(msg amqp.Delivery) {
	// 1. Parse Event Envelope
	var envelope domain.EventEnvelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		c.log.Errorf("Failed to unmarshal event envelope: %v", err)
		msg.Ack(false)
		return
	}

	// 2. Extract Data (Generic Map)
	dataBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		c.log.Errorf("Failed to re-marshal data field: %v", err)
		msg.Ack(false)
		return
	}

	// We only need showcase_id from the data
	var eventData struct {
		ShowcaseID string `json:"showcase_id"`
	}
	if err := json.Unmarshal(dataBytes, &eventData); err != nil {
		c.log.Errorf("Failed to unmarshal counter event data: %v", err)
		msg.Ack(false)
		return
	}

	if eventData.ShowcaseID == "" {
		c.log.Error("ShowcaseID is empty in counter event")
		msg.Ack(false)
		return
	}

	// 3. Determine Deltas based on Routing Key
	viewDelta := 0
	likeDelta := 0
	commentDelta := 0
	isCommentEvent := false

	switch msg.RoutingKey {
	case "showcase.viewed":
		viewDelta = 1
	case "showcase.liked":
		likeDelta = 1
	case "showcase.unliked":
		likeDelta = -1
	case "comment.created":
		commentDelta = 1
		isCommentEvent = true
	case "comment.deleted":
		commentDelta = -1
		isCommentEvent = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 4. Update Elasticsearch Atomically
	var updateErr error
	if isCommentEvent {
		updateErr = c.repo.UpdateCommentCount(ctx, eventData.ShowcaseID, commentDelta)
	} else {
		updateErr = c.repo.UpdateShowcaseCounters(ctx, eventData.ShowcaseID, viewDelta, likeDelta)
	}

	if updateErr != nil {
		c.log.Errorf("Failed to update counters for showcase %s: %v", eventData.ShowcaseID, updateErr)
		// Retry logic (same as before)
		msg.Ack(false)
		return
	}

	c.log.Infof("Updated counters for showcase %s (View: %+d, Like: %+d, Comment: %+d)", eventData.ShowcaseID, viewDelta, likeDelta, commentDelta)
	msg.Ack(false)
}
