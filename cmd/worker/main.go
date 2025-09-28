package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/taman9333/issue-estimate-reminder/internal/app"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
	"github.com/taman9333/issue-estimate-reminder/internal/github"
	"github.com/taman9333/issue-estimate-reminder/internal/idempotency"
	"github.com/taman9333/issue-estimate-reminder/internal/queue"
	"github.com/taman9333/issue-estimate-reminder/internal/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Redis client
	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Idempotency service
	idempotencySvc := idempotency.NewService(redisClient)

	// App
	githubClient := github.New(cfg)
	application := app.New(githubClient)

	// Asynq server (worker)
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.GetRedisAddr(),
			Password: cfg.RedisPassword,
		},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"default": 10,
			},
		},
	)

	// Register handlers
	mux := asynq.NewServeMux()
	processor := queue.NewWebhookProcessor(application, idempotencySvc)
	mux.HandleFunc(queue.TypeWebhook, processor.ProcessTask)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down worker")
		srv.Shutdown()
	}()

	log.Println("Worker started, processing tasks")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}

	log.Println("Worker exited")
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
