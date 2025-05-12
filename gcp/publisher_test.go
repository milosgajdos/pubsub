package gcp

import (
	"context"
	"testing"

	basepubsub "github.com/milosgajdos/pubsub"
)

func TestNewPublisher(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid project ID", func(t *testing.T) {
		_, err := NewPublisher(ctx, "", basepubsub.WithTopic("test-topic"))
		if err != ErrInvalidProject {
			t.Errorf("expected ErrInvalidProject, got %v", err)
		}
	})

	t.Run("missing topic ID", func(t *testing.T) {
		_, err := NewPublisher(ctx, "test-project")
		if err != ErrInvalidTopic {
			t.Errorf("expected ErrInvalidTopic, got %v", err)
		}
	})
}
