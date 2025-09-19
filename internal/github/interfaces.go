package github

import (
	"context"

	"github.com/google/go-github/v74/github"
)

//go:generate mockgen -source=interfaces.go -destination=../../test/mocks/github_mocks.go -package=mocks

// GitHubClientInterface defines what we need from GitHub client
type GitHubClientInterface interface {
	CreateComment(ctx context.Context, owner, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
	CreateInstallationToken(ctx context.Context, installationID int64, opts *github.InstallationTokenOptions) (*github.InstallationToken, *github.Response, error)
}

// GitHubFactoryInterface creates GitHub clients
type GitHubFactoryInterface interface {
	CreateInstallationClient(installationID int64) (GitHubClientInterface, error)
	CreateAppClient() (GitHubClientInterface, error)
}

// AuthInterface handles GitHub authentication
type AuthInterface interface {
	GenerateJWT() (string, error)
}
