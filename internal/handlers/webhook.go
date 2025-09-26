package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/taman9333/issue-estimate-reminder/internal/app"
	"github.com/taman9333/issue-estimate-reminder/internal/idempotency"
	"github.com/taman9333/issue-estimate-reminder/internal/utils"
)

type WebhookHandler struct {
	app         app.AppInterface
	idempotency idempotency.Service
}

func NewWebhookHandler(app app.AppInterface,
	idempotencySvc idempotency.Service) *WebhookHandler {
	return &WebhookHandler{
		app:         app,
		idempotency: idempotencySvc,
	}
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deliveryID := r.Header.Get("X-GitHub-Delivery")
	if deliveryID == "" {
		log.Println("Warning: No X-GitHub-Delivery header found")
		http.Error(w, "Missing X-GitHub-Delivery header", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	if !utils.VerifyWebhookSignature(body, r.Header.Get("X-Hub-Signature-256"), h.app.GetWebhookSecret()) {
		log.Println("Invalid webhook signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	log.Printf("Received %s event", eventType)

	if eventType != "issues" {
		log.Printf("Ignoring %s event", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload github.IssuesEvent
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Error unmarshaling payload: %v", err)
		http.Error(w, "Error parsing payload", http.StatusBadRequest)
		return
	}

	if payload.GetAction() != "opened" {
		log.Printf("Ignoring issues %s action", payload.GetAction())
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check idempotency before processing
	processed, err := h.idempotency.IsProcessed(r.Context(), deliveryID)
	if err != nil {
		log.Printf("Error checking idempotency: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if processed {
		log.Printf("Webhook delivery %s already processed, skipping", deliveryID)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Processing new issue #%d: %s",
		payload.GetIssue().GetNumber(),
		payload.GetIssue().GetTitle())

	if err := h.app.HandleIssueOpened(&payload); err != nil {
		log.Printf("Error handling issue opened event: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Mark as processed after successful handling
	if err := h.idempotency.MarkProcessed(r.Context(), deliveryID, 7*24*time.Hour); err != nil {
		log.Printf("Error marking delivery as processed: %v", err)
	} else {
		log.Printf("Marked delivery %s as processed", deliveryID)
	}

	w.WriteHeader(http.StatusOK)
}
