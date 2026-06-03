# docs-and-examples Задачи

## Phase Contract

Inputs: `docs/specs/docs-and-examples/plan.md` (pass), `docs/specs/docs-and-examples/spec.md` (17 AC, 10 RQ), `docs/specs/docs-and-examples/data-model.md` (no-change), `CONSTITUTION.md` (Clean Architecture, ru-docs, mock-обязательство).
Outputs: исполнимые задачи с `Touches:` для каждой, `Surface Map`, `Implementation Context`, AC coverage.
Stop if: coverage не удаётся сопоставить — не наш случай (17 AC ↔ 18 задач однозначно мапятся).

## Surface Map

| Surface | Tasks |
|---------|-------|
| `examples/shared/mock.go` (новый) | T1.2 |
| `examples/shared/print.go` (новый) | T1.4 |
| `examples/shared/{mock,print}_test.go` (новые) | T1.5 |
| `examples/*/main.go` (buildComponents inline switch) | T1.3 (→T2.1–T3.3) |
| `examples/memory/main.go` (новый) | T2.1 |
| `examples/memory/.env.example` (новый) | T2.1 |
| `examples/memory/README.md` (новый) | T2.1 |
| `examples/pgvector/main.go` (рефактор) | T2.2 |
| `examples/pgvector/docker-compose.yml` (новый) | T2.2 |
| `examples/pgvector/.env.example` (новый) | T2.2 |
| `examples/pgvector/README.md` (обновление) | T2.2 |
| `examples/qdrant/main.go` (рефактор) | T2.3 |
| `examples/qdrant/docker-compose.yml` (новый) | T2.3 |
| `examples/qdrant/.env.example` (новый) | T2.3 |
| `examples/qdrant/README.md` (обновление) | T2.3 |
| `examples/chromadb/main.go` (новый) | T3.1 |
| `examples/chromadb/docker-compose.yml` (новый) | T3.1 |
| `examples/chromadb/.env.example` (новый) | T3.1 |
| `examples/chromadb/README.md` (новый) | T3.1 |
| `examples/weaviate/main.go` (новый) | T3.2 |
| `examples/weaviate/docker-compose.yml` (новый) | T3.2 |
| `examples/weaviate/.env.example` (новый) | T3.2 |
| `examples/weaviate/README.md` (новый) | T3.2 |
| `examples/milvus/main.go` (новый) | T3.3 |
| `examples/milvus/docker-compose.yml` (новый, multi-service) | T3.3 |
| `examples/milvus/.env.example` (новый) | T3.3 |
| `examples/milvus/README.md` (новый) | T3.3 |
| `.github/workflows/examples-smoke.yml` (новый) | T2.4, T3.4 |
| `README.md` (обновление) | T3.5 |
| `docs/vector-stores.md` (обновление) | T3.6 |
| `ROADMAP.md` (обновление) | T3.7 |
| `docs/tutorials/ru/01-quickstart.md` (новый) | T4.1 |
| `docs/tutorials/ru/02-basic-rag.md` (новый) | T4.1 |
| `docs/tutorials/ru/03-hybrid-search.md` (новый) | T4.1 |
| `docs/tutorials/ru/04-metadata-filter.md` (новый) | T4.2 |
| `docs/tutorials/ru/05-streaming.md` (новый) | T4.2 |
| `docs/tutorials/ru/06-atomic-update.md` (новый) | T4.2 |
| `docs/tutorials/ru/07-citations.md` (новый) | T4.3 |
| `docs/tutorials/ru/08-observability.md` (новый) | T4.3 |
| `docs/tutorials/ru/09-evaluation.md` (новый) | T4.3 |
| `docs/tutorials/ru/10-production-hardening.md` (новый) | T4.3 |

## Implementation Context

- Цель MVP: разработчик с Docker + Go запускает RAG-чат на одном из 6 бэкендов за ≤10 мин, читая `docs/tutorials/ru/01-quickstart.md`.
- Границы приемки: AC-006 (memory, no Docker), AC-009 (compose-validate), AC-016 (zero-diff pkg/internal), AC-017 (existing tests pass) — закрываются в MVP; остальные AC — в Phase 3-4.
- Ключевые правила: ноль правок в `pkg/drafrag/` и `internal/` (AC-016); только публичный API; mock-LLM обязателен (CONSTITUTION: каждый публичный интерфейс имеет мок); русский язык в docs/README/tutorials, английский — в коде/командах.
- Инварианты: каждый `examples/<backend>/main.go` = `func main()` + читает env через `examples/shared.LoadConfig` + строит pipeline через `examples/shared.BuildPipeline` + индексирует 10 demo-документов + отвечает на 1-2 sample-запроса; docker-compose.yml = один service (multi для milvus) с healthcheck + pinned image; `.env.example` = все env-переменные с дефолтами и комментариями; README = quickstart (3 шага) + env-таблица + troubleshooting.
- Контракты/протокол: env-контракт = `LLM_PROVIDER ∈ {mock,ollama,openai,anthropic}` + `*_API_KEY` / `*_BASE_URL` / `*_MODEL` / `EMBEDDING_DIM` (default 1536) / `TABLE_NAME` (pgvector) / `COLLECTION_NAME` (chromadb/weaviate/milvus); mock-LLM stdout-prefix = `[mock]` для grep в CI.
- Границы scope: не трогаем `pkg/`, `internal/`, `examples/chat/`, `examples/index-dir/`, существующий `Makefile`, `.github/workflows/ci.yml`; новый CI workflow — отдельный `examples-smoke.yml`.
- Proof signals: `go run ./examples/<backend>/` exit 0 + stdout содержит `[mock]`; `docker compose -f examples/<b>/docker-compose.yml config` exit 0; `go build ./examples/...` exit 0; CI matrix 6 jobs зелёные; `git diff --stat main -- pkg/ internal/` пустой; `go test ./...` exit 0; coverage не падает.
- References: `DEC-001` (shared package), `DEC-002` (6 директорий), `DEC-003` (pinned versions), `DEC-004` (CI matrix), `DEC-005` (mock implements interface), `DEC-006` (tutorial 10 единый), `DEC-007` (mock embedder), `DEC-008` (milvus в CI через compose); `RQ-001..RQ-010`; `AC-001..AC-017`.

## Фаза 1: Основа

Цель: подготовить `examples/shared/` — Go-пакет, разделяемый всеми 6 примерами, чтобы они не дублировали env-loading, pipeline-building, mock-LLM/Embedder и print-helpers.

- [x] ~~T1.1 Создать `examples/shared/config.go` — удалён при рефакторинге. Каждая main.go читает env напрямую через `os.Getenv`/`envOr` helpers.~~
- [x] T1.2 Реализовать `examples/shared/mock.go` — `mockLLM` реализует `domain.LLMProvider`; `mockEmbedder` реализует `domain.Embedder`; детерминированный хэш в `[-1, 1]` размерности `Config.Dimension`; echo-ответы LLM с префиксом `[mock] <truncated question>`. `var _ domain.LLMProvider = (*mockLLM)(nil)` compile-time check. Touches: `examples/shared/mock.go`. DEC-005, DEC-007. AC-008.
- [x] ~~T1.3 Реализовать `examples/shared/build.go` — удалён при рефакторинге. Каждая `main.go` содержит inline `buildComponents` с switch по `LLM_PROVIDER` (mock|ollama|openai|anthropic). Touches: каждая `examples/*/main.go`. DEC-001. AC-007.~~
- [x] T1.4 Реализовать `examples/shared/print.go` — `PrintAnswer(question string, answer string, sources []domain.RetrievalResult)`: формат `[Q] ... \n[A] ... \n[Sources] N chunks`. Touches: `examples/shared/print.go`.
- [x] T1.5 Добавить `examples/shared/mock_test.go` и `examples/shared/print_test.go` — table-driven tests: `mockEmbedder.Embed("foo") == mockEmbedder.Embed("foo")` (детерминизм); compile-time interface checks; `PrintAnswer` snapshot на простом input. Touches: `examples/shared/{mock,print}_test.go`. AC-017 (новые тесты).

## Фаза 2: MVP Slice

Цель: поставить минимально демонстрируемую ценность — memory example (без Docker) + рефакторинг pgvector/qdrant под shared package + первичный CI workflow (compose-validate). Закрывает AC-006, AC-009, AC-016, AC-017.

- [x] T2.1 Создать `examples/memory/{main.go, .env.example, README.md}` — `main.go` использует `examples/shared/`; `draftrag.NewInMemoryStore()` (без Docker); индексирует 10 demo-документов про Go; спрашивает "Что такое goroutine?"; exit 0. `.env.example` минимальный (только `LLM_PROVIDER`, `EMBEDDING_DIM`, `OLLAMA_HOST` опционально). README = quickstart (3 строки: `cd examples/memory && cp .env.example .env && go run .`). Touches: `examples/memory/main.go`, `examples/memory/.env.example`, `examples/memory/README.md`. AC-006.
- [x] T2.2 Рефакторить `examples/pgvector/main.go` под `examples/shared/`; добавить `examples/pgvector/.env.example`; docker-compose.yml уже существовал — обновлён README. Touches: `examples/pgvector/main.go`, `examples/pgvector/.env.example`, `examples/pgvector/README.md`. AC-001.
- [x] T2.3 Рефакторить `examples/qdrant/main.go` под `examples/shared/`; добавить `examples/qdrant/docker-compose.yml` (image: `qdrant/qdrant:v1.12.4`, healthcheck, port 6333) и `examples/qdrant/.env.example`. Обновить `examples/qdrant/README.md`. Touches: `examples/qdrant/main.go`, `examples/qdrant/docker-compose.yml`, `examples/qdrant/.env.example`, `examples/qdrant/README.md`. AC-002.
- [x] T2.4 Создать первичный `.github/workflows/examples-smoke.yml` — `on: pull_request, push to master`; 2 jobs: `compose-validate` (matrix pgvector, qdrant) + `examples-build` (`go build ./examples/...` + `go vet ./examples/...` + `go test ./examples/shared/`). Матрица по 5 backend'ам добавляется в T3.4. Touches: `.github/workflows/examples-smoke.yml`. AC-009.

## Фаза 3: Основная реализация

Цель: добавить 3 новых бэкенда (chromadb, weaviate, milvus), расширить CI до полной matrix с mock-LLM, обновить README/vector-stores.md/ROADMAP.md. Закрывает AC-003, AC-004, AC-005, AC-013, AC-014, AC-015.

- [x] T3.1 Создать `examples/chromadb/{main.go, docker-compose.yml, .env.example, README.md}` — image: `chromadb/chroma:0.5.20`, port 8000, healthcheck `curl /api/v1/heartbeat`. `main.go` использует `examples/shared/` + `draftrag.NewChromaDBStore`. README quickstart + ссылка на tutorial 04. Touches: `examples/chromadb/main.go`, `examples/chromadb/docker-compose.yml`, `examples/chromadb/.env.example`, `examples/chromadb/README.md`. AC-003.
- [x] T3.2 Создать `examples/weaviate/{main.go, docker-compose.yml, .env.example, README.md}` — image: `semitechnologies/weaviate:1.27.5`, port 8080, без text2vec-модуля (внешний embedder), `ENABLE_MODULES: ""`, `AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: "true"`. `main.go` использует `examples/shared/` + `draftrag.NewWeaviateStore`. README + ссылка на tutorial 03. Touches: `examples/weaviate/main.go`, `examples/weaviate/docker-compose.yml`, `examples/weaviate/.env.example`, `examples/weaviate/README.md`. AC-004.
- [x] T3.3 Создать `examples/milvus/{main.go, docker-compose.yml, .env.example, README.md}` — multi-service compose: `milvusdb/milvus:v2.4.10` (standalone) + `quay.io/coreos/etcd:v3.5.5` + `minio/minio:RELEASE.2023-09-04T19-57-53Z`, healthcheck'и на каждом. `main.go` использует `examples/shared/` + `draftrag.NewMilvusStore` (внутренний API, документирован как "API в разработке" + использование через internal). README с предупреждением "Milvus = самый ресурсоёмкий бэкенд, требует ~2 GB RAM". Touches: `examples/milvus/main.go`, `examples/milvus/docker-compose.yml`, `examples/milvus/.env.example`, `examples/milvus/README.md`. AC-005.
- [x] T3.4 Расширить `.github/workflows/examples-smoke.yml` — добавить 3-ю job `examples-smoke` с matrix `backend ∈ {memory, pgvector, qdrant, chromadb, weaviate, milvus}` × `LLM_PROVIDER=mock`; для non-memory backend'ов: `services:` блок (или `docker compose up -d` для milvus, см. DEC-008) + `go run ./examples/<backend>/` с проверкой `exit 0` + grep `[mock]` в stdout; memory — без services. Touches: `.github/workflows/examples-smoke.yml`. AC-014.
- [x] T3.5 Обновить `README.md` — добавить секции: «Быстрый старт» (≤10 строк с командой `go run ./examples/memory/`), «Примеры» (таблица 6 строк: backend → ссылка на `examples/<b>/`, описание в одну строку, env-переменные), «Tutorials» (список 01..10 со ссылками), «Провайдеры LLM» (таблица ollama/openai/anthropic/mock + какие API ключи нужны). Существующие секции не удалять. Touches: `README.md`. AC-013.
- [x] T3.6 Обновить `docs/vector-stores.md` — в capability-таблице в первой колонке (название бэкенда) добавить markdown-ссылку `[memory](examples/memory/)` (и аналогично для pgvector, qdrant, chromadb, weaviate, milvus). RQ-007 явно зафиксировано: только первая колонка, остальные ячейки без ссылок. Touches: `docs/vector-stores.md`. AC-015.
- [x] T3.7 Обновить `ROADMAP.md` — в секции «Реализовано ✅» добавить запись "Обширные examples + tutorials (docs-and-examples): 6 backend'ов × Docker Compose, 10 tutorials в `docs/tutorials/ru/`, CI smoke matrix". Touches: `ROADMAP.md`.

## Фаза 4: Документация + Проверка

Цель: написать 10 tutorials и доказать, что фича работает (verify gate). Закрывает AC-010, AC-011, AC-012, AC-016, AC-017.

- [x] T4.1 Создать `docs/tutorials/ru/01-quickstart.md`, `02-basic-rag.md`, `03-hybrid-search.md` — каждый с frontmatter (`title`, `related_examples: [examples/...]`, `prerequisites`); введение (1-2 абзаца); 3-7 шагов с code-snippets; ссылка на релевантный пример; ссылка на следующий tutorial. 01 = memory + mock (zero-config), 02 = qdrant + ollama (real LLM), 03 = weaviate + hybrid search. Touches: `docs/tutorials/ru/01-quickstart.md`, `docs/tutorials/ru/02-basic-rag.md`, `docs/tutorials/ru/03-hybrid-search.md`. AC-010, AC-011, AC-012.
- [x] T4.2 Создать `docs/tutorials/ru/04-metadata-filter.md`, `05-streaming.md`, `06-atomic-update.md` — 04 = chromadb + metadata filter через `SearchBuilder.WithMetadataFilter`; 05 = qdrant + `SearchBuilder.Stream()`; 06 = pgvector + `Pipeline.UpdateDocument` + atomic semantics. Touches: `docs/tutorials/ru/04-metadata-filter.md`, `docs/tutorials/ru/05-streaming.md`, `docs/tutorials/ru/06-atomic-update.md`. AC-010, AC-012.
- [x] T4.3 Создать `docs/tutorials/ru/07-citations.md`, `08-observability.md`, `09-evaluation.md`, `10-production-hardening.md` — 07 = `SearchBuilder.Cite()` / `InlineCite()`; 08 = `draftrag/otel.NewHooks` + stdout exporter; 09 = `pkg/draftrag/eval` harness + 2-3 case; 10 = единый tutorial с подсекциями 10.1 resilience (retry + circuit breaker через `draftrag.NewRetryLLMProvider`), 10.2 observability (OTel), 10.3 redaction (`domain.Redactor`). Touches: `docs/tutorials/ru/07-citations.md`, `docs/tutorials/ru/08-observability.md`, `docs/tutorials/ru/09-evaluation.md`, `docs/tutorials/ru/10-production-hardening.md`. AC-010, AC-012.
- [x] T4.4 Финальный gate — выполнить: `git diff --stat master -- pkg/ internal/` (пустой вывод, AC-016); `go test ./...` (exit 0, coverage не падает ниже domain 100% / application ≥83.3% / vectorstore ≥60.7%, AC-017); `go build ./examples/...` (exit 0); `docker compose -f examples/<b>/docker-compose.yml config` для всех 5 backend'ов (exit 0); `ls docs/tutorials/ru/*.md | wc -l` (равно 10, AC-010); `grep -l 'examples/' docs/tutorials/ru/*.md | wc -l` (равно 10, AC-012); проверка ссылок в docs/vector-stores.md: `grep -oP 'examples/[a-z-]+/\)' docs/vector-stores.md | sort -u | wc -l` (≥ 6, AC-015). Touches: вся репа (read-only verification). AC-016, AC-017.

## Покрытие критериев приемки

- AC-001 -> T2.2
- AC-002 -> T2.3
- AC-003 -> T3.1
- AC-004 -> T3.2
- AC-005 -> T3.3
- AC-006 -> T2.1
- AC-007 -> T1.3
- AC-008 -> T1.2
- AC-009 -> T2.4
- AC-010 -> T4.1
- AC-010 -> T4.2
- AC-010 -> T4.3
- AC-011 -> T4.1
- AC-012 -> T4.1
- AC-012 -> T4.2
- AC-012 -> T4.3
- AC-013 -> T3.5
- AC-014 -> T3.4
- AC-015 -> T3.6
- AC-016 -> T4.4
- AC-017 -> T1.5
- AC-017 -> T4.4

## Заметки

- Порядок задач = порядок коммитов: Phase 1 (shared foundation) → Phase 2 (MVP с memory + pgvector + qdrant + initial CI) → Phase 3 (chromadb/weaviate/milvus + matrix CI + README/docs) → Phase 4 (tutorials + verify).
- Фаза 1 → Фаза 2 строго последовательны (shared нужен примерам). Внутри Фазы 2: T2.1, T2.2, T2.3 независимы (разные backend'ы) — могут параллелиться. T2.4 зависит от T2.2 + T2.3 (compose-файлы должны существовать).
- Внутри Фазы 3: T3.1, T3.2, T3.3 независимы (разные backend'ы) — могут параллелиться. T3.4 зависит от T3.1-T3.3 (matrix должен включать все backend'ы). T3.5, T3.6, T3.7 — независимые markdown-правки.
- Внутри Фазы 4: T4.1, T4.2, T4.3 независимы (10 разных тем). T4.4 — последняя задача поверх всех.
- Каждый task ID = phase-scoped (`T<phase>.<index>`). Phase 1 = foundation; Phase 2 = MVP; Phase 3 = examples + CI + docs; Phase 4 = tutorials + verify.
- Trace-маркеры `@sk-task docs-and-examples#T<n>.<m>` / `@sk-test docs-and-examples#T<n>.<m>` обязательны для marking задачи выполненной (см. AGENTS.md). Размещение — над owning function/method/test/type в `examples/shared/*.go` и `examples/<backend>/main.go`, НЕ на package/import/file-header.
- Не редактируйте исходный код `pkg/` или `internal/` на фазе implement (AC-016). Если обнаружится, что какого-то публичного API не хватает для удобного примера — это блокер, который возвращает в spec-фазу для уточнения.
- Implement-агент читает только `tasks.md` + файлы из `Touches:` активной задачи + `## Implementation Context` (без перечитывания `spec.md`/`plan.md`/`data-model.md`).
- Если задача получает новые файлы (например, новый helper в `examples/shared/`) — добавляйте их в Surface Map в patch-режиме, не переписывая tasks.md целиком.
- Для shared package: пакет размещается в `examples/shared/` (не `internal/`), чтобы `go build ./pkg/...` его не захватывал, но `go build ./examples/...` работал. Импортируется как `github.com/bzdvdn/draftrag/examples/shared`.
- Для tutorial 10: единый файл ≤400 строк с подсекциями 10.1/10.2/10.3 + якорные ссылки в начале файла. Не расщеплять на 10a/10b/10c.
- Для env-контракта: `LLM_PROVIDER=mock` — режим по умолчанию для CI и zero-config запуска. Если в `.env` не задан `LLM_PROVIDER` явно — `LoadConfig` возвращает ошибку с сообщением формата `error: required env var LLM_PROVIDER not set; set LLM_PROVIDER=mock to run without API key` (как требует AC-011).
