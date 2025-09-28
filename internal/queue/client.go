package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TypeWebhook = "webhook:process"
)

type WebhookPayload struct {
	DeliveryID string `json:"delivery_id"`
	EventType  string `json:"event_type"`
	Payload    []byte `json:"payload"`
}

type Client struct {
	client *asynq.Client
}

// Ensure Client implements QueueClient interface
var _ QueueClient = (*Client)(nil)

func NewClient(redisAddr, redisPassword string, redisDB int) *Client {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	})

	return &Client{client: client}
}

func (c *Client) EnqueueWebhook(ctx context.Context, payload *WebhookPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeWebhook, data)

	info, err := c.client.EnqueueContext(ctx, task,
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	fmt.Printf("Enqueued task: id=%s queue=%s\n", info.ID, info.Queue)
	return nil
}

func (c *Client) Close() error {
	return c.client.Close()
}
