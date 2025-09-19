package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
)

type Client struct {
	config *config.Config
	auth   AuthInterface
}

func New(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		auth:   NewAuth(cfg),
	}
}

func NewWithAuth(cfg *config.Config, auth AuthInterface) *Client {
	return &Client{
		config: cfg,
		auth:   auth,
	}
}

func (c *Client) CreateInstallationClient(installationID int64) (GitHubClientInterface, error) {
	token, err := c.auth.GenerateJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %v", err)
	}

	appClient := github.NewClient(nil).WithAuthToken(token)

	installationToken, _, err := appClient.Apps.CreateInstallationToken(
		context.Background(),
		installationID,
		&github.InstallationTokenOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create installation token: %v", err)
	}

	client := github.NewClient(nil).WithAuthToken(installationToken.GetToken())
	return NewGitHubClientWrapper(client), nil
}
