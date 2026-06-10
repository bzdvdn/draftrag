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
	"go.opentelemetry.io/otel/trace"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test arch-quality-pass#T4.2: StageStart создаёт span, StageEnd завершает (AC-005)
func TestHooks_StageStart_CreatesSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	hooks, err := NewHooks(HooksOptions{
		TracerProvider: tp,
		MeterProvider:  noop.NewMeterProvider(),
	})
	require.NoError(t, err)

	parentTracer := tp.Tracer("test")
	ctx, parent := parentTracer.Start(context.Background(), "parent")

	spanCtx := hooks.StageStart(ctx, domain.StageStartEvent{
		Operation: "Answer",
		Stage:     domain.HookStageGenerate,
	})

	span := trace.SpanFromContext(spanCtx)
	require.True(t, span.IsRecording(), "StageStart должен создать активный span в контексте")

	hooks.StageEnd(spanCtx, domain.StageEndEvent{
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
	require.NotNil(t, stageSpan, "должен быть span draftrag.generate")

	require.Equal(t, parent.SpanContext().SpanID(), stageSpan.Parent().SpanID())
	require.Equal(t, "Answer", attrString(stageSpan.Attributes(), SpanAttributeOperation))
	require.Equal(t, "generate", attrString(stageSpan.Attributes(), SpanAttributeStage))
	require.Equal(t, codes.Error, stageSpan.Status().Code)
	require.Equal(t, "boom", stageSpan.Status().Description)

	endTime := stageSpan.EndTime()
	startTime := stageSpan.StartTime()
	require.False(t, startTime.IsZero(), "StartTime должен быть заполнен")
	require.False(t, endTime.IsZero(), "EndTime должен быть заполнен")
	require.WithinRange(t, endTime, startTime, time.Now())
}

// @sk-test arch-quality-pass#T4.2: StageEnd без StageStart — no-op (AC-005)
func TestHooks_StageEnd_NoStart_NoOp(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	hooks, err := NewHooks(HooksOptions{
		TracerProvider: tp,
		MeterProvider:  noop.NewMeterProvider(),
	})
	require.NoError(t, err)

	hooks.StageEnd(context.Background(), domain.StageEndEvent{
		Operation: "Answer",
		Stage:     domain.HookStageGenerate,
		Duration:  100 * time.Millisecond,
	})

	ended := recorder.Ended()
	require.Empty(t, ended, "без StageStart span не должен создаваться")
}

func attrString(attrs []attribute.KeyValue, key string) string {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value.AsString()
		}
	}
	return ""
}
