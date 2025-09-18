package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/joho/godotenv"
)

type App struct {
	config Config
}

type Config struct {
	AppID          int64
	PrivateKeyPath string
	WebhookSecret  string
	Port           string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := Config{
		AppID:          getEnvAsInt("GITHUB_APP_ID"),
		PrivateKeyPath: getEnv("GITHUB_PRIVATE_KEY_PATH", "./app.pem"),
		WebhookSecret:  getEnv("WEBHOOK_SECRET", ""),
		Port:           getEnv("PORT", "8080"),
	}

	if config.AppID == 0 {
		log.Fatal("GITHUB_APP_ID is required")
	}
	if config.WebhookSecret == "" {
		log.Fatal("WEBHOOK_SECRET is required")
	}

	app := &App{config: config}

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/webhook", app.handleWebhook)

	log.Printf("Server starting on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (a *App) handleWebhook(w http.ResponseWriter, r *http.Request) {
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

	if !a.verifySignature(body, r.Header.Get("X-Hub-Signature-256")) {
		log.Println("Invalid webhook signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	log.Printf("Received %s event with body: %s", eventType, string(body))

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

	// only handle "opened" action based on the assessment's requirements
	if payload.GetAction() != "opened" {
		log.Printf("Ignoring issues %s action", payload.GetAction())
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Processing new issue #%d: %s",
		payload.GetIssue().GetNumber(),
		payload.GetIssue().GetTitle())

	if err := a.handleIssueOpened(&payload); err != nil {
		log.Printf("Error handling issue opened event: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return 0
}

func (a *App) verifySignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(a.config.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

func (a *App) handleIssueOpened(payload *github.IssuesEvent) error {
	issue := payload.GetIssue()

	log.Printf("Processing issue #%d: %s", issue.GetNumber(), issue.GetTitle())

	if a.hasEstimate(issue.GetBody()) {
		log.Printf("Issue #%d has an estimate", issue.GetNumber())
		return nil
	}

	log.Printf("Issue #%d missing estimate", issue.GetNumber())
	// TODO: post comment
	return nil
}

func (a *App) hasEstimate(body string) bool {
	// check for "Estimate: X days" format (case insensitive)
	estimatePattern := regexp.MustCompile(`(?i)estimate:\s*\d+(?:\.\d+)?\s*days?`)
	return estimatePattern.MatchString(body)
}
