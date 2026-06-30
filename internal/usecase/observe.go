package usecase

import (
	"context"
	"time"

	"github.com/endge-lab/service-template-go/internal/util"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type observedUseCase struct {
	tracer  trace.Tracer
	logger  *zap.Logger
	metrics *UseCaseMetrics
}

type observedOperation struct {
	ctx       context.Context
	logger    *zap.Logger
	metrics   *UseCaseMetrics
	op        string
	startedAt time.Time
	step      *util.TraceStep
}

func newObservedUseCase(tracer trace.Tracer, logger *zap.Logger, metrics *UseCaseMetrics) observedUseCase {
	if logger == nil {
		logger = zap.NewNop()
	}

	return observedUseCase{
		tracer:  tracer,
		logger:  logger,
		metrics: metrics,
	}
}

func (u *observedUseCase) startObservedOperation(
	ctx context.Context,
	op string,
	attrs []attribute.KeyValue,
	logFields []zap.Field,
) (context.Context, *observedOperation) {
	startedAt := time.Now()
	spanAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	spanAttrs = append(spanAttrs, attribute.String("usecase", op))
	spanAttrs = append(spanAttrs, attrs...)

	ctx, step := util.StartTrace(ctx, u.tracer, u.logger, "usecase."+op+".execute", spanAttrs...)

	logger := util.LoggerWithTrace(ctx, u.logger)
	if len(logFields) > 0 {
		logger = logger.With(logFields...)
	}

	return ctx, &observedOperation{
		ctx:       ctx,
		logger:    logger,
		metrics:   u.metrics,
		op:        op,
		startedAt: startedAt,
		step:      step,
	}
}

func (o *observedOperation) End(err *error) {
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
		o.step.EndTrace(actualErr)
	}
}

func (o *observedOperation) Logger() *zap.Logger {
	if o == nil || o.logger == nil {
		return zap.NewNop()
	}

	return o.logger
}
