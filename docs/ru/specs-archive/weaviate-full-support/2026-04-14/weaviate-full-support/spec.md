# Weaviate Full Support

## Scope Snapshot

- **In scope:** Стабилизация Weaviate до production-ready статуса с полным паритетом возможностей с другими vector stores (pgvector, Qdrant, ChromaDB)
- **Out of scope:** Добавление новых features специфичных для Weaviate (например, modular AI integrations)

## Цель

Weaviate должен стать production-ready vector store с тем же уровнем поддержки и консистентности API, как у pgvector и Qdrant. Пользователи смогут использовать Weaviate в production без опасений о breaking changes, с гарантированным покрытием тестами и документацией.

## Основной сценарий

1. **Стартовая точка:** Weaviate имеет experimental статус, API с префиксом `Weaviate*`, нет hybrid search, покрытие тестами частичное
2. **Основное действие:** Стабилизировать API, добавить недостающие функции (hybrid search), обеспечить полное тестовое покрытие, обновить документацию
3. **Результат:** Weaviate помечается как stable в `compatibility.md`, API консистентен с другими хранилищами, все тесты проходят, документация production-ready
4. **Fallback-путь:** Если hybrid search невозможно реализовать нативно, явно документировать ограничение (как для ChromaDB)

## Scope

### Входит

- Переименование функций для консистентности с другими хранилищами (breaking change до v1.0)
- Реализация hybrid search (BM25) если Weaviate поддерживает нативно
- Полное покрытие тестами (timeout, auth, errors, edge cases)
- Обновление документации до production уровня
- Изменение статуса с experimental → stable в `compatibility.md`
- Обновление `ROADMAP.md` — отметка Weaviate как production-ready

### Поверхность

- `pkg/draftrag/weaviate.go` — публичный API
- `pkg/draftrag/weaviate_test.go` — тесты
- `docs/weaviate.md` — документация
- `docs/compatibility.md` — статус backend'а
- `ROADMAP.md` — статус фичи
- `internal/infrastructure/vectorstore/weaviate.go` — внутренняя реализация (если требуется)

## Контекст

- **Ограничение:** Продукт ещё не зарелизился (v0.x), breaking changes допустимы
- **Существующий поток:** Weaviate уже имеет базовую функциональность (retrieval, фильтрация, управление коллекциями)
- **Предположение:** Weaviate поддерживает BM25 или альтернативный hybrid search нативно; если нет — нужно явно документировать

## Требования

### RQ-001: Консистентность именования API

Переименовать функции Weaviate для consistency с другими хранилищами:

```go
// Было (с префиксом Weaviate) → Станет (без префикса или с коротким префиксом)
WeaviateCollectionExists → CollectionExists (или WeaviateCollectionExists для уникальности в пакете)
CreateWeaviateCollection → CreateCollection (или CreateWeaviateCollection)
DeleteWeaviateCollection → DeleteCollection (или DeleteWeaviateCollection)
```

**Обоснование:** Продукт еще не зарелизился, breaking changes допустимы. Единообразие API важнее временной совместимости.

**Примечание:** Если в пакете уже есть `CreateCollection`/`DeleteCollection`/`CollectionExists` для Qdrant, использовать короткий префикс `Weaviate*` вместо `WeaviateDB*` (как было сделано для ChromaDB).

### RQ-002: Hybrid search (BM25)

Реализовать hybrid search (BM25 + semantic) для Weaviate, если нативно поддерживается.

- Если Weaviate поддерживает BM25 нативно → реализовать hybrid search через Weaviate API
- Если Weaviate НЕ поддерживает BM25 → явно документировать ограничение в `docs/weaviate.md` и `compatibility.md`

**Обоснование:** Parity с pgvector (который имеет hybrid search) важен для консистентности vector store API.

### RQ-003: Полное тестовое покрытие

Добавить тесты для всех сценариев:

- ✅ Базовые операции (создание/удаление/проверка коллекции)
- ✅ Retrieval с фильтрами
- ✅ Timeout и context cancellation
- ✅ Authentication (401/403 errors)
- ✅ Error handling (404, 500, network errors)
- ✅ Edge cases (пустые данные, дубликаты)

**Обоснование:** Production-ready статус требует надежности через тесты.

### RQ-004: Production документация

Обновить `docs/weaviate.md` до production уровня:

- Production checklist (deployment, monitoring, best practices)
- Performance guidance (batch size, timeouts, indexing)
- Migration guide (если есть breaking changes)
- Troubleshooting guide (расширенный)

### RQ-005: Статус backend'а

Изменить статус Weaviate с `experimental` → `stable` в `docs/compatibility.md` после выполнения всех AC.

## Вне scope

- Добавление Weaviate-specific features (например, GraphQL mutations, custom modules)
- Интеграция с Weaviate Cloud специфичными API (если они не являются частью core functionality)
- Performance tuning за пределами разумных defaults (например, advanced indexing strategies)

## Критерии приемки

### AC-001: API консистентность

- **Почему это важно:** Единообразие API упрощает обучение и использование разных vector stores
- **Given:** Weaviate функции имеют префикс `Weaviate*`
- **When:** Выполняется переименование функций
- **Then:** Имена функций консистентны с другими хранилищами (с префиксом для уникальности в пакете)
- **Evidence:** `pkg/draftrag/weaviate.go` содержит функции с консистентными именами, все godoc обновлены

### AC-002: Hybrid search или документирование ограничения

- **Почему это важно:** Parity с pgvector обеспечивает предсказуемый UX
- **Given:** Weaviate имеет базовый semantic search
- **When:** Проверяется поддержка BM25 в Weaviate API
- **Then:** Либо реализован hybrid search, либо явно документировано ограничение
- **Evidence:** `docs/weaviate.md` содержит секцию о hybrid search с явным статусом (поддерживается/не поддерживается)

### AC-003: Тестовое покрытие

- **Почему это важно:** Production-ready код требует надежности
- **Given:** Существующие тесты покрывают базовые сценарии
- **When:** Добавляются тесты для всех сценариев
- **Then:** Все тесты проходят, покрытие >= 90%
- **Evidence:** `go test ./pkg/draftrag -run Weaviate` проходит, `go test ./...` зеленый

### AC-004: Production документация

- **Почему это важно:** Production использование требует clear guidance
- **Given:** `docs/weaviate.md` содержит базовый quickstart
- **When:** Документация расширяется до production уровня
- **Then:** Документация содержит production checklist, performance guidance, troubleshooting
- **Evidence:** `docs/weaviate.md` содержит все секции из RQ-004

### AC-005: Статус backend'а

- **Почему это важно:** Пользователи должны знать, можно ли использовать в production
- **Given:** Weaviate имеет experimental статус
- **When:** Все AC выполнены
- **Then:** Статус изменен на stable в `compatibility.md`
- **Evidence:** `docs/compatibility.md` показывает Weaviate = stable

## Допущения

- Weaviate API стабилен и не будет иметь breaking changes в ближайшее время
- Пользователи имеют доступ к Weaviate инстансу (self-hosted или cloud)
- Hybrid search можно реализовать через Weaviate API или явно задокументировать невозможность

## Критерии успеха

- **SC-001:** Test coverage >= 90% для `pkg/draftrag/weaviate.go`
- **SC-002:** Все тесты проходят за < 30s на CI

## Краевые случаи

- **Пустое состояние:** Коллекция не существует → `WeaviateCollectionExists` возвращает false, не error
- **Timeout:** Context cancellation корректно обрабатывается во всех операциях
- **Auth errors:** 401/403 возвращаются как явные errors, не panic
- **Network errors:** Retry behavior documented (если применимо)

## Открытые вопросы

- **Q1:** Поддерживает ли Weaviate BM25 нативно? Если да, через какой API? [NEEDS CLARIFICATION: требуется проверка Weaviate docs]
- **Q2:** Если BM25 не поддерживается, стоит ли реализовать hybrid search через external index (как рассматривалось для ChromaDB)? [NEEDS CLARIFICATION: решение продукта]
- **Q3:** Стоит ли использовать короткий префикс `Weaviate*` или вообще убрать префикс (если возможно через разные packages)? [NEEDS CLARIFICATION: архитектурное решение]

---

**Статус:** Черновик  
**Создан:** 2026-04-14  
**Автор:** speckeep
