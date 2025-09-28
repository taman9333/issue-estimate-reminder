package queue

import "context"

//go:generate mockgen -source=interfaces.go -destination=../../test/mocks/queuemocks/queue_mocks.go -package=queuemocks

type QueueClient interface {
	EnqueueWebhook(ctx context.Context, payload *WebhookPayload) error
	Close() error
}
