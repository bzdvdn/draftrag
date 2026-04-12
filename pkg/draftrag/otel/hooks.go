package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// HooksOptions задаёт конфигурацию OTel hooks.
type HooksOptions struct {
	// TracerProvider — источник tracer'а. Если nil, используется глобальный.
	TracerProvider trace.TracerProvider
	// MeterProvider — источник meter'а. Если nil, используется глобальный.
	MeterProvider metric.MeterProvider

	// TracerName — имя tracer'а. Пустое → дефолт.
	TracerName string
	// MeterName — имя meter'а. Пустое → дефолт.
	MeterName string
}

func (o HooksOptions) tracer() trace.Tracer {
	tp := o.TracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	name := o.TracerName
	if name == "" {
		name = "github.com/bzdvdn/draftrag/pkg/draftrag/otel"
	}
	return tp.Tracer(name)
}

func (o HooksOptions) meter() metric.Meter {
	mp := o.MeterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}
	name := o.MeterName
	if name == "" {
		name = "github.com/bzdvdn/draftrag/pkg/draftrag/otel"
	}
	return mp.Meter(name)
}

// Hooks — OpenTelemetry-реализация `draftrag.Hooks`.
//
// Контракт (v1):
// - spans: атрибуты `draftrag.operation`, `draftrag.stage`;
// - metrics: `draftrag.pipeline.stage.duration_ms` и `draftrag.pipeline.stage.errors` с labels `operation`, `stage`.
//
// Важно: hooks вызываются синхронно, поэтому exporter/SDK должны быть неблокирующими.
type Hooks struct {
	tracer trace.Tracer

	duration metric.Float64Histogram
	errors   metric.Int64Counter
}

var _ domain.Hooks = (*Hooks)(nil)

// NewHooks создаёт OpenTelemetry hooks для стадий pipeline draftRAG.
//
// Hooks опциональны: подключите их через `draftrag.PipelineOptions{Hooks: ...}`.
// Использование не требует форка библиотеки и не меняет поведение pipeline.
func NewHooks(opts HooksOptions) (*Hooks, error) {
	m := opts.meter()

	duration, err := m.Float64Histogram(
		MetricStageDurationMS,
		metric.WithUnit("ms"),
		metric.WithDescription("Длительность стадий pipeline draftRAG (chunking/embed/search/generate)"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := m.Int64Counter(
		MetricStageErrors,
		metric.WithUnit("1"),
		metric.WithDescription("Количество ошибок стадий pipeline draftRAG"),
	)
	if err != nil {
		return nil, err
	}

	return &Hooks{
		tracer:   opts.tracer(),
		duration: duration,
		errors:   errors,
	}, nil
}

// StageStart реализует `Hooks` интерфейс pipeline (domain.Hooks).
//
// В v1 мы не создаём span на старте, т.к. интерфейс hooks не возвращает `context.Context`.
// Вместо этого создаём ретроспективный stage span на `StageEnd` по измеренной длительности.
func (h *Hooks) StageStart(ctx context.Context, ev domain.StageStartEvent) {
	_ = ctx
	_ = ev
}

// StageEnd реализует `Hooks` интерфейс pipeline (domain.Hooks).
func (h *Hooks) StageEnd(ctx context.Context, ev domain.StageEndEvent) {
	if ctx == nil {
		ctx = context.Background()
	}

	operation := ev.Operation
	stage := string(ev.Stage)

	spanAttrs := []attribute.KeyValue{
		attribute.String(SpanAttributeOperation, operation),
		attribute.String(SpanAttributeStage, stage),
	}

	metricAttrs := []attribute.KeyValue{
		attribute.String(MetricLabelOperation, operation),
		attribute.String(MetricLabelStage, stage),
	}

	// Пишем метрики.
	h.duration.Record(ctx, float64(ev.Duration.Milliseconds()), metric.WithAttributes(metricAttrs...))
	if ev.Err != nil {
		h.errors.Add(ctx, 1, metric.WithAttributes(metricAttrs...))
	}

	// Пишем tracing: ретроспективный span со start/end timestamp.
	startTime := time.Now().Add(-ev.Duration)
	spanCtx, span := h.tracer.Start(
		ctx,
		"draftrag."+stage,
		trace.WithTimestamp(startTime),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(spanAttrs...),
	)
	if ev.Err != nil {
		span.RecordError(ev.Err)
		span.SetStatus(codes.Error, ev.Err.Error())
	}
	span.End(trace.WithTimestamp(startTime.Add(ev.Duration)))
	_ = spanCtx
}
