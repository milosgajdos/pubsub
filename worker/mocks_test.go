package worker

import (
	"context"
	"sync"

	"github.com/milosgajdos/pubsub"
)

// MockSubscriber implements pubsub.Subscriber for testing
type MockSubscriber struct {
	SubscribeFn        func(ctx context.Context, handler pubsub.MessageHandler) error
	SubscribeCallCount int
	mu                 sync.Mutex
}

func (m *MockSubscriber) Subscribe(ctx context.Context, handler pubsub.MessageHandler) error {
	m.mu.Lock()
	m.SubscribeCallCount++
	m.mu.Unlock()
	if m.SubscribeFn != nil {
		return m.SubscribeFn(ctx, handler)
	}
	return nil
}

// MockProcessor implements pubsub.MessageProcessor for testing
type MockProcessor struct {
	RegisterFn        func(msgType pubsub.MessageType, handler pubsub.MessageHandler) error
	RegisterCallCount int
	ProcessFn         func(ctx context.Context, msg pubsub.MessageAcker) error
	ProcessCallCount  int
	mu                sync.Mutex
}

func (m *MockProcessor) Register(msgType pubsub.MessageType, handler pubsub.MessageHandler) error {
	m.mu.Lock()
	m.RegisterCallCount++
	m.mu.Unlock()
	if m.RegisterFn != nil {
		return m.RegisterFn(msgType, handler)
	}
	return nil
}

func (m *MockProcessor) Process(ctx context.Context, msg pubsub.MessageAcker) error {
	m.mu.Lock()
	m.ProcessCallCount++
	m.mu.Unlock()
	if m.ProcessFn != nil {
		return m.ProcessFn(ctx, msg)
	}
	return nil
}

// MockMessage implements pubsub.MessageAcker for testing
type MockMessage struct {
	id         string
	data       []byte
	attributes map[string]string
	metadata   map[string]any
	ackCalled  bool
	nackCalled bool
}

func NewMockMessage(id string, data []byte) *MockMessage {
	return &MockMessage{
		id:         id,
		data:       data,
		attributes: make(map[string]string),
		metadata:   make(map[string]any),
	}
}

func (m *MockMessage) ID() string                    { return m.id }
func (m *MockMessage) Data() []byte                  { return m.data }
func (m *MockMessage) Attributes() map[string]string { return m.attributes }
func (m *MockMessage) Metadata() map[string]any      { return m.metadata }
func (m *MockMessage) Ack()                          { m.ackCalled = true }
func (m *MockMessage) Nack()                         { m.nackCalled = true }
