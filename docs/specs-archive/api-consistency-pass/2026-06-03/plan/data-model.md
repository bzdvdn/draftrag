# api-consistency-pass Модель данных

## Scope

- Связанные `AC-*`: AC-016 (coverage gate)
- Связанные `DEC-*`: DEC-005 (новый capability-интерфейс)
- Статус: `no-change`

## Сущности

Нет. Фича не вводит, не изменяет и не удаляет persisted entities, value objects, state transitions или contract-relevant payload shapes. Все изменения — в Go-коде (refactor + new optional capability interface), не в runtime-данных pipeline.

## Связи

Нет.

## Производные правила

Нет.

## Переходы состояний

Нет.

## Вне scope

- `TransactionalDocumentStore` — это Go-интерфейс, не persisted entity. Определяется в `internal/domain/interfaces.go` и описывает контракт на работу с SQL-транзакциями в `pgvector`. Не является частью data model.
- Новые поля в `PipelineOptions` (`StreamBufferSize`, `IndexBatchRateLimitPerWorker`) — конфигурационные, не persisted.
- Новый sentinel `ErrUpdateNotAtomic` — runtime error, не persisted.

## No-Change Stub

- Статус: `no-change`
- Причина: фича `api-consistency-pass` фокусируется на архитектурном hardening (refactor роутинга, нормализация ошибок, worker pool reuse, atomic UpdateDocument, streaming backpressure, rate-limiter semantics, синхронизация документации). Никакие persisted entities, value objects, state transitions или contract-relevant payload shapes не затрагиваются. Все новые Go-типы — это интерфейсы, конфиги, sentinel-ошибки и helper-функции, не данные.
- Revisit triggers:
  - `TransactionalDocumentStore` начинает использоваться для хранения метаданных транзакции (rollback-журнал, audit log) — это уже data model concern
  - В `PipelineOptions` добавляется поле, описывающее persisted state (например, `PersistResults bool`)
  - Вводится новая таблица в pgvector-миграции для atomic update tracking
  - Добавляется новый тип `Chunk` или `Document` поле
