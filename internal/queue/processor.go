package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/hibiken/asynq"
)

//go:generate mockgen -destination=../../test/mocks/queuemocks/queue_mocks.go -package=queuemocks . IdempotencyChecker,IssueOpenedHandler

// IdempotencyChecker is used for webhook processing
type IdempotencyChecker interface {
	IsProcessed(ctx context.Context, deliveryID string) (bool, error)
	MarkProcessed(ctx context.Context, deliveryID string, ttl time.Duration) error
}

// IssueOpenedHandler defines the capability to handle a new GitHub issue opened event.
type IssueOpenedHandler interface {
	HandleIssueOpened(payload *github.IssuesEvent) error
}

type WebhookProcessor struct {
	app         IssueOpenedHandler
	idempotency IdempotencyChecker
}

func NewWebhookProcessor(app IssueOpenedHandler, idempotencySvc IdempotencyChecker) *WebhookProcessor {
	return &WebhookProcessor{
		app:         app,
		idempotency: idempotencySvc,
	}
}

func (p *WebhookProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload WebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Processing webhook: delivery_id=%s event_type=%s", payload.DeliveryID, payload.EventType)

	// Check idempotency
	processed, err := p.idempotency.IsProcessed(ctx, payload.DeliveryID)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if processed {
		log.Printf("Webhook %s already processed, skipping", payload.DeliveryID)
		return nil
	}

	var issuesEvent github.IssuesEvent
	if err := json.Unmarshal(payload.Payload, &issuesEvent); err != nil {
		return fmt.Errorf("failed to parse GitHub payload: %w", err)
	}

	if issuesEvent.GetAction() != "opened" {
		log.Printf("Ignoring action: %s", issuesEvent.GetAction())
		return nil
	}

	// Process webhook
	if err := p.app.HandleIssueOpened(&issuesEvent); err != nil {
		return fmt.Errorf("failed to handle issue: %w", err)
	}

	// Mark as processed
	if err := p.idempotency.MarkProcessed(ctx, payload.DeliveryID, 168*time.Hour); err != nil {
		log.Printf("Failed to mark as processed: %v", err)
	}

	log.Printf("Successfully processed webhook %s", payload.DeliveryID)
	return nil
}
