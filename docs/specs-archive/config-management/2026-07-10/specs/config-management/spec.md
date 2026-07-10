# Unified Config Management

## Scope Snapshot

- In scope: единый `Config` struct, который можно заполнить из YAML и/или env-переменных, и фабричный метод `NewPipelineFromConfig`, создающий полностью сконфигурированный Pipeline.
- Out of scope: hot-reload, remote config backends (etcd/consul), CLI-флаги, secrets management, встроенный HTTP-сервер.

## Цель

Пользователь библиотеки (Go-разработчик RAG-приложения) получает единственную точку входа для конфигурации всех компонентов Pipeline — VectorStore, Embedder, LLMProvider, Chunker, Reranker, resilience — через YAML-файл и/или переменные окружения. Успех фичи измеряется тем, что типовой сценарий «сконфигурировать и запустить Pipeline» требует не более одного YAML-файла + одной строки Go-кода (`draftrag.NewPipelineFromConfig(ctx, cfg)`), а типы ошибок конфигурации (неизвестный ключ, отсутствие обязательного поля) возвращаются на этапе создания, не в рантайме.

## Основной сценарий

1. Пользователь создаёт YAML-файл с настройками store, embedder, llm и общих опций Pipeline.
2. Пользователь вызывает `draftrag.LoadConfig(path)` — загрузка из файла с оверрайдами из env (префикс `DRAFTRAG_`).
3. Пользователь вызывает `draftrag.NewPipelineFromConfig(ctx, cfg)` — валидация + конструирование.
4. Результат: готовый `*draftrag.Pipeline`.
5. Ошибка: если YAML содержит неизвестный ключ — `ErrUnknownConfigKey`. Если не хватает обязательного поля — `ErrMissingRequiredField`.

## User Stories

- P1 Story: разработчик может описать всю конфигурацию RAG-пайплайна в одном YAML-файле и получить рабочий Pipeline одной функцией.
- P2 Story: разработчик может переопределить отдельные поля через переменные окружения (например, `DRAFTRAG_LLM_API_KEY`), не меняя YAML.

## MVP Slice

- Один top-level Config struct, охватывающий: PipelineOptions + ровно один store + ровно один embedder + ровно один LLM. YAML-загрузка c `yaml.Unmarshal` + env-оверрайды с `os.LookupEnv`. Фабрика `NewPipelineFromConfig`. Обработка неизвестных ключей через `yaml.DisallowUnknownFields`.

## First Deployable Outcome

- Загруженный из YAML файла конфиг, распечатанный `fmt.Printf("%+v", cfg)`, показывает корректные значения. `NewPipelineFromConfig` создаёт Pipeline без ошибок для memory-store + ollama-embedder + ollama-llm. Unit-тесты покрывают: YAML → struct, YAML + env override, missing required field, unknown key.

## Scope

- `Config` struct с sub-configs для: `Pipeline`, `VectorStore` (один), `Embedder`, `LLMProvider`, `Chunker`, `Reranker`, `Resilience`, `CostTracking`.
- Функция `LoadConfig(path string) (Config, error)` — YAML + env-оверрайды.
- Конструктор `NewPipelineFromConfig(ctx context.Context, cfg Config) (*Pipeline, error)`.
- Пакет `pkg/draftrag/config/` (или размещение рядом с `PipelineOptions` для минимального surface).
- Env-префикс `DRAFTRAG_` с hierarchical key mapping (например, `DRAFTRAG_PGVECTOR_TABLE_NAME`).
- Валидация обязательных полей + запрет неизвестных YAML-ключей.
- Go-документация для всех публичных полей и функций.

## Контекст

- Текущая кодовая база использует отдельные Options struct на каждый компонент (`PipelineOptions`, `PGVectorOptions`, `OllamaEmbedderOptions` и т.д.) — без единого config root.
- Ни YAML, ни env binding не реализованы — новое поведение, а не изменение существующего.
- Конституция: «Нет встроенного HTTP-сервера или CLI — только библиотека» → Config и LoadConfig — чисто библиотечная функция.
- Конституция: «Простота > расширяемость» → один Config struct без deep inheritance.
- Интерфейсы для внешних зависимостей (http.Client, sql.DB) не могут быть сериализованы → передаются отдельно или через функциональные опции.

## Зависимости

- Зависит от внутренних интерфейсов `domain.VectorStore`, `domain.Embedder`, `domain.LLMProvider`, `domain.Chunker`, `domain.Reranker` — уже существуют.
- Внешняя библиотека: `gopkg.in/yaml.v3` для YAML-разбора (уже есть в экосистеме; вендоринг через `go mod`).
- `none` меж-спековых зависимостей.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять единый `Config` struct, агрегирующий конфигурацию всех компонентов Pipeline.
- RQ-002 Система ДОЛЖНА загружать `Config` из YAML-файла через функцию `LoadConfig(path string) (Config, error)`. При пустом path файл не читается — Config заполняется только из env.
- RQ-003 Система ДОЛЖНА применять env-оверрайды с префиксом `DRAFTRAG_` для любых полей Config после YAML-загрузки.
- RQ-004 Система ДОЛЖНА возвращать `ErrUnknownConfigKey` при наличии в YAML ключа, не мапящегося ни на одно поле `Config`.
- RQ-005 Система ДОЛЖНА возвращать `ErrMissingRequiredField` с именем поля при отсутствии обязательного поля (Model для embedder/LLM, EmbeddingDimension для pgvector).
- RQ-006 Фабрика `NewPipelineFromConfig(ctx, Config)` ДОЛЖНА создать `*Pipeline` с корректной инициализацией всех компонентов.
- RQ-007 Фабрика ДОЛЖНА поддерживать store type dispatch по строковому полю (type: "pgvector" / "memory" / "qdrant" / "chromadb" / "weaviate" / "milvus").
- RQ-008 Фабрика ДОЛЖНА валидировать Config на этапе конструирования, не откладывая ошибки на runtime.

## Out of Scope

- Hot-reload / watch изменений YAML-файла.
- Remote config (etcd, consul, k8s ConfigMap watch).
- Secrets management (Vault, AWS Secrets Manager) — API-ключи читаются из env или YAML как plain text.
- CLI/флаги — библиотека не имеет CLI.
- Автоматическая генерация YAML-схемы / JSON Schema.
- Поддержка нескольких store/embedder/llm в одном Config (только один компонент каждого типа).
- Managed identity / IAM-роли для облачных провайдеров.

## Критерии приемки

### AC-001 YAML → Config struct

- Почему это важно: пользователь должен описать конфигурацию в читаемом формате без написания Go-кода инициализации.
- **Given** валидный YAML-файл с полной конфигурацией (store: memory, embedder: ollama, llm: ollama)
- **When** вызвана `LoadConfig(path)`
- **Then** возвращён `Config` с корректно заполненными полями
- Evidence: поля Config соответствуют значениям из YAML; `Config.Store.Type == "memory"`

### AC-002 Env override поверх YAML

- Почему это важно: разработчик должен переопределять чувствительные/окруженческие параметры без правки YAML.
- **Given** YAML-файл с `llm.api_key: "placeholder"` и установленная env `DRAFTRAG_LLM_API_KEY=real-key`
- **When** вызвана `LoadConfig(path)`
- **Then** в Config поле `LLM.APIKey` равно `"real-key"`
- Evidence: `cfg.LLM.APIKey == "real-key"`

### AC-003 Unknown YAML key → ошибка

- Почему это важно: опечатки в YAML не должны молча игнорироваться.
- **Given** YAML-файл с несуществующим ключом `store.ttl: 3600`
- **When** вызвана `LoadConfig(path)`
- **Then** возвращена ошибка `ErrUnknownConfigKey`, содержащая имя ключа
- Evidence: `errors.Is(err, draftrag.ErrUnknownConfigKey)` и `err.Error()` содержит `"ttl"`

### AC-004 Missing required field → ошибка

- Почему это важно: невалидная конфигурация должна обнаруживаться до вызова конструктора.
- **Given** YAML-файл с pgvector store без поля `embedding_dimension`
- **When** вызвана `NewPipelineFromConfig(ctx, cfg)`
- **Then** возвращена ошибка `ErrMissingRequiredField`, содержащая `"embedding_dimension"`
- Evidence: `errors.Is(err, draftrag.ErrMissingRequiredField)`

### AC-005 NewPipelineFromConfig создаёт рабочий Pipeline

- Почему это важно: фабрика — главная точка входа; она должна возвращать готовый к использованию Pipeline.
- **Given** `Config` с корректными настройками memory-store + ollama-embedder + ollama-llm
- **When** вызвана `NewPipelineFromConfig(ctx, cfg)`
- **Then** возвращён `*Pipeline` без ошибки, и `pipeline.Query(ctx, "test")` не паникует (ожидаемая ошибка от недоступного ollama — но не паника)
- Evidence: `pipeline != nil`, ошибка Query — транспортная, не конфигурационная

### AC-006 Store type dispatch

- Почему это важно: пользователь выбирает бэкенд строкой, без ручного вызова конструктора.
- **Given** Config с `store.type: "memory"` и Config с `store.type: "pgvector"` + pgvector-specific поля
- **When** вызвана `NewPipelineFromConfig(ctx, cfg)` для обоих
- **Then** memory-Config успешен, pgvector-Config без `*sql.DB` возвращает ошибку (не панику)
- Evidence: memory — успех; pgvector — `ErrMissingRequiredField` для `db`

### AC-007 Env override без YAML (только env)

- Почему это важно: можно сконфигурировать Pipeline вообще без YAML-файла.
- **Given** только env-переменные с префиксом `DRAFTRAG_` (`DRAFTRAG_STORE_TYPE=memory`, `DRAFTRAG_EMBEDDER_TYPE=ollama`, `DRAFTRAG_EMBEDDER_MODEL=nomic-embed-text`, `DRAFTRAG_LLM_TYPE=ollama`, `DRAFTRAG_LLM_MODEL=llama3`)
- **When** вызвана `LoadConfig("")`
- **Then** Config заполнен из env
- Evidence: `cfg.Store.Type == "memory"`, `cfg.Embedder.Model == "nomic-embed-text"`

## Допущения

- YAML-файл читается однократно при старте, без watch/refresh.
- Тип store/embedder/llm задаётся строковым полем `type` в соответствующей секции.
- Внешние зависимости (*sql.DB, *http.Client) передаются отдельно через функциональные опции или поля `External` в Config.
- Используется `gopkg.in/yaml.v3` (уже стандарт в Go-экосистеме).
- Имена env-переменных образуются: `DRAFTRAG_` + UPPER_SNAKE_CASE иерархического пути (например, `DRAFTRAG_PGVECTOR_TABLE_NAME`).
- Config не содержит секретов в plain-text после загрузки (ожидается, что пользователь сам управляет безопасностью).

## Критерии успеха

- SC-001 Написать unit-тест конфигурации memory-store pipeline (загрузка + создание) можно за 3 строки Go-кода (без подготовки intermediate structs).

## Краевые случаи

- Пустой YAML-файл: все поля принимают zero-значения; `NewPipelineFromConfig` возвращает `ErrMissingRequiredField`.
- Env-переменная с пустым значением: не переопределяет YAML-значение (пустая строка = "не задано").
- Пересечение env и YAML: env имеет приоритет.
- Пустой path в `LoadConfig("")`: файл не читается, Config заполняется только из env (RQ-002, AC-007).
- Nil `*sql.DB` для pgvector: `ErrMissingRequiredField` на этапе `NewPipelineFromConfig`.
- Некорректный YAML (синтаксис): ошибка парсинга от yaml.v3, не маскируется.
- Config с resilience-секцией без единой стратегии: resilience отсутствует (no-op).

## Открытые вопросы

1. Какой env naming convention использовать для вложенных структур? Варианты: `DRAFTRAG_LLM_API_KEY` (flat) vs `DRAFTRAG_LLM__API_KEY` (double underscore как разделитель). Выбран flat: `DRAFTRAG_LLM_API_KEY`. Если возникнет конфликт имён (например, поле `api_key` в двух разных подструктурах с одинаковым путём) — это решится на code review.
2. Передавать ли `*sql.DB` / `*http.Client` через сам Config или отдельным аргументом `NewPipelineFromConfig`? Предварительно: отдельный аргумент `externals` или функциональные опции, так как эти типы не сериализуемы.
