package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/taman9333/issue-estimate-reminder/test/mocks/queuemocks"
	"github.com/taman9333/issue-estimate-reminder/test/testutils"
	"go.uber.org/mock/gomock"
)

// Test Cases

func TestWebhookProcessor_ProcessTask_Success(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-123"
	payload := createWebhookPayload(deliveryID, "opened", "Bug without estimate")
	task := createAsynqTask(payload)

	// Expectations
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(false, nil).
		Times(1)

	s.mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Return(nil).
		Times(1)

	s.mockIdempotency.EXPECT().
		MarkProcessed(gomock.Any(), deliveryID, 168*time.Hour).
		Return(nil).
		Times(1)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert
	assert.NoError(t, err)
}

func TestWebhookProcessor_ProcessTask_AlreadyProcessed(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-456"
	payload := createWebhookPayload(deliveryID, "opened", "Bug without estimate")
	task := createAsynqTask(payload)

	// Expectations - already processed
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(true, nil).
		Times(1)

	// Should NOT process or mark as processed
	s.mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Times(0)

	s.mockIdempotency.EXPECT().
		MarkProcessed(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert - should succeed without processing
	assert.NoError(t, err)
}

func TestWebhookProcessor_ProcessTask_IdempotencyCheckError(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-789"
	payload := createWebhookPayload(deliveryID, "opened", "Bug without estimate")
	task := createAsynqTask(payload)

	// Expectations - idempotency check fails
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(false, errors.New("redis connection failed")).
		Times(1)

	// Should NOT proceed to processing
	s.mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Times(0)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert - should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "idempotency check failed")
}

func TestWebhookProcessor_ProcessTask_IgnoreNonOpenedAction(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-closed"
	payload := createWebhookPayload(deliveryID, "closed", "Bug report")
	task := createAsynqTask(payload)

	// Expectations
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(false, nil).
		Times(1)

	// Should NOT process closed actions
	s.mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Times(0)

	// Should NOT mark as processed (we didn't do anything)
	s.mockIdempotency.EXPECT().
		MarkProcessed(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert - should succeed without processing
	assert.NoError(t, err)
}

func TestWebhookProcessor_ProcessTask_HandleIssueOpenedError(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-error"
	payload := createWebhookPayload(deliveryID, "opened", "Bug without estimate")
	task := createAsynqTask(payload)

	// Expectations
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(false, nil).
		Times(1)

	s.mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Return(errors.New("github api error")).
		Times(1)

	// Should NOT mark as processed if processing fails
	s.mockIdempotency.EXPECT().
		MarkProcessed(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert - should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle issue")
}

func TestWebhookProcessor_ProcessTask_MarkProcessedError(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-mark-error"
	payload := createWebhookPayload(deliveryID, "opened", "Bug without estimate")
	task := createAsynqTask(payload)

	// Expectations
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(false, nil).
		Times(1)

	s.mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Return(nil).
		Times(1)

	s.mockIdempotency.EXPECT().
		MarkProcessed(gomock.Any(), deliveryID, 168*time.Hour).
		Return(errors.New("redis error")).
		Times(1)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert - should succeed even if marking fails (work is done)
	assert.NoError(t, err)
}

func TestWebhookProcessor_ProcessTask_InvalidPayloadJSON(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	// Create invalid task payload
	task := asynq.NewTask(TypeWebhook, []byte("invalid json"))

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert - should fail to unmarshal
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal payload")
}

func TestWebhookProcessor_ProcessTask_InvalidGitHubPayloadJSON(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-invalid-github"

	// Create webhook payload with invalid GitHub payload
	invalidPayload := &WebhookPayload{
		DeliveryID: deliveryID,
		EventType:  "issues",
		Payload:    []byte("invalid github json"),
	}
	task := createAsynqTask(invalidPayload)

	// Expectations
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(false, nil).
		Times(1)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert - should fail to parse GitHub payload
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse GitHub payload")
}

func TestWebhookProcessor_ProcessTask_WithEstimate(t *testing.T) {
	s := setupProcessorTest(t)
	defer s.cleanup()

	deliveryID := "test-delivery-with-estimate"
	payload := createWebhookPayload(deliveryID, "opened", "Bug report\nEstimate: 3 days")
	task := createAsynqTask(payload)

	// Expectations
	s.mockIdempotency.EXPECT().
		IsProcessed(gomock.Any(), deliveryID).
		Return(false, nil).
		Times(1)

	// App should handle the issue (it will check for estimate internally)
	s.mockApp.EXPECT().
		HandleIssueOpened(gomock.Any()).
		Return(nil).
		Times(1)

	s.mockIdempotency.EXPECT().
		MarkProcessed(gomock.Any(), deliveryID, 168*time.Hour).
		Return(nil).
		Times(1)

	// Execute
	err := s.processor.ProcessTask(context.Background(), task)

	// Assert
	assert.NoError(t, err)
}

// Test helpers
type processorTestSetup struct {
	ctrl            *gomock.Controller
	mockApp         *queuemocks.MockIssueOpenedHandler
	mockIdempotency *queuemocks.MockIdempotencyChecker
	processor       *WebhookProcessor
}

func setupProcessorTest(t *testing.T) *processorTestSetup {
	ctrl := gomock.NewController(t)
	mockApp := queuemocks.NewMockIssueOpenedHandler(ctrl)
	mockIdempotency := queuemocks.NewMockIdempotencyChecker(ctrl)
	processor := NewWebhookProcessor(mockApp, mockIdempotency)

	return &processorTestSetup{
		ctrl:            ctrl,
		mockApp:         mockApp,
		mockIdempotency: mockIdempotency,
		processor:       processor,
	}
}

func (s *processorTestSetup) cleanup() {
	s.ctrl.Finish()
}

func createWebhookPayload(deliveryID, action, issueBody string) *WebhookPayload {
	githubPayload := testutils.CreateWebhookPayload(action, issueBody)

	payloadBytes, _ := json.Marshal(githubPayload)

	return &WebhookPayload{
		DeliveryID: deliveryID,
		EventType:  "issues",
		Payload:    payloadBytes,
	}
}

func createAsynqTask(payload *WebhookPayload) *asynq.Task {
	data, _ := json.Marshal(payload)
	return asynq.NewTask(TypeWebhook, data)
}
