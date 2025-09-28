package app

import "github.com/google/go-github/v74/github"

//go:generate mockgen -source=interfaces.go -destination=../../test/mocks/appmocks/app_mocks.go -package=appmocks

// AppInterface defines what handlers need from the app
type AppInterface interface {
	HandleIssueOpened(payload *github.IssuesEvent) error
	GetWebhookSecret() string
}
