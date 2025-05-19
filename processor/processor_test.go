package processor

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/milosgajdos/pubsub"
)

// Simple test extractor that always returns a fixed type
func testExtractor(_ []byte) (pubsub.MessageType, error) {
	return pubsub.MessageType("test-type"), nil
}

// Test extractor that always returns an error
func errorExtractor(_ []byte) (pubsub.MessageType, error) {
	return "", errors.New("extraction failed")
}

func TestNew(t *testing.T) {
	t.Run("DefaultOptions", func(t *testing.T) {
		p := New()
		if p == nil {
			t.Fatal("New() returned nil")
		}
		if p.handlers == nil {
			t.Error("handlers map was not initialized")
		}
		if p.extractor == nil {
			t.Error("extractor was not set")
		}
	})

	t.Run("CustomExtractor", func(t *testing.T) {
		p := New(pubsub.WithExtractor(testExtractor))
		if p == nil {
			t.Fatal("New() with custom extractor returned nil")
		}

		msgType, _ := p.extractor([]byte("test"))
		if msgType != "test-type" {
			t.Errorf("Custom extractor not set correctly")
		}
	})
}

func TestDefaultExtractor(t *testing.T) {
	t.Run("ValidData", func(t *testing.T) {
		msgType := "test-message-type"
		data := []byte(`{"type":"` + msgType + `"}`)
		extractedType, err := DefaultExtractor(data)
		if err != nil {
			t.Errorf("DefaultExtractor returned unexpected error: %v", err)
		}
		if extractedType != pubsub.MessageType(msgType) {
			t.Errorf("DefaultExtractor returned incorrect type: got %s, want %s", extractedType, msgType)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		data := []byte(`{invalid json`)
		msgType, err := DefaultExtractor(data)
		if !errors.Is(err, ErrUnknownMessageType) {
			t.Errorf("Expected %v, got: %v", ErrUnknownMessageType, err)
		}
		if msgType != "" {
			t.Errorf("Expected empty message type, got: %s", msgType)
		}
	})

	t.Run("MissingTypeField", func(t *testing.T) {
		data := []byte(`{"message":"no type field"}`)
		msgType, err := DefaultExtractor(data)
		if !errors.Is(err, ErrUnknownMessageType) {
			t.Errorf("Expected %v, got: %v", ErrUnknownMessageType, err)
		}
		if msgType != "" {
			t.Errorf("Expected empty message type, got: %s", msgType)
		}
	})
}

func TestRegister(t *testing.T) {
	t.Run("ValidRegistration", func(t *testing.T) {
		p := New()
		msgType := pubsub.MessageType("test-type")
		handler := func(context.Context, pubsub.MessageAcker) error {
			return nil
		}

		err := p.Register(msgType, handler)
		if err != nil {
			t.Errorf("Register returned unexpected error: %v", err)
		}

		p.mu.RLock()
		registeredHandler, exists := p.handlers[msgType]
		p.mu.RUnlock()

		if !exists {
			t.Error("Handler was not registered")
		}
		// gotta make sure we get the same handler back
		if reflect.ValueOf(registeredHandler).Pointer() != reflect.ValueOf(handler).Pointer() {
			t.Error("Registered handler does not match the provided handler")
		}
	})

	t.Run("EmptyMessageType", func(t *testing.T) {
		p := New()
		err := p.Register("", func(context.Context, pubsub.MessageAcker) error {
			return nil
		})

		if err != ErrInvalidMessageType {
			t.Errorf("Expected ErrInvalidMessageType, got: %v", err)
		}
	})

	t.Run("NilHandler", func(t *testing.T) {
		p := New()
		err := p.Register("test-type", nil)
		if !errors.Is(err, ErrInvalidHandler) {
			t.Errorf("Expected %v, got: %v", ErrInvalidHandler, err)
		}
	})
}

func TestProcess(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessfulProcessing", func(t *testing.T) {
		p := New(pubsub.WithExtractor(testExtractor))

		// Prepare test data
		msgData := []byte("test message data")
		msg := NewMockMessage("test-id", msgData, nil, nil)
		handlerCalled := false

		err := p.Register("test-type", func(context.Context, pubsub.MessageAcker) error {
			handlerCalled = true
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		err = p.Process(ctx, msg)
		if err != nil {
			t.Errorf("Process returned unexpected error: %v", err)
		}

		// Verify handler was called
		if !handlerCalled {
			t.Error("Handler was not called")
		}
	})

	t.Run("ExtractorError", func(t *testing.T) {
		p := New(pubsub.WithExtractor(errorExtractor))

		msg := NewMockMessage("test-id", []byte("test data"), nil, nil)

		err := p.Process(ctx, msg)
		if err == nil {
			t.Error("Process should have returned an error")
		}

		// Verify message was nacked
		if !msg.nackCalled {
			t.Error("Message should have been nacked")
		}
	})

	t.Run("NoHandlerForType", func(t *testing.T) {
		p := New(pubsub.WithExtractor(testExtractor))

		msg := NewMockMessage("test-id", []byte("test data"), nil, nil)

		// No handler registered, so Process should return an error
		err := p.Process(ctx, msg)
		if err == nil {
			t.Error("Process should have returned an error")
		}

		// Verify message was nacked
		if !msg.nackCalled {
			t.Error("Message should have been nacked")
		}
	})

	t.Run("HandlerError", func(t *testing.T) {
		p := New(pubsub.WithExtractor(testExtractor))

		// Prepare test data
		msgData := []byte("test message data")
		msg := NewMockMessage("test-id", msgData, nil, nil)
		expectedErr := errors.New("handler error")

		// Register handler that returns an error
		err := p.Register("test-type", func(context.Context, pubsub.MessageAcker) error {
			return expectedErr
		})
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		// Process message
		err = p.Process(ctx, msg)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Process returned wrong error: got %v, want %v", err, expectedErr)
		}
	})
}
