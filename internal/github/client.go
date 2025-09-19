package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
)

type Client struct {
	config *config.Config
	auth   *Auth
}

func New(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		auth:   NewAuth(cfg),
	}
}

func (c *Client) CreateInstallationClient(installationID int64) (*github.Client, error) {
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

	return github.NewClient(nil).WithAuthToken(installationToken.GetToken()), nil
}
