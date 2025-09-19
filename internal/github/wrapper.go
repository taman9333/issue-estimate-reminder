// internal/github/wrapper.go
package github

import (
	"context"

	"github.com/google/go-github/v74/github"
)

type GitHubClientWrapper struct {
	client *github.Client
}

func NewGitHubClientWrapper(client *github.Client) *GitHubClientWrapper {
	return &GitHubClientWrapper{client: client}
}

func (w *GitHubClientWrapper) CreateComment(ctx context.Context, owner, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	return w.client.Issues.CreateComment(ctx, owner, repo, number, comment)
}
