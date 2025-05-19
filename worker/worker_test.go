package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/milosgajdos/pubsub"
)

func TestNew(t *testing.T) {
	t.Run("ValidCreation", func(t *testing.T) {
		subscriber := &MockSubscriber{}
		processor := &MockProcessor{}

		worker, err := New(subscriber, processor)
		if err != nil {
			t.Errorf("New() returned unexpected error: %v", err)
		}

		if worker == nil {
			t.Fatal("New() returned nil worker")
		}

		if worker.subscriber != subscriber {
			t.Error("Worker has incorrect subscriber")
		}

		if worker.processor != processor {
			t.Error("Worker has incorrect processor")
		}
	})

	t.Run("NilSubscriber", func(t *testing.T) {
		processor := &MockProcessor{}

		worker, err := New(nil, processor)
		if err != ErrNilSubscriber {
			t.Errorf("Expected ErrNilSubscriber, got: %v", err)
		}

		if worker != nil {
			t.Error("Worker should be nil when subscriber is nil")
		}
	})

	t.Run("NilProcessor", func(t *testing.T) {
		subscriber := &MockSubscriber{}

		worker, err := New(subscriber, nil)
		if err != ErrNilProcessor {
			t.Errorf("Expected ErrNilProcessor, got: %v", err)
		}

		if worker != nil {
			t.Error("Worker should be nil when processor is nil")
		}
	})
}

func TestWorkerStart(t *testing.T) {
	t.Run("SuccessfulStart", func(t *testing.T) {
		subscribeCalled := false
		subscriber := &MockSubscriber{
			SubscribeFn: func(ctx context.Context, _ pubsub.MessageHandler) error {
				subscribeCalled = true
				// Keep subscriber running until context is cancelled
				<-ctx.Done()
				return ctx.Err()
			},
		}

		processor := &MockProcessor{}

		worker, err := New(subscriber, processor)
		if err != nil {
			t.Fatalf("Failed to create worker: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		if err != nil {
			t.Errorf("Start() returned unexpected error: %v", err)
		}

		// Give goroutine time to start
		time.Sleep(50 * time.Millisecond)

		if !subscribeCalled {
			t.Error("Subscribe should have been called")
		}

		if !worker.IsRunning() {
			t.Error("Worker should be running")
		}

		// Clean up
		err = worker.Stop(100 * time.Millisecond)
		if err != nil {
			t.Errorf("Stop() returned unexpected error: %v", err)
		}
	})

	t.Run("AlreadyRunning", func(t *testing.T) {
		subscriber := &MockSubscriber{
			SubscribeFn: func(ctx context.Context, _ pubsub.MessageHandler) error {
				// Keep subscriber running until context is cancelled
				<-ctx.Done()
				return ctx.Err()
			},
		}

		processor := &MockProcessor{}

		worker, err := New(subscriber, processor)
		if err != nil {
			t.Fatalf("Failed to create worker: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		if err != nil {
			t.Errorf("First Start() returned unexpected error: %v", err)
		}

		// Try to start again
		err = worker.Start(ctx)
		if err != ErrWorkerRunning {
			t.Errorf("Expected ErrWorkerRunning, got: %v", err)
		}

		// Clean up
		err = worker.Stop(100 * time.Millisecond)
		if err != nil {
			t.Errorf("Stop() returned unexpected error: %v", err)
		}
	})

	t.Run("SubscribeError", func(t *testing.T) {
		expectedErr := errors.New("subscribe error")
		subscriber := &MockSubscriber{
			SubscribeFn: func(context.Context, pubsub.MessageHandler) error {
				return expectedErr
			},
		}

		processor := &MockProcessor{}

		worker, err := New(subscriber, processor)
		if err != nil {
			t.Fatalf("Failed to create worker: %v", err)
		}

		err = worker.Start(context.Background())
		if err != nil {
			t.Errorf("Start() returned unexpected error: %v", err)
		}

		// Give goroutine time to run and fail
		time.Sleep(50 * time.Millisecond)

		if worker.IsRunning() {
			t.Error("Worker should not be running after Subscribe fails")
		}
	})
}

func TestWorkerStop(t *testing.T) {
	t.Run("SuccessfulStop", func(t *testing.T) {
		subscribeCalled := false
		subscriber := &MockSubscriber{
			SubscribeFn: func(ctx context.Context, _ pubsub.MessageHandler) error {
				subscribeCalled = true
				// Keep subscriber running until context is cancelled
				<-ctx.Done()
				return ctx.Err()
			},
		}

		processor := &MockProcessor{}

		worker, err := New(subscriber, processor)
		if err != nil {
			t.Fatalf("Failed to create worker: %v", err)
		}

		err = worker.Start(context.Background())
		if err != nil {
			t.Errorf("Start() returned unexpected error: %v", err)
		}

		// Give goroutine time to start
		time.Sleep(50 * time.Millisecond)

		if !subscribeCalled {
			t.Error("Subscribe should have been called")
		}

		err = worker.Stop(100 * time.Millisecond)
		if err != nil {
			t.Errorf("Stop() returned unexpected error: %v", err)
		}

		if worker.IsRunning() {
			t.Error("Worker should not be running after Stop()")
		}
	})

	t.Run("NotRunning", func(t *testing.T) {
		subscriber := &MockSubscriber{}
		processor := &MockProcessor{}

		worker, err := New(subscriber, processor)
		if err != nil {
			t.Fatalf("Failed to create worker: %v", err)
		}

		err = worker.Stop(100 * time.Millisecond)
		if err != ErrWorkerNotRunning {
			t.Errorf("Expected ErrWorkerNotRunning, got: %v", err)
		}
	})
}

func TestMessageProcessing(t *testing.T) {
	t.Run("MessageToProcessor", func(t *testing.T) {
		// Create a subscriber that delivers a message to the handler
		subscriber := &MockSubscriber{
			SubscribeFn: func(ctx context.Context, handler pubsub.MessageHandler) error {
				// Create a test message
				msg := NewMockMessage("test-id", []byte("test-data"))

				// Call the handler with the message
				_ = handler(ctx, msg)

				// Wait until context is cancelled
				<-ctx.Done()
				return ctx.Err()
			},
		}

		// Track if Process was called
		processCalled := false
		processor := &MockProcessor{
			ProcessFn: func(_ context.Context, msg pubsub.MessageAcker) error {
				processCalled = true
				msg.Ack()
				return nil
			},
		}

		worker, err := New(subscriber, processor)
		if err != nil {
			t.Fatalf("Failed to create worker: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		if err != nil {
			t.Errorf("Start() returned unexpected error: %v", err)
		}

		// Give some time for the handler to be called
		time.Sleep(50 * time.Millisecond)

		// Verify processor was called
		if !processCalled {
			t.Error("Process should have been called")
		}

		// Clean up
		err = worker.Stop(100 * time.Millisecond)
		if err != nil {
			t.Errorf("Stop() returned unexpected error: %v", err)
		}
	})
}
