package gcp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	basepubsub "github.com/milosgajdos/pubsub"

	"cloud.google.com/go/pubsub"
	"github.com/milosgajdos/pubsub/gcp"
)

func TestSubscriberIntegration(t *testing.T) {
	// Set up the emulator, topic, and subscription
	client, topic, _, cleanup := gcp.SetupPubSub(t, gcp.TestTopicID, gcp.TestSubscriptionID)
	defer cleanup()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a subscriber
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

			// Acknowledge the message
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

		// Small delay to avoid overwhelming the emulator
		time.Sleep(100 * time.Millisecond)
	}

	// Wait a bit to ensure messages are processed
	time.Sleep(2 * time.Second)

	// Cancel the subscription context
	cancel()

	// Wait for subscription to finish
	wg.Wait()

	// Verify received messages
	messagesMutex.Lock()
	defer messagesMutex.Unlock()

	if len(receivedMessages) != numMessages {
		t.Errorf("Expected to receive %d messages, got %d", numMessages, len(receivedMessages))
	}

	// Verify message contents
	for i, msg := range receivedMessages {
		if string(msg.Data()) == "" {
			t.Errorf("Message %d has empty data", i)
		}

		if val, ok := msg.Attributes()["sequence"]; ok {
			t.Logf("Received message with sequence %s: %s", val, string(msg.Data()))
		} else {
			t.Errorf("Message %d is missing sequence attribute", i)
		}

		// Check message has metadata
		if _, ok := msg.Metadata()["publishTime"]; !ok {
			t.Errorf("Message %d is missing publishTime in metadata", i)
		}
	}
}
