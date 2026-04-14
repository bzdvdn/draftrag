# Сводка архива

## Спецификация

- snapshot: добавлены DeleteDocument и UpdateDocument во все 4 vector stores и Pipeline
- slug: document-lifecycle
- archived_at: 2026-04-09
- status: completed

## Причина

Без возможности удалить или обновить документ пользователи были вынуждены пересоздавать весь store при изменении контента. Особенно критично для систем с изменяемым контентом (wiki, knowledge base).

## Результат

- `domain.DocumentStore` интерфейс с `DeleteByParentID`.
- `DeleteByParentID` реализован в InMemory, PGVector, Qdrant, ChromaDB.
- Compile-time assertions в Qdrant и ChromaDB.
- `Pipeline.DeleteDocument` / `UpdateDocument` с capability check → `ErrDeleteNotSupported`.
- Тесты: 4 теста для Qdrant + 4 для ChromaDB с mock HTTP servers.
- Документация в `docs/pipeline.md`.

## Продолжение

- Транзакционный UpdateDocument (delete + index в одной операции).
- Batch delete ([]docID за один вызов).
- Soft delete с tombstone записями.
