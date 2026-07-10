# Unified Config Management План

## Phase Contract

Inputs: spec (config-management), inspect report (pass), repo map.
Outputs: plan, data model stub.
Stop if: spec содержит неразрешимые неоднозначности — нет.

## Цель

Добавить единый `Config` struct с загрузкой из YAML + env и фабрику `NewPipelineFromConfig` в публичный API. Вся работа — в `pkg/draftrag/`; ни один существующий компонент не меняет свой публичный интерфейс.

## MVP Slice

- Core `Config` struct, покрывающий: PipelineOptions + один store (memory) + один embedder (ollama) + один LLM (ollama).
- `LoadConfig` / `LoadConfigFromEnv` + env-оверрайды.
- `NewPipelineFromConfig` с type dispatch для store/embedder/llm.
- AC, закрываемые MVP: AC-001, AC-005, AC-007.

## First Validation Path

1. Создать YAML: `store: {type: memory}, embedder: {type: ollama, model: nomic-embed-text}, llm: {type: ollama, model: llama3}`.
2. `cfg, err := draftrag.LoadConfig("test.yaml")` → cfg заполнен.
3. `p, err := draftrag.NewPipelineFromConfig(ctx, cfg)` → p != nil, err == nil.
4. `_, err = p.Query(ctx, "test")` → транспортная ошибка (ollama недоступен), не паника.

## Scope

- `pkg/draftrag/config.go` — Config struct, LoadConfig, LoadConfigFromEnv, env-binding, NewPipelineFromConfig.
- `pkg/draftrag/config_test.go` — юнит-тесты.
- `pkg/draftrag/errors.go` — добавление `ErrUnknownConfigKey`, `ErrMissingRequiredField`.
- `go.mod` — продвижение `gopkg.in/yaml.v3` из indirect в direct.
- Существующие Options struct, constructors, domain, application — без изменений.

## Performance Budget

- `none` — конфигурация загружается однократно при старте, не на горячем пути.

## Implementation Surfaces

| Surface | Роль | Тип |
|---|---|---|
| `pkg/draftrag/config.go` | Config struct, sub-configs, LoadConfig, LoadConfigFromEnv, NewPipelineFromConfig | NEW |
| `pkg/draftrag/errors.go` | Добавить `ErrUnknownConfigKey`, `ErrMissingRequiredField` | CHANGED |
| `pkg/draftrag/config_test.go` | Unit-тесты: YAML→struct, env override, missing field, unknown key, store dispatch | NEW |
| `go.mod` | yaml.v3: indirect → direct | CHANGED |

## Bootstrapping Surfaces

- `none` — все файлы создаются как часть реализации, внешних bootstrap-шагов нет.

## Влияние на архитектуру

- Локальное: новый файл в `pkg/draftrag/`, без изменений в `internal/`.
- Ни одна существующая публичная функция/тип не меняет сигнатуру.
- `gopkg.in/yaml.v3` становится прямой зависимостью.

## Acceptance Approach

- AC-001: `LoadConfig` с YAML-файлом → сравнение полей Config. Surface: `config.go`.
- AC-002: `LoadConfig` с YAML + env override → `cfg.LLM.APIKey == "real-key"`. Surface: `config.go` (env binding).
- AC-003: YAML с неизвестным ключом → `errors.Is(err, ErrUnknownConfigKey)`. Surface: `config.go` (DisallowUnknownFields).
- AC-004: Config с pgvector без `embedding_dimension` → `ErrMissingRequiredField`. Surface: `config.go` (валидация в NewPipelineFromConfig).
- AC-005: `NewPipelineFromConfig` с memory+ollama → `pipeline.Query` не паникует. Surface: `config.go`.
- AC-006: memory vs pgvector dispatch → memory success, pgvector без `*sql.DB` → `ErrMissingRequiredField`. Surface: `config.go`.
- AC-007: `LoadConfig("")` с установленными env → поля из env. Surface: `config.go` (LoadConfigFromEnv logic).

## Данные и контракты

- `data-model.md`: status = no-change (Config struct — новая модель, не меняющая существующие).

## Стратегия реализации

### DEC-001: Config в main-пакете, не sub-package

- **Why**: spec упоминает `draftrag.LoadConfig(path)` и `draftrag.NewPipelineFromConfig(ctx, cfg)`. Вынос в `pkg/draftrag/config/` заставит пользователя импортировать два пакета. Размещение в `pkg/draftrag/` даёт единый импорт при той же читаемости. Один файл `config.go` (~250 строк) — допустимый размер.
- **Tradeoff**: небольшое увеличение `pkg/draftrag/`; при росте можно вынести sub-config-типы в `config_types.go`.
- **Affects**: `pkg/draftrag/config.go`, `pkg/draftrag/errors.go`.
- **Validation**: `import "github.com/bzdvdn/draftrag"; draftrag.LoadConfig("…")` компилируется.

### DEC-002: Внешние зависимости через отдельный аргумент `NewPipelineFromConfig`

- **Why**: `*sql.DB`, `*http.Client` не сериализуемы в YAML. Спека оставляет два варианта: поля в Config (yaml:"-") vs отдельный аргумент. Второй чище: Config остаётся чистой data-моделью. Добавляем опциональный аргумент `deps ...ExternalDeps` (variadic для backward-compat).
- **Tradeoff**: дополнительный параметр у конструктора. `ExternalDeps` — struct с опциональными полями.
- **Affects**: `pkg/draftrag/config.go` — тип `ExternalDeps`, `NewPipelineFromConfig(ctx, cfg, deps ...ExternalDeps)`.
- **Validation**: `NewPipelineFromConfig(ctx, cfg)` без deps работает (memory store не требует *sql.DB).

### DEC-003: Flat env naming с hierarchical mapping

- **Why**: spec выбирает `DRAFTRAG_LLM_API_KEY` (flat, single underscore). Mapping: YAML-путь `llm.api_key` → upper + replace `.` → `LLM_API_KEY` → с префиксом `DRAFTRAG_LLM_API_KEY`. Алгоритм: рефлексивно обойти Config, для каждого поля построить путь из yaml-тегов, upper-case, join через `_`.
- **Tradeoff**: коллизия если два поля на разных уровнях дают одинаковый env-key (например, `llm.api_key` и `llm_api_key` на одном уровне). В текущей структуре Config таких коллизий нет.
- **Affects**: `pkg/draftrag/config.go` — функция `applyEnvOverrides`.
- **Validation**: env `DRAFTRAG_LLM_API_KEY=secret` переопределяет `cfg.LLM.APIKey`.

## Incremental Delivery

### MVP (Первая ценность)

- Config struct (Pipeline + MemoryStore + OllamaEmbedder + OllamaLLM).
- LoadConfig с yaml.Unmarshal + DisallowUnknownFields.
- Env-overrides через рефлексивный обход.
- NewPipelineFromConfig для memory-store + ollama-embedder + ollama-llm.
- Sentinel errors: ErrUnknownConfigKey, ErrMissingRequiredField.
- Covered AC: AC-001, AC-002, AC-003, AC-005, AC-007.

### Итеративное расширение

- Добавить store dispatch (pgvector, qdrant, chromadb, weaviate, milvus) + ExternalDeps с `*sql.DB`, `*http.Client` — AC-004, AC-006.
- Добавить embedder dispatch (openai-compatible, mistral).
- Добавить llm dispatch (openai-compatible, anthropic, deepseek, mistral).
- Добавить Chunker, Reranker, Resilience, CostTracking sub-configs.

## Порядок реализации

1. `config.go`: Config struct + sub-configs (Pipeline, Store, Embedder, LLM) + yaml tags.
2. `errors.go`: добавить ErrUnknownConfigKey, ErrMissingRequiredField.
3. `config.go`: LoadConfig (yaml.Unmarshal + DisallowUnknownFields).
4. `config.go`: applyEnvOverrides (рефлексивный обход).
5. `config.go`: LoadConfigFromEnv + LoadConfig объединение.
6. `config.go`: NewPipelineFromConfig (type dispatch для memory-store, ollama-embedder, ollama-llm).
7. `config_test.go`: тесты на все AC.
8. `go.mod`: promote yaml.v3.

Шаги 1–6 в одном PR, шаги 7–8 в том же PR.

## Риски

- **Риск: рефлексивный env-binding может быть хрупким**  
  Mitigation: покрыть unit-тестами все типы полей (string, int, bool, struct, pointer). Использовать `reflect` только для flatten-пути, assign через `os.LookupEnv`.
- **Риск: yaml.v3 — indirect deps, promotion может конфликтовать**  
  Mitigation: yaml.v3 v3.0.1 уже в go.sum; `go mod tidy` решит продвижение. Никакой другой пакет в проекте не требует yaml.v3 напрямую.

## Rollout and compatibility

- Полностью аддитивно: Config, LoadConfig, NewPipelineFromConfig — новые символы. Старые конструкторы (NewPipeline, NewPipelineWithOptions) не меняются.
- Специальных rollout-действий не требуется.

## Проверка

- `go test ./pkg/draftrag/ -run TestConfig` — unit-тесты config.go.
- `go vet ./pkg/draftrag/config.go` — статический анализ.
- AC-001–007: каждый покрыт отдельным тестом.
- DEC-001–003: каждый подтверждён через `go build ./...` + тесты.

## Соответствие конституции

- нет конфликтов
