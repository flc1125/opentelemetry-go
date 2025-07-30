// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package tracetest is a testing helper package.
package tracetest_test

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ExampleInMemoryExporter demonstrates how to use the InMemoryExporter
// to collect and verify spans in unit tests.
func ExampleInMemoryExporter() {
	// Create an in-memory exporter to capture spans
	exporter := tracetest.NewInMemoryExporter()

	// Create a tracer provider with the in-memory exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("example-service"),
		)),
	)

	// Get a tracer from the provider
	tracer := tp.Tracer("example-tracer")

	// Create some spans to demonstrate testing
	ctx := context.Background()
	ctx, span1 := tracer.Start(ctx, "operation-1")
	span1.SetAttributes(attribute.String("key1", "value1"))
	span1.End()

	ctx, span2 := tracer.Start(ctx, "operation-2")
	span2.SetAttributes(attribute.String("key2", "value2"))
	span2.End()

	// Force flush to ensure spans are exported
	_ = tp.ForceFlush(ctx)

	// Retrieve and verify the captured spans
	spans := exporter.GetSpans()
	fmt.Printf("Captured %d spans:\n", len(spans))
	for _, span := range spans {
		fmt.Printf("- %s\n", span.Name)
	}

	// Output:
	// Captured 2 spans:
	// - operation-1
	// - operation-2
}

// ExampleSpanRecorder demonstrates how to use the SpanRecorder
// to capture span lifecycle events for testing.
func ExampleSpanRecorder() {
	// Create a span recorder to capture span events
	recorder := tracetest.NewSpanRecorder()

	// Create a tracer provider with the recorder as a span processor
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(recorder),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("example-service"),
		)),
	)

	// Get a tracer from the provider
	tracer := tp.Tracer("example-tracer",
		oteltrace.WithInstrumentationAttributes(attribute.String("library.version", "1.0.0")),
	)

	// Create a span to demonstrate testing
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "recorded-operation")
	span.SetAttributes(attribute.String("operation.type", "test"))
	span.End()

	// Verify the recorded spans
	startedSpans := recorder.Started()
	endedSpans := recorder.Ended()

	fmt.Printf("Started spans: %d\n", len(startedSpans))
	fmt.Printf("Ended spans: %d\n", len(endedSpans))

	if len(endedSpans) > 0 {
		fmt.Printf("First ended span: %s\n", endedSpans[0].Name())
	}

	// Output:
	// Started spans: 1
	// Ended spans: 1
	// First ended span: recorded-operation
}

// This example shows how to test custom instrumentation by verifying
// that the expected spans are created with the correct attributes.
func ExampleInMemoryExporter_testingInstrumentation() {
	// Setup: Create an in-memory exporter and tracer provider
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("instrumentation-test"),
		)),
	)
	otel.SetTracerProvider(tp)

	// Simulate some instrumented code
	instrumentedFunction := func(ctx context.Context) error {
		tracer := otel.Tracer("myapp",
			oteltrace.WithInstrumentationAttributes(attribute.String("version", "1.0.0")),
		)

		ctx, span := tracer.Start(ctx, "business-operation")
		defer span.End()

		span.SetAttributes(
			attribute.String("operation.type", "business"),
			attribute.Int("items.count", 42),
		)

		// Simulate nested operation
		ctx, childSpan := tracer.Start(ctx, "database-query")
		childSpan.SetAttributes(attribute.String("db.statement", "SELECT * FROM users"))
		childSpan.End()

		return nil
	}

	// Execute the instrumented function
	ctx := context.Background()
	_ = instrumentedFunction(ctx)

	// Force flush to ensure all spans are exported
	_ = tp.ForceFlush(ctx)

	// Test: Verify the spans were created correctly
	spans := exporter.GetSpans()
	fmt.Printf("Total spans: %d\n", len(spans))

	for _, span := range spans {
		fmt.Printf("Span: %s, Tracer: %s\n", span.Name, span.InstrumentationScope.Name)
	}

	// Output:
	// Total spans: 2
	// Span: database-query, Tracer: myapp
	// Span: business-operation, Tracer: myapp
}

// Custom exporter that prints span information for demonstration
type printExporter struct{}

func (e printExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	for _, span := range spans {
		attrs := ""
		for _, attr := range span.Attributes() {
			if attrs != "" {
				attrs += ", "
			}
			attrs += fmt.Sprintf("%s=%v", attr.Key, attr.Value.AsInterface())
		}
		fmt.Fprintf(os.Stdout, "span=%s attrs=[%s]\n", span.Name(), attrs)
	}
	return nil
}

func (e printExporter) Shutdown(context.Context) error   { return nil }
func (e printExporter) ForceFlush(context.Context) error { return nil }

// ExampleNoopExporter demonstrates the NoopExporter for performance testing
// where you want to measure instrumentation overhead without export costs.
func ExampleNoopExporter() {
	// Create a no-op exporter that discards all spans
	exporter := tracetest.NewNoopExporter()

	// Create a tracer provider with the no-op exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("performance-test"),
		)),
	)

	tracer := tp.Tracer("benchmark-tracer")

	// Create spans that will be discarded (useful for performance testing)
	for i := 0; i < 3; i++ {
		_, span := tracer.Start(context.Background(), fmt.Sprintf("operation-%d", i))
		span.SetAttributes(attribute.Int("iteration", i))
		span.End()
	}

	fmt.Println("All spans were discarded by NoopExporter")

	// Output:
	// All spans were discarded by NoopExporter
}
