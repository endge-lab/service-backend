package usecase

import (
	"context"
	"time"

	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type UseCaseMetrics struct {
	executionsTotal metric.Int64Counter
	durationMs      metric.Float64Histogram
}

func NewUseCaseMetrics(meter metric.Meter) (*UseCaseMetrics, error) {
	executionsTotal, err := meter.Int64Counter(
		"service_template.usecase.executions_total",
		metric.WithDescription("Total number of executed use cases"),
	)
	if err != nil {
		return nil, err
	}

	durationMs, err := meter.Float64Histogram(
		"service_template.usecase.duration_ms",
		metric.WithDescription("Use case execution duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &UseCaseMetrics{
		executionsTotal: executionsTotal,
		durationMs:      durationMs,
	}, nil
}

func (m *UseCaseMetrics) Record(ctx context.Context, useCaseName string, startedAt time.Time, err error) {
	if m == nil {
		return
	}

	status := "ok"
	if err != nil {
		status = "error"
	}

	attrs := metric.WithAttributes(
		attribute.String("usecase", useCaseName),
		attribute.String("status", status),
	)
	if err != nil {
		attrs = metric.WithAttributes(
			attribute.String("usecase", useCaseName),
			attribute.String("status", status),
			attribute.String("error.code", domainerrors.CodeOf(err)),
		)
	}

	m.executionsTotal.Add(ctx, 1, attrs)
	m.durationMs.Record(ctx, float64(time.Since(startedAt).Milliseconds()), attrs)
}
