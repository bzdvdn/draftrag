---
title: Наблюдаемость (OpenTelemetry)
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Наблюдаемость через OpenTelemetry

draftRAG поддерживает OpenTelemetry для трассировки и метрик каждого этапа пайплайна: чанкинг, эмбеддинг, поиск, генерация.

## 1. Подключите OTel hooks

```go
import (
    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/otel"
)
```

## 2. Создайте stdout-экспортёр

Для разработки экспортируйте трейсы и метрики в stdout:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/trace"
)

func setupOTel() (*trace.TracerProvider, *metric.MeterProvider) {
    traceExporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
    tp := trace.NewTracerProvider(trace.WithBatcher(traceExporter))
    otel.SetTracerProvider(tp)

    metricExporter, _ := stdoutmetric.New()
    mp := metric.NewMeterProvider(metric.WithReader(
        metric.NewPeriodicReader(metricExporter)))
    otel.SetMeterProvider(mp)

    return tp, mp
}
```

## 3. Создайте hooks и подключите к пайплайну

```go
tp, mp := setupOTel()

hooks, err := otel.NewHooks(otel.HooksOptions{
    TracerProvider: tp,
    MeterProvider:  mp,
})
if err != nil {
    log.Fatal(err)
}

pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Chunker: chunker,
    Hooks:   hooks,
})
```

## 4. Анализируйте вывод

После запроса в stdout появятся:

- **Span** для каждого этапа: `draftrag.pipeline.stage` с атрибутами `operation` и `stage`
- **Метрики**: `draftrag.pipeline.stage.duration_ms` (гистограмма), `draftrag.pipeline.stage.errors` (counter)

```json
{
  "Name": "draftrag.pipeline.stage.duration_ms",
  "Data": {
    "DataPoints": [{
      "Attributes": [{"Key": "stage", "Value": "search"}],
      "Value": 42.5
    }]
  }
}
```

## 5. Мониторинг в production

Для production подключите OTLP-экспортёр в Jaeger или Grafana:

```go
import (
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)
```

## Что дальше?

Переходите к [09-evaluation.md](09-evaluation.md) — оценка качества RAG.
