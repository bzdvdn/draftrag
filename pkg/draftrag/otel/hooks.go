package otel

import (
	"context"

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
// @sk-task arch-quality-pass#T1.2: возвращает ctx для compatibility; span создаётся в T3.1 (AC-001)
// @sk-task arch-quality-pass#T3.1: OTel StageStart создаёт span (AC-001, AC-005)
func (h *Hooks) StageStart(ctx context.Context, ev domain.StageStartEvent) context.Context {
	spanCtx, span := h.tracer.Start(ctx, "draftrag."+string(ev.Stage),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(SpanAttributeOperation, ev.Operation),
			attribute.String(SpanAttributeStage, string(ev.Stage)),
		),
	)
	_ = span
	return spanCtx
}

// StageEnd реализует `Hooks` интерфейс pipeline (domain.Hooks).
//
// @sk-task arch-quality-pass#T3.1: StageEnd завершает span из context (AC-001, AC-005)
func (h *Hooks) StageEnd(ctx context.Context, ev domain.StageEndEvent) {
	if ctx == nil {
		ctx = context.Background()
	}

	operation := ev.Operation
	stage := string(ev.Stage)

	metricAttrs := []attribute.KeyValue{
		attribute.String(MetricLabelOperation, operation),
		attribute.String(MetricLabelStage, stage),
	}

	// Пишем метрики.
	h.duration.Record(ctx, float64(ev.Duration.Milliseconds()), metric.WithAttributes(metricAttrs...))
	if ev.Err != nil {
		h.errors.Add(ctx, 1, metric.WithAttributes(metricAttrs...))
	}

	// Завершаем span, созданный в StageStart.
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		if ev.Err != nil {
			span.RecordError(ev.Err)
			span.SetStatus(codes.Error, ev.Err.Error())
		}
		span.End()
	}
}
