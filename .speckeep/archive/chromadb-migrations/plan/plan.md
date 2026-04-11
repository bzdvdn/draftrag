# ChromaDB: миграции коллекций — План

## Phase Contract

**Inputs**: spec, inspect report, контекст существующих реализаций Qdrant и ChromaDB  
**Outputs**: plan.md, data-model.md, contracts/api.md  
**Stop conditions**: нет — spec одобрен, implementation surface ясен

---

## Цель

Расширить публичный API пакета `draftrag` функциями управления коллекциями ChromaDB (`CreateCollection`, `DeleteCollection`, `CollectionExists`), консистентно с существующим API для Qdrant. Добавить `Validate()` к `ChromaDBOptions` и обновить `NewChromaDBStore` для его использования.

---

## Scope

- **Входит**: `pkg/draftrag/chromadb.go` — новые функции и метод `Validate()`
- **Входит**: `pkg/draftrag/chromadb_test.go` — тесты для миграций
- **Входит**: Обновление `NewChromaDBStore` для использования `Validate()`
- **Не входит**: Изменения в `internal/infrastructure/vectorstore/chromadb.go` — HTTP-логика остаётся в infrastructure, API-обёртки в pkg
- **Не входит**: ChromaDB hybrid search (вне scope spec)

---

## Implementation Surfaces

| Surface | Тип | Описание |
|---|---|---|
| `pkg/draftrag/chromadb.go` | Существующий | Добавить `Validate()`, `CreateCollection`, `DeleteCollection`, `CollectionExists` |
| `pkg/draftrag/chromadb_test.go` | Новый | Тесты для миграций с HTTP mock-сервером (как `qdrant_test.go`) |

---

## Влияние на архитектуру

- **Локальное**: Только в API слое (`pkg/draftrag/`)
- **Чистая архитектура**: HTTP-логика остаётся в infrastructure, в pkg только публичные обёртки с контекстом
- **Compatibility**: `NewChromaDBStore` сохраняет сигнатуру, меняется только внутренняя валидация (использует `Validate()` вместо inline проверок)
- **Консистентность**: API ChromaDB миграций полностью соответствует API Qdrant миграций

---

## Acceptance Approach

### AC-001 Создание коллекции
- **Реализация**: `CreateCollection(ctx, opts)` в `chromadb.go`
- **Surface**: `pkg/draftrag/chromadb.go`
- **Подход**: HTTP POST `/api/v1/collections` с JSON body `{name, metadata, ...}`; проверка статуса 200
- **Evidence**: Тест с mock-сервером проверяет URL, метод, body; ошибка при статусе ≠ 200

### AC-002 Удаление коллекции
- **Реализация**: `DeleteCollection(ctx, opts)` в `chromadb.go`
- **Surface**: `pkg/draftrag/chromadb.go`
- **Подход**: HTTP DELETE `/api/v1/collections/{name}`; 404 считается success (идемпотентность)
- **Evidence**: Тест проверяет URL, метод; 404 возвращает `nil` ошибки

### AC-003/004 Проверка существования
- **Реализация**: `CollectionExists(ctx, opts)` в `chromadb.go`
- **Surface**: `pkg/draftrag/chromadb.go`
- **Подход**: HTTP GET `/api/v1/collections/{name}`; 200 → `true`, 404 → `false`
- **Evidence**: Тесты для обоих сценариев; проверка корректного парсинга ответа

### AC-005 Валидация опций
- **Реализация**: `(o ChromaDBOptions) Validate() error` в `chromadb.go`
- **Surface**: `pkg/draftrag/chromadb.go`
- **Подход**: Проверка `Collection != ""` и `Dimension > 0`; возврат понятных ошибок
- **Evidence**: Тесты на каждое условие валидации; `NewChromaDBStore` вызывает `Validate()`

### AC-006 Контекстная отмена
- **Реализация**: Проброс `ctx` в `http.NewRequestWithContext` во всех функциях
- **Surface**: `pkg/draftrag/chromadb.go`
- **Подход**: Стандартный механизм Go HTTP client с context
- **Evidence**: Тест с `context.WithTimeout` показывает `DeadlineExceeded`

---

## Данные и контракты

**Data model**: Нет изменений — сущности `Chunk`, `Document` и другие не меняются. Коллекция ChromaDB — это external state, не domain entity.

**API Contracts**: См. `contracts/api.md` — HTTP endpoints ChromaDB REST API v1.

---

## Стратегия реализации

### DEC-001 ChromaDB API endpoint: POST /api/v1/collections для CreateCollection
- **Why**: ChromaDB использует другой endpoint чем Qdrant (Qdrant: PUT /collections/{name}, ChromaDB: POST /api/v1/collections с body)
- **Tradeoff**: Необходимо знать ChromaDB API; нельзя просто скопировать Qdrant реализацию
- **Affects**: `CreateCollection` в `chromadb.go`
- **Validation**: Тест проверяет корректный URL и body

### DEC-002 ChromaDB API endpoint: GET /api/v1/collections/{name} для CollectionExists
- **Why**: ChromaDB не имеет отдельного `/exists` endpoint как Qdrant; проверка через GET
- **Tradeoff**: 404 = не существует, любой другой error = реальная ошибка
- **Affects**: `CollectionExists` в `chromadb.go`
- **Validation**: Тесты на 200 и 404 ответы

### DEC-003 Добавить Timeout в ChromaDBOptions (как у QdrantOptions)
- **Why**: Консистентность API; production-ready таймауты
- **Tradeoff**: Расширение структуры, но backward compatible (zero value = default)
- **Affects**: `ChromaDBOptions` struct, все migration функции
- **Validation**: Тест на таймаут с коротким deadline

---

## Incremental Delivery

### MVP (Первая ценность)

1. **Добавить `Validate()` и обновить `NewChromaDBStore`**
   - AC-005 покрыт
   - Быстрая проверка: `go test` на валидацию

2. **Реализовать `CreateCollection` + тест**
   - AC-001 покрыт
   - ChromaDB коллекция создаётся программно

### Итеративное расширение

3. **Реализовать `DeleteCollection` + тест**
   - AC-002 покрыт

4. **Реализовать `CollectionExists` + тест**
   - AC-003, AC-004 покрыты

5. **Добавить тест на контекстную отмену**
   - AC-006 покрыт

---

## Порядок реализации

1. **`Validate()` для `ChromaDBOptions`** — блокер для остальных функций, foundation
2. **`CreateCollection`** — основная ценность, можно тестировать
3. **`DeleteCollection`** — параллельно с Create, независимая функция
4. **`CollectionExists`** — зависит от понимания ChromaDB GET endpoint
5. **Обновление `NewChromaDBStore`** — рефакторинг с использованием `Validate()`
6. **Тест на cancellation** — можно параллелить с пунктами 2-4

---

## Риски

| Риск | Mitigation |
|---|---|
| ChromaDB API 409 на duplicate collection | Документировать поведение: возвращаем ошибку (как ChromaDB), пользователь может вызвать `CollectionExists` перед созданием |
| Невалидный JSON от ChromaDB | Оборачивать в `fmt.Errorf("chromadb decode: %w", err)` с контекстом |
| Отличие ChromaDB API от Qdrant | Проверять ChromaDB REST API документацию; тесты с реальными HTTP responses |

---

## Rollout и compatibility

- **Backward compatible**: `NewChromaDBStore` сохраняет сигнатуру, только меняет internal implementation
- **Breaking changes**: Нет
- **Migration**: Не требуется
- **Feature flags**: Не требуются

---

## Проверка

| Проверка | AC/DEC | Метод |
|---|---|---|
| `go test ./pkg/draftrag/...` | Все AC | Автоматический |
| Покрытие >80% для новых функций | SC-001 | `go test -cover` |
| Консистентность с Qdrant API | SC-002 | Ручной code review |
| Пример использования | SC-003 | Ручная проверка примера или создание `examples/chromadb/` |

---

## Соответствие конституции

| Принцип | Применение | Статус |
|---|---|---|
| Интерфейсная абстракция | Новые функции — обёртки над HTTP API, не меняют domain интерфейсы | ✅ |
| Чистая архитектура | HTTP-логика в pkg (API layer), не проникает в domain/application | ✅ |
| Контекстная безопасность | Все функции принимают `context.Context` первым параметром | ✅ |
| Тестируемость | HTTP mock-сервер в тестах, как у Qdrant | ✅ |
| Языковая политика | Комментарии на русском, godoc на русском | ✅ |

**Конфликтов нет**.

---

**Slug**: `chromadb-migrations`  
**Следующая команда**: `/draftspec.tasks chromadb-migrations`
