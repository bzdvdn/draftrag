# Weaviate documentation

## Scope Snapshot

- In scope: добавить публичную документацию по Weaviate vector store: как подключить `NewWeaviateStore`, как создавать/проверять/удалять коллекцию, какие возможности поддерживаются (filters, metadata filters), и где это найти (ссылки из существующих docs).
- Out of scope: изменения реализации Weaviate store, добавление новых возможностей (hybrid/streaming и т.п.), изменение публичного API, добавление CI/integration тестов с реальным Weaviate.

## Цель

Пользователь должен иметь “production-minded” инструкцию по Weaviate: быстрый старт, требования к схеме/коллекции, типовые ошибки (collection missing, auth, timeouts), и понимание ограничений.

## Основной сценарий

1. Пользователь выбирает Weaviate как vector store.
2. Пользователь открывает документацию и находит раздел/страницу про Weaviate.
3. Пользователь копипастит пример: создаёт store, подготавливает коллекцию (deploy job/init), индексирует документы и делает поиск.
4. При ошибках пользователь понимает, какие проверки сделать (collection exists, APIKey, timeout).

## Scope

- Добавить новый документ `docs/weaviate.md` на русском языке.
- Добавить ссылки на документ:
  - из `docs/vector-stores.md` (в списке хранилищ) и/или в таблице сравнения;
  - из `docs/compatibility.md` (исправить примечание “нет дока”, если оно есть).
- Документ должен описывать только публичный API `pkg/draftrag` (без ссылок на `internal/`).

## Требования

- RQ-001 ДОЛЖЕН существовать документ `docs/weaviate.md` (русский) с быстрым стартом.
- RQ-002 Документ ДОЛЖЕН показывать, как создать store через `draftrag.NewWeaviateStore(draftrag.WeaviateOptions{...})`.
- RQ-003 Документ ДОЛЖЕН описывать управление коллекцией через публичные функции `WeaviateCollectionExists`, `CreateWeaviateCollection`, `DeleteWeaviateCollection` и подчеркнуть, что DDL/schema в production обычно делается отдельным шагом деплоя.
- RQ-004 Документ ДОЛЖЕН явно описать поддерживаемые возможности Weaviate store (минимум: поиск, `Filter/ParentIDs` и metadata filters) и ограничения (например, нет hybrid search BM25).
- RQ-005 ДОЛЖНЫ быть добавлены ссылки на `docs/weaviate.md` из существующих docs (как минимум `docs/vector-stores.md`), чтобы документ был обнаруживаемым.
- RQ-006 Изменения не ДОЛЖНЫ требовать правок кода библиотеки (только docs).

## Критерии приемки

### AC-001 Документ Weaviate добавлен и обнаруживаем

- **Given** пользователь читает `docs/vector-stores.md`
- **When** он ищет Weaviate
- **Then** он находит ссылку на `docs/weaviate.md` и может открыть документ
- Evidence: `docs/weaviate.md` существует + есть явная ссылка в `docs/vector-stores.md`

### AC-002 Quickstart покрывает жизненный цикл: подготовка → store → index → retrieve

- **Given** пользователь хочет запустить RAG на Weaviate
- **When** он следует quickstart в `docs/weaviate.md`
- **Then** он понимает как: (1) подготовить коллекцию, (2) создать store, (3) индексировать, (4) выполнить поиск
- Evidence: в документе есть компилируемый пример кода (можно частями), использующий `context.WithTimeout`

### AC-003 Документ фиксирует возможности/ограничения и типовые ошибки

- **Given** пользователь сравнивает backend’ы
- **When** он читает `docs/weaviate.md`
- **Then** он видит: что поддерживается (filters/metadata), что не поддерживается (hybrid), и что делать при типовых ошибках (404/collection missing, auth, timeout)
- Evidence: отдельные секции “Возможности/ограничения” и “Типовые ошибки”

## Допущения

- Документация основана на текущем публичном API и существующих docs; без “обещаний SLA”.

## Открытые вопросы

- none

