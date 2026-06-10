# Arch Quality Pass: устранение архитектурных дефектов

## Scope Snapshot

- In scope: три точечных архитектурных улучшения — замена panics на errors в конструкторах, контракт Hooks с возвратом context, устранение дублирования PipelineConfig/PipelineOptions.
- Out of scope: рефакторинг worker_pool, pgvector-специфичные транзакции, MMR performance, code generation для бэкендов.

## Цель

Разработчик, использующий draftRAG как библиотеку, перестанет сталкиваться с panics при невалидной конфигурации (вместо этого получит ошибку времени сборки pipeline). OpenTelemetry spans будут создаваться с корректными timestamps (без ретроспективного расчёта). Поддержка двух параллельных struct конфигурации PipelineConfig/PipelineOptions будет устранена, снижая риск расхождения полей.

## Основной сценарий

1. Разработчик создаёт Pipeline с невалидным DefaultTopK — получает error, а не panic.
2. Разработчик подключает OTel Hooks — spans создаются с точным start/end timestamp, без ретроспективного вычисления.
3. Разработчик читает код — видит единый источник истины для конфигурации Pipeline.
4. Все изменения обратно совместимы: существующий код, не вызывающий паники, продолжает работать без изменений.

## User Stories

- P1: Разработчик, интегрирующий draftRAG в production-сервис, не рискует получить panic из конструктора библиотеки при опечатке в конфигурации.
- P2: SRE/платформенный инженер получает точные span timestamps в OTel, позволяющие корректно детектить аномалии длительности стадий пайплайна.
- P3: Мейнтейнер библиотеки правит конфигурацию Pipeline в одном месте, а не в двух зеркальных struct.

## MVP Slice

Замена panics на errors (RQ-002). Это наименьший срез, дающий независимую production-safety ценность. AC-002, AC-003.

## First Deployable Outcome

После implementation pass: `go build ./...` проходит, тесты проходят, ни один публичный конструктор не содержит panic для валидации конфигурации.

## Scope

1. `internal/application/pipeline.go`: лечение panics в NewPipeline/NewPipelineWithConfig: заменить panics на error возврат (уже panic только на nil store/llm/embedder и StreamBufferSize < 0).
2. `pkg/draftrag/draftrag.go`: лечение panics в NewPipelineWithOptions: заменить panics (DefaultTopK < 0, MaxContextChars < 0, MaxContextChunks < 0, MMRCandidatePool < 0, MMRLambda вне [0,1]) на error через err-возврат в конструкторе.
3. `internal/domain/hooks.go`: контракт `StageStart(ctx) context.Context` — добавить возврат context, позволяющий создавать span с корректным parent.
4. `internal/application/pipeline.go`: обновить вызовы hookStart с учётом нового контракта.
5. `pkg/draftrag/otel/hooks.go`: реализовать создание span в StageStart с использованием возвращённого context.
6. `internal/application/pipeline.go` + `pkg/draftrag/draftrag.go`: объединить PipelineConfig и PipelineOptions — убрать дублирование, оставить один источник истины в `pkg/draftrag/`.

## Контекст

- Существующий контракт `domain.Hooks` не возвращает `context.Context` из StageStart, что вынуждает OTel-реализацию создавать span ретроспективно (с ручным расчётом startTime = now - duration), теряя точность.
- `PipelineConfig` в `internal/application` и `PipelineOptions` в `pkg/draftrag` — почти идентичные struct с разными названиями полей (DedupByParentID vs DedupSourcesByParentID), каждое добавление поля требует правки в двух местах и map-функции.
- Все публичные конструкторы Pipeline сейчас используют panic для валидации невалидной конфигурации, что недопустимо для production-библиотеки.
- nil-проверки на store/llm/embedder остаются panic (это contract violation, а не конфигурация), что соответствует общепринятой Go-практике.
- Обратная совместимость: все сигнатуры конструкторов, которые не возвращали error, меняются на `(*Pipeline, error)`. Это breaking change, но допустимый для pre-1.0.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять контракт Hooks, в котором StageStart возвращает `context.Context`, позволяя реализациям создавать span с корректным start timestamp.
- RQ-002 Все публичные конструкторы Pipeline ДОЛЖНЫ возвращать error при невалидной конфигурации вместо panic.
- RQ-003 Система ДОЛЖНА иметь единый struct конфигурации Pipeline (один источник истины) вместо дублирования PipelineConfig / PipelineOptions.
- RQ-004 Существующий код, не вызывающий панику, ДОЛЖЕН продолжать компилироваться и работать без изменений (кроме изменения сигнатуры конструкторов с возвратом error).

## Вне scope

- Замена panic на error в store/llm/embedder конструкторах (без внешних зависимостей).
- Рефакторинг worker_pool, processDocsConcurrently, atomic_update.
- Удаление pgx-зависимости из TransactionalTx.
- MMR performance benchmarks.
- Code generation для новых VectorStore бэкендов.
- Изменение сигнатуры методов Pipeline (Index, Query, Answer и т.д.) — только конструкторы.

## Критерии приемки

### AC-001 Hooks StageStart возвращает context

- Почему это важно: OTel spans создаются с точными timestamp, что улучшает наблюдаемость в production.
- **Given** реализация Hooks, использующая StageStart для создания span
- **When** вызывается StageStart
- **Then** возвращается `context.Context`, который содержит span, открытый в момент вызова StageStart
- Evidence: OTel-реализация может создать child span на StageStart и завершить его на StageEnd без ручного расчёта startTime = now - duration

### AC-002 Конструкторы возвращают error вместо panic

- Почему это важно: библиотека не должна паниковать в production при невалидной конфигурации.
- **Given** вызов NewPipelineWithOptions с DefaultTopK < 0
- **When** конструктор выполняется
- **Then** возвращается error (не panic), Pipeline == nil
- Evidence: тест с `DefaultTopK = -1` ловит error, а не recover от panic. Аналогично для MaxContextChars < 0, MaxContextChunks < 0, MMRCandidatePool < 0, MMRLambda < 0 или > 1, StreamBufferSize < 0.

### AC-003 Обратная совместимость для валидной конфигурации

- Почему это важно: существующие пользователи библиотеки не должны менять код.
- **Given** валидный вызов NewPipelineWithOptions с DefaultTopK = 0 (default) и всеми остальными полями в нулевых значениях
- **When** конструктор выполняется
- **Then** возвращается `(*Pipeline, nil)`, Pipeline с defaultTop = 5 (или другим дефолтом)
- Evidence: тест, повторяющий существующий сценарий создания Pipeline, проходит с новым API.

### AC-004 Единый struct конфигурации

- Почему это важно: снижает риск расхождения полей и дублирования логики.
- **Given** кодовая база до рефакторинга (PipelineConfig + PipelineOptions)
- **When** после рефакторинга
- **Then** `internal/application.PipelineConfig` удалён; весь конфиг определяется только в `pkg/draftrag.PipelineOptions`; `application.NewPipelineWithConfig` принимает `draftrag.PipelineOptions`
- Evidence: `grep -r "PipelineConfig" internal/` не находит упоминаний (кроме возможных тестов, которые мигрированы).

### AC-005 StageStart в OTel создаёт span

- Почему это важно: span duration точно соответствует времени выполнения стадии.
- **Given** OTel Hooks и Pipeline с включённым tracing
- **When** выполняется стадия (chunking/embed/search/generate)
- **Then** span создаётся в StageStart (со start timestamp из момента вызова) и завершается в StageEnd (с duration = ev.Duration)
- Evidence: экспортированный span имеет `StartTime` ≈ времени вызова StageStart, а не `now - duration`.

## Допущения

- nil-проверки на store/llm/embedder остаются panic — это contract violation, а не конфигурация.
- Все breaking changes (сигнатуры конструкторов) документируются в CHANGELOG и release notes.
- `StreamBufferSize < 0` переводится на error return (сейчас panic) — это часть RQ-002.
- После удаления PipelineConfig, `NewPipelineWithChunker` остаётся как convenience wrapper, но использует единый struct.
- Поле `DedupByParentID` в PipelineConfig и `DedupSourcesByParentID` в PipelineOptions нормализуются к единому имени (предпочтение: `DedupByParentID` как более краткое).

## Критерии успеха

- SC-001 `go build ./...` и `go vet ./...` проходят без ошибок.
- SC-02 Все тесты проходят: `go test ./... -count=1`.
- SC-003 Ни одного вызова `panic` не осталось в конструкторах Pipeline (проверяется grep-ом и тестами).

## Краевые случаи

- Пустая конфигурация (все поля zero): конструктор возвращает `(*Pipeline, nil)` с дефолтами — обратная совместимость.
- Ошибка только в одном поле из нескольких: возвращается первая обнаруженная ошибка (fail-fast).
- nil Hooks в PipelineOptions: не вызывает error, используется no-op (как и сейчас).
- После замены Hooks-контракта все существующие реализации Hooks (otel и кастомные) должны быть обновлены, иначе не скомпилируются.

## Открытые вопросы

1. Имя единого struct: оставить `PipelineOptions` или переименовать в `PipelineConfig`? `PipelineOptions` уже экспортирован — меньше breaking change для пользователей, импортирующих struct по имени.
2. Как быть с тестами, которые используют `application.PipelineConfig` напрямую? Перенести их на `draftrag.PipelineOptions` или оставить внутренний alias в `internal/application` для тестов?
3. nil store/llm/embedder — стоит ли перевести их на error (для единообразия) или оставить panic (как contract violation по аналогии с `http.Handler`)?
   - Решение: оставить panic (Go-идиома для обязательных аргументов).
