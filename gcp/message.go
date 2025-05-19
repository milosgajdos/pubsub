package gcp

import (
	gps "cloud.google.com/go/pubsub"

	"github.com/milosgajdos/pubsub"
)

// Message implements the pubsub.MessageAcker interface for GCP Cloud Pub/Sub
type Message struct {
	msg      *gps.Message
	metadata map[string]any
}

// NewMessage creates a new message from a Google Cloud Pub/Sub message
func NewMessage(msg *gps.Message, options ...pubsub.MessageOption) *Message {
	opts := &pubsub.MessageOptions{}
	for _, option := range options {
		option(opts)
	}
	if opts.Metadata == nil {
		opts.Metadata = make(map[string]any)
	}

	return &Message{
		msg:      msg,
		metadata: opts.Metadata,
	}
}

// ID returns the message ID
func (m *Message) ID() string {
	return m.msg.ID
}

// Data returns the message payload
func (m *Message) Data() []byte {
	return m.msg.Data
}

// Attributes returns the message attributes
func (m *Message) Attributes() map[string]string {
	return m.msg.Attributes
}

// Metadata returns the message metadata
func (m *Message) Metadata() map[string]any {
	if m.metadata == nil {
		m.metadata = make(map[string]any)
	}
	return m.metadata
}

// Ack acknowledges the message
func (m *Message) Ack() {
	m.msg.Ack()
}

// Nack negatively acknowledges the message
func (m *Message) Nack() {
	m.msg.Nack()
}
