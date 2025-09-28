package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/taman9333/issue-estimate-reminder/internal/app"
	"github.com/taman9333/issue-estimate-reminder/test/mocks/appmocks"
	"github.com/taman9333/issue-estimate-reminder/test/testutils"
	"go.uber.org/mock/gomock"
)

func TestApp_HandleIssueOpened_WithEstimate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGitHubFactory := appmocks.NewMockGitHubClientFactory(ctrl)
	application := app.New(mockGitHubFactory)

	// Create issue with estimate
	issue := testutils.CreateTestIssue(1, "Bug Report", "This is a bug\nEstimate: 3 days")
	event := testutils.CreateTestIssuesEvent("opened", issue)

	// Should not call GitHub API since issue has estimate
	// No expectations set for mockGitHubFactory

	err := application.HandleIssueOpened(event)

	assert.NoError(t, err)
}

func TestApp_HandleIssueOpened_WithoutEstimate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGitHubFactory := appmocks.NewMockGitHubClientFactory(ctrl)
	mockGitHubCommenter := appmocks.NewMockGitHubCommenter(ctrl)
	application := app.New(mockGitHubFactory)

	// Create issue without estimate
	issue := testutils.CreateTestIssue(1, "Bug Report", "This is a bug without estimate")
	event := testutils.CreateTestIssuesEvent("opened", issue)

	// Set up expectations
	mockGitHubFactory.EXPECT().
		CreateInstallationClient(int64(67890)).
		Return(mockGitHubCommenter, nil).
		Times(1)

	mockGitHubCommenter.EXPECT().
		CreateComment(
			gomock.Any(), // context
			"test-owner", // owner
			"test-repo",  // repo
			1,            // issue number
			gomock.Any(), // comment
		).
		Return(
			testutils.CreateSuccessfulIssueComment(123, app.ReminderMessage),
			testutils.CreateSuccessfulResponse(201),
			nil,
		).
		Times(1)

	err := application.HandleIssueOpened(event)

	assert.NoError(t, err)
}
