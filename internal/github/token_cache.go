package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/taman9333/issue-estimate-reminder/internal/app"
)

type TokenCache struct {
	mu     sync.RWMutex
	tokens map[int64]*CachedToken
	auth   *Auth
}

type CachedToken struct {
	Token     string
	ExpiresAt time.Time
	Client    app.GitHubCommenter
}

func NewTokenCache(auth *Auth) *TokenCache {
	return &TokenCache{
		tokens: make(map[int64]*CachedToken),
		auth:   auth,
	}
}

func (tc *TokenCache) GetInstallationClient(installationID int64) (app.GitHubCommenter, error) {
	tc.mu.RLock()
	cached, exists := tc.tokens[installationID]
	tc.mu.RUnlock()

	// Check if we have a valid cached token (with 5-minute buffer before expiration)
	if exists && time.Now().Before(cached.ExpiresAt.Add(-5*time.Minute)) {
		return cached.Client, nil
	}

	// Need to create a new token
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// maybe another goroutine might have updated it
	if cached, exists := tc.tokens[installationID]; exists && time.Now().Before(cached.ExpiresAt.Add(-5*time.Minute)) {
		return cached.Client, nil
	}

	// Generate new token
	client, token, expiresAt, err := tc.createNewInstallationClient(installationID)
	if err != nil {
		return nil, err
	}

	// Cache the token
	tc.tokens[installationID] = &CachedToken{
		Token:     token,
		ExpiresAt: expiresAt,
		Client:    client,
	}

	return client, nil
}

func (tc *TokenCache) createNewInstallationClient(installationID int64) (app.GitHubCommenter, string, time.Time, error) {
	jwt, err := tc.auth.GenerateJWT()
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("failed to generate JWT: %v", err)
	}

	appClient := github.NewClient(nil).WithAuthToken(jwt)

	installationToken, _, err := appClient.Apps.CreateInstallationToken(
		context.Background(),
		installationID,
		&github.InstallationTokenOptions{},
	)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("failed to create installation token: %v", err)
	}

	client := github.NewClient(nil).WithAuthToken(installationToken.GetToken())
	wrappedClient := NewGitHubClientWrapper(client)

	return wrappedClient, installationToken.GetToken(), installationToken.GetExpiresAt().Time, nil
}
