package postgres

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RepositoryMetrics records database operation metrics.
type RepositoryMetrics struct {
	executionsTotal metric.Int64Counter
	durationMs      metric.Float64Histogram
}

func NewRepositoryMetrics(meter metric.Meter) (*RepositoryMetrics, error) {
	executionsTotal, err := meter.Int64Counter("service_template.repository.executions_total")
	if err != nil {
		return nil, err
	}
	durationMs, err := meter.Float64Histogram("service_template.repository.duration_ms", metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	return &RepositoryMetrics{executionsTotal: executionsTotal, durationMs: durationMs}, nil
}

func (m *RepositoryMetrics) Record(ctx context.Context, operation string, startedAt time.Time, err error) {
	if m == nil {
		return
	}
	status := "ok"
	if err != nil {
		status = "error"
	}
	attrs := metric.WithAttributes(attribute.String("repository", operation), attribute.String("status", status))
	m.executionsTotal.Add(ctx, 1, attrs)
	m.durationMs.Record(ctx, float64(time.Since(startedAt).Milliseconds()), attrs)
}
