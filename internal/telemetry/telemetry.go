// Package telemetry sets up OpenTelemetry tracing and Prometheus metrics for HTTP apps.
//
// Environment variables:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT  e.g. obs-otel-collector:4317 (gRPC: host:port only, no scheme)
//	OTEL_SERVICE_NAME             e.g. notification-service
//
// When OTEL_EXPORTER_OTLP_ENDPOINT is not set, a no-op tracer is used so the
// app still starts cleanly in local development without a collector running.
package telemetry

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Init configures the global OpenTelemetry TracerProvider.
// Returns a shutdown function that must be deferred by the caller.
func Init(defaultServiceName string) func() {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = defaultServiceName
	}

	if endpoint == "" {
		log.Println("[telemetry] OTEL_EXPORTER_OTLP_ENDPOINT not set — tracing disabled")
		return func() {}
	}

	// gRPC exporter requires host:port — strip http:// or https:// if present
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		log.Printf("[telemetry] Resource creation failed: %v — tracing disabled", err)
		return func() {}
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("[telemetry] OTLP exporter failed: %v — tracing disabled", err)
		return func() {}
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	log.Printf("[telemetry] Tracing configured — service=%s endpoint=%s", serviceName, endpoint)

	return func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			log.Printf("[telemetry] Shutdown error: %v", err)
		}
	}
}

// RegisterMetrics adds Prometheus /metrics endpoint to the HTTP mux.
func RegisterMetrics(mux interface{ Handle(string, http.Handler) }) {
	mux.Handle("/metrics", promhttp.Handler())
	log.Println("[telemetry] Prometheus /metrics endpoint registered")
}
