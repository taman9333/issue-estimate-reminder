package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/taman9333/issue-estimate-reminder/test/mocks"
	"github.com/taman9333/issue-estimate-reminder/test/testutils"
	"go.uber.org/mock/gomock"
)

func TestWebhookHandler_Handle_Success(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-123"
	payload := createTestPayload("opened", "Bug without estimate")
	payloadBytes, _ := json.Marshal(payload)
	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	// Expectations
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret").Times(1)
	s.mockIdempotency.EXPECT().IsProcessed(gomock.Any(), deliveryID).Return(false, nil).Times(1)
	s.mockApp.EXPECT().HandleIssueOpened(gomock.Any()).Return(nil).Times(1)
	s.mockIdempotency.EXPECT().MarkProcessed(gomock.Any(), deliveryID, 7*24*time.Hour).Return(nil).Times(1)

	// Execute
	req := createTestRequest(t, payload, deliveryID, signature, "issues")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestWebhookHandler_Handle_AlreadyProcessed(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-456"
	payload := createTestPayload("opened", "Bug without estimate")
	payloadBytes, _ := json.Marshal(payload)
	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	// Expectations - webhook already processed
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret").Times(1)
	s.mockIdempotency.EXPECT().IsProcessed(gomock.Any(), deliveryID).Return(true, nil).Times(1)
	s.mockApp.EXPECT().HandleIssueOpened(gomock.Any()).Times(0)
	s.mockIdempotency.EXPECT().MarkProcessed(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// Execute
	req := createTestRequest(t, payload, deliveryID, signature, "issues")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestWebhookHandler_Handle_IdempotencyCheckError(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-789"
	payload := createTestPayload("opened", "Bug without estimate")
	payloadBytes, _ := json.Marshal(payload)
	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	// Expectations - Redis error
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret").Times(1)
	s.mockIdempotency.EXPECT().IsProcessed(gomock.Any(), deliveryID).Return(false, assert.AnError).Times(1)

	// Execute
	req := createTestRequest(t, payload, deliveryID, signature, "issues")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestWebhookHandler_Handle_InvalidSignature(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-invalid"
	payload := createTestPayload("opened", "Invalid signature")

	// Expectations
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret").Times(1)

	// Execute
	req := createTestRequest(t, payload, deliveryID, "sha256=invalid_signature", "issues")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid signature")
}

func TestWebhookHandler_Handle_IgnoreNonOpenedActions(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-closed"
	payload := createTestPayload("closed", "Bug report")
	payloadBytes, _ := json.Marshal(payload)
	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	// Expectations
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret")

	// Execute
	req := createTestRequest(t, payload, deliveryID, signature, "issues")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, recorder.Code)
}

// Test helpers
type testSetup struct {
	ctrl            *gomock.Controller
	mockApp         *mocks.MockAppInterface
	mockIdempotency *mocks.MockService
	handler         *WebhookHandler
}

func setupTest(t *testing.T) *testSetup {
	ctrl := gomock.NewController(t)
	mockApp := mocks.NewMockAppInterface(ctrl)
	mockIdempotency := mocks.NewMockService(ctrl)
	handler := NewWebhookHandler(mockApp, mockIdempotency)

	return &testSetup{
		ctrl:            ctrl,
		mockApp:         mockApp,
		mockIdempotency: mockIdempotency,
		handler:         handler,
	}
}

func (s *testSetup) cleanup() {
	s.ctrl.Finish()
}

func createTestPayload(action string, issueBody string) map[string]interface{} {
	return map[string]interface{}{
		"action": action,
		"issue": map[string]interface{}{
			"number": 1,
			"title":  "Test Issue",
			"body":   issueBody,
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
}

func createTestRequest(t *testing.T, payload map[string]interface{},
	deliveryID, signature, eventType string) *http.Request {
	var payloadBytes []byte
	var err error

	payloadBytes, err = json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("X-GitHub-Event", eventType)
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("X-GitHub-Delivery", deliveryID)

	return req
}
