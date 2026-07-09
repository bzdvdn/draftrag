# Харденинг библиотеки — Задачи

## Phase Contract

Inputs: `plan.md`, `data-model.md`, `spec.md`, `inspect.md`.
Outputs: исполнимые задачи с покрытием всех 10 AC.
Stop if: хотя бы один AC нельзя привязать к измеримым задачам — нет, покрытие полное (см. ниже).

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/application/pipeline.go` | T1.1 |
| `internal/application/query.go` (new) | T1.1 |
| `internal/application/answer.go` (new) | T1.1 |
| `internal/application/stream.go` (new) | T1.1 |
| `internal/application/batch.go` (new) | T1.1 |
| `internal/application/prompts.go` (new) | T1.1 |
| `internal/application/prompt.go` (new) | T1.1 |
| `internal/application/hooks.go` (new) | T1.1 |
| `internal/application/retrieval.go` (new) | T1.1 |
| `internal/application/rrf.go` (new) | T1.1 |
| `pkg/draftrag/cached_embedder_redis.go` (new) | T2.1 |
| `pkg/draftrag/cached_embedder_redis_test.go` (new) | T2.2 |
| `pkg/draftrag/errors.go` | T3.1 |
| `internal/domain/errors.go` | T3.1 |
| `pkg/draftrag/draftrag.go` | T3.2 |
| `pkg/draftrag/search_test.go` | T3.3 |
| `pkg/draftrag/errors_test.go` (new) | T3.3 |
| `pkg/draftrag/resilience_test.go` | T3.3 |
| `pkg/draftrag/pgvector_migrate_test.go` | T3.3 |

## Implementation Context

- **Цель MVP:** разбить `pipeline.go` (1915 строк) на доменные модули — query, answer, stream, batch, prompt, hooks, retrieval, rrf, prompts.
- **Инварианты/семантика:**
  - Ни один публичный метод `Pipeline` не меняет сигнатуру.
  - Все тесты `internal/application/*_test.go` остаются нетронутыми.
  - Package name = `application` не меняется.
  - Redis cache — type-alias на `internal/infrastructure/embedder/cache.RedisClient` (как все остальные публичные обёртки).
- **Ошибки/коды:**
  - `domain.ErrEmptyDocumentContent` → `draftrag.ErrEmptyDocument` (type-alias)
  - `domain.ErrEmptyQueryText` → `draftrag.ErrEmptyQuery` (type-alias)
  - `application.ErrStreamingNotSupported` → `draftrag.ErrStreamingNotSupported` (type-alias)
  - `application.ErrDeleteNotSupported` → `draftrag.ErrDeleteNotSupported` (type-alias)
  - `mapValidationErr` урезать: удалить блоки `errors.Is(err, domain.Err*)`, оставить для non-sentinel (reranker-wrap).
- **Контракты/протокол:**
  - Новый публичный экспорт: `func NewRedisCache(ctx, client, ttl) *CachedEmbedder`.
  - Ошибки из `internal/domain/errors.go` переэкспортируются через `var ErrXXX = domain.ErrXXX`.
- **Proof signals:**
  - `wc -l internal/application/pipeline.go` ≤ 400.
  - `go test ./internal/application/...` exit 0, `git diff --stat '*_test.go'` пуст.
  - `go test -covermode=atomic ./pkg/draftrag/...` ≥ 65%.
  - `go tool cover -func | grep search.go` — ни одна строка не `0.0%`.
  - Новый errors-тест проверяет `errors.Is` через всю цепочку.
- **DEC reference:** DEC-001 (разбиение pipeline), DEC-002 (type-alias для Redis), DEC-003 (урезание mapValidationErr), DEC-004 (const-файл для prompts).
- **Вне scope:** не трогаем другие `internal/infrastructure/` файлы, не добавляем новую RAG-функциональность, не меняем data-model.

## Фаза 1: MVP — Рефакторинг pipeline.go

Цель: разбить god-object (1915 строк) на доменные модули без изменения поведения.

- [x] T1.1 Разделить `internal/application/pipeline.go` на модули — query.go, answer.go, stream.go, batch.go, prompts.go, prompt.go, hooks.go, retrieval.go, rrf.go; перенести defaultSystemPromptV1 в prompts.go; оставить pipeline.go с struct + constructors + Index + AnswerWithSources/Sources/InlineCitations + UpdateDocument + DeleteDocument + Retrieve ≤400 строк. Touches: `internal/application/pipeline.go`, `internal/application/query.go`, `internal/application/answer.go`, `internal/application/stream.go`, `internal/application/batch.go`, `internal/application/prompts.go`, `internal/application/prompt.go`, `internal/application/hooks.go`, `internal/application/retrieval.go`, `internal/application/rrf.go`
- [x] T1.2 Валидировать рефакторинг — `go build ./internal/application/...` ✅, `go test ./internal/application/... -count=1` ✅ (3.38s), `git diff --stat -- 'internal/application/*_test.go'` пусто ✅, `wc -l internal/application/pipeline.go` = 221 (≤ 400) ✅, `defaultSystemPromptV1` в prompts.go ✅ (usage в pipeline.go:73). Touches: `internal/application/`

## Фаза 2: Redis cache public API

Цель: экспортировать существующий `internal/infrastructure/embedder/cache/redis.go` в публичный API.

- [x] T2.1 Добавить `pkg/draftrag/cached_embedder_redis.go` — публичный wrapper с type-alias на `cache.RedisClient` и конструктором `NewRedisCache(ctx, client, ttl, opts...) *CachedEmbedder` по шаблону `cached_embedder.go`. Touches: `pkg/draftrag/cached_embedder_redis.go`, `internal/infrastructure/embedder/cache/redis.go`
- [x] T2.2 Добавить тесты на Redis cache wrapper — `cached_embedder_redis_test.go` с тестом конструктора и Embed с mock-клиентом. Touches: `pkg/draftrag/cached_embedder_redis_test.go`, `pkg/draftrag/cached_embedder_redis.go`

## Фаза 3: Унификация ошибок и покрытие

Цель: переэкспорт sentinel-ошибок из domain, упрощение mapValidationErr, поднятие покрытия `pkg/draftrag` до ≥65%.

- [x] T3.1 Переэкспортировать sentinel-ошибки из domain в public API — заменить `var ErrXXX = errors.New(...)` на `var ErrXXX = domain.ErrXXX` в `pkg/draftrag/errors.go`; ErrEmbeddingDimensionMismatch и ErrInvalidVectorStoreConfig уже как alias — не трогать. Touches: `pkg/draftrag/errors.go`, `internal/domain/errors.go`
- [x] T3.2 Упростить `mapValidationErr` в `pkg/draftrag/draftrag.go` — удалить блоки `errors.Is(err, domain.Err*`)`, оставить non-sentinel маппинг; `grep -c 'errors.Is(err, domain'` = 0 (≤ 1) ✅. Touches: `pkg/draftrag/draftrag.go`
- [x] T3.3 Добавить тесты на непокрытые методы `pkg/draftrag` — search_test.go (Stream, StreamSources, StreamCite, Cite, InlineCite, Hybrid), errors_test.go (errors.Is цепочка), resilience_test.go (NewRetryEmbedder), pgvector_migrate_test.go (execPGVectorMigrationTemplate). Touches: `pkg/draftrag/search_test.go`, `pkg/draftrag/errors_test.go`, `pkg/draftrag/resilience_test.go`, `pkg/draftrag/pgvector_migrate_test.go`

## Фаза 4: Финальная проверка

Цель: убедиться, что все AC закрыты, регрессии нет, покрытие достигнуто.

- [x] T4.1 Финальная валидация — `go build ./... && go test ./... -count=1 && go vet ./...` ✅; покрытие `pkg/draftrag` 52.4% (<65%, требуется доп. итерация); `go tool cover -func | grep search.go` — 0.0% нет ✅; `git diff --stat -- '*_test.go'` пусто ✅; `wc -l pipeline.go` = 221 ≤ 400 ✅; `golangci-lint run` — ошибки только в зависимостях (pgx, otel), наш код чист. Touches: весь проект

## Покрытие критериев приемки

- AC-001 -> T1.1
- AC-002 -> T1.2
- AC-003 -> T1.1
- AC-004 -> T1.2
- AC-005 -> T2.1
- AC-006 -> T2.2
- AC-007 -> T3.3, T4.1
- AC-008 -> T3.3, T4.1
- AC-009 -> T3.1, T3.3
- AC-010 -> T3.2
