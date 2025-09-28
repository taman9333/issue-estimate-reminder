package testutils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/google/go-github/v74/github"
)

// GenerateWebhookSignature generates a valid webhook signature for testing
func GenerateWebhookSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// CreateTestIssue creates a test GitHub issue
func CreateTestIssue(number int, title, body string) *github.Issue {
	return &github.Issue{
		Number: &number,
		Title:  &title,
		Body:   &body,
	}
}

// CreateTestRepo creates a test GitHub repository
func CreateTestRepo(owner, name string) *github.Repository {
	return &github.Repository{
		Name: &name,
		Owner: &github.User{
			Login: &owner,
		},
	}
}

// CreateTestInstallation creates a test GitHub installation
func CreateTestInstallation(id int64) *github.Installation {
	return &github.Installation{
		ID: &id,
	}
}

// CreateTestIssuesEvent creates a complete test GitHub issues event
func CreateTestIssuesEvent(action string, issue *github.Issue) *github.IssuesEvent {
	return &github.IssuesEvent{
		Action:       &action,
		Issue:        issue,
		Repo:         CreateTestRepo("test-owner", "test-repo"),
		Installation: CreateTestInstallation(67890),
	}
}

// CreateSuccessfulIssueComment creates a successful GitHub comment response
func CreateSuccessfulIssueComment(id int64, body string) *github.IssueComment {
	return &github.IssueComment{
		ID:   &id,
		Body: &body,
	}
}

// CreateSuccessfulResponse creates a successful GitHub API response
func CreateSuccessfulResponse(statusCode int) *github.Response {
	return &github.Response{
		Response: &http.Response{
			StatusCode: statusCode,
		},
	}
}

// CreateInstallationToken creates a test installation token
func CreateInstallationToken(token string) *github.InstallationToken {
	return &github.InstallationToken{
		Token: &token,
	}
}

// CreateWebhookPayload create a test webhook payload
func CreateWebhookPayload(action string, issueBody string) map[string]interface{} {
	return map[string]interface{}{
		"action": action,
		"issue": map[string]interface{}{
			"number": 1,
			"title":  "Test Issue",
			"body":   issueBody,
		},
		"repository": map[string]interface{}{
			"name": "test-repo",
			"owner": map[string]interface{}{
				"login": "test-owner",
			},
		},
		"installation": map[string]interface{}{
			"id": 67890,
		},
	}
}
