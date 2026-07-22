package observability

import (
	"context"
	"strings"
	"time"

	"github.com/endge-lab/service-kit-go/pkg/logging"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Layer identifies the application layer that owns an observed operation.
type Layer string

const (
	LayerHandler    Layer = "handler"
	LayerUseCase    Layer = "usecase"
	LayerRepository Layer = "repository"
)

// Recorder records metrics for a completed operation. Implementations stay
// layer-specific; the shared observer does not know metric names or labels.
type Recorder interface {
	Record(ctx context.Context, operation string, startedAt time.Time, err error)
}

// Core is immutable shared tracing and logging infrastructure. It does not
// hold request state, an active span, or layer-specific metric behaviour.
type Core struct {
	tracer trace.Tracer
	logger *zap.Logger
}

func NewCore(tracer trace.Tracer, logger *zap.Logger) *Core {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Core{tracer: tracer, logger: logger}
}

// For creates an observer scoped to one component and application layer.
// The returned observer uses a no-op recorder until WithRecorder is called.
func (c *Core) For(layer Layer, component string) Observer {
	if c == nil {
		c = NewCore(nil, nil)
	}

	logger := c.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	logger = logger.With(
		zap.String("layer", string(layer)),
		zap.String("component", strings.TrimSpace(component)),
	)

	return Observer{
		layer:    layer,
		tracer:   c.tracer,
		logger:   logger,
		recorder: noopRecorder{},
	}
}

type noopRecorder struct{}

func (noopRecorder) Record(context.Context, string, time.Time, error) {}

// Observer starts operations for one component. It is safe to reuse because
// request-specific state is held only by Operation.
type Observer struct {
	layer    Layer
	tracer   trace.Tracer
	logger   *zap.Logger
	recorder Recorder
}

// WithRecorder returns a copy of the observer that records completed
// operations through recorder. Neither the source observer nor Core is mutated.
func (o Observer) WithRecorder(recorder Recorder) Observer {
	if recorder == nil {
		o.recorder = noopRecorder{}
		return o
	}

	o.recorder = recorder
	return o
}

// WithFields returns a copy of the observer whose logger includes fields.
func (o Observer) WithFields(fields ...zap.Field) Observer {
	if len(fields) > 0 {
		o.logger = o.Logger().With(fields...)
	}

	return o
}

func (o Observer) Tracer() trace.Tracer { return o.tracer }

func (o Observer) Logger() *zap.Logger {
	if o.logger == nil {
		return zap.NewNop()
	}
	return o.logger
}

// Start begins a child span of the active span in ctx and returns a new context.
func (o Observer) Start(
	ctx context.Context,
	operation string,
	attrs []attribute.KeyValue,
	logFields []zap.Field,
) (context.Context, *Operation) {
	startedAt := time.Now()
	operation = strings.TrimSpace(operation)
	spanAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	spanAttrs = append(spanAttrs, attribute.String(string(o.layer), operation))
	spanAttrs = append(spanAttrs, attrs...)

	ctx, step := telemetry.StartTrace(ctx, o.tracer, o.logger, string(o.layer)+"."+operation+".execute", spanAttrs...)
	logger := logging.WithContext(ctx, o.logger)
	if len(logFields) > 0 {
		logger = logger.With(logFields...)
	}

	return ctx, &Operation{
		ctx:       ctx,
		logger:    logger,
		recorder:  o.recorder,
		operation: operation,
		startedAt: startedAt,
		step:      step,
	}
}

// Operation owns request-specific observation state.
type Operation struct {
	ctx       context.Context
	logger    *zap.Logger
	recorder  Recorder
	operation string
	startedAt time.Time
	step      *telemetry.Step
}

func (o *Operation) End(err *error) {
	if o == nil {
		return
	}

	var actualErr error
	if err != nil {
		actualErr = *err
	}
	if o.recorder != nil {
		o.recorder.Record(o.ctx, o.operation, o.startedAt, actualErr)
	}
	if o.step != nil {
		o.step.End(actualErr)
	}
}

func (o *Operation) Logger() *zap.Logger {
	if o == nil || o.logger == nil {
		return zap.NewNop()
	}

	return o.logger
}

func (o *Operation) AddEvent(name string, attrs ...attribute.KeyValue) {
	if o == nil || o.step == nil {
		return
	}

	o.step.Event(name, attrs...)
}
