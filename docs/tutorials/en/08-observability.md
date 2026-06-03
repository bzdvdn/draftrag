---
title: Observability (OpenTelemetry)
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Observability with OpenTelemetry

draftRAG supports OpenTelemetry for tracing and metrics across all pipeline stages: chunking, embedding, search, generation.

## 1. Import OTel hooks

```go
import (
    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/otel"
)
```

## 2. Create stdout exporter

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

## 3. Wire hooks into pipeline

```go
tp, mp := setupOTel()
hooks, _ := otel.NewHooks(otel.HooksOptions{
    TracerProvider: tp,
    MeterProvider:  mp,
})

pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Chunker: chunker,
    Hooks:   hooks,
})
```

## 4. Metrics emitted

Each pipeline stage produces:
- **Span** with attributes `operation` and `stage`
- **Histogram**: `draftrag.pipeline.stage.duration_ms`
- **Counter**: `draftrag.pipeline.stage.errors`

## 5. Production OTLP

For production, use OTLP exporters to Jaeger or Grafana:

```go
import (
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)
```

## Next

Proceed to [09-evaluation.md](09-evaluation.md) — evaluating RAG quality.
