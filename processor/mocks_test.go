package processor

// MockMessage implements the pubsub.MessageAcker interface for testing
type MockMessage struct {
	id         string
	data       []byte
	attributes map[string]string
	metadata   map[string]any
	ackCalled  bool
	nackCalled bool
}

func NewMockMessage(id string, data []byte, attributes map[string]string, metadata map[string]any) *MockMessage {
	return &MockMessage{
		id:         id,
		data:       data,
		attributes: attributes,
		metadata:   metadata,
	}
}

func (m *MockMessage) ID() string                    { return m.id }
func (m *MockMessage) Data() []byte                  { return m.data }
func (m *MockMessage) Attributes() map[string]string { return m.attributes }
func (m *MockMessage) Metadata() map[string]any      { return m.metadata }
func (m *MockMessage) Ack()                          { m.ackCalled = true }
func (m *MockMessage) Nack()                         { m.nackCalled = true }
