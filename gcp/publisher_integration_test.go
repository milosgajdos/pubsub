package gcp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	gps "cloud.google.com/go/pubsub"

	"github.com/milosgajdos/pubsub"
	"github.com/milosgajdos/pubsub/gcp"
)

func TestPublisherIntegration(t *testing.T) {
	pubTopicID := "pub-test-topic"
	pubSubID := "pub-test-subscription"

	// Set up the emulator, client, topic, and subscription
	client, _, sub, cleanup := gcp.SetupPubSub(t, pubTopicID, pubSubID)
	defer cleanup()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a publisher
	publisher, err := gcp.NewPublisher(
		ctx,
		gcp.TestProjectID,
		pubsub.WithTopic(pubTopicID),
		pubsub.WithBatch(pubsub.Batch{Size: 10}),
	)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Define common variables for both subtests
	numMessages := 5
	var receivedMutex sync.Mutex
	receivedMessages := make([]*gps.Message, 0, numMessages*2)

	// Receive message setup
	receiveMessages := func(timeout time.Duration) []*gps.Message {
		receivedMutex.Lock()
		receivedMessages = make([]*gps.Message, 0, numMessages*2)
		receivedMutex.Unlock()

		receiveCtx, receiveCancel := context.WithTimeout(ctx, timeout)
		defer receiveCancel()

		// Start receiving messages
		t.Log("Starting to receive messages...")
		err = sub.Receive(receiveCtx, func(_ context.Context, msg *gps.Message) {
			receivedMutex.Lock()
			receivedMessages = append(receivedMessages, msg)
			receivedMutex.Unlock()
			msg.Ack()
		})
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Error receiving messages: %v", err)
		}

		receivedMutex.Lock()
		result := make([]*gps.Message, len(receivedMessages))
		copy(result, receivedMessages)
		receivedMutex.Unlock()

		return result
	}

	// Run single message publishing subtest
	t.Run("SinglePublish", func(t *testing.T) {
		publishedIDs := make([]string, numMessages)

		// Publish messages individually
		for i := range numMessages {
			pubsubMsg := &gps.Message{
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

		// Receive and verify messages
		messages := receiveMessages(10 * time.Second)

		t.Logf("Received %d messages", len(messages))

		// Count individual messages
		individualCount := 0
		for _, msg := range messages {
			if _, ok := msg.Attributes["publisher-test"]; ok {
				individualCount++
				t.Logf("Received individual message: %s", string(msg.Data))
			}
		}

		t.Logf("Received %d individual messages", individualCount)

		if individualCount < numMessages {
			t.Errorf("Expected %d individual messages, got %d", numMessages, individualCount)
		}
	})

	// Run batch publishing subtest
	t.Run("BatchPublish", func(t *testing.T) {
		// Prepare batch messages
		batchMessages := make([]pubsub.Message, numMessages)
		for i := range numMessages {
			pubsubMsg := &gps.Message{
				Data: fmt.Appendf(nil, "test-batch-message-%d", i),
				Attributes: map[string]string{
					"batch-test": "true",
					"sequence":   fmt.Sprintf("%d", i),
					"id":         uuid.New().String(),
				},
			}
			batchMessages[i] = gcp.NewMessage(pubsubMsg)
		}

		// Publish batch
		batchIDs, err := publisher.PublishBatch(ctx, batchMessages)
		if err != nil {
			t.Fatalf("Failed to publish batch: %v", err)
		}

		for i, id := range batchIDs {
			t.Logf("Published batch message %d with ID: %s", i, id)
		}

		// Receive and verify messages
		messages := receiveMessages(10 * time.Second)

		t.Logf("Received %d messages", len(messages))

		// Count batch messages
		batchCount := 0
		for _, msg := range messages {
			if _, ok := msg.Attributes["batch-test"]; ok {
				batchCount++
				t.Logf("Received batch message: %s", string(msg.Data))
			}
		}

		t.Logf("Received %d batch messages", batchCount)

		if batchCount < numMessages {
			t.Errorf("Expected %d batch messages, got %d", numMessages, batchCount)
		}
	})
}
