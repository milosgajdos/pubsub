# Go PubSub Library

This is a Go module which provides an abstract interface for PubSub systems, allowing you to create both batch and single message publishers and subscribers. The library is designed to be cloud-provider agnostic, but currently includes an implementation for Google Cloud Pub/Sub.

## Features

- Abstract interfaces for PubSub operations
- Support for both single and batch message publishing
- Configurable concurrency for subscribers
- Support for message filtering
- Retry configuration
- Schema configuration
- Idiomatic Go API

## Installation

```bash
go get github.com/milosgajdos/pubsub
```

## Usage

### Google Cloud Pub/Sub

#### Creating a Client

```go
import (
    "context"
    "log"
    "github.com/milosgajdos/pubsub/gcp"
)

func main() {
    ctx := context.Background()
    
    // Create a new GCP client
    client, err := gcp.NewClient(ctx, "your-gcp-project-id")
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()
    
    // Use the client to create publishers and subscribers
}
```

#### Publishing Messages

```go
import (
    "context"
    "log"
    "github.com/milosgajdos/pubsub"
    "github.com/milosgajdos/pubsub/gcp"
)

func PublishExample(client *gcp.Client) {
    ctx := context.Background()
    
    // Create a publisher with options
    publisher, err := client.NewPublisher(ctx,
        pubsub.WithTopic("my-topic"),
        pubsub.WithBatch(pubsub.Batch{Size: 100}),
        pubsub.WithPubRetry(pubsub.Retry{
            MinBackoffSeconds: 1,
            MaxBackoffSeconds: 10,
        }),
    )
    if err != nil {
        log.Fatalf("Failed to create publisher: %v", err)
    }
    
    // Create a message
    data := []byte("Hello, PubSub!")
    attributes := map[string]string{
        "event_type": "greeting",
        "source": "example",
    }
    metadata := map[string]any{
        "timestamp": time.Now().Unix(),
    }
    
    message := gcp.NewMessage("message-id", data, attributes, metadata)
    
    // Publish a single message
    messageID, err := publisher.Publish(ctx, message)
    if err != nil {
        log.Fatalf("Failed to publish message: %v", err)
    }
    log.Printf("Published message with ID: %s", messageID)
}
```

#### Batch Publishing

```go
func PublishBatchExample(client *gcp.Client) {
    ctx := context.Background()
    
    // Create a publisher with batch options
    publisher, err := client.NewPublisher(ctx,
        pubsub.WithTopic("my-topic"),
        pubsub.WithBatch(pubsub.Batch{Size: 100}),
    )
    if err != nil {
        log.Fatalf("Failed to create publisher: %v", err)
    }
    
    // Create a batch of messages
    var messages []pubsub.Message
    for i := 0; i < 10; i++ {
        data := []byte(fmt.Sprintf("Batch message %d", i))
        attributes := map[string]string{
            "index": fmt.Sprintf("%d", i),
            "batch": "true",
        }
        
        message := gcp.NewMessage(fmt.Sprintf("batch-msg-%d", i), data, attributes, nil)
        messages = append(messages, message)
    }
    
    // Publish the batch
    messageIDs, err := publisher.PublishBatch(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to publish batch: %v", err)
    }
    log.Printf("Published %d messages", len(messageIDs))
}
```

#### Subscribing to Messages

```go
func SubscribeExample(client *gcp.Client) {
    ctx := context.Background()
    
    // Create a subscriber with options
    subscriber, err := client.NewSubscriber(ctx,
        pubsub.WithSub("my-subscription"),
        pubsub.WithFilter(`attributes.event_type = "greeting"`),
        pubsub.WithConcurrency(5),
    )
    if err != nil {
        log.Fatalf("Failed to create subscriber: %v", err)
    }
    
    // Create a context that can be canceled
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()
    
    // Handle OS signals for graceful shutdown
    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, os.Interrupt)
    go func() {
        <-signalCh
        cancel()
    }()
    
    // Define a message handler
    messageHandler := func(ctx context.Context, msg pubsub.MessageAcker) error {
        log.Printf("Received message: %s", string(msg.Data()))
        
        // Process message...
        
        // Acknowledge the message
        msg.Ack()
        return nil
    }
    
    // Start consuming messages
    log.Println("Starting to consume messages...")
    err = subscriber.Subscribe(ctx, messageHandler)
    if err != nil && err != context.Canceled {
        log.Fatalf("Subscription error: %v", err)
    }
}
```

## Extending the Library

The library is designed to be extensible, allowing you to implement the interfaces for other Pub/Sub systems. The main interfaces to implement are:

- `Message`: Represents a message in the PubSub system
- `MessageAcker`: Extends `Message` with methods to acknowledge messages
- `Publisher`: For publishing messages
- `Subscriber`: For consuming messages

## License

[MIT License](LICENSE)