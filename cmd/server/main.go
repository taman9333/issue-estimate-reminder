package main

import (
	"log"
	"net/http"

	"github.com/taman9333/issue-estimate-reminder/internal/app"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
	"github.com/taman9333/issue-estimate-reminder/internal/handlers"
	"github.com/taman9333/issue-estimate-reminder/internal/idempotency"
	"github.com/taman9333/issue-estimate-reminder/internal/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	idempotencySvc := idempotency.NewService(redisClient)

	app := app.New(cfg)
	webhookHandler := handlers.NewWebhookHandler(app, idempotencySvc)

	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/webhook", webhookHandler.Handle)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

func initRedis(cfg *config.Config) (*redis.Client, error) {
	redisClient, err := redis.NewClient(redis.Config{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to Redis at %s", cfg.GetRedisAddr())
	return redisClient, nil
}
