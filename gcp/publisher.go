package gcp

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"

	gps "cloud.google.com/go/pubsub"
	vkit "cloud.google.com/go/pubsub/apiv1"
	gax "github.com/googleapis/gax-go/v2"

	"github.com/milosgajdos/pubsub"
	"github.com/milosgajdos/pubsub/tracing"
)

// Publisher implements the pubsub.Publisher interface for Google Cloud Pub/Sub
// TODO: client and topic should be interfaces so we can write unit tests.
type Publisher struct {
	projectID      string
	client         *gps.Client
	topic          *gps.Topic
	settings       *gps.PublishSettings
	tracingEnabled bool
}

// NewPublisher creates a new Google Cloud Pub/Sub publisher
func NewPublisher(ctx context.Context, projectID string, options ...pubsub.PubOption) (*Publisher, error) {
	if projectID == "" {
		return nil, ErrInvalidProject
	}

	opts := &pubsub.PubOptions{}
	for _, option := range options {
		option(opts)
	}

	if opts.Topic == "" {
		return nil, ErrInvalidTopic
	}

	// Create a client with the retry configuration
	clientConfig := pubClientConfig(opts.Retry)
	client, err := gps.NewClientWithConfig(ctx, projectID, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher pubsub client: %w", err)
	}

	// Create or get topic
	topic := client.Topic(opts.Topic)

	// Configure topic settings based on options
	publishSettings := &gps.PublishSettings{}

	if opts.Batch != nil {
		publishSettings.CountThreshold = opts.Batch.Size
	}

	topic.PublishSettings = *publishSettings

	return &Publisher{
		projectID:      projectID,
		client:         client,
		topic:          topic,
		settings:       publishSettings,
		tracingEnabled: opts.TracingEnabled,
	}, nil
}

// Publish publishes a message to the topic
func (p *Publisher) Publish(ctx context.Context, message pubsub.Message) (string, error) {
	// Create message attributes
	attributes := message.Attributes()

	var span trace.Span
	// Add tracing context to attributes if tracing is enabled
	if p.tracingEnabled {
		ctx, span = tracing.StartPublishSpan(ctx, p.topic.ID())
		defer span.End()

		// Inject tracing context into message attributes
		attributes = tracing.InjectTracingContext(ctx, attributes)
	}

	result := p.topic.Publish(ctx, &gps.Message{
		Data:       message.Data(),
		Attributes: attributes,
	})

	// Block until the result is returned and ID is assigned
	id, err := result.Get(ctx)

	// Record any errors in the span if tracing is enabled
	if err != nil && p.tracingEnabled {
		span.RecordError(err)
	}

	return id, err
}

// PublishBatch publishes a batch of messages to the topic
func (p *Publisher) PublishBatch(ctx context.Context, messages []pubsub.Message) ([]string, error) {
	// Start a batch publish span if tracing is enabled
	var span trace.Span
	if p.tracingEnabled {
		ctx, span = tracing.StartPublishSpan(ctx, p.topic.ID()+"-batch")
		defer span.End()
	}

	ids := make([]string, 0, len(messages))
	var lastErr error

	for _, msg := range messages {
		id, err := p.Publish(ctx, msg)
		if err != nil {
			lastErr = err
			continue
		}
		ids = append(ids, id)
	}

	if lastErr != nil && len(ids) < len(messages) {
		// Record error in span if tracing is enabled
		if p.tracingEnabled {
			span.RecordError(lastErr)
		}
		return ids, fmt.Errorf("failed to publish some messages: %w", lastErr)
	}

	return ids, nil
}

// Close closes the publisher
func (p *Publisher) Close() error {
	p.topic.Stop()
	return p.client.Close()
}

// pubClientConfig creates a publisher client config with retry settings
func pubClientConfig(retry *pubsub.Retry) *gps.ClientConfig {
	initialRetry := DefaultInitRetryBackoff
	if retry != nil {
		initialRetry = time.Duration(retry.InitBackoffSeconds) * time.Second
	}
	maxBackoff := DefaultMaxRetryBackoff
	if retry != nil {
		maxBackoff = time.Duration(retry.MaxBackoffSeconds) * time.Second
	}

	return &gps.ClientConfig{
		PublisherCallOptions: &vkit.PublisherCallOptions{
			Publish: []gax.CallOption{
				gax.WithRetry(func() gax.Retryer {
					return gax.OnCodes([]codes.Code{
						codes.Aborted,
						codes.Canceled,
						codes.Internal,
						codes.ResourceExhausted,
						codes.Unknown,
						codes.Unavailable,
						codes.DeadlineExceeded,
					}, gax.Backoff{
						Initial:    initialRetry,
						Max:        maxBackoff,
						Multiplier: 1.45,
					})
				}),
			},
		},
	}
}
