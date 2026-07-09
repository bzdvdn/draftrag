# docs-and-examples: обширная документация и легко запускаемые примеры

## Scope Snapshot

- In scope: запускаемые примеры для всех VectorStore-бэкендов и LLM-провайдеров, серия tutorials, обновлённый README, CI smoke-test matrix, ноль изменений в `pkg/draftrag/`.
- Out of scope: новая функциональность библиотеки, новые бэкенды, перевод документации на английский, hosted/web RAG.

## Цель

Go-разработчик, впервые увидевший draftRAG, должен иметь возможность за <10 минут от клона репозитория дойти до работающего RAG-чата на своём ноутбуке. Это достигается через: (1) по одной самодостаточной директории `examples/<backend>/` для каждого VectorStore с `docker-compose.yml` + `.env.example` + `README.md` с quickstart, (2) серию `docs/tutorials/ru/NN-*.md` от quickstart до production-hardening, (3) матричный CI, который доказывает, что каждый бэкенд поднимается и пример отрабатывает с mock-LLM. Успех измеряется тем, что новый контрибьютор может сменить VectorStore и LLM, поменяв только env-переменные, без правки Go-кода.

## Основной сценарий

1. Стартовая точка: разработчик клонирует репозиторий, читает корневой `README.md`.
2. Основное действие: выбирает VectorStore (например, Qdrant), копирует `examples/qdrant/.env.example` в `.env`, вписывает один API-ключ (Ollama, OpenAI или Anthropic), выполняет `docker compose up -d && go run ./examples/qdrant/`.
3. Результат: 10 демо-документов проиндексированы, пример отвечает на пользовательский вопрос в интерактивном REPL или batch-режиме; exit 0.
4. Ошибка/fallback: если API-ключ не задан, пример автоматически переключается на встроенный mock-LLM (детерминированный echo) и пишет предупреждение в stderr; пользователь всё равно видит работу пайплайна end-to-end.

## User Stories

- P1 Story: новый Go-разработчик запускает RAG-чат с Qdrant + Ollama локально за <10 минут.
- P2 Story: существующий пользователь переключает свой проект с Qdrant на pgvector, проходя tutorial 02 и меняя только env.
- P3 Story: contributor запускает `make examples-smoke` и за 5 минут проверяет, что его изменения не сломали ни один бэкенд.

## MVP Slice

Минимальный срез = 4 примера (memory, pgvector, qdrant, chromadb) + 2 tutorial'а (01-quickstart, 02-basic-rag) + 1 CI job для этих 4 бэкендов. Эти AC обязаны быть зелёными первыми: AC-001, AC-003 (pgvector), AC-006 (compose validate), AC-007, AC-008 (01-quickstart), AC-014 (existing tests pass).

## First Deployable Outcome

После первого implementation pass можно показать: `cd examples/qdrant && docker compose up -d && cp .env.example .env && go run .` — пользователь получает работающий RAG-чат; CI job `examples-smoke` зелёный на main.

## Scope

- Новые директории `examples/{memory,chromadb,weaviate,milvus}/` (pgvector и qdrant уже существуют — рефакторинг под общий шаблон).
- Новые файлы `examples/<backend>/docker-compose.yml`, `.env.example`, `README.md`, `main.go`.
- Новые файлы `examples/shared/` — пакет Go-утилит (`loadConfig`, `mockLLM`, `mockEmbedder`, `printAnswer`), импортируемый всеми примерами.
- 10 файлов `docs/tutorials/ru/NN-*.md` (01..10).
- Обновления `README.md`, `docs/vector-stores.md`, `ROADMAP.md` (индексы, ссылки).
- Новый GitHub Actions workflow `.github/workflows/examples-smoke.yml` — матрица по 6 бэкендам, mock-LLM.
- Осознанно НЕ включаем: правки `pkg/draftrag/`, `internal/`, изменение существующих примеров `chat/`, `index-dir/` (они функционально отличаются — оставляем как есть).

## Контекст

- Ограничение: репозиторий — Go-библиотека без HTTP-сервера и CLI (CONSTITUTION.md). Значит examples = единственный "running" smoke для нового пользователя.
- Существующий поток: уже есть `examples/{chat,index-dir,pgvector,qdrant}`. Они написаны в разном стиле и без `docker-compose.yml` для qdrant. Шаблонизируем под общий вид: `main.go + docker-compose.yml + .env.example + README.md + Makefile` (опционально).
- Конституция требует: "Все новые функции ДОЛЖНЫ иметь unit-тесты" — для examples это означает CI smoke-job, который реально запускает каждый бэкенд.
- Конституция требует: "Язык документации: русский" — все README и tutorials пишутся на русском; код и команды shell — на английском.
- Существующая фича api-consistency-pass добавила: SearchBuilder routing, atomic UpdateDocument, bounded streaming backpressure, per-worker rate-limit, hybrid search, capability table 6×6. Tutorial'ы 03-10 должны покрывать именно эти capability, чтобы документация шла в ногу с реализованной функциональностью.
- Предположение: все 6 бэкендов (memory, pgvector, qdrant, chromadb, weaviate, milvus) доступны как публичные Docker images (true: ankane/pgvector, qdrant/qdrant, chromadb/chroma, semitechnologies/weaviate, milvusdb/milvus). Ollama — `ollama/ollama`.
- Предположение: размерность эмбеддингов 1536 — де-факто стандарт (OpenAI ada-002, многие open-source модели). Делаем настраиваемым через env.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять по одной директории `examples/<backend>/` для каждого из 6 VectorStore-бэкендов: `memory`, `pgvector`, `qdrant`, `chromadb`, `weaviate`, `milvus`. Каждая директория ДОЛЖНА содержать `main.go`, `docker-compose.yml` (кроме `memory`), `.env.example`, `README.md`.
- RQ-002 Каждый пример ДОЛЖЕН поддерживать одни и те же LLM-провайдеры через env: `LLM_PROVIDER=ollama|openai|anthropic|mock`. Смена провайдера НЕ ДОЛЖНА требовать изменения Go-кода.
- RQ-003 Режим `LLM_PROVIDER=mock` ДОЛЖЕН работать без внешних API-ключей и Docker (только для бэкенда, требующего Docker). Mock-эмбеддер возвращает детерминированные 1536-мерные векторы; mock-LLM возвращает echo-ответы с префиксом `[mock]`. Используется для CI smoke и для разработчиков без API-ключей.
- RQ-004 В `docs/tutorials/ru/` ДОЛЖНЫ быть 10 файлов: `01-quickstart.md`, `02-basic-rag.md`, `03-hybrid-search.md`, `04-metadata-filter.md`, `05-streaming.md`, `06-atomic-update.md`, `07-citations.md`, `08-observability.md`, `09-evaluation.md`, `10-production-hardening.md`. Каждый tutorial ДОЛЖЕН иметь frontmatter (title, related_examples, prerequisites) и ссылки на релевантный пример.
- RQ-005 Корневой `README.md` ДОЛЖЕН содержать секции: «Быстрый старт» (≤10 строк), «Примеры» (таблица с бэкендами и ссылками), «Tutorials» (ссылки на 01..10), «Векторные хранилища» (ссылка на capability table), «Провайдеры LLM» (ссылка на tutorials).
- RQ-006 CI workflow `.github/workflows/examples-smoke.yml` ДОЛЖЕН запускать matrix-job по 6 бэкендам + mock-LLM, поднимать соответствующий Docker-сервис, запускать `go run ./examples/<backend>/` с `LLM_PROVIDER=mock` и тестовым запросом, проверять exit 0 и наличие ожидаемого слова в выводе.
- RQ-007 `docs/vector-stores.md` capability-таблица ДОЛЖНА в каждой строке-бэкенде в первой колонке (название бэкенда) содержать markdown-ссылку на `examples/<backend>/`. Ячейки остальных колонок ссылок не содержат.
- RQ-008 Никаких изменений в `pkg/draftrag/` или `internal/`. Все примеры используют только публичный API.
- RQ-009 Все существующие тесты ДОЛЖНЫ продолжать проходить (`go test ./...` exit 0). Покрытие не должно падать.
- RQ-010 Каждый `docker-compose.yml` ДОЛЖЕН использовать healthcheck для своего сервиса и pin-версию image (например, `qdrant/qdrant:v1.12.4`, не `:latest`).

## Вне scope

- Изменение публичного API библиотеки (это потребовало бы отдельной спеки и breaking change в major-версии).
- Добавление новых VectorStore-бэкендов или LLM-провайдеров.
- Перевод документации на английский (конституция требует русский).
- Web UI / SaaS / hosted RAG (это не библиотека).
- Бенчмарки производительности (могут быть отдельной спекой).
- Замена существующих примеров `chat/` и `index-dir/` — они служат другим целям (chat = интерактивный REPL; index-dir = batch ingestion директории); оставляем как есть.
- Mock-эмбеддер для real-mode (только для mock-провайдера).

## Критерии приемки

### AC-001: pgvector example запускается end-to-end

- Почему это важно: pgvector — самый популярный open-source VectorStore; разработчик с PostgreSQL должен иметь zero-friction опыт.
- **Given** PostgreSQL с pgvector поднят через `docker compose up -d` из `examples/pgvector/`
- **When** разработчик выполняет `cp .env.example .env`, устанавливает `LLM_PROVIDER=mock` (или реальный ключ), затем `go run ./examples/pgvector/`
- **Then** программа индексирует 10 демо-документов, принимает запрос пользователя, возвращает ответ и завершается с exit 0
- Evidence: `examples/pgvector/README.md` quickstart; CI job `examples-smoke` для `pgvector` PASS с ожидаемой подстрокой в stdout

### AC-002: qdrant example запускается end-to-end

- Аналогично AC-001, но для Qdrant.
- **Given** Qdrant поднят через `docker compose up -d` из `examples/qdrant/`
- **When** `go run ./examples/qdrant/` с `LLM_PROVIDER=mock`
- **Then** exit 0, ответ содержит маркер mock-провайдера
- Evidence: README + CI

### AC-003: chromadb example запускается end-to-end

- Аналогично AC-001, но для ChromaDB.
- **Given** ChromaDB поднят через `docker compose up -d` из `examples/chromadb/`
- **When** `go run ./examples/chromadb/`
- **Then** exit 0
- Evidence: README + CI

### AC-004: weaviate example запускается end-to-end

- Аналогично AC-001, но для Weaviate.
- **Given** Weaviate поднят через `docker compose up -d` из `examples/weaviate/`
- **When** `go run ./examples/weaviate/`
- **Then** exit 0
- Evidence: README + CI

### AC-005: milvus example запускается end-to-end

- Аналогично AC-001, но для Milvus.
- **Given** Milvus поднят через `docker compose up -d` из `examples/milvus/` (standalone mode)
- **When** `go run ./examples/milvus/`
- **Then** exit 0
- Evidence: README + CI

### AC-006: in-memory example запускается без Docker

- Почему это важно: разработчик без Docker должен иметь возможность попробовать библиотеку.
- **Given** Go установлен, Docker не требуется
- **When** `go run ./examples/memory/`
- **Then** программа работает в in-memory режиме, exit 0
- Evidence: README quickstart без упоминания Docker

### AC-007: выбор LLM-провайдера через env без правок Go-кода

- **Given** `examples/qdrant/.env` с `LLM_PROVIDER=ollama`, `OLLAMA_HOST=http://localhost:11434`
- **When** разработчик меняет только `LLM_PROVIDER=anthropic` + `ANTHROPIC_API_KEY=...`
- **Then** тот же `main.go` использует Anthropic без пересборки
- Evidence: код в `examples/shared/llm.go` (фабрика по `LLM_PROVIDER`); README объясняет env-switching

### AC-008: mock-провайдер работает без API-ключей

- **Given** `LLM_PROVIDER=mock` в `.env`
- **When** пример запускается на любом бэкенде
- **Then** mock-эмбеддер возвращает детерминированные 1536-мерные векторы; mock-LLM возвращает echo с префиксом `[mock]`; программа не обращается к внешним сервисам; exit 0
- Evidence: код `examples/shared/mock.go`; CI использует `LLM_PROVIDER=mock`

### AC-009: все docker-compose.yml синтаксически валидны

- **Given** все 6 `examples/<backend>/docker-compose.yml` (кроме memory)
- **When** `docker compose -f examples/pgvector/docker-compose.yml config` (и т.д.) запускается локально или в CI
- **Then** exit 0; список сервисов корректный
- Evidence: `make compose-validate` или эквивалент в CI

### AC-010: 10 tutorials существуют с правильной структурой

- **Given** директория `docs/tutorials/ru/`
- **When** разработчик открывает любой из 10 файлов
- **Then** файл содержит: frontmatter (`title`, `related_examples`, `prerequisites`), введение, пошаговые инструкции, code-snippets, ссылку на релевантный пример, ссылку на следующий tutorial
- Evidence: `ls docs/tutorials/ru/ | wc -l` = 10; каждая ссылка на `examples/<backend>/` резолвится

### AC-011: tutorial 01-quickstart проходим за ≤10 минут

- **Given** новый разработчик, клонировавший репозиторий
- **When** следует шагам `docs/tutorials/ru/01-quickstart.md` без отступлений
- **Then** в пределах 10 минут получает работающий RAG-чат (с Ollama) или видит stderr-строку с указанием отсутствующей переменной окружения (формат: `error: required env var <NAME> not set; set LLM_PROVIDER=mock to run without API key`)
- Evidence: tutorial содержит quickstart-команды копипастой; ссылки на рабочий пример

### AC-012: каждый tutorial ссылается хотя бы на один example

- **Given** 10 tutorials
- **When** `grep -l 'examples/' docs/tutorials/ru/*.md` запускается
- **Then** все 10 файлов содержат ссылку на `examples/`
- Evidence: grep output

### AC-013: README обновлён с индексами

- **Given** новый посетитель GitHub-репозитория
- **When** открывает `README.md`
- **Then** в первых 50 строках видит: «Быстрый старт» (5..10 строк), «Примеры» (таблица), «Tutorials» (список 01..10), «Векторные хранилища», «Провайдеры»
- Evidence: README секции по якорям; ссылки валидны

### AC-014: CI matrix examples-smoke проходит

- **Given** PR в main
- **When** GitHub Actions запускает `.github/workflows/examples-smoke.yml`
- **Then** 6 параллельных jobs (по одному на бэкенд) поднимают Docker-сервис, запускают example с `LLM_PROVIDER=mock`, проверяют exit 0 и наличие `[mock]` в stdout; все 6 jobs зелёные
- Evidence: workflow файл + зелёные галочки в PR checks

### AC-015: docs/vector-stores.md capability-таблица ссылается на examples

- **Given** `docs/vector-stores.md` секция с capability-таблицей
- **When** читатель смотрит на строку "qdrant"
- **Then** видит ссылку на `examples/qdrant/`
- Evidence: grep `examples/qdrant` в `docs/vector-stores.md`

### AC-016: ноль изменений в pkg/draftrag и internal/

- **Given** репозиторий до и после фичи
- **When** `git diff --stat main -- pkg/ internal/` запускается
- **Then** пустой вывод (0 строк изменений)
- Evidence: git diff

### AC-017: существующие тесты проходят без изменений

- **Given** существующая test suite (`go test ./...`)
- **When** фича реализована
- **Then** exit 0; coverage не падает ниже текущих уровней (domain 100%, application ≥83.3%, vectorstore ≥60.7%)
- Evidence: `go test -cover ./...` output

## Допущения

- Docker доступен на машинах разработчиков и в CI (GitHub-hosted runners имеют Docker).
- `ollama/ollama`, `qdrant/qdrant`, `chromadb/chroma`, `semitechnologies/weaviate`, `milvusdb/milvus`, `pgvector/pgvector` — публичные образы с лицензиями, совместимыми с MIT/Apache (проверить в plan).
- GitHub Actions имеет достаточно ресурсов (7 ГБ RAM) для поднятия milvus standalone (требует ~2 ГБ).
- Разработчик имеет один из API-ключей (или Ollama локально) для non-mock режима. Mock режим покрывает тех, у кого ничего нет.
- Размерность эмбеддингов 1536 (OpenAI ada-002) — де-факто стандарт; для других размеров пользователь правит `EMBEDDING_DIM` в `.env`.
- Mock-эмбеддер детерминирован: `mockEmbed(text) = hash(text) -> 1536 floats в диапазоне [-1, 1]`. Это даёт стабильный (хотя и бессмысленный) retrieval для smoke-тестов.
- Tutorial 08-observability.md покрывает OTel hooks (`pkg/draftrag/otel`) и `domain.Hooks` callback; не требует running collector (только локальный stdout exporter).
- Tutorial 10-production-hardening.md покрывает resilience (`pkg/draftrag/resilience`: retry, circuit breaker), redactor (`domain.Redactor`) и observability вместе; работает поверх in-memory store для простоты.

## Критерии успеха

- SC-001 Time-to-first-answer для нового разработчика: ≤10 минут (clone → docker compose up → cp .env.example .env → set one API key → go run → ответ на вопрос).
- SC-002 CI examples-smoke matrix: ≤15 минут wall-clock на main runner (6 jobs параллельно).
- SC-003 Каждый из 10 tutorials читается за ≤5 минут; содержит минимум 1 работающий code-snippet.
- SC-004 Zero changes в `pkg/` и `internal/` (verify через `git diff --stat`).
- SC-005 Все 6 бэкендов имеют зелёный smoke-job в CI.

## Краевые случаи

- Разработчик без Docker: AC-006 (in-memory) покрывает. Остальные 5 бэкендов требуют Docker — README явно это упоминает.
- Разработчик без API-ключей и без Ollama: AC-008 (mock mode) покрывает все 6 бэкендов.
- Docker pull fail в CI: cache Docker images, retry on transient errors; не делать blocking.
- Milvus требует etcd + minio + standalone — отдельный compose с тремя сервисами; README объясняет ресурсоёмкость.
- ChromaDB новые версии (1.x) изменили API; pin к `chromadb/chroma:0.5.x` для стабильности.
- Weaviate требует модуль `text2vec-transformers` для встроенных эмбеддингов; в нашем case эмбеддер внешний, модуль не нужен — pin к `semitechnologies/weaviate:1.27.x`.
- Ollama требует предварительной загрузки модели (`ollama pull llama3.2`); README объясняет.

## Открытые вопросы

- RQ-007: достаточно ли одной ссылки "examples/qdrant" в строке capability-таблицы, или каждая capability-ячейка должна ссылаться? — отложено в plan: дефолт = одна ссылка на бэкенд в первой колонке; уточнить при ревью.
- Tutorial 09-evaluation: использовать существующий `pkg/draftrag/eval` (harness, metrics) — этого достаточно? Или нужен более простой пример? — отложено в plan: дефолт = использовать `eval` пакет, показать 2-3 простых case.
- Tutorial 10-production-hardening: один длинный tutorial или split на 10a-resilience, 10b-observability, 10c-redaction? — отложено в plan: дефолт = один tutorial 10, упоминающий все три; уточнить при ревью.
- Нужен ли отдельный `examples/cli/` (запуск через `go run ./examples/cli/index.go --store=qdrant --llm=ollama`)? Или 6 отдельных директорий — это уже достаточно? — отложено в plan: дефолт = 6 директорий, без CLI-обёртки; уточнить при ревью.

Готово к: /speckeep.inspect docs-and-examples
