package otel

// Стабильные имена/ключи observability контракта (v1).

const (
	// SpanAttributeOperation — атрибут span: имя операции pipeline.
	SpanAttributeOperation = "draftrag.operation"
	// SpanAttributeStage — атрибут span: стадия pipeline.
	SpanAttributeStage = "draftrag.stage"
)

const (
	// MetricStageDurationMS — histogram длительности стадии в миллисекундах.
	MetricStageDurationMS = "draftrag.pipeline.stage.duration_ms"
	// MetricStageErrors — counter ошибок стадии.
	MetricStageErrors = "draftrag.pipeline.stage.errors"
)

const (
	// MetricLabelOperation — label метрик: имя операции pipeline.
	MetricLabelOperation = "operation"
	// MetricLabelStage — label метрик: стадия pipeline.
	MetricLabelStage = "stage"
)
