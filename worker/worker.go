package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/milosgajdos/pubsub"
)

// Worker continuously processes messages from a Subscriber
// using a MessageProcessor to route them to appropriate handlers
type Worker struct {
	subscriber pubsub.Subscriber
	processor  pubsub.MessageProcessor
	running    atomic.Bool
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// New creates a new worker
func New(subscriber pubsub.Subscriber, processor pubsub.MessageProcessor) (*Worker, error) {
	if subscriber == nil {
		return nil, ErrNilSubscriber
	}

	if processor == nil {
		return nil, ErrNilProcessor
	}

	return &Worker{
		subscriber: subscriber,
		processor:  processor,
	}, nil
}

// Start begins processing messages
func (w *Worker) Start(ctx context.Context) error {
	if !w.running.CompareAndSwap(false, true) {
		return ErrWorkerRunning
	}

	w.mu.Lock()
	ctx, cancel := context.WithCancel(ctx)
	w.cancelFunc = cancel
	w.mu.Unlock()

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		err := w.subscriber.Subscribe(ctx, func(msgCtx context.Context, msg pubsub.MessageAcker) error {
			return w.processor.Process(msgCtx, msg)
		})

		// When Subscribe returns, mark as not running
		w.running.Store(false)

		if err != nil && ctx.Err() == nil {
			fmt.Printf("Subscriber error: %v\n", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop(timeout time.Duration) error {
	if !w.running.Load() {
		return ErrWorkerNotRunning
	}

	w.mu.Lock()
	if w.cancelFunc == nil {
		w.mu.Unlock()
		return ErrWorkerNotRunning
	}

	// Signal cancellation
	cancel := w.cancelFunc
	w.mu.Unlock()

	// Cancel the context to stop receiving
	cancel()

	// Wait for graceful shutdown with timeout
	c := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		return nil
	case <-time.After(timeout):
		return errors.New("worker shutdown timed out")
	}
}

// IsRunning returns whether the worker is currently running
func (w *Worker) IsRunning() bool {
	return w.running.Load()
}
