package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

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

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/webhook", handleWebhook)

	log.Printf("Server starting on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
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
