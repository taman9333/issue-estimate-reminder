package utils

import (
	"testing"

	"github.com/taman9333/issue-estimate-reminder/test/testutils"
)

func TestHasEstimate(t *testing.T) {
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
			result := HasEstimate(tt.body)
			if result != tt.expected {
				t.Errorf("hasEstimate() = %v, expected %v for body: %s", result, tt.expected, tt.body)
			}
		})
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	payload := `{"test":"data"}`
	webhookSecret := "test_secret"

	tests := []struct {
		name      string
		payload   string
		signature string
		secret    string
		expected  bool
	}{
		{
			name:      "Valid signature",
			payload:   payload,
			signature: testutils.GenerateWebhookSignature([]byte(payload), webhookSecret),
			secret:    webhookSecret,
			expected:  true,
		},
		{
			name:      "Invalid signature",
			payload:   payload,
			signature: "sha256=invalid_signature",
			secret:    webhookSecret,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyWebhookSignature([]byte(tt.payload), tt.signature, tt.secret)
			if result != tt.expected {
				t.Errorf("VerifyWebhookSignature() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
