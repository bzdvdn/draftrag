# docs-and-examples План

## Phase Contract

Inputs: `docs/specs/docs-and-examples/spec.md` (pass), `docs/specs/docs-and-examples/inspect.md` (pass), `CONSTITUTION.md` (Clean Architecture, ru-docs, mock-обязательство для каждого публичного интерфейса), текущее состояние `examples/`, `.github/workflows/`, `README.md`.
Outputs: `plan.md` (этот файл), `data-model.md` (no-change stub).
Stop if: spec расплывчата для безопасного планирования — не наш случай (inspect = pass, 17 AC, 0 unresolved markers).

## Цель

Реализовать фичу "обширная документация и легко запускаемые примеры" как чисто examples/docs/CI работу, без правок `pkg/draftrag/` и `internal/`. План фиксирует: 6 директорий `examples/<backend>/` под общим шаблоном, 1 shared Go-пакет `examples/shared/`, 10 tutorials, обновления 3 markdown-файлов, 1 новый CI workflow с матрицей. Подход безопасен, т.к. examples уже существуют и работают (`pgvector`, `qdrant`, `chat`, `index-dir`); план их рефакторит, а не переписывает с нуля. Mock-LLM режим позволяет CI smoke-test без API-ключей, что решает проблему "как проверить RAG end-to-end в CI".

## MVP Slice

- Минимальный срез = 4 примера (memory + pgvector + qdrant + chromadb) + 1 tutorial 01-quickstart + 1 CI-job `compose-validate` (синтаксис compose-файлов) + 1 CI-job `examples-build` (`go build ./examples/...`).
- AC, обязательные до расширения scope: AC-006 (memory example, без Docker — самый дешёвый smoke), AC-009 (compose-validate), AC-016 (zero-diff pkg/internal), AC-017 (existing tests pass).
- После MVP: chromadb/weaviate/milvus, tutorials 02..10, полный CI matrix с mock-LLM.

## First Validation Path

```bash
# 1. Branch
git checkout feature/docs-and-examples

# 2. Shared package + memory example (no Docker)
go run ./examples/memory/
# → exit 0, [mock] echo-ответ на тестовый вопрос

# 3. Compose validation (fast, no services started)
docker compose -f examples/pgvector/docker-compose.yml config
docker compose -f examples/qdrant/docker-compose.yml config
# ... и т.д. для всех 5

# 4. Build all examples
go build ./examples/...

# 5. CI smoke matrix (на main push)
gh workflow run examples-smoke.yml
# → 6 jobs параллельно, mock LLM, exit 0 в каждом

# 6. Zero-diff gate
git diff --stat main -- pkg/ internal/
# → пустой вывод
```

## Scope

- Зона 1: 6 директорий `examples/<backend>/` под общим шаблоном (4 новых: memory/chromadb/weaviate/milvus; 2 рефакторинг: pgvector/qdrant).
- Зона 2: 1 shared Go-пакет `examples/shared/` (loadConfig, buildPipeline, mockLLM, mockEmbedder, printAnswer).
- Зона 3: 10 файлов `docs/tutorials/ru/NN-*.md` (01-quickstart, 02-basic-rag, 03-hybrid-search, 04-metadata-filter, 05-streaming, 06-atomic-update, 07-citations, 08-observability, 09-evaluation, 10-production-hardening).
- Зона 4: 1 новый CI workflow `.github/workflows/examples-smoke.yml` (matrix 6 × mock).
- Зона 5: точечные правки `README.md` (добавить индексы), `docs/vector-stores.md` (ссылки в capability-таблице), `ROADMAP.md` (упоминание).
- Нетронутая граница: `pkg/draftrag/*.go`, `internal/**/*.go` — zero changes (AC-016); `examples/chat/`, `examples/index-dir/` — не трогаем (другая функция); `Makefile` — не трогаем (его `make test/lint/build` уже покрывают либу; для examples добавим отдельный `make examples-smoke` в самой новой CI workflow, не в корневой Makefile).

## Implementation Surfaces

- `examples/shared/` (новый Go-пакет) — почему новый: 6 примеров должны разделять loadConfig (env-чтение + дефолты), buildPipeline (фабрика по LLM_PROVIDER), printAnswer (форматированный вывод), mockLLM/mockEmbedder. Дублировать 6 раз = 6x расходы на поддержку; централизация в одном пакете устраняет это. Не делаем internal-пакет, т.к. examples в той же Go-module (`github.com/bzdvdn/draftrag`), и `examples/shared` доступен через обычный import path. Пакет размещается в `examples/` (не `internal/`), чтобы явно отделить его от библиотечного кода — `go build ./pkg/...` его не захватывает.
- `examples/<backend>/main.go` (4 новых, 2 рефактор) — каждый = `func main()`, читает env через `examples/shared/loadConfig`, строит pipeline через `examples/shared/buildPipeline`, индексирует 10 demo-документов, делает 1-2 sample-запроса, печатает ответы через `examples/shared/printAnswer`. ~80..150 LOC каждый.
- `examples/<backend>/docker-compose.yml` (5 новых; memory без) — один сервис (для chromadb, qdrant, weaviate, pgvector) или multi-service (для milvus: standalone + etcd + minio). Healthcheck обязателен (RQ-010). Pin версии образа.
- `examples/<backend>/.env.example` (6 новых) — все env-переменные с дефолтами и комментариями. Один источник правды для env-контракта.
- `examples/<backend>/README.md` (6) — quickstart (3 шага), env-таблица, troubleshooting (порт занят, Docker не установлен, etc.), ссылка на релевантный tutorial.
- `docs/tutorials/ru/NN-*.md` (10) — frontmatter (title, related_examples, prerequisites) + введение + 3..7 шагов с code-snippets + ссылка на релевантный пример + ссылка на следующий tutorial.
- `.github/workflows/examples-smoke.yml` (новый) — `on: pull_request, push to main`; `strategy.matrix.backend: [pgvector, qdrant, chromadb, weaviate, milvus, memory]`; `services:` (для non-memory) с `image:`, `ports:`, `options: --health-cmd ...`; `steps:` checkout + setup-go + `go run ./examples/<backend>/` с `LLM_PROVIDER=mock` + `go test` встроенный sanity (ожидаем подстроку в stdout). Memory example — без services. Milvus — отдельный compose-up подход, т.к. GitHub services не поддерживает multi-container compose.
- `README.md` (правка) — добавить секции "Примеры" (6-row таблица), "Tutorials" (список 01..10), "Провайдеры LLM" (таблица ollama/openai/anthropic + mock). Существующие секции не удалять.
- `docs/vector-stores.md` (правка) — в capability-таблице в первой колонке (название бэкенда) добавить markdown-ссылку `[memory](examples/memory/)` и т.д. AC-015.
- `ROADMAP.md` (правка) — добавить запись в секции "Реализовано ✅" про новые examples + tutorials.

## Bootstrapping Surfaces

- Создать директорию `examples/shared/` с пустым `go.mod`-compatible пакетом (без отдельного go.mod; внутри module `github.com/bzdvdn/draftrag`).
- Создать `examples/memory/main.go` как первый reference implementation — на нём отлаживаем shared package.
- Создать `docs/tutorials/ru/.gitkeep` (если директория новая).
- Существующие `examples/pgvector/main.go` и `examples/qdrant/main.go` остаются — рефакторим их под shared-пакет в задачах T2.x.
- Проверить `.github/workflows/ci.yml` (уже существует) — добавить туда или рядом `examples-smoke.yml`. Не модифицировать существующий ci.yml (его `go test` + lint покрывают либу; example-smoke — отдельный workflow).

## Влияние на архитектуру

- Локальное: ноль влияния на архитектуру библиотеки. `pkg/draftrag/` не меняется. `internal/` не меняется.
- Примеры потребляют только публичный API: `draftrag.NewPipeline`, `draftrag.New<Backend>Store`, `draftrag.New<Provider>Embedder`, `draftrag.New<Provider>LLM`, `draftrag.NewBasicChunker`, `draftrag.NewHooks`. Если в ходе implementation обнаружится, что какого-то конструктора или опции не хватает для удобного примера — это блокер, который возвращает в spec-фазу для уточнения (а не молча добавляет в `pkg/`).
- Миграции/совместимость: не применимо (ноль изменений в публичном API).
- Rollout: новая CI workflow добавляется как `examples-smoke.yml` (отдельный файл); существующий `ci.yml` не трогаем. Merge в main → оба workflow (ci.yml + examples-smoke.yml) начинают работать.

## Acceptance Approach

- AC-001 (pgvector) → подход: рефакторинг `examples/pgvector/main.go` под `examples/shared/` + новый `docker-compose.yml` (уже есть) + новый `README.md` (уже есть, обновить) + новый `.env.example` (создать). Surfaces: `examples/pgvector/{main.go,docker-compose.yml,.env.example,README.md}`. Наблюдаемо: `cd examples/pgvector && docker compose up -d && LLM_PROVIDER=mock go run .` → exit 0 + stdout содержит `[mock]`.
- AC-002 (qdrant) → аналогично AC-001. Surfaces: `examples/qdrant/*`. Существующий `main.go` использует qdrant без docker-compose; добавляем compose + .env.example.
- AC-003 (chromadb) → новый. Surfaces: `examples/chromadb/{main.go,docker-compose.yml,.env.example,README.md}`. Image: `chromadb/chroma:0.5.20`.
- AC-004 (weaviate) → новый. Surfaces: `examples/weaviate/*`. Image: `semitechnologies/weaviate:1.27.5`. Без модуля text2vec-* (эмбеддер внешний).
- AC-005 (milvus) → новый. Surfaces: `examples/milvus/*`. Multi-service compose: standalone + etcd + minio. Image: `milvusdb/milvus:v2.4.10`.
- AC-006 (memory, без Docker) → новый. Surfaces: `examples/memory/{main.go,.env.example,README.md}` (без compose). Использует `draftrag.NewInMemoryStore()`.
- AC-007 (LLM provider switching) → `examples/shared/llm.go::buildLLM(env) (LLMProvider, error)`. Surfaces: `examples/shared/llm.go`. Наблюдаемо: README объясняет; CI проверяет 3 из 4 провайдеров (mock + ollama + openai; anthropic — опционально, требует API key, не в CI).
- AC-008 (mock LLM без ключей) → `examples/shared/mock.go::mockLLM`, `mockEmbedder`. Surfaces: `examples/shared/mock.go`. Наблюдаемо: CI smoke с `LLM_PROVIDER=mock` exit 0.
- AC-009 (compose validate) → CI job `compose-validate` в `examples-smoke.yml`; `docker compose -f examples/<b>/docker-compose.yml config`. Surfaces: `.github/workflows/examples-smoke.yml`.
- AC-010 (10 tutorials) → `docs/tutorials/ru/{01..10}-*.md`. Surfaces: `docs/tutorials/ru/`. Наблюдаемо: `ls docs/tutorials/ru/*.md | wc -l` = 10.
- AC-011 (tutorial 01 quickstart ≤10 мин) → контент `docs/tutorials/ru/01-quickstart.md`. Наблюдаемо: ручной прогон (developer-experience metric; SC-001).
- AC-012 (каждый tutorial ссылается на example) → `grep -l 'examples/' docs/tutorials/ru/*.md` возвращает 10 строк. Surfaces: `docs/tutorials/ru/*.md`.
- AC-013 (README индексы) → секции в `README.md`. Surfaces: `README.md`. Наблюдаемо: grep `^## ` в README.
- AC-014 (CI matrix зелёный) → `.github/workflows/examples-smoke.yml`. 6 jobs. Surfaces: тот же файл.
- AC-015 (capability-таблица линки) → `docs/vector-stores.md`. Surfaces: тот же файл.
- AC-016 (zero-diff pkg/internal) → `git diff --stat main -- pkg/ internal/`. Не требует surface; это gate после implement.
- AC-017 (existing tests) → `go test ./...` exit 0. Не требует surface; это gate после implement.

## Данные и контракты

- AC-016, AC-017: `data-model.md` = no-change (см. `docs/specs/docs-and-examples/data-model.md`). Никаких persisted entities, value objects, state transitions или contract-relevant payload shapes не вводится/модифицируется.
- Env-контракт (новый, но не persisted): `examples/<backend>/.env.example` определяет переменные `LLM_PROVIDER`, `*_API_KEY`, `*_BASE_URL`, `*_MODEL`, `EMBEDDING_DIM`, `TABLE_NAME` (pgvector), `COLLECTION_NAME` (chromadb, weaviate, milvus). Это контракт между example и внешней средой, не часть публичного API библиотеки. Документируется в каждом `.env.example` с комментариями.
- API contracts: не меняются (AC-016).
- Event contracts: не применимо.
- `data-model.md` = no-change stub (см. `data-model.md`).

## Стратегия реализации

### DEC-001: Shared Go-пакет в `examples/shared/` (а не дублирование в каждом example)

- Why: 6 примеров должны разделять env-loading, pipeline-building, mock-LLM/mock-Embedder. Дублирование = 6x расходы на поддержку при багфиксах; централизация = одна точка истины.
- Tradeoff: добавляется один новый internal Go-пакет; примеры теперь зависят от него. Если пакет плохо спроектирован — проблема во всех 6 примерах сразу. Митигируется: пакет = thin wrappers (≤200 LOC), unit-тестируется отдельно.
- Affects: `examples/shared/{config.go,llm.go,embedder.go,mock.go,print.go}` + импорты во всех 6 `examples/<backend>/main.go`.
- Validation: `go build ./examples/...` exit 0; `examples/shared/shared_test.go` (новый) — юнит-тесты на `loadConfig`, `buildLLM` с разными env-значениями.

### DEC-002: 6 отдельных директорий (а не unified `examples/cli/` с флагами)

- Why: каждый бэкенд имеет свой docker-compose, env-набор, edge cases. CLI-фасад с `--store=X` потребовал бы условной логики запуска разных docker-compose файлов + env-mapping — больше кода, меньше прозрачности.
- Tradeoff: 6 директорий вместо одной. Но каждая самодостаточна, README читается изолированно, можно скопировать одну директорию к себе и запустить standalone.
- Affects: `examples/<backend>/` × 6.
- Validation: каждая директория собирается независимо (`cd examples/<b> && go build .` exit 0).

### DEC-003: Pin конкретных версий Docker images (не `:latest`)

- Why: воспроизводимость. RQ-010 явно требует pin. `:latest` ломает CI при обновлении upstream images.
- Tradeoff: приходится периодически обновлять pin (одна строка). Митигируется: pinned версии в `.env.example` или compose; Dependabot/Renovate не настраиваем (out of scope).
- Affects: `examples/<backend>/docker-compose.yml`.
- Validation: `docker compose pull` + `docker compose up -d` в CI; образ стартует с pinned версией.

### DEC-004: CI matrix — 6 параллельных jobs с mock-LLM

- Why: гарантирует, что каждый из 6 бэкендов реально поднимается и example с ним работает. Без matrix — нет гарантии, что новый коммит не сломал chromadb, например.
- Tradeoff: matrix использует CI-минуты (6 × 5 мин = 30 мин на PR). GitHub Actions free tier = 2000 мин/мес; оставим margin. Альтернатива: один job с `docker compose up` для всех сразу — но тогда ошибка в одном бэкенде валит остальные.
- Affects: `.github/workflows/examples-smoke.yml` (новый файл).
- Validation: 6 зелёных jobs в PR check.

### DEC-005: Mock-LLM реализует `domain.LLMProvider` и `domain.Embedder` напрямую

- Why: композиция — example строит pipeline как обычно (`NewPipeline(store, mockLLM, mockEmbedder, ...)`); не нужны отдельные mock-функции в pipeline.
- Tradeoff: mock должен корректно реализовать интерфейс (signature-compatible); при добавлении нового метода в `domain.LLMProvider` mock тоже нужно обновить. Митигируется: `var _ domain.LLMProvider = (*mockLLM)(nil)` compile-time check.
- Affects: `examples/shared/mock.go`.
- Validation: `examples/shared/shared_test.go::TestMockImplementsInterfaces` (compile + minimal smoke).

### DEC-006: Tutorial 10 — единый файл с подсекциями 10.1/10.2/10.3

- Why: resilience + observability + redaction — все три относятся к "production hardening"; расщепление на 10a/10b/10c раздувает индекс (13 tutorials вместо 10).
- Tradeoff: длинный файл (≈300 строк). Митигируется: якорные ссылки `#resilience`, `#observability`, `#redaction` в начале файла; frontmatter `prerequisites: 09-evaluation`.
- Affects: `docs/tutorials/ru/10-production-hardening.md`.
- Validation: `wc -l docs/tutorials/ru/10-production-hardening.md` ≤ 400; каждая подсекция имеет code-snippet.

### DEC-007: Mock embedder — детерминированный хэш, размерность 1536 (configurable через EMBEDDING_DIM)

- Why: детерминизм нужен для reproducible smoke-тестов; 1536 — де-факто стандарт (OpenAI ada-002), но пользователь может переопределить. Хэш в `[-1, 1]` даёт косинусное расстояние, не нулевое.
- Tradeoff: retrieval-качество mock'а — не реалистичное (random-ish, но стабильное). Это нормально для smoke; production требует реальный эмбеддер.
- Affects: `examples/shared/mock.go::mockEmbedder.Embed`.
- Validation: `shared_test.go` проверяет детерминизм (`mockEmbed("foo") == mockEmbed("foo")`).

### DEC-008: Milvus в CI запускается через `docker compose up` в одном job (не через `services:`)

- Why: GitHub Actions `services:` поддерживает только один контейнер; Milvus требует три (standalone + etcd + minio). Multi-container = `docker compose`.
- Tradeoff: один job (не параллелится с другими бэкендами внутри одного job'а); но job всё равно параллелен с остальными 5 в matrix. Реальное wall-clock ≤ 15 мин на job (митигируется RQ-010 + DEC-003 — pinned versions ускоряют pull).
- Affects: `.github/workflows/examples-smoke.yml` для `backend=milvus` — `run: docker compose -f examples/milvus/docker-compose.yml up -d`.
- Validation: job `examples-smoke (backend: milvus)` exit 0.

## Incremental Delivery

### MVP (Первая ценность)

- Задачи: T1.1 (shared package скелет), T1.2 (mockLLM + mockEmbedder), T2.1 (memory example), T3.1 (compose-validate CI), T3.2 (examples-build CI).
- AC покрываются: AC-006 (memory), AC-009 (compose-validate), AC-016, AC-017.
- Проверка: `go run ./examples/memory/` exit 0; `docker compose -f examples/pgvector/docker-compose.yml config` exit 0; `go test ./...` exit 0; `git diff --stat main -- pkg/ internal/` пустой.

### Итеративное расширение

- Step A: refactor pgvector + qdrant под shared package → AC-001, AC-002.
- Step B: новые chromadb, weaviate, milvus examples → AC-003, AC-004, AC-005.
- Step C: CI matrix с mock-LLM (6 jobs параллельно) → AC-008, AC-014.
- Step D: README + vector-stores.md + ROADMAP.md правки → AC-013, AC-015.
- Step E: 10 tutorials → AC-010, AC-011, AC-012.
- Step F: LLM_PROVIDER switch (ollama/openai/anthropic) документирован и в example коде → AC-007.
- Каждый step — отдельный коммит, тесты зелёные, MVP из previous step не сломан.

## Порядок реализации

- Первый: `examples/shared/` (foundation — без него примеры не запустятся).
- Параллельно после shared: 6 примеров независимы (разные backend → разные поверхности) — могут делаться параллельными ветками или последовательно в одном.
- Параллельно с примерами: 10 tutorials — независимы по содержимому (разные темы), могут делаться в любом порядке после MVP.
- CI workflow — последним (после того, как все examples существуют).
- Что за флагом / guarded rollout: ничего (новые workflow + новые директории, ноль impact на существующие CI).
- Что требует merge в main до rollout: ничего (CI workflow активируется автоматически при push в main).

## Риски

- R-1 Milvus CI: требует ~2 GB RAM на job, запуск 60-90s.
  Mitigation: pinned version v2.4.10 (стабильнее latest); cache Docker layers; в README явно отмечено "Milvus = самый ресурсоёмкий бэкенд".
- R-2 Weaviate startup time: 30-60s до ready healthcheck.
  Mitigation: healthcheck в compose с `start_period: 60s`; retry budget 10.
- R-3 ChromaDB 0.5 vs 1.x breaking changes: API может измениться.
  Mitigation: pin к `chromadb/chroma:0.5.20` (последняя стабильная 0.5.x); в README "Если у вас chroma 1.x, см. migration guide" (но это не в scope).
- R-4 Mock embedder dimension mismatch: если пользователь выберет модель с dim≠1536, retrieval сломается.
  Mitigation: `.env.example` явно говорит "EMBEDDING_DIM должен совпадать с моделью"; mock embedder читает `EMBEDDING_DIM` из env (default 1536).
- R-5 Ollama в CI: модель не скачана, ollama pull занимает 5+ мин + 3-5 GB.
  Mitigation: ollama НЕ включается в CI matrix (mock достаточно); в README tutorial 02 объясняет `ollama pull llama3.2`.
- R-6 Tutorial links rot: если рефакторим examples (rename), tutorial-ссылки ломаются.
  Mitigation: shared CI job `check-tutorial-links` — `grep` ломающихся путей в `docs/tutorials/ru/*.md` и валидация `examples/<b>/main.go` exists.
- R-7 CI minute budget: 6 jobs × 5 мин = 30 мин/PR + другие jobs (lint, test) ≈ 5 мин = 35 мин/PR.
  Mitigation: matrix jobs кэшируют Go modules; per-job timeout 15 мин. Если превысим free tier — рассматриваем self-hosted runner (out of scope сейчас).
- R-8 examples/shared импортирует pkg/draftrag, который импортирует internal/infrastructure, который импортирует pgx. Если CI не имеет network для `go mod download`, build ломается.
  Mitigation: existing `ci.yml` уже делает `go mod download`; новый workflow наследует тот же `actions/setup-go@v5` с кэшем.

## Rollout и compatibility

- Backfill: не применимо.
- Migration: не применимо.
- Feature flag: не применимо.
- Compatibility: ноль изменений в публичном API (AC-016).
- Monitoring/auditability после rollout: новый CI job `examples-smoke` добавляется в GitHub Actions; его статус виден на каждом PR. Если упадёт — ревьюер блокирует merge.
- Специальных rollout-действий не требуется: новый CI workflow активируется автоматически при первом push в main с этим файлом; существующий CI (`ci.yml`) не меняется.

## Проверка

- Automated tests: новые `examples/shared/{config_test,llm_test,mock_test,print_test}.go` — unit-тесты shared-пакета.
- Targeted manual checks:
  - `cd examples/<b> && docker compose up -d && LLM_PROVIDER=mock go run .` для каждого из 6 бэкендов → exit 0 + stdout содержит `[mock]`.
  - `cd examples/<b> && docker compose down -v` — cleanup.
  - Tutorial 01 (memory + mock) — пройти за ≤10 мин.
  - Tutorial 02 (qdrant + ollama) — пройти с локальным ollama.
- `AC-*` и `DEC-*` подтверждение:
  - `go test ./examples/...` exit 0 → DEC-001, AC-006, AC-017.
  - `docker compose -f examples/<b>/docker-compose.yml config` exit 0 для всех 5 → AC-009, DEC-003.
  - CI matrix 6 jobs зелёные → AC-008, AC-014, DEC-004.
  - `git diff --stat main -- pkg/ internal/` пустой → AC-016.
  - `wc -l docs/tutorials/ru/*.md | tail -1` = 10 файлов → AC-010.
  - `grep -l 'examples/' docs/tutorials/ru/*.md | wc -l` = 10 → AC-012.
  - `grep -c '^\[.*\](examples/.*/)' docs/vector-stores.md` ≥ 6 → AC-015.
  - `go test -cover ./...` показывает те же уровни coverage, что до фичи → AC-017.

## Соответствие конституции

- ✅ "Код ДОЛЖЕН следовать принципам Clean Architecture" — examples используют только `pkg/draftrag` (публичный API); ноль правок `internal/`. (AC-016 как gate.)
- ✅ "Каждый публичный интерфейс ДОЛЖЕН иметь мок-реализацию" — DEC-005 + `examples/shared/mock.go` (mockLLM, mockEmbedder реализуют `domain.LLMProvider` и `domain.Embedder`).
- ✅ "Все операции ДОЛЖНЫ принимать context.Context" — examples передают `ctx` во все `Pipeline.*` методы.
- ✅ "Все новые функции ДОЛЖНЫ иметь unit-тесты" — `examples/shared/*_test.go` покрывают shared-пакет; CI smoke покрывает сами examples.
- ✅ "Документация: каждый публичный тип и функция ДОЛЖНЫ иметь godoc-комментарий на русском языке" — не затрагивается (ноль новых публичных типов).
- ✅ "Язык документации: русский" — все README, .env.example комментарии, tutorials на русском; код Go и команды shell на английском.
- ✅ "Каждая фича ДОЛЖНА разрабатываться в отдельной git-ветке с префиксом feature/<slug>" — `feature/docs-and-examples` создана в spec-фазе.
- ✅ "Время сборки: go build ./... ДОЛЖЕН завершаться <5 секунд" — 6 новых `main.go` собираются параллельно; инкрементальный build остаётся быстрым.
- ✅ "Тестовое покрытие: ≥80% для domain и application слоёв" — не должно упасть; existing tests + новые shared_*_test.
- Нет конфликтов с конституцией.
