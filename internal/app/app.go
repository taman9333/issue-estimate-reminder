package app

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v74/github"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
	githubclient "github.com/taman9333/issue-estimate-reminder/internal/github"
	"github.com/taman9333/issue-estimate-reminder/internal/utils"
)

type App struct {
	config       *config.Config
	githubClient githubclient.GitHubFactoryInterface
}

var reminderMessage = `Hello! Please add a time estimate to this issue.

Format: Estimate: X days

Example: Estimate: 3 days

Thanks!`

func New(cfg *config.Config) *App {
	return &App{
		config:       cfg,
		githubClient: githubclient.New(cfg),
	}
}

func NewWithGitHubClient(cfg *config.Config, githubClient githubclient.GitHubFactoryInterface) *App {
	return &App{
		config:       cfg,
		githubClient: githubClient,
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

	client, err := a.githubClient.CreateInstallationClient(installation.GetID())
	if err != nil {
		return fmt.Errorf("failed to create installation client: %v", err)
	}

	comment := &github.IssueComment{
		Body: &reminderMessage,
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

func (a *App) GetWebhookSecret() string {
	return a.config.WebhookSecret
}
