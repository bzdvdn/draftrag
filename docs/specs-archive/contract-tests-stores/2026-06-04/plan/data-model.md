# Data Model — Контрактные тесты VectorStore

## Status

**no-change**

## Причина

Contract-тесты — исключительно test-инфраструктура. Никакие доменные типы (Chunk, RetrievalResult, MetadataFilter и т.д.) не меняются. Единственный новый контракт — `StoreFactory func() domain.VectorStore` — живёт только в test-файлах и не является частью data model.
