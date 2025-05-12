package gcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc/codes"

	vkit "cloud.google.com/go/pubsub/apiv1"
	gax "github.com/googleapis/gax-go/v2"
	basepubsub "github.com/milosgajdos/pubsub"
)

// DefaultMaxOutstandingMessages is the default maximum number of outstanding messages
const DefaultMaxOutstandingMessages = 10

// Subscriber implements the basepubsub.Subscriber interface for Google Cloud Pub/Sub
type Subscriber struct {
	projectID      string
	subscriptionID string
	isSubscribed   bool
	client         *pubsub.Client
	subscription   *pubsub.Subscription
	opts           *pubsub.ReceiveSettings
	mu             sync.Mutex
}

// NewSubscriber creates a new Google Cloud Pub/Sub subscriber
func NewSubscriber(ctx context.Context, projectID string, options ...basepubsub.SubOption) (*Subscriber, error) {
	if projectID == "" {
		return nil, ErrInvalidProject
	}

	opts := &basepubsub.SubOptions{}
	for _, option := range options {
		option(opts)
	}

	if opts.Sub == "" {
		return nil, ErrInvalidSubsciption
	}

	// Create a client with the retry configuration
	clientConfig := subClientConfig(opts.Retry)
	client, err := pubsub.NewClientWithConfig(ctx, projectID, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber pubsub client: %w", err)
	}

	// Get the subscription
	subscription := client.Subscription(opts.Sub)

	// Configure subscription settings based on options
	receiveSettings := pubsub.ReceiveSettings{
		MaxOutstandingMessages: DefaultMaxOutstandingMessages,
	}

	if opts.Concurrency > 0 {
		receiveSettings.NumGoroutines = opts.Concurrency
		// MaxOutstandingMessages should be at least 2x the number of goroutines
		// to avoid goroutines being idle
		receiveSettings.MaxOutstandingMessages = 2 * opts.Concurrency
	}

	subscription.ReceiveSettings = receiveSettings

	return &Subscriber{
		projectID:      projectID,
		subscriptionID: opts.Sub,
		client:         client,
		subscription:   subscription,
		opts:           &receiveSettings,
	}, nil
}

// Subscribe starts consuming messages from the subscription
// Subscribe is a blocking call intended to run in a goroutine
func (s *Subscriber) Subscribe(ctx context.Context, handler basepubsub.MessageHandler) error {
	s.mu.Lock()
	if s.isSubscribed {
		s.mu.Unlock()
		return ErrAlreadySubscribed
	}
	s.isSubscribed = true
	s.mu.Unlock()

	// Start receiving messages
	return s.subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Create metadata for the message
		metadata := map[string]any{
			"publishTime":     msg.PublishTime,
			"deliveryAttempt": msg.DeliveryAttempt,
			"orderingKey":     msg.OrderingKey,
		}

		// Create a message that implements the MessageAcker interface
		message := NewMessage(msg, metadata)

		// Process the message with the provided handler
		err := handler(ctx, message)
		if err != nil {
			// If handler returns an error, we nack the message
			// This will cause the message to be redelivered according to
			// the subscription's retry policy
			msg.Nack()
		}
		// If handler succeeds without error, we expect it to have called
		// message.Ack() explicitly. If it didn't, the message will be redelivered
		// when the ack deadline expires.
	})
}

// Close closes the subscriber and releases resources
func (s *Subscriber) Close() error {
	s.mu.Lock()
	s.isSubscribed = false
	s.mu.Unlock()
	return s.client.Close()
}

// subClientConfig creates a subscriber client config with retry settings
func subClientConfig(retry *basepubsub.Retry) *pubsub.ClientConfig {
	// TODO: figure out retries for different types of messages
	// Right now these are uniformly applied to all kinds of messages
	initialRetry := DefaultInitRetryBackoff
	if retry != nil {
		initialRetry = time.Duration(retry.InitBackoffSeconds) * time.Second
	}
	maxBackoff := DefaultMaxRetryBackoff
	if retry != nil {
		maxBackoff = time.Duration(retry.MaxBackoffSeconds) * time.Second
	}

	return &pubsub.ClientConfig{
		SubscriberCallOptions: &vkit.SubscriberCallOptions{
			Pull: []gax.CallOption{
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
			Acknowledge: []gax.CallOption{
				gax.WithRetry(func() gax.Retryer {
					return gax.OnCodes([]codes.Code{
						codes.Aborted,
						codes.Internal,
						codes.ResourceExhausted,
						codes.Unknown,
						codes.Unavailable,
					}, gax.Backoff{
						Initial:    initialRetry,
						Max:        maxBackoff,
						Multiplier: 1.3,
					})
				}),
			},
			ModifyAckDeadline: []gax.CallOption{
				gax.WithRetry(func() gax.Retryer {
					return gax.OnCodes([]codes.Code{
						codes.Aborted,
						codes.Internal,
						codes.ResourceExhausted,
						codes.Unknown,
						codes.Unavailable,
					}, gax.Backoff{
						Initial:    initialRetry,
						Max:        maxBackoff,
						Multiplier: 1.3,
					})
				}),
			},
		},
	}
}
