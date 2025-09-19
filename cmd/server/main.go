package main

import (
	"log"
	"net/http"

	"github.com/taman9333/issue-estimate-reminder/internal/app"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
	"github.com/taman9333/issue-estimate-reminder/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	app := app.New(cfg)
	webhookHandler := handlers.NewWebhookHandler(app)

	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/webhook", webhookHandler.Handle)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
