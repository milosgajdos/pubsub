package gcp

import (
	"context"
	"testing"

	"github.com/milosgajdos/pubsub"
)

func TestNewSubscriber(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid project ID", func(t *testing.T) {
		_, err := NewSubscriber(ctx, "", pubsub.WithSub("test-subscription"))
		if err != ErrInvalidProject {
			t.Errorf("expected %v, got %v", ErrInvalidProject, err)
		}
	})

	t.Run("missing subscription ID", func(t *testing.T) {
		_, err := NewSubscriber(ctx, "test-project")
		if err != ErrInvalidSubsciption {
			t.Errorf("expected %v, got %v", ErrInvalidSubsciption, err)
		}
	})
}

func TestSubscriberAlreadySubscribed(t *testing.T) {
	// Create a subscriber with the isSubscribed flag set to true
	subscriber := &Subscriber{
		isSubscribed: true,
	}

	// Create a mock MessageHandler
	handler := func(_ context.Context, _ pubsub.MessageAcker) error {
		return nil
	}

	err := subscriber.Subscribe(context.Background(), handler)
	if err != ErrAlreadySubscribed {
		t.Errorf("expected %v, got %v", ErrAlreadySubscribed, err)
	}
}
