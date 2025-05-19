package processor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/milosgajdos/pubsub"
)

// TestEvent is a simple event structure used for testing
type TestEvent struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// ComplexTestEvent is a more complex event for testing
type ComplexTestEvent struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Items     []string `json:"items"`
	Processed bool     `json:"processed"`
}

func TestRegisterTyped(t *testing.T) {
	t.Run("SuccessfulHandling", func(t *testing.T) {
		p := New()

		// Test data
		expectedID := "test-123"
		expectedMessage := "hello world"

		// Track if handler is called
		handlerCalled := false
		var capturedEvent TestEvent

		// Register typed handler
		err := Register(p, "test-event",
			func(_ context.Context, event TestEvent) error {
				handlerCalled = true
				capturedEvent = event
				return nil
			},
		)

		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Create test message
		testEvent := TestEvent{
			ID:      expectedID,
			Message: expectedMessage,
		}

		eventData, _ := json.Marshal(testEvent)

		// Create a mock message
		mockMessage := &MockMessage{
			id:   "msg-1",
			data: eventData,
		}

		// Mock the extractor
		p.extractor = func(_ []byte) (pubsub.MessageType, error) {
			return "test-event", nil
		}

		err = p.Process(context.Background(), mockMessage)
		if err != nil {
			t.Errorf("Process returned error: %v", err)
		}

		// Verify handler was called with correct data
		if !handlerCalled {
			t.Error("Handler was not called")
		}

		if capturedEvent.ID != expectedID {
			t.Errorf("Wrong ID: got %s, want %s", capturedEvent.ID, expectedID)
		}

		if capturedEvent.Message != expectedMessage {
			t.Errorf("Wrong message: got %s, want %s", capturedEvent.Message, expectedMessage)
		}

		// Verify message was acked
		if !mockMessage.ackCalled {
			t.Error("Message was not acked")
		}
	})

	t.Run("ComplexEventHandling", func(t *testing.T) {
		p := New()

		// Test data
		complexEvent := ComplexTestEvent{
			UserID:    "user-123",
			Email:     "test@example.com",
			Items:     []string{"item1", "item2", "item3"},
			Processed: true,
		}
		complexEventType := pubsub.MessageType("complex-event")

		// Track if handler is called
		handlerCalled := false
		var capturedEvent ComplexTestEvent

		// Register typed handler
		err := Register(p, complexEventType,
			func(_ context.Context, event ComplexTestEvent) error {
				handlerCalled = true
				capturedEvent = event
				return nil
			},
		)

		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Create message data
		eventData, err := json.Marshal(complexEvent)
		if err != nil {
			t.Fatalf("Failed to Marshal event data: %v", err)
		}

		// Create a mock message
		mockMessage := &MockMessage{
			id:   "complex-msg-1",
			data: eventData,
		}

		// Mock the extractor
		p.extractor = func(_ []byte) (pubsub.MessageType, error) {
			return complexEventType, nil
		}

		// Process the message
		err = p.Process(context.Background(), mockMessage)
		if err != nil {
			t.Errorf("Process returned error: %v", err)
		}

		// Verify handler was called with correct data
		if !handlerCalled {
			t.Error("Handler was not called")
		}

		if capturedEvent.UserID != complexEvent.UserID {
			t.Errorf("Wrong UserID: got %s, want %s", capturedEvent.UserID, complexEvent.UserID)
		}

		if capturedEvent.Email != complexEvent.Email {
			t.Errorf("Wrong Email: got %s, want %s", capturedEvent.Email, complexEvent.Email)
		}

		if len(capturedEvent.Items) != len(complexEvent.Items) {
			t.Errorf("Wrong number of items: got %d, want %d", len(capturedEvent.Items), len(complexEvent.Items))
		}

		// Verify message was acked
		if !mockMessage.ackCalled {
			t.Error("Message was not acked")
		}
	})

	t.Run("UnmarshalError", func(t *testing.T) {
		p := New()

		// Register typed handler
		testEventType := pubsub.MessageType("test-event")
		err := Register(p, testEventType,
			func(context.Context, TestEvent) error {
				// This should not be called
				t.Error("Handler should not be called for invalid data")
				return nil
			},
		)

		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Invalid JSON data
		invalidData := []byte(`{invalid json`)

		mockMessage := &MockMessage{
			id:   "msg-1",
			data: invalidData,
		}

		// Mock the extractor
		p.extractor = func(_ []byte) (pubsub.MessageType, error) {
			return testEventType, nil
		}

		err = p.Process(context.Background(), mockMessage)
		if err == nil {
			t.Error("Process should have returned an error for invalid JSON")
		}

		// Verify message was nacked
		if !mockMessage.nackCalled {
			t.Error("Message was not nacked")
		}
	})

	t.Run("HandlerError", func(t *testing.T) {
		p := New()

		expectedErr := errors.New("handler error")
		testEventType := pubsub.MessageType("test-event")

		// Register typed handler that returns an error
		err := Register(p, testEventType,
			func(context.Context, TestEvent) error {
				return expectedErr
			},
		)

		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Create test event
		testEvent := TestEvent{
			ID:      "test-123",
			Message: "hello world",
		}

		eventData, _ := json.Marshal(testEvent)

		mockMessage := &MockMessage{
			id:   "msg-1",
			data: eventData,
		}

		// Mock the extractor
		p.extractor = func(_ []byte) (pubsub.MessageType, error) {
			return testEventType, nil
		}

		err = p.Process(context.Background(), mockMessage)
		if err != expectedErr {
			t.Errorf("Process returned wrong error: got %v, want %v", err, expectedErr)
		}

		// Verify message was nacked
		if !mockMessage.nackCalled {
			t.Error("Message was not nacked")
		}
	})
}
