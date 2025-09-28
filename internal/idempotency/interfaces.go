package idempotency

import (
	"context"
	"time"
)

//go:generate mockgen -source=interfaces.go -destination=../../test/mocks/idempotencymocks/idempotency_mocks.go -package=idempotencymocks

// Service handles webhook idempotency using delivery IDs
type Service interface {
	// IsProcessed checks if a delivery ID has already been processed
	IsProcessed(ctx context.Context, deliveryID string) (bool, error)

	// MarkProcessed marks a delivery ID as processed with TTL
	MarkProcessed(ctx context.Context, deliveryID string, ttl time.Duration) error
}
