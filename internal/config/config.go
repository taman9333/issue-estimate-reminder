package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
	// GitHub Configuration
	AppID          int64
	PrivateKeyPath string
	WebhookSecret  string
	// Redis Configuration
	RedisHost     string
	RedisPort     string
	RedisPassword string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := &Config{
		Port: getEnv("PORT", "8080"),

		AppID:          getEnvAsInt("GITHUB_APP_ID"),
		PrivateKeyPath: getEnv("GITHUB_PRIVATE_KEY_PATH", "./app.pem"),
		WebhookSecret:  getEnv("WEBHOOK_SECRET", ""),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.AppID == 0 {
		return fmt.Errorf("GITHUB_APP_ID is required")
	}
	if c.WebhookSecret == "" {
		return fmt.Errorf("WEBHOOK_SECRET is required")
	}

	return nil
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
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
