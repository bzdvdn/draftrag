# Unified Config Management Задачи

## Phase Contract

Inputs: plan (config-management), data-model stub, spec.
Outputs: упорядоченные исполнимые задачи с покрытием всех 7 AC.
Stop if: задачи получаются расплывчатыми или AC без покрытия — нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/config.go` | T1.1, T2.1, T2.2, T2.3, T3.1, T3.2, T3.3 |
| `pkg/draftrag/errors.go` | T1.2 |
| `pkg/draftrag/config_test.go` | T2.4, T3.4, T4.1 |
| `go.mod` | T1.2 |

## Implementation Context

- **Цель MVP:** единый Config struct + LoadConfig (YAML + env) + NewPipelineFromConfig для memory-store + ollama-embedder + ollama-llm.
- **Инварианты:**
  - Config в `pkg/draftrag/` (один import), не sub-package (DEC-001).
  - `*sql.DB` / `*http.Client` передаются через опциональный `ExternalDeps` (DEC-002).
  - Env-ключи: `DRAFTRAG_` + UPPER_SNAKE_CASE иерархического yaml-пути (DEC-003).
- **Ошибки:** `ErrUnknownConfigKey` (yaml.DisallowUnknownFields), `ErrMissingRequiredField` (валидация в NewPipelineFromConfig).
- **Proof signals:** `LoadConfig(path)` → Config заполнен; `NewPipelineFromConfig(ctx, cfg)` → рабочий Pipeline; env-переменные переопределяют YAML.
- **Вне scope:** hot-reload, remote config, secrets management, CLI-флаги, sub-package.

## Фаза 1: Основа (структуры и зависимости)

Цель: создать data-типы Config и настроить зависимости, чтобы последующие фазы работали с готовым каркасом.

- [x] T1.1 Создать `Config` struct с sub-configs (Pipeline, Store, Embedder, LLM) + YAML-теги + тип `ExternalDeps` для runtime-зависимостей (`*sql.DB`, `*http.Client`). Touches: `pkg/draftrag/config.go`

- [x] T1.2 Добавить sentinel-ошибки `ErrUnknownConfigKey` и `ErrMissingRequiredField` в `pkg/draftrag/errors.go` + продвинуть `gopkg.in/yaml.v3` из indirect в direct в `go.mod`. Touches: `pkg/draftrag/errors.go`, `go.mod`

## Фаза 2: MVP Slice

Цель: минимальная самостоятельно демонстрируемая ценность — загрузить Config из YAML/env и создать Pipeline для memory-store + ollama.

- [x] T2.1 Реализовать `LoadConfig(path string) (Config, error)` — чтение YAML-файла с `yaml.Unmarshal` + `DisallowUnknownFields`. AC-001, AC-003. Touches: `pkg/draftrag/config.go`

- [x] T2.2 Реализовать env-binding: функцию `applyEnvOverrides` (рефлексивный обход Config, построение env-ключа из yaml-тегов, префикс `DRAFTRAG_`) + `LoadConfigFromEnv()`. AC-002, AC-007. Touches: `pkg/draftrag/config.go`

- [x] T2.3 Реализовать `NewPipelineFromConfig(ctx, cfg Config, deps ...ExternalDeps) (*Pipeline, error)` с type dispatch для memory-store, ollama-embedder, ollama-llm. AC-005. Touches: `pkg/draftrag/config.go`

- [x] T2.4 Написать тесты MVP-пути: YAML→Config (AC-001), YAML+env override (AC-002), unknown YAML key (AC-003), NewPipelineFromConfig не паникует (AC-005), env-only (AC-007). Touches: `pkg/draftrag/config_test.go`

## Фаза 3: Основная реализация

Цель: расширить dispatch на все store/embedder/llm провайдеры и добавить валидацию required fields.

- [x] T3.1 Добавить store dispatch (pgvector, qdrant, chromadb, weaviate, milvus) + интеграция с `ExternalDeps` (`*sql.DB` для pgvector). AC-006, RQ-007. Touches: `pkg/draftrag/config.go`

- [x] T3.2 Добавить embedder dispatch (openai-compatible, mistral) и LLM dispatch (openai-compatible, anthropic, deepseek, mistral). AC-004 (required field validation для model/api_key и т.д.). Touches: `pkg/draftrag/config.go`

- [x] T3.3 Добавить валидацию required fields для всех dispatch-типов: Model для embedder/LLM, EmbeddingDimension для pgvector, URL для qdrant, CollectionName для chromadb. AC-004, RQ-005. Touches: `pkg/draftrag/config.go`

- [x] T3.4 Написать тесты dispatch-путей: store dispatch (AC-006), missing required fields (AC-004), pgvector без `*sql.DB`, все embedder/llm variants. Touches: `pkg/draftrag/config_test.go`

## Фаза 4: Проверка

Цель: доказать, что фича работает, и оставить пакет в reviewable состоянии.

- [x] T4.1 Добавить edge-тесты: пустой YAML → ErrMissingRequiredField, пустая env-переменная не переопределяет, некорректный YAML (синтаксис) → ошибка парсинга, Config с resilience-секцией без стратегии → no-op. Touches: `pkg/draftrag/config_test.go`

- [x] T4.2 Выполнить `go vet ./pkg/draftrag/` + `go build ./...` + финальная проверка, что нет regressions в существующих тестах (`go test ./pkg/draftrag/`). Touches: `pkg/draftrag/config.go`, `pkg/draftrag/errors.go`, `go.mod`

## Покрытие критериев приемки

- AC-001 (YAML → Config) -> T2.1, T2.4
- AC-002 (env override) -> T2.2, T2.4
- AC-003 (unknown key) -> T2.1, T2.4
- AC-004 (missing required) -> T3.3, T3.4
- AC-005 (NewPipelineFromConfig) -> T2.3, T2.4
- AC-006 (store dispatch) -> T3.1, T3.4
- AC-007 (env-only) -> T2.2, T2.4

## Заметки

- Все фазы в одном PR — никаких ветвлений между фазами.
- T1.1 и T1.2 можно параллелить.
- T2.1 → T2.2 → T2.3 обязательный порядок (зависимость по данным).
- T3.1 и T3.2 независимы, можно параллелить после T2.3.
- T4.1 и T4.2 — после завершения всех предшествующих фаз.
