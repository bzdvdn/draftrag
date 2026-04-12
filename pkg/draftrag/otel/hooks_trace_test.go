package otel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestHooks_StageEnd_CreatesSpanWithAttributesAndError(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	hooks, err := NewHooks(HooksOptions{
		TracerProvider: tp,
		MeterProvider:  noop.NewMeterProvider(),
	})
	require.NoError(t, err)

	parentTracer := tp.Tracer("test")
	ctx, parent := parentTracer.Start(context.Background(), "parent")

	hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: "Answer",
		Stage:     domain.HookStageGenerate,
		Duration:  150 * time.Millisecond,
		Err:       errors.New("boom"),
	})
	parent.End()

	ended := recorder.Ended()
	require.NotEmpty(t, ended)

	var stageSpan sdktrace.ReadOnlySpan
	for _, s := range ended {
		if s.Name() == "draftrag.generate" {
			stageSpan = s
			break
		}
	}
	require.NotNil(t, stageSpan)

	require.Equal(t, parent.SpanContext().SpanID(), stageSpan.Parent().SpanID())
	require.Equal(t, "Answer", attrString(stageSpan.Attributes(), SpanAttributeOperation))
	require.Equal(t, "generate", attrString(stageSpan.Attributes(), SpanAttributeStage))
	require.Equal(t, codes.Error, stageSpan.Status().Code)
	require.Equal(t, "boom", stageSpan.Status().Description)
}

func attrString(attrs []attribute.KeyValue, key string) string {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value.AsString()
		}
	}
	return ""
}
