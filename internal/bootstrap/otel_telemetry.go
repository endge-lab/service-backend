package bootstrap

import (
	"context"
	"time"

	"github.com/endge-lab/service-template-go/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func newTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func registerTextMapPropagator(propagator propagation.TextMapPropagator) {
	otel.SetTextMapPropagator(propagator)
}

func newTraceProvider(lc fx.Lifecycle, cfg *config.Config, telemetryResource *resource.Resource, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !cfg.Telemetry.Enabled || cfg.Telemetry.OTLPEndpoint == "" {
		logger.Info("trace exporter disabled")
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(telemetryResource),
			sdktrace.WithSampler(sdktrace.NeverSample()),
		)
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	exp, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(append([]otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Telemetry.OTLPEndpoint),
			otlptracegrpc.WithDialOption(grpc.WithBlock()),
		}, insecureOptions(cfg.Telemetry.OTLPInsecure)...)...),
	)
	if err != nil {
		logger.Warn("trace exporter disabled", zap.Error(err), zap.String("endpoint", cfg.Telemetry.OTLPEndpoint))
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(telemetryResource),
			sdktrace.WithSampler(sdktrace.NeverSample()),
		)
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(telemetryResource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down tracer provider")
			return tp.Shutdown(ctx)
		},
	})

	logger.Info("trace exporter enabled", zap.String("endpoint", cfg.Telemetry.OTLPEndpoint))
	return tp, nil
}

func newTracer(cfg *config.Config, tp *sdktrace.TracerProvider) trace.Tracer {
	return tp.Tracer(cfg.AppName)
}

func newMeterProvider(lc fx.Lifecycle, cfg *config.Config, telemetryResource *resource.Resource, logger *zap.Logger) (*sdkmetric.MeterProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !cfg.Telemetry.Enabled || cfg.Telemetry.OTLPEndpoint == "" {
		logger.Info("metric exporter disabled")
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(telemetryResource),
		)
		otel.SetMeterProvider(mp)
		return mp, nil
	}

	exp, err := otlpmetricgrpc.New(
		ctx,
		append([]otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.Telemetry.OTLPEndpoint),
			otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
		}, insecureMetricOptions(cfg.Telemetry.OTLPInsecure)...)...,
	)
	if err != nil {
		logger.Warn("metric exporter disabled", zap.Error(err), zap.String("endpoint", cfg.Telemetry.OTLPEndpoint))
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(telemetryResource),
		)
		otel.SetMeterProvider(mp)
		return mp, nil
	}

	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(telemetryResource),
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down meter provider")
			return mp.Shutdown(ctx)
		},
	})

	logger.Info("metric exporter enabled", zap.String("endpoint", cfg.Telemetry.OTLPEndpoint))
	return mp, nil
}

func newMeter(cfg *config.Config, mp *sdkmetric.MeterProvider) otelmetric.Meter {
	return mp.Meter(cfg.AppName)
}

func insecureOptions(enabled bool) []otlptracegrpc.Option {
	if enabled {
		return []otlptracegrpc.Option{otlptracegrpc.WithInsecure()}
	}

	return nil
}

func insecureMetricOptions(enabled bool) []otlpmetricgrpc.Option {
	if enabled {
		return []otlpmetricgrpc.Option{otlpmetricgrpc.WithInsecure()}
	}

	return nil
}
