# Поддержка Milvus как векторного хранилища — План

## Phase Contract

Inputs: spec, inspect report, узкий контекст репозитория (qdrant.go, weaviate.go — pattern reference).
Outputs: plan.md, data-model.md.
Stop if: spec слишком расплывчата для безопасного планирования.

## Цель

Добавить два новых файла в `internal/infrastructure/vectorstore/`: `milvus.go` со структурой `MilvusStore` и `milvus_test.go` с unit-тестами на мок-HTTP-сервере. `MilvusStore` взаимодействует с Milvus исключительно через REST API v2 (`/v2/vectordb/entities/`), без SDK, по аналогии с `WeaviateStore` и `QdrantStore`. Никакие существующие файлы не меняются.

## Scope

- Новый файл `internal/infrastructure/vectorstore/milvus.go`
- Новый файл `internal/infrastructure/vectorstore/milvus_test.go`
- Граница `internal/domain/` остаётся нетронутой — интерфейсы уже содержат все нужные методы
- `go.mod` / `go.sum` не меняются (только stdlib)

## Implementation Surfaces

- `internal/infrastructure/vectorstore/milvus.go` — **новая** surface; нет существующей реализации Milvus, создаётся с нуля по паттерну `weaviate.go` / `qdrant.go`
- `internal/infrastructure/vectorstore/milvus_test.go` — **новая** surface; unit-тесты с `httptest.NewServer` по паттерну `weaviate_test.go`

## Влияние на архитектуру

- Локальное: добавляется новый файл в существующий пакет `vectorstore`; никаких изменений в публичном API пакета или domain-слое
- Границы: только `net/http` ↔ Milvus REST API v2; никаких новых inter-package зависимостей
- Migration / compatibility: не требуются — фича только добавляет новый провайдер

## Acceptance Approach

- **AC-001** (Upsert) → `Upsert` сериализует `domain.Chunk` в тело `{"collectionName": ..., "data": [{...}]}` и отправляет POST `/v2/vectordb/entities/upsert`; тест проверяет тело запроса через мок-сервер
- **AC-002** (Delete) → `Delete` отправляет POST `/v2/vectordb/entities/delete` с `{"filter": "id == \"<id>\"", ...}`; тест проверяет фильтр-выражение
- **AC-003** (Search) → `Search` отправляет POST `/v2/vectordb/entities/search` с вектором и `limit`; ответ `{"data": [...]}` десериализуется в `domain.RetrievalResult`; тест проверяет количество чанков
- **AC-004** (SearchWithFilter) → `SearchWithFilter` добавляет `"filter": "parent_id in [\"a\",\"b\"]"` к телу поиска; при пустом `ParentIDFilter.ParentIDs` — поле `filter` опускается; тест проверяет наличие/отсутствие фильтра
- **AC-005** (SearchWithMetadataFilter) → `SearchWithMetadataFilter` строит AND-выражение вида `metadata["k"] == "v" && ...`; при пустом `Fields` — поле `filter` опускается; тест проверяет точное выражение
- **AC-006** (DeleteByParentID) → `DeleteByParentID` отправляет POST `/v2/vectordb/entities/delete` с `{"filter": "parent_id == \"<id>\"", ...}`; тест проверяет фильтр
- **AC-007** (Compile-time) → три строки `var _ domain.X = (*MilvusStore)(nil)` в начале файла; проверяется `go build ./...`
- **AC-008** (Ошибки) → внутренний `doRequest` хелпер разбирает HTTP-статус и поле `code` из ответа; ненулевой `code` или 4xx/5xx → возвращается `fmt.Errorf("milvus: code=%d msg=%s", ...)`; тест покрывает error-path

## Данные и контракты

- Новых persisted сущностей нет — MilvusStore не управляет состоянием внутри Go-процесса. Схема коллекции Milvus (поля `id`, `text`, `parent_id`, `metadata`, `vector`) определяется пользователем до использования библиотеки.
- Data model: см. `data-model.md` — зафиксирован mapping `domain.Chunk` → Milvus REST payload и формат ответа.
- Contracts: API boundary — Milvus REST API v2 (external); Go-интерфейс domain.VectorStore/VectorStoreWithFilters/DocumentStore (internal) — не меняется. Файл `contracts/api.md` не создаётся, т.к. фича не вводит новых публичных Go-API; Milvus REST boundary полностью описана в data-model.md.
- Эта фича не публикует и не потребляет события — `contracts/events.md` не создаётся.

## Стратегия реализации

- **DEC-001** Raw HTTP без SDK
  Why: конституция требует минимума зависимостей; `milvus-sdk-go` тянет gRPC и добавляет ~10 косвенных зависимостей; Milvus REST API v2 стабилен начиная с 2.3 и покрывает весь нужный набор операций
  Tradeoff: JSON payload строится вручную; нет типизированных DTO из SDK — компенсируется локальными struct-типами внутри `milvus.go`
  Affects: `milvus.go` (только)
  Validation: `go.mod` не содержит новых зависимостей после реализации; SC-003

- **DEC-002** Bearer-токен аутентификация
  Why: Milvus REST API v2 принимает `Authorization: Bearer <token>`; пустая строка означает отсутствие аутентификации — это покрывает оба use-case (с токеном и без)
  Tradeoff: нет поддержки username/password basic-auth — остаётся вне scope
  Affects: `NewMilvusStore`, внутренний `doRequest` хелпер
  Validation: unit-тест AC-008 проверяет наличие заголовка при непустом токене и его отсутствие при пустом

- **DEC-003** Metadata как JSON-поле Milvus
  Why: `domain.Chunk.Metadata` — это `map[string]string`; Milvus JSON-type field поддерживает key-access в фильтрах вида `metadata["key"] == "value"`; это стандартный механизм Milvus 2.3+
  Tradeoff: при Upsert `metadata` сериализуется в JSON-строку (`json.Marshal`); при поиске десериализуется обратно через `json.Unmarshal`
  Affects: `Upsert` (serialize), `Search`/`SearchWithFilter`/`SearchWithMetadataFilter` (deserialize output fields), `SearchWithMetadataFilter` (build filter expr)
  Validation: AC-005 unit-тест проверяет, что тело запроса содержит `metadata["source"] == "wiki"` при `Fields: {"source": "wiki"}`

- **DEC-004** Внутренний `doRequest` хелпер
  Why: все методы MilvusStore делают POST-запросы с одинаковым JSON-телом и одинаковой обработкой ответа; хелпер устраняет дублирование и централизует error-wrapping
  Tradeoff: один хелпер для upsert/delete/search — при будущем расширении (создание коллекций) может потребоваться перегрузка
  Affects: только `milvus.go` (package-private)
  Validation: AC-008 unit-тест покрывает HTTP 4xx/5xx и `code != 0`

## Incremental Delivery

### MVP (Первая ценность)

- `MilvusStore` struct + `NewMilvusStore` + `doRequest` + compile-time assertions
- `Upsert`, `Delete`, `Search`
- Error-path handling
- Покрывает: AC-001, AC-002, AC-003, AC-007, AC-008
- Критерий готовности: `go build ./...` проходит; базовые тесты зелёные

### Итеративное расширение

- `SearchWithFilter`, `SearchWithMetadataFilter`, `DeleteByParentID`
- Покрывает: AC-004, AC-005, AC-006
- Можно реализовывать после MVP без риска сломать уже написанное

## Порядок реализации

1. **MilvusStore struct + NewMilvusStore + doRequest + assertions** — основа для всех остальных методов; без этого ничего не компилируется
2. **Upsert + Delete + Search** — базовый `domain.VectorStore`; независимы между собой, можно писать параллельно
3. **SearchWithFilter + SearchWithMetadataFilter + DeleteByParentID** — расширения; зависят от Search и Delete как шаблонов
4. **Unit-тесты** — можно писать параллельно с реализацией каждого метода; мок-сервер создаётся один раз в тест-файле

Никаких feature flags или migration steps нет.

## Риски

- **Milvus REST API response schema** — структура `{"code": 0, "data": [...]}` может незначительно отличаться между патч-версиями
  Mitigation: тесты используют только мок-сервер, задавая нужный response самостоятельно; production-совместимость обеспечивается документированным требованием Milvus ≥ 2.3 (spec Допущения)

- **MetadataFilter expression syntax** — неправильный синтаксис фильтра приведёт к ошибке на стороне Milvus
  Mitigation: DEC-003 фиксирует точный формат; AC-005 unit-тест валидирует строку фильтра до отправки; пользователь увидит Milvus error message через AC-008 error wrapping

## Rollout и compatibility

Специальных rollout-действий не требуется. Фича добавляет новый файл в существующий пакет; никакого изменения публичного Go API или поведения существующих провайдеров нет. Backward compatibility сохранена полностью.

## Проверка

- Unit-тесты с `httptest.NewServer` в `milvus_test.go` (AC-001..AC-008)
- `go build ./...` — compile-time assertions (AC-007)
- `go vet ./internal/infrastructure/vectorstore/` — нет новых предупреждений (SC-001)
- `go test -cover ./internal/infrastructure/vectorstore/ -run TestMilvus` — coverage ≥60% (SC-002)
- `grep -c "require" go.mod` не увеличился (SC-003, DEC-001)

## Соответствие конституции

| Ограничение конституции | Применение | Статус |
|---|---|---|
| Интерфейсная абстракция: внешние зависимости через интерфейсы | `MilvusStore` реализует `domain.VectorStore`, `VectorStoreWithFilters`, `DocumentStore` — compile-time assertions | ✓ выполняется |
| Чистая архитектура: infrastructure-слой | Файл в `internal/infrastructure/vectorstore/` | ✓ выполняется |
| Минимальные зависимости | Только stdlib (`net/http`, `encoding/json`) — DEC-001 | ✓ выполняется |
| Context safety | Все методы принимают `context.Context` первым параметром (наследуется от domain-интерфейса) | ✓ выполняется |
| Тестируемость: ≥60% coverage для infrastructure | Unit-тесты с мок-сервером на все 8 AC; SC-002 | ✓ выполняется |
| Godoc на русском | RQ-007; все публичные типы и функции документируются на русском | ✓ выполняется |
| Go 1.21+ (фактически 1.23.5) | Только stdlib, совместима с любой версией Go 1.21+ | ✓ выполняется |
