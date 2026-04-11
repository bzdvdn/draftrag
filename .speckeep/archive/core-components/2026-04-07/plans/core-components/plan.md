# Core компоненты пакета draftRAG — План

## Phase Contract

Inputs: `.draftspec/specs/core-components/spec.md`, `.draftspec/specs/core-components/inspect.md`, конституция проекта
Outputs: `plan.md`, `data-model.md`
Stop if: spec слишком расплывчата для безопасного планирования

## Цель

Реализовать ядро пакета draftRAG через Clean Architecture слои: domain (интерфейсы и модели), application (use-cases), infrastructure (in-memory реализации), и публичный API в `pkg/draftrag/`. Изменения затрагивают только новые файлы — существующий код не модифицируется.

## Scope

- Domain-слой: интерфейсы `VectorStore`, `LLMProvider`, `Embedder`, `Chunker` и модели `Document`, `Chunk`, `Query`, `RetrievalResult`, `Embedding`
- Application-слой: use-case `Pipeline` с методами `Index(ctx, docs)` и `Query(ctx, question)`
- Infrastructure-слой: in-memory реализация `VectorStore` для тестирования
- Публичный API: `pkg/draftrag/` с фабриками и конструкторами
- Явно вне scope: внешние провайдеры (pgvector, Qdrant, OpenAI и др.), HTTP/CLI интерфейсы

## Implementation Surfaces

- `internal/domain/interfaces.go` — новый файл, определяет core-интерфейсы пакета (VectorStore, LLMProvider, Embedder, Chunker)
- `internal/domain/models.go` — новый файл, определяет domain-модели (Document, Chunk, Query, RetrievalResult, Embedding)
- `internal/application/pipeline.go` — новый файл, use-case Pipeline с композицией интерфейсов
- `internal/infrastructure/vectorstore/memory.go` — новый файл, in-memory реализация VectorStore для тестирования
- `pkg/draftrag/draftrag.go` — новый файл, публичный API с фабриками и экспортом интерфейсов
- `pkg/draftrag/errors.go` — новый файл, пакетные ошибки и типы валидации

## Влияние на архитектуру

- Локальное влияние: создаётся новая структура каталогов `internal/domain/`, `internal/application/`, `internal/infrastructure/`, `pkg/draftrag/`
- Нет влияния на существующие интеграции — пакет создаётся с нуля
- Нет migration или compatibility последствий — первый релиз библиотеки

## Acceptance Approach

- AC-001 -> создание `internal/domain/interfaces.go` с godoc-комментариями на русском; проверка через `go doc pkg/draftrag.VectorStore`
- AC-002 -> создание `internal/domain/models.go` с моделями Document/Chunk; unit-тест демонстрирует Index + Search цикл
- AC-003 -> все методы в `internal/application/pipeline.go` принимают context.Context; тест с отменённым контекстом возвращает context.Canceled
- AC-004 -> создание `internal/infrastructure/vectorstore/memory.go`; тест TestInMemoryStore_BasicSearch проходит
- AC-005 -> создание `pkg/draftrag/draftrag.go` с NewPipeline(); integration-тест показывает полный цикл

## Данные и контракты

- Сошлитесь на `AC-002`: сущности Document, Chunk, Query, RetrievalResult, Embedding определяются в `internal/domain/models.go`
- Изменения data model: создаются 5 новых структур с полями согласно spec (RQ-004, RQ-005)
- API contracts: нет внешних API boundaries — пакет используется как Go library
- Event contracts: нет событийной модели в core-компонентах

## Стратегия реализации

- DEC-001 Clean Architecture слоистая структура
  Why: конституция требует Clean Architecture с направлением зависимостей внутрь; domain не должен импортировать внешние пакеты
  Tradeoff: больше файлов и каталогов, но чёткое разделение ответственности и тестируемость
  Affects: `internal/domain/`, `internal/application/`, `internal/infrastructure/`
  Validation: `go build ./...` без ошибок; domain-пакет не импортирует infrastructure

- DEC-002 Интерфейсы в domain-слое, реализации в infrastructure
  Why: конституция требует интерфейсную абстракцию для внешних зависимостей; позволяет заменять провайдеров без изменения client code
  Tradeoff: дополнительные interface-типы, но моки для тестирования создаются естественно
  Affects: `internal/domain/interfaces.go`, `internal/infrastructure/vectorstore/memory.go`
  Validation: in-memory реализация удовлетворяет интерфейсу VectorStore; unit-тесты проходят

- DEC-003 In-memory store использует cosine similarity по умолчанию
  Why: cosine similarity — стандартная метрика для RAG; простая реализация без внешних зависимостей
  Tradeoff: нет настраиваемой метрики в v1, но интерфейс позволяет добавить параметр позже
  Affects: `internal/infrastructure/vectorstore/memory.go`
  Validation: тест BasicSearch возвращает результаты с корректным score (cosine similarity в диапазоне [-1, 1])

- DEC-004 Ошибки валидации через отдельный пакет `pkg/draftrag/errors`
  Why: стандарт Go — ошибки определяются там, где используются; пакетные ошибки удобнее для client-side checks
  Tradeoff: отдельный файл для ошибок, но чёткая граница API contract
  Affects: `pkg/draftrag/errors.go`
  Validation: пустой document/content возвращает ErrEmptyDocument; nil context вызывает panic

## Порядок реализации

1. Domain-слой (`internal/domain/interfaces.go`, `internal/domain/models.go`) — определяет core-абстракции, от которых зависят все остальные слои
2. Infrastructure-слой (`internal/infrastructure/vectorstore/memory.go`) — реализует in-memory store для тестирования domain-логики
3. Application-слой (`internal/application/pipeline.go`) — композиция интерфейсов в use-case Pipeline
4. Публичный API (`pkg/draftrag/draftrag.go`, `pkg/draftrag/errors.go`) — экспорт функциональности для клиентов
5. Unit-тесты для каждого слоя параллельно с реализацией

## Риски

- Риск 1: Недостаточная гибкость интерфейсов для будущих провайдеров
  Mitigation: интерфейсы проектируются минимальными (только необходимые методы); расширение через новые методы с breaking change в major-версии
- Риск 2: Cosine similarity в in-memory store может быть медленным для больших коллекций
  Mitigation: in-memory store предназначен только для тестов; production-реализации (pgvector, Qdrant) будут использовать индексированный поиск
- Риск 3:godoc на русском может быть непривычен для англоязычных разработчиков
  Mitigation: конституция фиксирует русский язык комментариев; имена типов и функций на английском сохраняют понятность

## Rollout и compatibility

- Нет migration, feature flags или operational follow-up — первый релиз библиотеки
- Совместимость: семантическое версионирование с v0.x для начальной разработки; breaking changes возможны до v1.0
- Monitoring: не применимо для библиотеки; клиенты сами обрабатывают ошибки

## Проверка

- AC-001: `go doc pkg/draftrag.VectorStore` выводит godoc на русском с описанием методов Upsert, Delete, Search
- AC-002: unit-тест `TestPipeline_IndexAndQuery` создаёт Document, вызывает Index, затем Query и проверяет результат
- AC-003: unit-тест `TestPipeline_ContextCancellation` с `context.WithCancel()` и немедленным `cancel()` возвращает context.Canceled
- AC-004: unit-тест `TestInMemoryStore_BasicSearch` Upsert документ, Search по похожему тексту, проверка score > 0
- AC-005: integration-тест `TestPipeline_FullCycle` демонстрирует NewPipeline + Index + Query
- DEC-001: `go build ./...` без ошибок; `go list ./internal/domain/...` показывает отсутствие внешних импортов
- DEC-002: `go test ./internal/infrastructure/vectorstore/...` проходит; memory.go реализует все методы VectorStore
- DEC-003: тест проверяет, что score в диапазоне [-1, 1] для cosine similarity
- DEC-004: тест `TestValidation_EmptyDocument` возвращает ErrEmptyDocument; тест с nil context вызывает panic

## Соответствие конституции

- нет конфликтов
- Clean Architecture: domain → application → infrastructure, зависимости направлены внутрь
- Интерфейсная абстракция: все внешние зависимости (VectorStore, LLMProvider, Embedder) через интерфейсы
- Контекстная безопасность: все операции принимают context.Context
- Тестируемость: in-memory реализации и моки для всех интерфейсов
- Языковая политика: godoc-комментарии на русском, имена на английском
