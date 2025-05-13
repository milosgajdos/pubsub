package gcp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	basepubsub "github.com/milosgajdos/pubsub"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"github.com/milosgajdos/pubsub/gcp"
)

func TestPublisherIntegration(t *testing.T) {
	pubTopicID := "pub-test-topic"
	pubSubID := "pub-test-subscription"

	// Set up the emulator, client, topic, and subscription
	client, _, sub, cleanup := gcp.SetupPubSub(t, pubTopicID, pubSubID)
	defer cleanup()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a publisher
	publisher, err := gcp.NewPublisher(
		ctx,
		gcp.TestProjectID,
		basepubsub.WithTopic(pubTopicID),
		basepubsub.WithBatch(basepubsub.Batch{Size: 10}),
	)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Test data for publishing
	numMessages := 5
	publishedIDs := make([]string, numMessages)

	// Publish messages individually
	for i := range numMessages {
		pubsubMsg := &pubsub.Message{
			Data: fmt.Appendf(nil, "test-publish-message-%d", i),
			Attributes: map[string]string{
				"publisher-test": "true",
				"sequence":       fmt.Sprintf("%d", i),
				"id":             uuid.New().String(),
			},
		}
		message := gcp.NewMessage(pubsubMsg)

		msgID, err := publisher.Publish(ctx, message)
		if err != nil {
			t.Fatalf("Failed to publish message: %v", err)
		}
		publishedIDs[i] = msgID
		t.Logf("Published message with ID: %s", msgID)
	}

	// Also test batch publishing
	batchMessages := make([]basepubsub.Message, numMessages)
	for i := range numMessages {
		pubsubMsg := &pubsub.Message{
			Data: fmt.Appendf(nil, "test-batch-message-%d", i),
			Attributes: map[string]string{
				"batch-test": "true",
				"sequence":   fmt.Sprintf("%d", i),
				"id":         uuid.New().String(),
			},
		}
		batchMessages[i] = gcp.NewMessage(pubsubMsg)
	}

	batchIDs, err := publisher.PublishBatch(ctx, batchMessages)
	if err != nil {
		t.Fatalf("Failed to publish batch: %v", err)
	}

	for i, id := range batchIDs {
		t.Logf("Published batch message %d with ID: %s", i, id)
	}

	// Setup receiving messages to verify publishing worked
	var receivedMutex sync.Mutex
	receivedMessages := make([]*pubsub.Message, 0, numMessages*2)
	receiveCtx, receiveCancel := context.WithTimeout(ctx, 10*time.Second)
	defer receiveCancel()

	// Start receiving messages
	t.Log("Starting to receive messages...")
	err = sub.Receive(receiveCtx, func(_ context.Context, msg *pubsub.Message) {
		receivedMutex.Lock()
		receivedMessages = append(receivedMessages, msg)
		receivedMutex.Unlock()
		msg.Ack()
	})
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		t.Errorf("Error receiving messages: %v", err)
	}

	// Verify received messages
	receivedMutex.Lock()
	defer receivedMutex.Unlock()

	t.Logf("Received %d messages", len(receivedMessages))

	if len(receivedMessages) < numMessages {
		t.Errorf("Expected at least %d messages, got %d", numMessages, len(receivedMessages))
	}

	// Count individual and batch messages
	individualCount := 0
	batchCount := 0

	for _, msg := range receivedMessages {
		if _, ok := msg.Attributes["publisher-test"]; ok {
			individualCount++
			t.Logf("Received individual message: %s", string(msg.Data))
		}

		if _, ok := msg.Attributes["batch-test"]; ok {
			batchCount++
			t.Logf("Received batch message: %s", string(msg.Data))
		}
	}

	t.Logf("Received %d individual messages and %d batch messages", individualCount, batchCount)

	if individualCount < numMessages {
		t.Errorf("Expected %d individual messages, got %d", numMessages, individualCount)
	}

	if batchCount < numMessages {
		t.Errorf("Expected %d batch messages, got %d", numMessages, batchCount)
	}
}
