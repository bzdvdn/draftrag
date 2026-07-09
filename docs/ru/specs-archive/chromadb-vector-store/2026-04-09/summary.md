---
slug: chromadb-vector-store
generated_at: 2026-04-09T13:25:00+03:00
---

## Goal

Реализация ChromaStore — backend для работы с ChromaDB через HTTP API для быстрого прототипирования RAG-систем.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Успешный upsert чанка | Тест: запись через прямой GET из ChromaDB |
| AC-002 | Поиск возвращает релевантные результаты | Тест: top-3 чанка с корректными score > 0 |
| AC-003 | Фильтрация по метаданным работает | Тест: только matching чанки возвращены |
| AC-004 | Удаление чанка по ID | Тест: поиск не возвращает удалённый чанк |
| AC-005 | Валидация размерности эмбеддинга | Тест: возврат ErrEmbeddingDimensionMismatch |
| AC-006 | Context cancellation прерывает операцию | Тест: context.DeadlineExceeded при timeout |
| AC-007 | Автосоздание коллекции | Тест: коллекция создана через API после операции |

## Out of Scope

- Гибридный поиск (BM25 + semantic)
- Batch-операции для массового upsert/delete
- Multi-tenant сценарии с tenant/workspace изоляцией
- Retry и circuit breaker логика
- Persistence настроек коллекции между перезапусками
