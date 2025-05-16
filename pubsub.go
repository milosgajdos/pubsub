package pubsub

import (
	"context"
)

// Message represents a message in the PubSub system
type Message interface {
	// ID returns the unique identifier of the message
	ID() string
	// Data returns the payload of the message
	Data() []byte
	// Metadata returns the metadata of the message
	Metadata() map[string]any
	// Attributes returns the attributes of the message
	Attributes() map[string]string
}

// MessageAcker is a message
// that can be acked or nacked
type MessageAcker interface {
	Message
	// Ack acknowledges the message
	Ack()
	// Nack nacks the message
	Nack()
}

// MessageHandler processes incoming messages from the PubSub system.
// The handler is responsible for acking or nacking messages
type MessageHandler func(ctx context.Context, msg MessageAcker) error

// MessageUnmarshaler defines how to unmarshal a message's data
type MessageUnmarshaler interface {
	// Unmarshal converts raw message data into a specific type
	Unmarshal(data []byte, v any) error
}

// MessageType is the type of the message that needs to be processed
type MessageType string

// MessageTypeExtractor extracts message type from data and returns it
type MessageTypeExtractor func(data []byte) (MessageType, error)

// MessageProcessor handles message routing to appropriate handlers
type MessageProcessor interface {
	// Register adds a handler for a specific message type
	Register(msgType MessageType, handler MessageHandler) error
	// Process routes a message to appropriate handler based on type
	Process(ctx context.Context, msg MessageAcker) error
}

// Publisher is responsible for publishing messages
type Publisher interface {
	// Publish publishes a message to the PubSub system
	Publish(ctx context.Context, message Message) (string, error)
	// PublishBatch publishes a batch of messages to the PubSub system
	PublishBatch(ctx context.Context, messages []Message) ([]string, error)
}

// Subscriber is responsible for consuming messages
type Subscriber interface {
	// Subscribe registers a handler for incoming messages
	Subscribe(ctx context.Context, handler MessageHandler) error
}
