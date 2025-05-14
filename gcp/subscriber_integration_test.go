package gcp_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/milosgajdos/pubsub/gcp"

	basepubsub "github.com/milosgajdos/pubsub"
)

func TestSubscriberIntegration(t *testing.T) {
	client, topic, _, cleanup := gcp.SetupPubSub(t, gcp.TestTopicID, gcp.TestSubscriptionID)
	defer cleanup()
	defer client.Close()

	t.Run("BasicSubscription", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Create a subscriber for this test
		subscriber, err := gcp.NewSubscriber(
			ctx,
			gcp.TestProjectID,
			basepubsub.WithSub(gcp.TestSubscriptionID),
			basepubsub.WithConcurrency(2),
		)
		if err != nil {
			t.Fatalf("Failed to create subscriber: %v", err)
		}
		defer subscriber.Close()

		// Setup message reception
		var wg sync.WaitGroup
		receivedMessages := make([]basepubsub.MessageAcker, 0)
		messagesMutex := &sync.Mutex{}

		// Start receiving messages
		wg.Add(1)
		go func() {
			defer wg.Done()

			subCtx, subCancel := context.WithTimeout(ctx, 10*time.Second)
			defer subCancel()

			err := subscriber.Subscribe(subCtx, func(_ context.Context, msg basepubsub.MessageAcker) error {
				messagesMutex.Lock()
				receivedMessages = append(receivedMessages, msg)
				messagesMutex.Unlock()

				msg.Ack()
				return nil
			})

			if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				t.Errorf("Subscribe returned error: %v", err)
			}
		}()

		// Publish test messages to the topic
		numMessages := 5
		sentMessages := make([]string, numMessages)

		for i := range numMessages {
			messageData := fmt.Appendf(nil, "test-message-%d", i)
			messageAttrs := map[string]string{"sequence": fmt.Sprintf("%d", i)}

			result := topic.Publish(ctx, &pubsub.Message{
				Data:       messageData,
				Attributes: messageAttrs,
			})

			id, err := result.Get(ctx)
			if err != nil {
				t.Fatalf("Failed to publish message: %v", err)
			}

			sentMessages[i] = id
			t.Logf("Published message with ID: %s", id)

			// Small delay to avoid overwhelming the emulator
			time.Sleep(100 * time.Millisecond)
		}

		// Wait a bit to ensure messages are processed
		time.Sleep(2 * time.Second)

		// Cancel the context and wait for subscription to finish
		cancel()
		wg.Wait()

		// Verify received messages
		messagesMutex.Lock()
		defer messagesMutex.Unlock()

		t.Logf("Received %d messages", len(receivedMessages))
		if len(receivedMessages) != numMessages {
			t.Errorf("Expected to receive %d messages, got %d", numMessages, len(receivedMessages))
		}

		// Verify message contents
		for i, msg := range receivedMessages {
			msgData := string(msg.Data())
			if msgData == "" {
				t.Errorf("Message %d has empty data", i)
			} else {
				t.Logf("Message data: %s", msgData)
			}

			if _, ok := msg.Attributes()["sequence"]; !ok {
				t.Errorf("Message %d is missing sequence attribute", i)
			}

			// Check message has metadata
			if _, ok := msg.Metadata()["publishTime"]; !ok {
				t.Errorf("Message %d is missing publishTime in metadata", i)
			}
		}
	})

	t.Run("ConcurrentSubscription", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Create a new subscriber for concurrent test with higher concurrency (5)
		subscriber, err := gcp.NewSubscriber(
			ctx,
			gcp.TestProjectID,
			basepubsub.WithSub(gcp.TestSubscriptionID),
			basepubsub.WithConcurrency(5),
		)
		if err != nil {
			t.Fatalf("Failed to create concurrent subscriber: %v", err)
		}
		defer subscriber.Close()

		// Setup for tracking received messages
		var receivedCount atomic.Int32

		// Create a context with timeout
		subCtx, subCancel := context.WithTimeout(ctx, 10*time.Second)
		defer subCancel()

		// Start the subscription in the background
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()

			err := subscriber.Subscribe(subCtx, func(_ context.Context, msg basepubsub.MessageAcker) error {
				t.Logf("Concurrent processing message: %s", string(msg.Data()))
				// Simulate processing time to test concurrency
				time.Sleep(200 * time.Millisecond)
				// Update received count
				receivedCount.Add(1)

				msg.Ack()
				return nil
			})

			if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				t.Errorf("Concurrent subscribe returned error: %v", err)
			}
		}()

		// Publish a batch of messages quickly to test concurrent processing
		batchSize := 10
		for i := range batchSize {
			messageData := fmt.Appendf(nil, "concurrent-msg-%d", i)
			result := topic.Publish(ctx, &pubsub.Message{
				Data: messageData,
				Attributes: map[string]string{
					"test": "concurrent",
					"idx":  fmt.Sprintf("%d", i),
				},
			})

			id, err := result.Get(ctx)
			if err != nil {
				t.Fatalf("Failed to publish concurrent message: %v", err)
			}

			t.Logf("Published concurrent message %d with ID: %s", i, id)
		}

		// Give some time for processing
		time.Sleep(5 * time.Second)

		// Cancel subscription and wait for it to finish
		subCancel()
		wg.Wait()

		// Check results
		count := receivedCount.Load()

		t.Logf("Received %d concurrent messages", count)

		if int(count) < batchSize {
			t.Errorf("Expected to receive at least %d concurrent messages, got %d", batchSize, count)
		}
	})
}
