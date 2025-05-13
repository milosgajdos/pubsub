package gcp

import (
	"reflect"
	"testing"

	basepubsub "github.com/milosgajdos/pubsub"

	"cloud.google.com/go/pubsub"
)

func TestNewMessage(t *testing.T) {
	// Create mock pubsub.Message
	mockPubsubMsg := &pubsub.Message{
		ID:         "test-id",
		Data:       []byte("test-data"),
		Attributes: map[string]string{"key": "value"},
	}

	metadata := map[string]any{"meta-key": "meta-value"}

	// Create a new Message
	msg := NewMessage(mockPubsubMsg, basepubsub.WithMetadata(metadata))

	// Verify the message was created correctly
	if msg == nil {
		t.Fatal("NewMessage returned nil")
	}

	if msg.msg != mockPubsubMsg {
		t.Errorf("Message does not contain the expected pubsub.Message")
	}

	if !reflect.DeepEqual(msg.metadata, metadata) {
		t.Errorf("Message metadata does not match expected: got %v, want %v", msg.metadata, metadata)
	}
}

func TestMessage_ID(t *testing.T) {
	expectedID := "test-message-id"
	mockPubsubMsg := &pubsub.Message{ID: expectedID}
	msg := &Message{msg: mockPubsubMsg}

	if id := msg.ID(); id != expectedID {
		t.Errorf("ID() = %v, want %v", id, expectedID)
	}
}

func TestMessage_Data(t *testing.T) {
	expectedData := []byte("test message data")
	mockPubsubMsg := &pubsub.Message{Data: expectedData}
	msg := &Message{msg: mockPubsubMsg}

	if data := msg.Data(); !reflect.DeepEqual(data, expectedData) {
		t.Errorf("Data() = %v, want %v", data, expectedData)
	}
}

func TestMessage_Attributes(t *testing.T) {
	expectedAttrs := map[string]string{
		"attr1": "value1",
		"attr2": "value2",
	}
	mockPubsubMsg := &pubsub.Message{Attributes: expectedAttrs}
	msg := &Message{msg: mockPubsubMsg}

	if attrs := msg.Attributes(); !reflect.DeepEqual(attrs, expectedAttrs) {
		t.Errorf("Attributes() = %v, want %v", attrs, expectedAttrs)
	}
}

func TestMessage_Metadata(t *testing.T) {
	// Test with existing metadata
	existingMetadata := map[string]any{"key": "value"}
	msg := &Message{metadata: existingMetadata}

	if metadata := msg.Metadata(); !reflect.DeepEqual(metadata, existingMetadata) {
		t.Errorf("Metadata() = %v, want %v", metadata, existingMetadata)
	}

	// Test with nil metadata (should initialize a new map)
	msg = &Message{metadata: nil}
	metadata := msg.Metadata()

	if metadata == nil {
		t.Error("Metadata() should initialize a new map when metadata is nil")
	}

	if len(metadata) != 0 {
		t.Errorf("Newly initialized metadata should be empty, got %v", metadata)
	}
}
