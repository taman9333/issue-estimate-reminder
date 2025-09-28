package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/taman9333/issue-estimate-reminder/internal/app"
	"github.com/taman9333/issue-estimate-reminder/internal/queue"
	"github.com/taman9333/issue-estimate-reminder/internal/utils"
)

type WebhookHandler struct {
	app         app.AppInterface
	queueClient queue.QueueClient
}

func NewWebhookHandler(app app.AppInterface,
	queueClient queue.QueueClient) *WebhookHandler {
	return &WebhookHandler{
		app:         app,
		queueClient: queueClient,
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

	payload := &queue.WebhookPayload{
		DeliveryID: deliveryID,
		EventType:  eventType,
		Payload:    body,
	}

	if err := h.queueClient.EnqueueWebhook(r.Context(), payload); err != nil {
		log.Printf("Failed to enqueue webhook: %v", err)
		http.Error(w, "Failed to queue", http.StatusInternalServerError)
		return
	}

	log.Printf("Queued webhook %s for processing", deliveryID)
	w.WriteHeader(http.StatusOK)
}
