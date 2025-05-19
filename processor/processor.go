package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/milosgajdos/pubsub"
)

// Processor implements MessageProcessor
type Processor struct {
	handlers  map[pubsub.MessageType]pubsub.MessageHandler
	extractor pubsub.MessageTypeExtractor
	mu        sync.RWMutex
}

// DefaultExtractor is the default message type extractor
// By default it will try to extract message type from a JSON message
// whose type field is "type".
func DefaultExtractor(data []byte) (pubsub.MessageType, error) {
	var msg struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &msg); err == nil && msg.Type != "" {
		return pubsub.MessageType(msg.Type), nil
	}

	return "", ErrUnknownMessageType
}

// New creates a new message processor
func New(options ...pubsub.ProcessorOption) *Processor {
	opts := &pubsub.ProcessorOptions{}
	for _, option := range options {
		option(opts)
	}

	extractor := opts.Extractor
	if extractor == nil {
		extractor = DefaultExtractor
	}

	return &Processor{
		handlers:  make(map[pubsub.MessageType]pubsub.MessageHandler),
		extractor: extractor,
	}
}

// Register adds a handler for a specific message type
func (p *Processor) Register(msgType pubsub.MessageType, handler pubsub.MessageHandler) error {
	if string(msgType) == "" {
		return ErrInvalidMessageType
	}

	if handler == nil {
		return ErrInvalidHandler
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.handlers[msgType] = handler
	return nil
}

// Process routes a message to its appropriate handler based on type
func (p *Processor) Process(ctx context.Context, msg pubsub.MessageAcker) error {
	msgType, err := p.extractor(msg.Data())
	if err != nil {
		msg.Nack()
		return fmt.Errorf("failed to extract message type: %w", err)
	}

	p.mu.RLock()
	handler, exists := p.handlers[msgType]
	p.mu.RUnlock()

	if !exists {
		msg.Nack()
		return fmt.Errorf("no handler registered for message type: %s", msgType)
	}

	return handler(ctx, msg)
}
