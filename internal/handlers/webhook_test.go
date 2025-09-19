package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/taman9333/issue-estimate-reminder/test/mocks"
	"github.com/taman9333/issue-estimate-reminder/test/testutils"
	"go.uber.org/mock/gomock"
)

func TestWebhookHandler_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := mocks.NewMockAppInterface(ctrl)
	handler := NewWebhookHandler(mockApp)

	mockApp.EXPECT().
		GetWebhookSecret().
		Return("test_secret").
		Times(1)

	mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Return(nil).
		Times(1)

	payload := map[string]interface{}{
		"action": "opened",
		"issue": map[string]interface{}{
			"number": 1,
			"title":  "Test Issue",
			"body":   "Bug without estimate",
		},
		"repository": map[string]interface{}{
			"name": "test-repo",
			"owner": map[string]interface{}{
				"login": "test-owner",
			},
		},
		"installation": map[string]interface{}{
			"id": 67890,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", signature)

	recorder := httptest.NewRecorder()

	handler.Handle(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestWebhookHandler_Handle_InvalidSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := mocks.NewMockAppInterface(ctrl)
	handler := NewWebhookHandler(mockApp)

	mockApp.EXPECT().
		GetWebhookSecret().
		Return("test_secret").
		Times(1)

	payload := `{"action":"opened"}`
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(payload)))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid_signature")

	recorder := httptest.NewRecorder()

	handler.Handle(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid signature")
}

func TestWebhookHandler_Handle_IgnoreNonOpenedActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := mocks.NewMockAppInterface(ctrl)
	handler := NewWebhookHandler(mockApp)

	mockApp.EXPECT().
		GetWebhookSecret().
		Return("test_secret")

	payload := map[string]interface{}{
		"action": "closed",
		"issue": map[string]interface{}{
			"number": 1,
			"title":  "Test Issue",
			"body":   "Bug report",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", signature)

	recorder := httptest.NewRecorder()

	handler.Handle(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}
