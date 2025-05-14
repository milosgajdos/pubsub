# GCP PubSub Integration Tests

This directory contains integration tests for the GCP PubSub implementation of the `pubsub` package. These tests use the [Testcontainers](https://testcontainers.com/) library to run the GCP Pub/Sub emulator in a Docker container.

## Prerequisites

To run the integration tests, you need to have:

1. Docker installed and running on your machine
2. Go 1.23 or later
3. Required Go packages installed (handled by go.mod)

## Running the Tests

### Running All Tests

You can run all tests, including the integration tests, with the following command:

```bash
cd pubsub
go test -v ./gcp
```

### Running Only Integration Tests

To run only the integration tests:

```bash
cd pubsub
go test -v ./gcp -run TestPublisherIntegration
go test -v ./gcp -run TestSubscriberIntegration
```

### Skipping Integration Tests

Integration tests can be skipped when running in short mode:

```bash
cd pubsub
go test -v ./gcp -short
```

## How It Works

The integration tests in this package use Testcontainers to:

1. Spin up a Docker container with the GCP Pub/Sub emulator
2. Create test topics and subscriptions in the emulator
3. Test the actual publisher and subscriber implementations against the emulator
4. Verify that messages are correctly published and received

This approach provides a more comprehensive test of the actual behavior of the components compared to unit tests with mocks.

## Troubleshooting

If you encounter any issues:

1. Ensure Docker is running and properly configured
2. Verify that you have enough permissions to create and run Docker containers
3. Check that no port conflicts exist with the Testcontainers ports
4. Ensure the PUBSUB_EMULATOR_HOST environment variable is not set in your environment, as it might interfere with the tests

## Adding New Integration Tests

When adding new integration tests:

1. Follow the existing pattern for setting up the emulator
2. Ensure tests are skipped when running in short mode
3. Clean up resources (containers, connections) when tests are complete
4. Handle errors appropriately and make tests informative when they fail
