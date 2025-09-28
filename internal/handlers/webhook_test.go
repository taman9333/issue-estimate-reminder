package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/taman9333/issue-estimate-reminder/test/mocks/appmocks"
	"github.com/taman9333/issue-estimate-reminder/test/mocks/queuemocks"
	"github.com/taman9333/issue-estimate-reminder/test/testutils"
	"go.uber.org/mock/gomock"
)

func TestWebhookHandler_Handle_Success(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-123"
	payload := testutils.CreateWebhookPayload("opened", "Bug without estimate")
	payloadBytes, _ := json.Marshal(payload)
	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	// Expectations
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret").Times(1)
	s.mockQueue.EXPECT().
		EnqueueWebhook(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// Execute
	req := createTestRequest(t, payload, deliveryID, signature, "issues")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestWebhookHandler_Handle_EnqueueFailure(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-123"
	payload := testutils.CreateWebhookPayload("opened", "Bug without estimate")
	payloadBytes, _ := json.Marshal(payload)
	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	// Expectations
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret").Times(1)
	s.mockQueue.EXPECT().
		EnqueueWebhook(gomock.Any(), gomock.Any()).
		Return(assert.AnError).
		Times(1)

	// Execute
	req := createTestRequest(t, payload, deliveryID, signature, "issues")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Failed to queue")
}

func TestWebhookHandler_Handle_InvalidSignature(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-invalid"
	payload := testutils.CreateWebhookPayload("opened", "Invalid signature")

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

func TestWebhookHandler_Handle_IgnoreNonIssuesEvent(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-pr"
	payload := testutils.CreateWebhookPayload("opened", "Bug report")
	payloadBytes, _ := json.Marshal(payload)
	signature := testutils.GenerateWebhookSignature(payloadBytes, "test_secret")

	// Expectations
	s.mockApp.EXPECT().GetWebhookSecret().Return("test_secret").Times(1)
	// No queue expectation - should return before enqueuing

	// Execute
	req := createTestRequest(t, payload, deliveryID, signature, "pull_request")
	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestWebhookHandler_Handle_MissingDeliveryID(t *testing.T) {
	s := setupTest(t)
	defer s.cleanup()

	// Execute - no delivery ID
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-GitHub-Event", "issues")
	// No X-GitHub-Delivery header

	recorder := httptest.NewRecorder()
	s.handler.Handle(recorder, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Missing X-GitHub-Delivery header")
}

// Test helpers
type testSetup struct {
	ctrl      *gomock.Controller
	mockApp   *appmocks.MockAppInterface
	mockQueue *queuemocks.MockQueueClient
	handler   *WebhookHandler
}

func setupTest(t *testing.T) *testSetup {
	ctrl := gomock.NewController(t)
	mockApp := appmocks.NewMockAppInterface(ctrl)
	mockQueue := queuemocks.NewMockQueueClient(ctrl)
	handler := NewWebhookHandler(mockApp, mockQueue)

	return &testSetup{
		ctrl:      ctrl,
		mockApp:   mockApp,
		mockQueue: mockQueue,
		handler:   handler,
	}
}

func (s *testSetup) cleanup() {
	s.ctrl.Finish()
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
