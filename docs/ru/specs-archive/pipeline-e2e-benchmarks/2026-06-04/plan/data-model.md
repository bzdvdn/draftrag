# Data Model — Pipeline E2E Benchmarks

## Status

**no-change**

## Причина

Benchmarks — исключительно test-инфраструктура. Никакие доменные типы (Document, Chunk, PipelineConfig и т.д.) не меняются. Helper-типы (benchEmbedder, benchLLM) живут только в test-файлах и не являются частью data model.
