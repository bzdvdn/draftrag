# ChromaDB: миграции коллекций — Задачи

## Phase Contract

**Inputs**: plan.md, spec.md, contracts/api.md  
**Outputs**: tasks.md с декомпозицией работы  
**Stop conditions**: нет — план конкретен, все AC покрываются

---

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/chromadb.go` | T1.1, T1.2, T2.1, T2.2, T2.3, T2.4 |
| `pkg/draftrag/chromadb_test.go` | T3.1, T3.2, T3.3, T3.4, T3.5 |

---

## Фаза 1: Основа — Validate и рефакторинг

**Цель**: Добавить `Validate()` к `ChromaDBOptions` и обновить `NewChromaDBStore` для консистентности с `QdrantOptions` — foundation для миграций.

- [x] **T1.1** Добавить метод `Validate()` к `ChromaDBOptions` — проверяет `Collection != ""` и `Dimension > 0`, возвращает понятные ошибки. Touches: `pkg/draftrag/chromadb.go`. AC-005, RQ-004

- [x] **T1.2** Обновить `NewChromaDBStore` для использования `opts.Validate()` вместо inline проверок — сохраняет сигнатуру, меняет только внутреннюю реализацию. Touches: `pkg/draftrag/chromadb.go`. AC-005, RQ-006

---

## Фаза 2: Основная реализация — миграции

**Цель**: Реализовать три функции миграций с HTTP-логикой, обработкой ошибок и контекстом.

- [x] **T2.1** Реализовать `CreateCollection(ctx, opts)` — HTTP POST `/api/v1/collections` с body `{name, metadata}`, проверка статуса 200, оборачивание ошибок. Touches: `pkg/draftrag/chromadb.go`. AC-001, RQ-001, DEC-001

- [x] **T2.2** Реализовать `DeleteCollection(ctx, opts)` — HTTP DELETE `/api/v1/collections/{name}`, 404 считается success (идемпотентность). Touches: `pkg/draftrag/chromadb.go`. AC-002, RQ-002

- [x] **T2.3** Реализовать `CollectionExists(ctx, opts)` — HTTP GET `/api/v1/collections/{name}`, возвращает `(true, nil)` при 200, `(false, nil)` при 404. Touches: `pkg/draftrag/chromadb.go`. AC-003, AC-004, RQ-003, DEC-002

- [x] **T2.4** Добавить поле `Timeout time.Duration` в `ChromaDBOptions` — консистентность с `QdrantOptions`, zero value = default 10s. Touches: `pkg/draftrag/chromadb.go`. DEC-003

---

## Фаза 3: Проверка — тесты

**Цель**: Покрыть все миграции unit-тестами с HTTP mock-сервером (как `qdrant_test.go`).

- [x] **T3.1** Добавить тест `TestChromaDBOptions_Validate` — проверка валидации для пустого Collection и невалидного Dimension. Touches: `pkg/draftrag/chromadb_test.go`. AC-005

- [x] **T3.2** Добавить тест `TestCreateCollection` — mock-сервер проверяет POST `/api/v1/collections`, корректный body, success path. Touches: `pkg/draftrag/chromadb_test.go`. AC-001

- [x] **T3.3** Добавить тест `TestDeleteCollection` — mock-сервер проверяет DELETE `/api/v1/collections/{name}`, 200 = success, 404 = nil error (идемпотентность). Touches: `pkg/draftrag/chromadb_test.go`. AC-002

- [x] **T3.4** Добавить тесты `TestCollectionExists` и `TestCollectionExists_NotFound` — mock-сервер проверяет GET `/api/v1/collections/{name}`, 200 → true, 404 → false. Touches: `pkg/draftrag/chromadb_test.go`. AC-003, AC-004

- [x] **T3.5** Добавить тест `TestCreateCollection_ContextTimeout` — проверка cancellation с `context.WithTimeout`. Touches: `pkg/draftrag/chromadb_test.go`. AC-006, RQ-005

---

## Покрытие критериев приемки

| AC | Задачи | Статус |
|---|---|---|
| AC-001 Создание коллекции | T2.1, T3.2 | ✅ |
| AC-002 Удаление коллекции | T2.2, T3.3 | ✅ |
| AC-003 Проверка существования (есть) | T2.3, T3.4 | ✅ |
| AC-004 Проверка существования (нет) | T2.3, T3.4 | ✅ |
| AC-005 Валидация опций | T1.1, T1.2, T3.1 | ✅ |
| AC-006 Контекстная отмена | T2.1, T2.2, T2.3, T3.5 | ✅ |

---

## Покрытие требований

| RQ | Задачи |
|---|---|
| RQ-001 CreateCollection | T2.1 |
| RQ-002 DeleteCollection | T2.2 |
| RQ-003 CollectionExists | T2.3 |
| RQ-004 Validate() | T1.1 |
| RQ-005 Context support | T2.1, T2.2, T2.3, T3.5 |
| RQ-006 NewChromaDBStore использует Validate | T1.2 |

---

## Заметки

- Все задачи привязаны к конкретным файлам (`pkg/draftrag/chromadb.go`, `pkg/draftrag/chromadb_test.go`)
- Тесты следуют паттерну из `qdrant_test.go` (httptest.Server, require.NoError, assert)
- T3.5 можно выполнить параллельно с T3.2-T3.4 — независимая проверка
