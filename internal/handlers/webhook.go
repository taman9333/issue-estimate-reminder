package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/taman9333/issue-estimate-reminder/internal/app"
	"github.com/taman9333/issue-estimate-reminder/internal/utils"
)

type WebhookHandler struct {
	app app.AppInterface // Use app.AppInterface instead of local interface
}

func NewWebhookHandler(app app.AppInterface) *WebhookHandler {
	return &WebhookHandler{app: app}
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	log.Printf("Processing new issue #%d: %s",
		payload.GetIssue().GetNumber(),
		payload.GetIssue().GetTitle())

	if err := h.app.HandleIssueOpened(&payload); err != nil {
		log.Printf("Error handling issue opened event: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
