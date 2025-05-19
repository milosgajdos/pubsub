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
		if !errors.Is(err, ErrNilSubscriber) {
			t.Errorf("Expected %v, got: %v", ErrNilSubscriber, err)
		}

		if worker != nil {
			t.Error("Worker should be nil when subscriber is nil")
		}
	})

	t.Run("NilProcessor", func(t *testing.T) {
		subscriber := &MockSubscriber{}

		worker, err := New(subscriber, nil)
		if !errors.Is(err, ErrNilProcessor) {
			t.Errorf("Expected %v, got: %v", ErrNilProcessor, err)
		}

		if worker != nil {
			t.Error("Worker should be nil when processor is nil")
		}
	})
}

func TestWorkerStart(t *testing.T) {
	t.Run("SuccessfulStart", func(t *testing.T) {
		subscribeCalled := make(chan struct{})
		subscriber := &MockSubscriber{
			SubscribeFn: func(ctx context.Context, _ pubsub.MessageHandler) error {
				// Signal that Subscribe was called
				close(subscribeCalled)
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

		// Wait for Subscribe to be called
		select {
		case <-subscribeCalled:
			// Subscribe was called, continue
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for Subscribe to be called")
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
		errorReturned := make(chan struct{})
		subscriber := &MockSubscriber{
			SubscribeFn: func(context.Context, pubsub.MessageHandler) error {
				defer close(errorReturned)
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

		// Wait for subscription to fail
		select {
		case <-errorReturned:
			// Error was returned, continue
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for Subscribe to return error")
		}

		if worker.IsRunning() {
			t.Error("Worker should not be running after Subscribe fails")
		}
	})
}

func TestWorkerStop(t *testing.T) {
	t.Run("SuccessfulStop", func(t *testing.T) {
		subscribeCalled := make(chan struct{})
		subscriber := &MockSubscriber{
			SubscribeFn: func(ctx context.Context, _ pubsub.MessageHandler) error {
				close(subscribeCalled)
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

		// Wait for Subscribe to be called
		select {
		case <-subscribeCalled:
			// Subscribe was called, continue
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for Subscribe to be called")
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
		processCalled := make(chan struct{})
		handlerCalled := make(chan struct{})

		// Create a subscriber that delivers a message to the handler
		subscriber := &MockSubscriber{
			SubscribeFn: func(ctx context.Context, handler pubsub.MessageHandler) error {
				msg := NewMockMessage("test-id", []byte("test-data"))
				// Call the handler with the message
				_ = handler(ctx, msg)
				close(handlerCalled)
				// Wait until context is cancelled
				<-ctx.Done()
				return ctx.Err()
			},
		}

		// Track if Process was called
		processor := &MockProcessor{
			ProcessFn: func(_ context.Context, msg pubsub.MessageAcker) error {
				close(processCalled)
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

		// Wait for processor to be called
		select {
		case <-processCalled:
			// Process was called, continue
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for Process to be called")
		}

		// Verify handler was called
		select {
		case <-handlerCalled:
			// Handler was called, continue
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for handler to be called")
		}

		// Clean up
		err = worker.Stop(100 * time.Millisecond)
		if err != nil {
			t.Errorf("Stop() returned unexpected error: %v", err)
		}
	})
}
