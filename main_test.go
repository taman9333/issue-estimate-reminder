package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestHasEstimate(t *testing.T) {
	app := &App{}

	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "No estimate",
			body:     "This is a bug that needs fixing",
			expected: false,
		},
		{
			name:     "Valid estimate - days",
			body:     "Bug in login system\nEstimate: 3 days",
			expected: true,
		},
		{
			name:     "Case insensitive",
			body:     "Bug report\nestimate: 2 days",
			expected: true,
		},
		{
			name:     "Invalid format",
			body:     "Bug report\nEstimate 3 days",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.hasEstimate(tt.body)
			if result != tt.expected {
				t.Errorf("hasEstimate() = %v, expected %v for body: %s", result, tt.expected, tt.body)
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	payload := `{"test":"data"}`
	webhookSecret := "test_secret"
	app := &App{
		config: Config{
			WebhookSecret: webhookSecret,
		},
	}

	tests := []struct {
		name      string
		payload   string
		signature string
		expected  bool
	}{
		{
			name:      "Valid signature",
			payload:   payload,
			signature: generateSignature([]byte(payload), webhookSecret),
			expected:  true,
		},
		{
			name:      "Invalid signature",
			payload:   payload,
			signature: "sha256=invalid_signature",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.verifySignature([]byte(tt.payload), tt.signature)
			if result != tt.expected {
				t.Errorf("verifySignature() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
