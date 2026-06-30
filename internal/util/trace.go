package util

import (
	"context"

	servicelogging "github.com/endge-lab/service-kit-go/logging"
	servicetelemetry "github.com/endge-lab/service-kit-go/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type TraceStep struct {
	inner *servicetelemetry.Step
}

func TraceFieldsFromContext(ctx context.Context) []zap.Field {
	return servicelogging.TraceFieldsFromContext(ctx)
}

func TraceFieldsFromSpan(span trace.Span) []zap.Field {
	return servicelogging.TraceFieldsFromSpan(span)
}

func LoggerWithTrace(ctx context.Context, logger *zap.Logger) *zap.Logger {
	return servicelogging.WithContext(ctx, logger)
}

func StartTrace(ctx context.Context, tracer trace.Tracer, logger *zap.Logger, name string, attrs ...attribute.KeyValue) (context.Context, *TraceStep) {
	ctx, step := servicetelemetry.StartTrace(ctx, tracer, logger, name, attrs...)
	return ctx, &TraceStep{inner: step}
}

func (s *TraceStep) EndTrace(err error) {
	if s == nil || s.inner == nil {
		return
	}

	s.inner.End(err)
}

func (s *TraceStep) Fail(err error) {
	if s == nil || s.inner == nil {
		return
	}

	s.inner.Fail(err)
}

func (s *TraceStep) Event(name string, attrs ...attribute.KeyValue) {
	if s == nil || s.inner == nil {
		return
	}

	s.inner.Event(name, attrs...)
}
