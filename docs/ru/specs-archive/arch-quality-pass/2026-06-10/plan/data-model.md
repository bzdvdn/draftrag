---
status: no-change
slug: arch-quality-pass
---

# Data Model: arch-quality-pass

## Статус

**no-change** — доменные сущности (`Document`, `Chunk`, `RetrievalResult`, `Query`, `Embedding`, `HybridConfig`, `IndexBatchResult`, `IndexBatchError`, `InlineCitation`) не меняются.

## Причина

Все три workstream-а spec затрагивают только:

1. **Hooks контракт** — метод `StageStart` начинает возвращать `context.Context`. События (`StageStartEvent`, `StageEndEvent`) не меняются.
2. **Panic→error** — только сигнатуры конструкторов. Новые типы ошибок не вводятся, существующие sentinel-ы переиспользуются.
3. **PipelineConfig удаление** — struct конфигурации перемещается (удаляется `internal/application.PipelineConfig`), но его поля и типы полей остаются идентичными. Единственное изменение: нормализация имени `DedupSourcesByParentID` → `DedupByParentID`.

Ни одна доменная сущность не расширяется и не удаляется.
