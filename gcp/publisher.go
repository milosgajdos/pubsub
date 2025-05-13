package gcp

import (
	"context"
	"fmt"
	"time"

	vkit "cloud.google.com/go/pubsub/apiv1"
	gax "github.com/googleapis/gax-go/v2"
	basepubsub "github.com/milosgajdos/pubsub"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc/codes"
)

// Publisher implements the basepubsub.Publisher interface for Google Cloud Pub/Sub
type Publisher struct {
	projectID string
	client    *pubsub.Client
	topic     *pubsub.Topic
	opts      *pubsub.PublishSettings
}

// NewPublisher creates a new Google Cloud Pub/Sub publisher
func NewPublisher(ctx context.Context, projectID string, options ...basepubsub.PubOption) (*Publisher, error) {
	if projectID == "" {
		return nil, ErrInvalidProject
	}

	opts := &basepubsub.PubOptions{}
	for _, option := range options {
		option(opts)
	}

	if opts.Topic == "" {
		return nil, ErrInvalidTopic
	}

	// Create a client with the retry configuration
	clientConfig := pubClientConfig(opts.Retry)
	client, err := pubsub.NewClientWithConfig(ctx, projectID, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher pubsub client: %w", err)
	}

	// Create or get topic
	topic := client.Topic(opts.Topic)

	// Configure topic settings based on options
	publishSettings := &pubsub.PublishSettings{}

	if opts.Batch != nil {
		publishSettings.CountThreshold = opts.Batch.Size
	}

	topic.PublishSettings = *publishSettings

	return &Publisher{
		projectID: projectID,
		client:    client,
		topic:     topic,
		opts:      publishSettings,
	}, nil
}

// Publish publishes a message to the topic
func (p *Publisher) Publish(ctx context.Context, message basepubsub.Message) (string, error) {
	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       message.Data(),
		Attributes: message.Attributes(),
	})

	// Block until the result is returned and ID is assigned
	return result.Get(ctx)
}

// PublishBatch publishes a batch of messages to the topic
func (p *Publisher) PublishBatch(ctx context.Context, messages []basepubsub.Message) ([]string, error) {
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
func pubClientConfig(retry *basepubsub.Retry) *pubsub.ClientConfig {
	initialRetry := DefaultInitRetryBackoff
	if retry != nil {
		initialRetry = time.Duration(retry.InitBackoffSeconds) * time.Second
	}
	maxBackoff := DefaultMaxRetryBackoff
	if retry != nil {
		maxBackoff = time.Duration(retry.MaxBackoffSeconds) * time.Second
	}

	return &pubsub.ClientConfig{
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
