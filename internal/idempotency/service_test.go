package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

const (
	testDeliveryID = "test-delivery-id-456"
)

func TestIsProcessed(t *testing.T) {
	mr, rdb, teardown := setupMiniredis(t)
	defer teardown()

	ctx := context.Background()
	svc := NewService(rdb)
	expectedKey := keyPrefix + testDeliveryID

	t.Run("NotProcessed_KeyDoesNotExist", func(t *testing.T) {
		assert.False(t, mr.Exists(expectedKey))

		isProcessed, err := svc.IsProcessed(ctx, testDeliveryID)

		assert.NoError(t, err)
		assert.False(t, isProcessed)
	})

	t.Run("Processed_KeyExists", func(t *testing.T) {
		mr.Set(expectedKey, "12345")

		isProcessed, err := svc.IsProcessed(ctx, testDeliveryID)

		assert.NoError(t, err)
		assert.True(t, isProcessed)
	})
}

func TestMarkProcessed(t *testing.T) {
	mr, rdb, teardown := setupMiniredis(t)
	defer teardown()

	ctx := context.Background()
	svc := NewService(rdb)
	expectedKey := keyPrefix + testDeliveryID

	t.Run("Success_CustomTTL", func(t *testing.T) {
		customTTL := 5 * time.Hour

		err := svc.MarkProcessed(ctx, testDeliveryID, customTTL)

		assert.NoError(t, err)
		// Verification 1: Check key exists
		assert.True(t, mr.Exists(expectedKey))
		// Verification 2: Check TTL is set correctly (miniredis uses seconds)
		ttl := mr.TTL(expectedKey)
		assert.InDelta(t, customTTL.Seconds(), ttl.Seconds(), 1.0, "TTL should match custom TTL")
	})

	t.Run("Success_DefaultTTL", func(t *testing.T) {
		mr.FlushAll()

		err := svc.MarkProcessed(ctx, testDeliveryID, 0) // TTL=0 triggers default

		assert.NoError(t, err)
		// Verification 1: Check key exists
		assert.True(t, mr.Exists(expectedKey))
		// Verification 2: Check TTL is set to default
		ttl := mr.TTL(expectedKey)
		assert.InDelta(t, defaultTTL.Seconds(), ttl.Seconds(), 1.0, "TTL should match default TTL")
	})

	t.Run("RedisError_Simulated", func(t *testing.T) {
		// To simulate an error, we can stop the server mid-call
		mr.Close()

		err := svc.MarkProcessed(ctx, testDeliveryID, 1*time.Hour)

		assert.Error(t, err)
	})
}

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, redis.Cmdable, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, rdb, func() {
		rdb.Close()
		mr.Close()
	}
}
