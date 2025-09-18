package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v74/github"
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

var reminderMessage = `Hello! Please add a time estimate to this issue.

Format: Estimate: X days

Example: Estimate: 3 days

Thanks!`

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
	repo := payload.GetRepo()
	installation := payload.GetInstallation()

	if installation == nil {
		return fmt.Errorf("no installation found in payload")
	}

	log.Printf("Processing issue #%d: %s", issue.GetNumber(), issue.GetTitle())

	if a.hasEstimate(issue.GetBody()) {
		log.Printf("Issue #%d has an estimate", issue.GetNumber())
		return nil
	}

	client, err := a.createInstallationClient(installation.GetID())
	if err != nil {
		return fmt.Errorf("failed to create installation client: %v", err)
	}

	comment := &github.IssueComment{
		Body: &reminderMessage,
	}

	_, _, err = client.Issues.CreateComment(
		context.Background(),
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		issue.GetNumber(),
		comment,
	)

	if err != nil {
		return fmt.Errorf("failed to create comment: %v", err)
	}

	log.Printf("Posted reminder comment on issue #%d", issue.GetNumber())
	return nil
}

func (a *App) hasEstimate(body string) bool {
	// check for "Estimate: X days" format (case insensitive)
	estimatePattern := regexp.MustCompile(`(?i)estimate:\s*\d+(?:\.\d+)?\s*days?`)
	return estimatePattern.MatchString(body)
}

func (a *App) generateJWT() (string, error) {
	keyData, err := os.ReadFile(a.config.PrivateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %v", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": a.config.AppID,
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(), // TODO: check later when this expires
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	return tokenString, nil
}

func (a *App) createInstallationClient(installationID int64) (*github.Client, error) {
	token, err := a.generateJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %v", err)
	}

	appClient := github.NewClient(nil).WithAuthToken(token)

	installationToken, _, err := appClient.Apps.CreateInstallationToken(
		context.Background(),
		installationID,
		&github.InstallationTokenOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create installation token: %v", err)
	}

	return github.NewClient(nil).WithAuthToken(installationToken.GetToken()), nil
}
