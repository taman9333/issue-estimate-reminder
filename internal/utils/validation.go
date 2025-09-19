package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

func HasEstimate(body string) bool {
	// check for "Estimate: X days" format (case insensitive)
	estimatePattern := regexp.MustCompile(`(?i)estimate:\s*\d+(?:\.\d+)?\s*days?`)
	return estimatePattern.MatchString(body)
}

func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}
