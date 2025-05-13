package gcp

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/api/option"
)

const (
	// Common test constants
	TestTimeout        = 90 * time.Second
	TestProjectID      = "test-project"
	TestTopicID        = "test-topic"
	TestSubscriptionID = "test-subscription"

	// Environment variables for PubSub emulator configuration
	EnvManualEmulator = "MANUAL_PUBSUB_EMULATOR" // Set to "true" to use a manually started emulator
	EnvEmulatorHost   = "PUBSUB_EMULATOR_HOST"   // The emulator host:port (e.g., "localhost:8085")
)

// SetupPubSub creates and starts a GCP PubSub emulator container and sets up a topic and subscription.
// It returns the client, topic, subscription, and a cleanup function that should be deferred.
func SetupPubSub(t *testing.T, topicID, subID string) (*pubsub.Client, *pubsub.Topic, *pubsub.Subscription, func()) {
	t.Helper()

	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	// Check for manual emulator mode first
	useManualEmulator := os.Getenv(EnvManualEmulator) == "true"
	emulatorHost := os.Getenv(EnvEmulatorHost)

	// If manual emulator is requested but host not set, use default localhost
	if useManualEmulator && emulatorHost == "" {
		emulatorHost = "localhost:8085"
		t.Setenv(EnvEmulatorHost, emulatorHost)
		t.Logf("Using manual emulator with default address: %s", emulatorHost)
		t.Logf("MANUAL EMULATOR MODE: Make sure you've started the pubsub emulator with:")
		t.Logf("gcloud beta emulators pubsub start --host-port=localhost:8085")
	}

	// If emulator host is already set, use it
	if emulatorHost != "" {
		t.Logf("Using existing emulator at: %s", emulatorHost)
		client, topic, sub := createTopicAndSub(t, ctx, emulatorHost, topicID, subID)
		cleanup := func() {}
		return client, topic, sub, cleanup
	}

	// Start the PubSub emulator container
	t.Log("Starting PubSub emulator container...")

	// Create container request
	// Note: We're using port 8085 inside the container and letting Docker assign a random host port
	req := testcontainers.ContainerRequest{
		Image:        "gcr.io/google.com/cloudsdktool/cloud-sdk:emulators",
		ExposedPorts: []string{"8085/tcp"},
		Env: map[string]string{
			"PUBSUB_PROJECT_ID": TestProjectID,
		},
		Cmd: []string{
			"gcloud", "beta", "emulators", "pubsub", "start",
			"--host-port=0.0.0.0:8085",
			"--project=" + TestProjectID,
			"--quiet",
		},
		WaitingFor: wait.ForLog("Server started").WithStartupTimeout(60 * time.Second),
	}

	// Start the container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start PubSub emulator: %v", err)
	}

	// Get the mapped port - note that we're looking for port "8085/tcp" here
	mappedPort, err := container.MappedPort(ctx, "8085/tcp")
	if err != nil {
		t.Fatalf("Failed to get mapped port: %v", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get host: %v", err)
	}

	// Set the emulator host environment variable
	emulatorHost = fmt.Sprintf("%s:%s", host, mappedPort.Port())
	t.Setenv(EnvEmulatorHost, emulatorHost)
	t.Logf("Emulator running at: %s", emulatorHost)

	// Return a cleanup function
	cleanup := func() {
		// Terminate the container
		t.Log("Terminating PubSub emulator container...")
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate emulator: %v", err)
		}
	}

	// Create the client, topic, and subscription
	client, topic, sub := createTopicAndSub(t, ctx, emulatorHost, topicID, subID)
	return client, topic, sub, cleanup
}

// createTopicAndSub creates a client, topic, and subscription
// nolint:revive
func createTopicAndSub(t *testing.T, ctx context.Context, endpoint, topicID, subID string) (*pubsub.Client, *pubsub.Topic, *pubsub.Subscription) {
	t.Helper()

	t.Logf("Creating PubSub client with endpoint: %s", endpoint)

	client, err := pubsub.NewClient(
		ctx,
		TestProjectID,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}

	topic, err := client.CreateTopic(ctx, topicID)
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	sub, err := client.CreateSubscription(ctx, subID, pubsub.SubscriptionConfig{
		Topic: topic,
	})
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	return client, topic, sub
}
