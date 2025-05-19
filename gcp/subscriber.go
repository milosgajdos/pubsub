package gcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"

	gps "cloud.google.com/go/pubsub"
	vkit "cloud.google.com/go/pubsub/apiv1"
	gax "github.com/googleapis/gax-go/v2"
	otelcodes "go.opentelemetry.io/otel/codes"

	"github.com/milosgajdos/pubsub"
	"github.com/milosgajdos/pubsub/tracing"
)

// Subscriber implements the pubsub.Subscriber interface for Google Cloud Pub/Sub
type Subscriber struct {
	projectID      string
	subscriptionID string
	isSubscribed   bool
	client         *gps.Client
	subscription   *gps.Subscription
	settings       *gps.ReceiveSettings
	tracingEnabled bool
	topicID        string
	mu             sync.Mutex
}

// NewSubscriber creates a new Google Cloud Pub/Sub subscriber
func NewSubscriber(ctx context.Context, projectID string, options ...pubsub.SubOption) (*Subscriber, error) {
	if projectID == "" {
		return nil, ErrInvalidProject
	}

	opts := &pubsub.SubOptions{}
	for _, option := range options {
		option(opts)
	}

	if opts.Sub == "" {
		return nil, ErrInvalidSubsciption
	}

	// Create a client with the retry configuration
	clientConfig := subClientConfig(opts.Retry)
	client, err := gps.NewClientWithConfig(ctx, projectID, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber pubsub client: %w", err)
	}

	// Get the subscription
	subscription := client.Subscription(opts.Sub)

	// Configure subscription settings based on options
	receiveSettings := &gps.ReceiveSettings{
		MaxOutstandingMessages: DefaultMaxOutstandingMessages,
	}

	if opts.Concurrency > 0 {
		receiveSettings.NumGoroutines = opts.Concurrency
		// MaxOutstandingMessages should be at least 2x the number of goroutines
		// to avoid goroutines being idle
		receiveSettings.MaxOutstandingMessages = 2 * opts.Concurrency
	}

	subscription.ReceiveSettings = *receiveSettings

	// Get the topic from the subscription
	topicID := ""
	subConfig, err := subscription.Config(ctx)
	if err == nil && subConfig.Topic != nil {
		topicID = subConfig.Topic.ID()
	}

	return &Subscriber{
		projectID:      projectID,
		subscriptionID: opts.Sub,
		client:         client,
		subscription:   subscription,
		settings:       receiveSettings,
		tracingEnabled: opts.TracingEnabled,
		topicID:        topicID,
	}, nil
}

// Subscribe starts consuming messages from the subscription
// Subscribe is a blocking call intended to run in a goroutine
func (s *Subscriber) Subscribe(ctx context.Context, handler pubsub.MessageHandler) error {
	s.mu.Lock()
	if s.isSubscribed {
		s.mu.Unlock()
		return ErrAlreadySubscribed
	}
	s.isSubscribed = true
	s.mu.Unlock()

	return s.subscription.Receive(ctx, func(ctx context.Context, msg *gps.Message) {
		// Extract message type from attributes if available
		messageType := "unknown"
		if msgType, ok := msg.Attributes["message_type"]; ok {
			messageType = msgType
		}

		// Create metadata for the message
		metadata := map[string]any{
			"publishTime":     msg.PublishTime,
			"deliveryAttempt": msg.DeliveryAttempt,
			"orderingKey":     msg.OrderingKey,
			"messageType":     messageType,
		}

		// Create a message that implements the MessageAcker interface
		message := NewMessage(msg, pubsub.WithMetadata(metadata))

		var span trace.Span
		if s.tracingEnabled {
			ctx = tracing.ExtractTracingContext(ctx, msg.Attributes)
			// Start a new span for message processing
			ctx, span = tracing.StartMessageProcessingSpan(ctx, message.ID(), s.topicID, s.subscriptionID, messageType)
			defer span.End()
		}

		// Process the message with the provided handler
		err := handler(ctx, message)
		if err != nil {
			// If handler returns an error, we nack the message
			// This will cause the message to be redelivered according to
			// the subscription's retry policy
			msg.Nack()

			if s.tracingEnabled {
				span.RecordError(err)
				span.SetStatus(otelcodes.Error, err.Error())
			}
		}
		// If handler succeeds without error, we expect it to have called
		// message.Ack() explicitly. If it didn't, the message will be redelivered
		// when the ack deadline expires.
		// This isnt the ideal design, but it's the choice made. The alternative
		// is either to have an ack deadline or to ack the message in receive which
		// might lead to double-acking of the same message which may have been acked
		// in the handler.
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
func subClientConfig(retry *pubsub.Retry) *gps.ClientConfig {
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

	return &gps.ClientConfig{
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
