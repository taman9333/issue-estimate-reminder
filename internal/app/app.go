package app

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v74/github"
	"github.com/taman9333/issue-estimate-reminder/internal/utils"
)

//go:generate mockgen -destination=../../test/mocks/appmocks/app_mocks.go -package=appmocks . GitHubClientFactory,GitHubCommenter

// GitHubClientFactory creates a GitHub client
type GitHubClientFactory interface {
	CreateInstallationClient(installationID int64) (GitHubCommenter, error)
}

// GitHubCommenter defines the capability to interact with issue/PR comments on GitHub
type GitHubCommenter interface {
	CreateComment(ctx context.Context, owner, repo string, number int,
		comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

type App struct {
	githubFactory GitHubClientFactory
}

var ReminderMessage = `Hello! Please add a time estimate to this issue.

Format: Estimate: X days

Example: Estimate: 3 days

Thanks!`

func New(githubClientFactory GitHubClientFactory) *App {
	return &App{
		githubFactory: githubClientFactory,
	}
}

func (a *App) HandleIssueOpened(payload *github.IssuesEvent) error {
	issue := payload.GetIssue()
	repo := payload.GetRepo()
	installation := payload.GetInstallation()

	if installation == nil {
		return fmt.Errorf("no installation found in payload")
	}

	log.Printf("Processing issue #%d: %s", issue.GetNumber(), issue.GetTitle())

	if utils.HasEstimate(issue.GetBody()) {
		log.Printf("Issue #%d has an estimate", issue.GetNumber())
		return nil
	}

	client, err := a.githubFactory.CreateInstallationClient(installation.GetID())
	if err != nil {
		return fmt.Errorf("failed to create installation client: %v", err)
	}

	comment := &github.IssueComment{
		Body: &ReminderMessage,
	}

	_, _, err = client.CreateComment(
		context.Background(),
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		issue.GetNumber(),
		comment,
	)

	if err != nil {
		return fmt.Errorf("failed to create comment: %v", err)
	}

	log.Printf("Posted reminder comment on issue #%d", issue.GetNumber())
	return nil
}
