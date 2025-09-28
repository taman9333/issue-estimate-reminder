package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/taman9333/issue-estimate-reminder/internal/config"
	"github.com/taman9333/issue-estimate-reminder/test/mocks/githubmocks"
	"github.com/taman9333/issue-estimate-reminder/test/testutils"
	"go.uber.org/mock/gomock"
)

func TestApp_HandleIssueOpened_WithEstimate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		AppID:         12345,
		WebhookSecret: "test_secret",
		Port:          "8080",
	}

	mockGitHubFactory := githubmocks.NewMockGitHubFactoryInterface(ctrl)
	app := NewWithGitHubClient(cfg, mockGitHubFactory)

	// Create issue with estimate
	issue := testutils.CreateTestIssue(1, "Bug Report", "This is a bug\nEstimate: 3 days")
	event := testutils.CreateTestIssuesEvent("opened", issue)

	// Should not call GitHub API since issue has estimate
	// No expectations set for mockGitHubFactory

	err := app.HandleIssueOpened(event)

	assert.NoError(t, err)
}

func TestApp_HandleIssueOpened_WithoutEstimate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		AppID:         12345,
		WebhookSecret: "test_secret",
		Port:          "8080",
	}

	mockGitHubFactory := githubmocks.NewMockGitHubFactoryInterface(ctrl)
	mockGitHubClient := githubmocks.NewMockGitHubClientInterface(ctrl)
	app := NewWithGitHubClient(cfg, mockGitHubFactory)

	// Create issue without estimate
	issue := testutils.CreateTestIssue(1, "Bug Report", "This is a bug without estimate")
	event := testutils.CreateTestIssuesEvent("opened", issue)

	// Set up expectations
	mockGitHubFactory.EXPECT().
		CreateInstallationClient(int64(67890)).
		Return(mockGitHubClient, nil).
		Times(1)

	mockGitHubClient.EXPECT().
		CreateComment(
			gomock.Any(), // context
			"test-owner", // owner
			"test-repo",  // repo
			1,            // issue number
			gomock.Any(), // comment
		).
		Return(
			testutils.CreateSuccessfulIssueComment(123, reminderMessage),
			testutils.CreateSuccessfulResponse(201),
			nil,
		).
		Times(1)

	err := app.HandleIssueOpened(event)

	assert.NoError(t, err)
}
