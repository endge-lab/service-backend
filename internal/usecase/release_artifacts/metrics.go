package release_artifacts

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type cacheMetrics struct {
	requests  metric.Int64Counter
	evictions metric.Int64Counter
	items     metric.Int64UpDownCounter
	bytes     metric.Int64UpDownCounter
	loadMs    metric.Float64Histogram
}

func newCacheMetrics(meter metric.Meter) (cacheMetrics, error) {
	requests, err := meter.Int64Counter("endge.release_artifact_cache.requests_total")
	if err != nil {
		return cacheMetrics{}, err
	}
	evictions, err := meter.Int64Counter("endge.release_artifact_cache.evictions_total")
	if err != nil {
		return cacheMetrics{}, err
	}
	items, err := meter.Int64UpDownCounter("endge.release_artifact_cache.items")
	if err != nil {
		return cacheMetrics{}, err
	}
	bytes, err := meter.Int64UpDownCounter("endge.release_artifact_cache.bytes", metric.WithUnit("By"))
	if err != nil {
		return cacheMetrics{}, err
	}
	loadMs, err := meter.Float64Histogram("endge.release_artifact_cache.load_duration_ms", metric.WithUnit("ms"))
	if err != nil {
		return cacheMetrics{}, err
	}
	return cacheMetrics{requests: requests, evictions: evictions, items: items, bytes: bytes, loadMs: loadMs}, nil
}

func (m cacheMetrics) recordRequest(ctx context.Context, operation, result string) {
	m.requests.Add(ctx, 1, metric.WithAttributes(m.attributes(operation, attribute.String("result", result))...))
}

func (m cacheMetrics) recordLoad(ctx context.Context, operation string, duration time.Duration) {
	m.loadMs.Record(ctx, float64(duration.Microseconds())/1000, metric.WithAttributes(m.attributes(operation)...))
}

func (m cacheMetrics) addItems(ctx context.Context, items, bytes int) {
	m.items.Add(ctx, int64(items))
	m.bytes.Add(ctx, int64(bytes))
}

func (m cacheMetrics) recordEviction(ctx context.Context) {
	m.evictions.Add(ctx, 1)
}

func (m cacheMetrics) attributes(operation string, extra ...attribute.KeyValue) []attribute.KeyValue {
	attributes := []attribute.KeyValue{attribute.String("operation", operation)}
	return append(attributes, extra...)
}
