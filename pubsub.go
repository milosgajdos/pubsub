package pubsub

import (
	"context"
)

// Message represents a message in the PubSub system
type Message interface {
	ID() string
	Data() []byte
	Metadata() map[string]any
	Attributes() map[string]string
}

// MessageAcker is a message
// that can be acked or nacked
type MessageAcker interface {
	Message
	Ack()
	Nack()
}

// Publisher is responsible for publishing messages
type Publisher interface {
	Publish(ctx context.Context, message Message) (string, error)
	PublishBatch(ctx context.Context, messages []Message) ([]string, error)
}

// MessageHandler processes incoming messages
// The handler is responsible for acking or nacking messages
type MessageHandler func(ctx context.Context, msg MessageAcker) error

// Subscriber is responsible for consuming messages
type Subscriber interface {
	Subscribe(ctx context.Context, handler MessageHandler) error
}
