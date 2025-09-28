package main

import (
	"log"
	"net/http"

	"github.com/taman9333/issue-estimate-reminder/internal/config"
	"github.com/taman9333/issue-estimate-reminder/internal/handlers"
	"github.com/taman9333/issue-estimate-reminder/internal/queue"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	queueClient := queue.NewClient(cfg.GetRedisAddr(), cfg.RedisPassword, 0)
	defer queueClient.Close()

	// app := app.New(cfg)
	webhookHandler := handlers.NewWebhookHandler(cfg.WebhookSecret, queueClient)

	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/webhook", webhookHandler.Handle)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
