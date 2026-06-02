# Харденинг библиотеки: рефакторинг, покрытие, публичный API, ошибки

## Scope Snapshot

- In scope: четыре ортогональных улучшения качества кода — рефакторинг god-object `internal/application/pipeline.go`, экспорт Redis cache в публичный API, повышение покрытия тестами `pkg/draftrag` до ≥65%, унификация sentinel-ошибок между domain и публичным API.
- Out of scope: новая функциональность (новые провайдеры, фичи RAG), health checks, Prometheus-экспорт, переработка публичного API (breaking changes), рефакторинг других файлов.

## Цель

Разработчики библиотеки и downstream-потребители получают поддерживаемую, измеряемую и консистентную кодовую базу: pipeline-слой перестаёт быть god-object (1915 → разбит на модули по доменам), Redis cache доступен через публичный API (сейчас только internal), покрытие `pkg/draftrag` поднимается до ≥65% (сейчас 48.8%), ошибки перестают дублироваться между `domain` и `pkg/draftrag`. Успех измеряется: прохождение `go vet`/`golangci-lint`/`go test ./...`, покрытие `pkg/draftrag` ≥65%, ни один тест не изменён (refactor-safe), Redis cache wrapper работает в `examples/`.

## Основной сценарий

1. Разработчик создаёт `Pipeline` с Redis cache embedder: `NewCachedEmbedder(inner, redis.NewRedisCache(...))` — достигается без импорта `internal/`.
2. Разработчик вносит правку в pipeline: открывает `internal/application/answer.go`, а не скроллит 1915 строк одного файла.
3. Разработчик вызывает `ErrEmptyDocumentID` или `ErrStreamingNotSupported` — ошибка проходит `errors.Is`-цепочку от `domain` до пользователя без `mapValidationErr`.
4. CI/CD прогоняет тесты `pkg/draftrag`: все методы `SearchBuilder.Stream*`, `Cite`, `Hybrid` покрыты, покрытие не ниже 65%.
5. Регрессии нет: все существующие тесты (`go test ./...`) проходят без изменений.

## User Stories

- **P1 Story**: Разработчик, поддерживающий pipeline, может найти нужный use-case за 3 секунды вместо скролла 1915 строк.
- **P2 Story**: Разработчик использует `NewRedisCache` из `pkg/draftrag`, не залезая в `internal/`.
- **P3 Story**: Разработчик полагается на `errors.Is(err, ErrStreamingNotSupported)` в своём коде — ошибка не теряется в `mapValidationErr`.
- **P4 Story**: CI/CD гарантирует, что каждый новый метод SearchBuilder покрыт тестом (покрытие ≥65%).

## MVP Slice

Наименьший независимый срез: **рефакторинг pipeline.go** (AC-A1–AC-A4) без изменения поведения и тестов. После него `internal/application/` разделён на модули, `go test ./...` зелёный, покрытие не упало. Возможность положить в master без остальных направлений.

Остальные AC имеют слабую связность с рефакторингом и могут добавляться итеративно.

## First Deployable Outcome

После первого implementation pass можно показать:
- `internal/application/` — 5+ файлов вместо одного (pipeline.go, query.go, answer.go, stream.go, batch.go, prompt.go, hooks.go)
- `pkg/draftrag/cached_embedder_redis.go` — публичный конструктор Redis cache
- `go test -coverprofile ./pkg/draftrag/` показывает ≥65%
- `go vet ./... && golangci-lint run` — чисто

## Scope

1. **Рефакторинг `internal/application/pipeline.go`**
   — Разбиение на модули по зонам ответственности (pipeline, query, answer, stream, batch, prompt, hooks, retrieval, rrf).
   — Сохранение обратной совместимости: ни один метод не переименован, структура `Pipeline` публично не меняется, сигнатуры не меняются.
   — Весь существующий тестовый код (22 файла) не редактируется, только перекомпилируется.
2. **Redis cache в публичном API**
   — Экспорт `RedisCache` из `internal/infrastructure/embedder/cache/redis.go` в `pkg/draftrag/`.
   — Follow существующему шаблону `cached_embedder.go` (type-aliases, конструкторы).
   — `RedisClient` — публичный интерфейс, либо пользователь может использовать `internal/infrastructure/embedder/cache.RedisClient` через type-aliases.
3. **Повышение покрытия `pkg/draftrag`**
   — Цель: ≥65% statements (сейчас 48.8%) без удаления существующих тестов.
   — Затрагиваемые файлы: `pkg/draftrag/search.go` (Stream\*, Cite\*, Hybrid, MultiQuery), `pkg/draftrag/resilience.go` (NewRetryEmbedder, toInternal), `pkg/draftrag/pgvector_migrate.go` (execTemplate, ensureIndex).
   — Новые тесты добавляются в `pkg/draftrag/*_test.go`.
4. **Унификация ошибок**
   — `pkg/draftrag/errors.go` — переэкспорт всех ошибок из `internal/domain/` вместо дублирования.
   — Упрощение/удаление `mapValidationErr` в `pkg/draftrag/draftrag.go`.
   — Сохранение обратной совместимости: старые sentinel-значения (`ErrEmptyDocument`, `ErrEmptyQuery`) продолжают работать.

## Контекст

- Репозиторий уже имеет полный workflow: constitution → spec → plan → tasks → implement → verify → archive. Данная спека следует ему.
- `internal/infrastructure/embedder/cache/redis.go` уже реализован и покрыт тестами (92.9%), но не экспортирован в `pkg/draftrag/`.
- `pkg/draftrag/search.go` уже содержит streaming-методы с низким покрытием (Stream 35.7%, StreamSources 28.6%, StreamCite 33.3%, Cite 40.7%) — это скрытые дефекты.
- Разбиение pipeline.go **не должно** затрагивать файлы вне `internal/application/` и `pkg/draftrag/` (последний — только через тесты).
- Конституция: Clean Architecture, интерфейсная абстракция, покрытие ≥60% для infrastructure, ≥80% для domain/application — не нарушаются.
- `authz на shield` не относится к этой спеке и остаётся вне scope.

## Требования

- RQ-001 Проект проходит `go test ./... -count=1` без ошибок после рефакторинга.
- RQ-002 `go vet ./...` и `golangci-lint run` без ошибок.
- RQ-003 Ни один существующий тест не изменён (только добавлены новые).
- RQ-004 Покрытие `pkg/draftrag` ≥65% statements.
- RQ-005 Redis cache доступен через `pkg/draftrag.NewRedisCache(ctx, client, ttl)` или аналогичный публичный конструктор.
- RQ-006 Ошибки `ErrEmptyDocument`, `ErrEmptyQuery`, `ErrInvalidTopK`, `ErrFiltersNotSupported`, `ErrStreamingNotSupported`, `ErrDeleteNotSupported` проходят `errors.Is` в цепочке от domain до пользователя.
- RQ-007 SystemPrompt v1 (defaultSystemPromptV1) вынесен из `internal/application/pipeline.go` в отдельный файл `internal/application/prompts.go` или `//go:embed`.
- RQ-008 `internal/application/pipeline.go` не превышает 400 строк после разбиения.

## Вне scope

- Новые RAG-фичи (query rewriting, cross-encoder reranker, hierarchical indices, sub-query decomposition) — согласно ROADMAP Priority 2/3, не являются частью харденинга.
- Health checks, Prometheus-экспорт, metrics endpoint — Priority 3 по ROADMAP, не затрагиваются.
- Breaking changes в публичном API — любой перенос/переименование метода Pipeline требует major-версии.
- Рефакторинг других файлов, кроме `internal/application/pipeline.go` и `pkg/draftrag/`. (Исключения: добавление `pkg/draftrag/cached_embedder_redis.go`, тестовые файлы `*_test.go`.)
- Изменение модели domain (`internal/domain/`) — только переэкспорт ошибок.
- `authz на shield` — не относится.

## Критерии приемки

### AC-001 pipeline.go разделён на модули по доменам

- Почему это важно: god-object 1915 строк затрудняет навигацию и поддержку.
- **Given** файл `internal/application/pipeline.go` существует с 1915 строками
- **When** рефакторинг завершён
- **Then** `internal/application/` содержит 5+ файлов: `pipeline.go` (≤400 строк), `query.go`, `answer.go`, `stream.go`, `batch.go`, `prompt.go`, `hooks.go` (или аналогичные)
- Evidence: `wc -l internal/application/pipeline.go` ≤ 400; `ls internal/application/*.go` показывает новые файлы; `go build ./...` проходит.

### AC-002 Все существующие тесты проходят без изменений

- Почему это важно: рефакторинг не должен менять поведение.
- **Given** 22 тестовых файла в `internal/application/`
- **When** `go test ./internal/application/... -count=1` выполнен
- **Then** exit code 0, ни один тест не изменён (проверка: `git diff --stat -- internal/application/*_test.go` — пусто)
- Evidence: `go test ./internal/application/ -count=1` успешен.

### AC-003 defaultSystemPromptV1 вынесен из pipeline.go

- Почему это важно: контент не должен быть в коде, мешает i18n и тестированию.
- **Given** `defaultSystemPromptV1` определён в `internal/application/pipeline.go`
- **When** рефакторинг завершён
- **Then** `defaultSystemPromptV1` находится в отдельном файле `internal/application/prompts.go`
- Evidence: `grep 'defaultSystemPromptV1' internal/application/prompts.go` — найдено; `grep 'defaultSystemPromptV1' internal/application/pipeline.go` — пусто.

### AC-004 golangci-lint и go vet проходят на изменённых файлах

- Почему это важно: качество кода не должно ухудшиться.
- **Given** рефакторинг pipeline.go завершён
- **When** `go vet ./internal/application/... && golangci-lint run ./internal/application/...`
- **Then** exit code 0
- Evidence: CI-анализ.

### AC-005 Redis cache доступен через pkg/draftrag

- Почему это важно: пользователь не должен импортировать `internal/`.
- **Given** `internal/infrastructure/embedder/cache/redis.go` реализует Redis cache с интерфейсом `RedisClient`
- **When** в `pkg/draftrag/` добавлен файл `cached_embedder_redis.go` (или аналогичный)
- **Then** пользователь может вызвать `draftrag.NewRedisCache(ctx, client, ttl)` без импорта `internal/`
- Evidence: пример кода `import "github.com/bzdvdn/draftrag"; cache := draftrag.NewRedisCache(ctx, client, 5*time.Minute)` компилируется.

### AC-006 Redis cache wrapper имеет тесты в pkg/draftrag

- Почему это важно: публичный API должен быть протестирован.
- **Given** файл `pkg/draftrag/cached_embedder_redis.go` добавлен
- **When** `go test ./pkg/draftrag/... -count=1` выполнен
- **Then** покрытие нового кода ≥70%
- Evidence: `go test -coverprofile ./pkg/draftrag/...` показывает покрытие новой функции.

### AC-007 Покрытие pkg/draftrag ≥65% statements

- Почему это важно: конституция требует ≥60% для infrastructure, текущее 48.8%.
- **Given** покрытие `pkg/draftrag` = 48.8%
- **When** добавлены тесты на SearchBuilder.Stream\*, Cite, Hybrid, NewRetryEmbedder, execPGVectorMigrationTemplate
- **Then** `go test -covermode=atomic ./pkg/draftrag/...` показывает ≥65%
- Evidence: `go tool cover -func=coverage.out | grep 'pkg/draftrag'` — итоговое покрытие ≥65%.

### AC-008 Методы Stream\*, Cite, Hybrid в search.go покрыты тестами

- Почему это важно: streaming и citations — ключевые пользовательские сценарии, на которые приходят баги.
- **Given** `pkg/draftrag/search.go` содержит методы `Stream`, `StreamSources`, `StreamCite`, `Cite`, `InlineCite`, `SearchBuilder.Hybrid`
- **When** тесты запущены
- **Then** каждый метод имеет хотя бы один тест (positive + error path), coverage каждой функции > 0%
- Evidence: `go tool cover -func=coverage.out | grep 'pkg/draftrag/search.go'` — ни одна строка не `0.0%`.

### AC-009 Sentinel-ошибки public API проходят errors.Is до domain

- Почему это важно: пользователь пишет `errors.Is(err, draftrag.ErrEmptyDocument)` и получает true.
- **Given** `pkg/draftrag/errors.go` определяет sentinel-ошибки
- **When** пользователь вызывает `errors.Is(err, draftrag.ErrEmptyDocument)` где err — ошибка из `internal/domain.ErrEmptyDocumentContent`
- **Then** результат true
- Evidence: тест в `pkg/draftrag/errors_test.go` (новый) проверяет errors.Is-цепочку для каждой публичной ошибки.

### AC-010 mapValidationErr удалён или упрощён

- Почему это важно: ad-hoc маппинг скрывает цепочку ошибок и усложняет поддержку.
- **Given** `mapValidationErr` в `pkg/draftrag/draftrag.go` содержит ручной маппинг domain-ошибок на API-ошибки
- **When** errors унифицированы
- **Then** либо `mapValidationErr` удалён, либо его код сокращён до 1-2 строк (только переэкспорт)
- Evidence: `grep -c 'errors.Is(err, domain' pkg/draftrag/draftrag.go` — счетчик уменьшился или 0.

## Допущения

- `internal/infrastructure/embedder/cache/redis.go` остаётся стабильным (его сигнатуры не меняются) — экспорт будет через type-aliases и тонкие конструкторы-обёртки.
- Разбиение pipeline.go сохраняет package name = `application` — сменить его нельзя, т.к. завязаны тесты.
- Ни один тест в `internal/application/` не использует внутренние неэкспортируемые функции (кроме `dedupRetrievedChunksByParentID`, `rrfMergeMultiple`, `buildUserMessageV1*`) — их можно перемещать между файлами пакета.
- Пользовательский код, использующий `errors.Is(err, draftrag.ErrXXX)`, ожидает, что sentinel-значения остаются теми же объектами.
- CI-окружение использует `GOPROXY=https://proxy.golang.org,direct` для тестов (учитывая текущие проблемы с пустым GOPROXY на машине разработчика, тесты должны работать на любом стандартном Go-окружении).

## Критерии успеха

- SC-001 Покрытие `pkg/draftrag` ≥65% (было 48.8%) — проверяется после каждого PR.
- SC-002 `internal/application/pipeline.go` ≤400 строк (было 1915).
- SC-003 Redis cache доступен как `draftrag.NewRedisCache` — документация на русском в godoc.
- SC-004 Все 6 публичных sentinel-ошибок проходят errors.Is-тест: `ErrEmptyDocument`, `ErrEmptyQuery`, `ErrInvalidTopK`, `ErrFiltersNotSupported`, `ErrStreamingNotSupported`, `ErrDeleteNotSupported`.

## Краевые случаи

- **Рефакторинг:** случайный перенос символа меняет observable behavior. Гарантия: `go test ./...` + `git diff --stat *_test.go` == nil.
- **Redis cache:** `RedisClient` может быть nil в runtime. Конструктор валидирует клиент (panic или error? — выбор за plan).
- **Покрытие:** добавление теста, который случайно вызывает настоящие API (HTTP). Тесты должны использовать mock-реализации `LLMProvider`, `Embedder`, `VectorStore`.
- **Ошибки:** `ErrEmbeddingDimensionMismatch` и `ErrInvalidVectorStoreConfig` уже экспортированы как alias (см. `errors.go:29-37`). Унификация не должна их сломать — regression test.
- **Streaming:** тесты на `Stream*` должны использовать мок `StreamingLLMProvider` с controllable-каналом.

## Открытые вопросы

1. `defaultSystemPromptV1` — отдельный .go-файл с константой (const), без `//go:embed`. Решение: AC-003 фиксирует отдельный файл `prompts.go`, embed избыточен для одной константы.
2. `mapValidationErr` — урезать, а не удалять. Оставить для ошибок без sentinel-значения (reranker-wrap, streaming-wrap). Решение зафиксировано для plan.
3. `RedisClient` интерфейс в `internal/infrastructure/embedder/cache/` — уже определён. Какой подход: type-alias (как VectorStore) или копия интерфейса? Желательно type-alias для консистентности с rest API библиотеки. Детали за plan.
4. Зависит ли рефакторинг pipeline.go от унификации ошибок? Нет, они ортогональны.
5. `none` — дополнительных уточнений от пользователя не требуется, кроме указанных выше двух NEEDS CLARIFICATION.
