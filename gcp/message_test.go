package gcp

import (
	"reflect"
	"testing"

	gps "cloud.google.com/go/pubsub"

	"github.com/milosgajdos/pubsub"
)

func TestNeMessage(t *testing.T) {
	mockPubsubMsg := &gps.Message{
		ID:         "test-id",
		Data:       []byte("test-data"),
		Attributes: map[string]string{"key": "value"},
	}

	metadata := map[string]any{"meta-key": "meta-value"}

	msg := NewMessage(mockPubsubMsg, pubsub.WithMetadata(metadata))
	if msg == nil {
		t.Fatal("NeMessage returned nil")
	}

	if !reflect.DeepEqual(msg.msg, mockPubsubMsg) {
		t.Error("Message does not contain the expected pubsub.Message")
	}

	if !reflect.DeepEqual(msg.metadata, metadata) {
		t.Errorf("Message metadata does not match expected: got %v, want %v", msg.metadata, metadata)
	}
}

func TestMessage(t *testing.T) {
	t.Run("ID", func(t *testing.T) {
		expectedID := "test-message-id"
		mockPubsubMsg := &gps.Message{ID: expectedID}
		msg := &Message{msg: mockPubsubMsg}

		if id := msg.ID(); id != expectedID {
			t.Errorf("ID() = %v, ant %v", id, expectedID)
		}
	})

	t.Run("Data", func(t *testing.T) {
		expectedData := []byte("test message data")
		mockPubsubMsg := &gps.Message{Data: expectedData}
		msg := &Message{msg: mockPubsubMsg}

		if data := msg.Data(); !reflect.DeepEqual(data, expectedData) {
			t.Errorf("Data() = %v, ant %v", data, expectedData)
		}
	})

	t.Run("Attributes", func(t *testing.T) {
		expectedAttrs := map[string]string{
			"attr1": "value1",
			"attr2": "value2",
		}
		mockPubsubMsg := &gps.Message{Attributes: expectedAttrs}
		msg := &Message{msg: mockPubsubMsg}

		if attrs := msg.Attributes(); !reflect.DeepEqual(attrs, expectedAttrs) {
			t.Errorf("Attributes() = %v, ant %v", attrs, expectedAttrs)
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		t.Run("ExistingMetadata", func(t *testing.T) {
			existingMetadata := map[string]any{"key": "value"}
			msg := &Message{metadata: existingMetadata}

			if metadata := msg.Metadata(); !reflect.DeepEqual(metadata, existingMetadata) {
				t.Errorf("Metadata() = %v, ant %v", metadata, existingMetadata)
			}
		})

		t.Run("NilMetadata", func(t *testing.T) {
			// Test ith nil metadata (should initialize a new map)
			msg := &Message{metadata: nil}
			metadata := msg.Metadata()

			if metadata == nil {
				t.Error("Metadata() should initialize a new map when metadata is nil")
			}

			if len(metadata) != 0 {
				t.Errorf("Nely initialized metadata should be empty, got %v", metadata)
			}
		})
	})
}
