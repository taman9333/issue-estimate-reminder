package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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
