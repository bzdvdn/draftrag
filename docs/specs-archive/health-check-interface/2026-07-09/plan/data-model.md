---
status: no-change
reason: Health Check Interface не добавляет новых полей в существующие модели данных. Единственное изменение — новый метод `Health(ctx context.Context) error` в трёх интерфейсах (VectorStore, Embedder, LLMProvider). Никакие структуры (Document, Chunk, RetrievalResult и т.д.) не расширяются.
---
