package idempotency

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix  = "webhook:delivery:"
	defaultTTL = 7 * 24 * time.Hour // keep events for 7 days
)

type service struct {
	redis redis.Cmdable
}

// NewService creates a new idempotency service
func NewService(redisClient redis.Cmdable) service {
	return service{
		redis: redisClient,
	}
}

// IsProcessed checks if a webhook delivery has already been processed
func (s service) IsProcessed(ctx context.Context, deliveryID string) (bool, error) {
	key := s.buildKey(deliveryID)

	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check delivery ID %s: %w", deliveryID, err)
	}

	return exists > 0, nil
}

// MarkProcessed marks a webhook delivery as processed
func (s service) MarkProcessed(ctx context.Context, deliveryID string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = defaultTTL
	}

	key := s.buildKey(deliveryID)
	timestamp := time.Now().Unix()

	err := s.redis.Set(ctx, key, timestamp, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to mark delivery %s as processed: %w", deliveryID, err)
	}

	return nil
}

func (s service) buildKey(deliveryID string) string {
	return keyPrefix + deliveryID
}
