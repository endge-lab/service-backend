package shared

import (
	"context"
	"time"

	"github.com/endge-lab/service-kit-go/pkg/logging"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ObservedUseCase struct {
	tracer  trace.Tracer
	logger  *zap.Logger
	metrics *UseCaseMetrics
}

type ObservedOperation struct {
	ctx       context.Context
	logger    *zap.Logger
	metrics   *UseCaseMetrics
	op        string
	startedAt time.Time
	step      *telemetry.Step
}

func NewObservedUseCase(tracer trace.Tracer, logger *zap.Logger, metrics *UseCaseMetrics) ObservedUseCase {
	if logger == nil {
		logger = zap.NewNop()
	}

	return ObservedUseCase{
		tracer:  tracer,
		logger:  logger,
		metrics: metrics,
	}
}

func (u *ObservedUseCase) StartObservedOperation(
	ctx context.Context,
	op string,
	attrs []attribute.KeyValue,

	logFields []zap.Field,
) (context.Context, *ObservedOperation) {
	startedAt := time.Now()
	spanAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	spanAttrs = append(spanAttrs, attribute.String("usecase", op))
	spanAttrs = append(spanAttrs, attrs...)

	ctx, step := telemetry.StartTrace(ctx, u.tracer, u.logger, "usecase."+op+".execute", spanAttrs...)

	logger := logging.WithContext(ctx, u.logger)
	if len(logFields) > 0 {
		logger = logger.With(logFields...)
	}

	return ctx, &ObservedOperation{
		ctx:       ctx,
		logger:    logger,
		metrics:   u.metrics,
		op:        op,
		startedAt: startedAt,
		step:      step,
	}
}

func (o *ObservedOperation) End(err *error) {
	if o == nil {
		return
	}

	var actualErr error
	if err != nil {
		actualErr = *err
	}

	if o.metrics != nil {
		o.metrics.Record(o.ctx, o.op, o.startedAt, actualErr)
	}
	if o.step != nil {
		o.step.End(actualErr)
	}
}

func (o *ObservedOperation) Logger() *zap.Logger {
	if o == nil || o.logger == nil {
		return zap.NewNop()
	}

	return o.logger
}
