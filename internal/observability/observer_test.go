package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type recorderStub struct {
	operation string
	err       error
}

func (r *recorderStub) Record(_ context.Context, operation string, _ time.Time, err error) {
	r.operation = operation
	r.err = err
}

func TestObserverCreatesChildSpanAndRecordsOperation(t *testing.T) {
	t.Parallel()

	spans := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(spans))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	ctx, parent := provider.Tracer("test").Start(context.Background(), "http.request")
	recorder := &recorderStub{}
	observer := NewCore(provider.Tracer("test"), zap.NewNop()).For(LayerUseCase, "projects_usecase").WithRecorder(recorder)
	_, operation := observer.Start(ctx, "project.create", nil, nil)
	operation.End(nil)
	parent.End()

	ended := spans.Ended()
	if len(ended) != 2 {
		t.Fatalf("ended spans = %d, want 2", len(ended))
	}
	if ended[0].Name() != "usecase.project.create.execute" || ended[0].Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Fatalf("unexpected child span: %#v", ended[0])
	}
	if recorder.operation != "project.create" || recorder.err != nil {
		t.Fatalf("unexpected recorder state: %#v", recorder)
	}
}

func TestObserverWithRecorderDoesNotMutateSourceObserver(t *testing.T) {
	t.Parallel()

	first := &recorderStub{}
	second := &recorderStub{}
	base := NewCore(nil, zap.NewNop()).For(LayerUseCase, "projects_usecase")
	firstObserver := base.WithRecorder(first)
	secondObserver := base.WithRecorder(second)

	_, firstOperation := firstObserver.Start(context.Background(), "project.create", nil, nil)
	firstOperation.End(nil)
	_, secondOperation := secondObserver.Start(context.Background(), "project.list", nil, nil)
	secondOperation.End(nil)

	if first.operation != "project.create" || second.operation != "project.list" {
		t.Fatalf("recorders received unexpected operations: first=%q second=%q", first.operation, second.operation)
	}
}

func TestOperationRecordStepWritesTraceEventAndInfoLog(t *testing.T) {
	spans := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(spans))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	logCore, logs := observer.New(zap.InfoLevel)
	core := NewCore(provider.Tracer("test"), zap.New(logCore))
	_, operation := core.For(LayerUseCase, "projects_usecase").Start(context.Background(), "project.create", nil, nil)

	operation.RecordStep(
		"project.create.persisted",
		"project persisted",
		[]attribute.KeyValue{attribute.String("project.identity", "demo")},
		zap.String("project_identity", "demo"),
	)
	operation.End(nil)

	ended := spans.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	events := ended[0].Events()
	if len(events) != 2 || events[0].Name != "project.create.persisted" || events[1].Name != "project.create.completed" {
		t.Fatalf("trace events = %#v, want persisted and completed events", events)
	}

	entries := logs.FilterMessage("project persisted").All()
	if len(entries) != 1 {
		t.Fatalf("info log entries = %d, want 1", len(entries))
	}
	if entries[0].ContextMap()["project_identity"] != "demo" {
		t.Fatalf("log fields = %#v, want project_identity=demo", entries[0].ContextMap())
	}
	if completed := logs.FilterMessage("use case operation completed").All(); len(completed) != 1 || completed[0].ContextMap()["operation"] != "project.create" {
		t.Fatalf("completion log entries = %#v, want project.create", completed)
	}
}

func TestOperationEndRecordsFailedCompletionWithoutSuccessLog(t *testing.T) {
	spans := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(spans))
	defer func() { _ = provider.Shutdown(context.Background()) }()

	logCore, logs := observer.New(zap.InfoLevel)
	_, operation := NewCore(provider.Tracer("test"), zap.New(logCore)).For(LayerUseCase, "projects_usecase").Start(context.Background(), "project.create", nil, nil)
	err := errors.New("repository unavailable")
	operation.End(&err)

	ended := spans.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	events := ended[0].Events()
	if len(events) == 0 || events[0].Name != "project.create.completed" {
		t.Fatalf("trace events = %#v, want failed completion event", events)
	}
	if events[0].Attributes[0].Key != "operation.status" || events[0].Attributes[0].Value.AsString() != "error" {
		t.Fatalf("completion event attributes = %#v, want operation.status=error", events[0].Attributes)
	}
	if completed := logs.FilterMessage("use case operation completed").All(); len(completed) != 0 {
		t.Fatalf("success completion logs = %#v, want none", completed)
	}
	if failed := logs.FilterMessage("span failed").All(); len(failed) != 1 {
		t.Fatalf("failed span logs = %#v, want one", failed)
	}
}
