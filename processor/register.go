package processor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/milosgajdos/pubsub"
)

// Handler is a type-safe handler for a specific message type.
type Handler[T any] func(ctx context.Context, data T) error

// RegisterTyped registers a type-safe handler for a specific message type
func Register[T any](p pubsub.MessageProcessor, msgType pubsub.MessageType, handler Handler[T]) error {
	// Create a wrapper handler that:
	// 1. Unmarshals the message data into type T
	// 2. Calls the handler with the typed data
	wrappedHandler := func(ctx context.Context, msg pubsub.MessageAcker) error {
		var data T
		if err := json.Unmarshal(msg.Data(), &data); err != nil {
			msg.Nack()
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		if err := handler(ctx, data); err != nil {
			msg.Nack()
			return err
		}

		msg.Ack()
		return nil
	}

	return p.Register(msgType, wrappedHandler)
}
