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
	InitBackoffSeconds int
	MaxBackoffSeconds  int
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
