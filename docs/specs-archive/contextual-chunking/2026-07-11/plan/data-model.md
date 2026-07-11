---
status: no-change
---

# Data Model: contextual-chunking

Data model не меняется. `ContextualChunker` работает с существующими типами `domain.Document`, `domain.Chunk`, `domain.Chunker` без расширения. Контекст сохраняется внутри `Chunk.Content` через шаблон — отдельного поля или типа не требуется.
