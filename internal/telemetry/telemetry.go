// Package telemetry configures OpenTelemetry tracing and metrics providers.
package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Provider manages OpenTelemetry tracing and metrics providers.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

// New creates and registers OpenTelemetry providers.
func New(ctx context.Context, endpoint, serviceName string) (*Provider, error) {
	traceExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	metricExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		return nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		_ = metricExporter.Shutdown(ctx)
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(metricExporter),
		),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	return &Provider{
		tracerProvider: tp,
		meterProvider:  mp,
	}, nil
}

// Shutdown gracefully shuts down the tracing and metrics providers.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}

	var shutdownErrors []error

	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	return errors.Join(shutdownErrors...)
}
