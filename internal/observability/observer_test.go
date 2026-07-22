package observability

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
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
