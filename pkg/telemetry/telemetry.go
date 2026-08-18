package telemetry

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Setup configures OTLP tracing when OTEL_EXPORTER_OTLP_ENDPOINT is present.
// The endpoint may be a full URL such as http://jaeger:4318 or a host:port.
func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	options := []otlptracehttp.Option{}
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		options = append(options, otlptracehttp.WithEndpoint(parsed.Host))
		if parsed.Scheme == "http" {
			options = append(options, otlptracehttp.WithInsecure())
		}
	} else {
		options = append(options, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}
	resource, err := resourceForService(serviceName)
	if err != nil {
		return nil, err
	}
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource),
	)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

func resourceForService(serviceName string) (*resource.Resource, error) {
	if strings.TrimSpace(serviceName) == "" {
		return nil, errors.New("telemetry service name is required")
	}
	return resource.New(context.Background(), resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)))
}
