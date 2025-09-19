package testutils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateWebhookSignature generates a valid webhook signature for testing
func GenerateWebhookSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
