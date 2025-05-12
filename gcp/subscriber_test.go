package gcp

import (
	"context"
	"testing"

	basepubsub "github.com/milosgajdos/pubsub"
)

func TestNewSubscriber(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid project ID", func(t *testing.T) {
		_, err := NewSubscriber(ctx, "", basepubsub.WithSub("test-subscription"))
		if err != ErrInvalidProject {
			t.Errorf("expected ErrInvalidProject, got %v", err)
		}
	})

	t.Run("missing subscription ID", func(t *testing.T) {
		_, err := NewSubscriber(ctx, "test-project")
		if err != ErrInvalidSubsciption {
			t.Errorf("expected ErrInvalidSubsciption, got %v", err)
		}
	})
}

func TestSubscriberAlreadySubscribed(t *testing.T) {
	ctx := context.Background()

	// Create a mock MessageHandler
	handler := func(_ context.Context, _ basepubsub.MessageAcker) error {
		return nil
	}

	// Create a subscriber with the isSubscribed flag set to true
	subscriber := &Subscriber{
		isSubscribed: true,
	}

	// Try to subscribe
	err := subscriber.Subscribe(ctx, handler)

	// Verify it returns the already subscribed error
	if err != ErrAlreadySubscribed {
		t.Errorf("expected ErrAlreadySubscribed, got %v", err)
	}
}
