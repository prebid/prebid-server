package main

import (
	"context"
	"fmt"
	"time"

	"github.com/prebid/prebid-server/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
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
	driver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(cfg.Endpoint),
	)

	exp, err := otlp.NewExporter(ctx, driver)
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

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	cont := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		controller.WithExporter(exp),
		controller.WithCollectPeriod(2*time.Second),
	)

	// LATER: currently do not report metrics
	// err = cont.Start(ctx)
	// if err != nil {
	// 	log.Fatalf("failed to initialize metric controller: %v", err)
	// }

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)
	global.SetMeterProvider(cont.MeterProvider())
	return func() {
		// Push any last metric events to the exporter.
		cont.Stop(context.Background())

		// Shutdown will flush any remaining spans and shut down the exporter.
		tracerProvider.Shutdown(ctx)
	}, nil
}
