package pubsub

// PubOptions is used for configuring publishers
type PubOptions struct {
	// Topic is topic name
	Topic string
	// Batch configures batch settings
	Batch *Batch
	// Retry configures retry settings
	Retry *Retry
}

// PubOption is used for configuring publishers
type PubOption func(*PubOptions)

// Batch is used for batch settings
type Batch struct {
	// Size is batch size
	Size int
}

// WithTopic is used for setting topic name
func WithTopic(topic string) PubOption {
	return func(o *PubOptions) {
		o.Topic = topic
	}
}

// WithBatch is used for setting batch size
func WithBatch(batch Batch) PubOption {
	return func(o *PubOptions) {
		o.Batch = &batch
	}
}

// WithPubRetry is used for setting retry settings
func WithPubRetry(retry Retry) PubOption {
	return func(o *PubOptions) {
		o.Retry = &retry
	}
}

// Retry is used for retry settings
type Retry struct {
	// InitBackoffSeconds is initial backoff in seconds
	InitBackoffSeconds int
	// MaxBackoffSeconds is maximum backoff in seconds
	MaxBackoffSeconds int
}

// SubOptions is used for configuring subscribers
type SubOptions struct {
	// Sub is subscription name
	Sub string
	// Retry configures retry settings
	Retry *Retry
	// Concurrency configures concurrency settings
	Concurrency int
}

// SubOption is used for configuring subscribers
type SubOption func(*SubOptions)

// WithSub is used for setting subscription name
func WithSub(sub string) SubOption {
	return func(o *SubOptions) {
		o.Sub = sub
	}
}

// WithSubRetry is used for setting retry settings
func WithSubRetry(retry Retry) SubOption {
	return func(o *SubOptions) {
		o.Retry = &retry
	}
}

// WithConcurrency is used for setting concurrency settings
func WithConcurrency(concurrency int) SubOption {
	return func(o *SubOptions) {
		o.Concurrency = concurrency
	}
}

// MessageOptions is used for configuring messages
type MessageOptions struct {
	// Metadata is message metadata
	Metadata map[string]any
}

// MessageOption is used for configuring messages
type MessageOption func(*MessageOptions)

// WithMetadata is used for setting message metadata
func WithMetadata(metadata map[string]any) MessageOption {
	return func(o *MessageOptions) {
		o.Metadata = metadata
	}
}

// ProcessorOptions is used for configuring processors
type ProcessorOptions struct {
	// Unmarshaler is used for unmarshaling messages
	Unmarshaler MessageUnmarshaler
	// Extractor is used for extracting message type
	Extractor MessageTypeExtractor
}

// ProcessorOption is used for configuring processors
type ProcessorOption func(*ProcessorOptions)

// WithUnmarshaler sets the unmarshaler for the processor
func WithUnmarshaler(unmarshaler MessageUnmarshaler) ProcessorOption {
	return func(p *ProcessorOptions) {
		p.Unmarshaler = unmarshaler
	}
}

// WithExtractor sets the extractor for the processor
func WithExtractor(extractor MessageTypeExtractor) ProcessorOption {
	return func(p *ProcessorOptions) {
		p.Extractor = extractor
	}
}
