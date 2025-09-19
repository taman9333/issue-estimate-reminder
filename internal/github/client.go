package github

import (
	"github.com/taman9333/issue-estimate-reminder/internal/config"
)

type Client struct {
	config     *config.Config
	auth       AuthInterface
	tokenCache *TokenCache
}

func New(cfg *config.Config) *Client {
	auth := NewAuth(cfg)
	return &Client{
		config:     cfg,
		auth:       NewAuth(cfg),
		tokenCache: NewTokenCache(auth),
	}
}

func (c *Client) CreateInstallationClient(installationID int64) (GitHubClientInterface, error) {
	return c.tokenCache.GetInstallationClient(installationID)
}
