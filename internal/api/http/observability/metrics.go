package http

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// HandlerMetrics records HTTP handler operation metrics.
type HandlerMetrics struct {
	executionsTotal metric.Int64Counter
	durationMs      metric.Float64Histogram
}

func NewHandlerMetrics(meter metric.Meter) (*HandlerMetrics, error) {
	executionsTotal, err := meter.Int64Counter("service_template.handler.executions_total")
	if err != nil {
		return nil, err
	}
	durationMs, err := meter.Float64Histogram("service_template.handler.duration_ms", metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	return &HandlerMetrics{executionsTotal: executionsTotal, durationMs: durationMs}, nil
}

func (m *HandlerMetrics) Record(ctx context.Context, operation string, startedAt time.Time, err error) {
	if m == nil {
		return
	}
	status := "ok"
	if err != nil {
		status = "error"
	}
	attrs := metric.WithAttributes(attribute.String("handler", operation), attribute.String("status", status))
	m.executionsTotal.Add(ctx, 1, attrs)
	m.durationMs.Record(ctx, float64(time.Since(startedAt).Milliseconds()), attrs)
}
