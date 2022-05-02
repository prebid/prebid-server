package main

import (
	"context"
	"fmt"

	"github.com/prebid/prebid-server/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

type DoneCallback func()

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func initProvider(cfg config.OpenTelemetry) (DoneCallback, error) {
	if !cfg.Enabled {
		return func() {}, fmt.Errorf("error getting opentelemetry configs")
	}

	ctx := context.Background()

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of `otelcol-gateway.observability-system`
	driver := otlpgrpc.NewClient(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(cfg.Endpoint),
	)

	exp, err := otlptrace.New(ctx, driver)
	if err != nil {
		return func() {}, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("prebid"),
			semconv.ServiceNamespaceKey.String("supply"),
		),
	)
	if err != nil {
		return func() {}, err
	}
	// Not using ratio based sampler for now. Composite tries parent and defaults to 0.
	ratioBasedSampler := sdktrace.TraceIDRatioBased(0.0)
	compositeBasedSampler := sdktrace.ParentBased(ratioBasedSampler)

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(compositeBasedSampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)
	return func() {
		// Shutdown will flush any remaining spans and shut down the exporter.
		tracerProvider.Shutdown(ctx)
	}, nil
}
