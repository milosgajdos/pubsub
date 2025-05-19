package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// PubSubTracerName is the name of the tracer for the PubSub system
	PubSubTracerName = "github.com/milosgajdos/pubsub"
)

// GlobalTracer provides access to the global tracer
var GlobalTracer trace.Tracer

// TraceConfig holds configuration for OpenTelemetry tracing
type TraceConfig struct {
	// Enabled indicates if tracing is enabled
	Enabled bool
	// ServiceName is the name of the service
	ServiceName string
	// ServiceVersion is the version of the service
	ServiceVersion string
	// Environment is the environment (e.g., "development", "production")
	Environment string
	// OTLPEndpoint is the endpoint for the OTLP exporter (e.g., "localhost:4317")
	OTLPEndpoint string
	// SampleRate is the fraction of traces to sample (0.0 to 1.0)
	SampleRate float64
}

// InitTracing initializes OpenTelemetry tracing
func InitTracing(ctx context.Context, config TraceConfig) (func(context.Context) error, error) {
	if !config.Enabled {
		// Return no-op cleanup function if tracing is disabled
		return func(context.Context) error { return nil }, nil
	}

	if config.ServiceName == "" {
		config.ServiceName = "pubsub-service"
	}

	if config.OTLPEndpoint == "" {
		config.OTLPEndpoint = "localhost:4317"
	}

	if config.SampleRate <= 0 {
		config.SampleRate = 1.0 // Default to 100% sampling
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP exporter
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create trace provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Setup context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Get global tracer
	GlobalTracer = otel.Tracer(PubSubTracerName)

	// Return cleanup function
	return func(ctx context.Context) error {
		// Shutdown trace provider with timeout
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctxWithTimeout)
	}, nil
}

// StartMessageProcessingSpan starts a new span for message processing
func StartMessageProcessingSpan(ctx context.Context, msgID, topic, subscription, msgType string) (context.Context, trace.Span) {
	if GlobalTracer == nil {
		// If tracing is not initialized, use noop tracer
		return ctx, trace.SpanFromContext(ctx)
	}

	// Try to extract existing context from attributes if available
	ctx, span := GlobalTracer.Start(ctx, "process_message",
		trace.WithAttributes(
			attribute.String("messaging.system", "pubsub"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.destination_kind", "topic"),
			attribute.String("messaging.pubsub.subscription", subscription),
			attribute.String("messaging.message_id", msgID),
			attribute.String("messaging.message_type", msgType),
		),
	)

	return ctx, span
}

// StartPublishSpan starts a new span for publishing a message
func StartPublishSpan(ctx context.Context, topic string) (context.Context, trace.Span) {
	if GlobalTracer == nil {
		// If tracing is not initialized, use noop tracer
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := GlobalTracer.Start(ctx, "publish_message",
		trace.WithAttributes(
			attribute.String("messaging.system", "pubsub"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.destination_kind", "topic"),
		),
	)

	return ctx, span
}

// InjectTracingContext injects tracing context into attributes map
func InjectTracingContext(ctx context.Context, attributes map[string]string) map[string]string {
	if attributes == nil {
		attributes = make(map[string]string)
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(attributes))
	return attributes
}

// ExtractTracingContext extracts tracing context from attributes map
func ExtractTracingContext(ctx context.Context, attributes map[string]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(attributes))
}
