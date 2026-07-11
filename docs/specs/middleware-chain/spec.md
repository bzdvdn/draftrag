# Middleware Chain — плагинная система между стадиями pipeline

## Scope Snapshot

- In scope: единая middleware-цепочка для сквозных concern'ов на границах стадий pipeline (PII, guardrails, логирование, валидация).
- Out of scope: реализация конкретных middleware (кроме демо); замена существующих Hooks или PIIDetector.

## Цель

Разработчик RAG-системы получает единый механизм для внедрения сквозных проверок и трансформаций на границах стадий pipeline — без необходимости дублировать логику или модифицировать ядро. Успех фичи определяется возможностью подключить произвольную последовательность middleware (например, PII-фильтр → guardrail → логгер) одной опцией в `PipelineOptions`, при этом каждая middleware может модифицировать или прервать поток.

## Основной сценарий

1. Пользователь конфигурирует `PipelineOptions` с цепочкой middleware (срез разнородных обработчиков).
2. При вызове `Index`, `Query`, `Retrieve`, `Answer` или `AnswerStream` pipeline для каждой стадии (chunking, embed, search, generate) вызывает middleware-цепочку: сначала pre-middleware (до выполнения стадии), затем основная логика стадии, затем post-middleware (после выполнения стадии).
3. Каждая middleware получает запрос и контекст выполнения, может прочитать/изменить входящие данные или вернуть ошибку.
4. При ошибке в middleware pipeline прерывается с этой ошибкой; успешный проход всех middleware передаёт управление следующей стадии.
5. Если middleware не сконфигурирована — pipeline работает без изменений (no-op).

## User Stories

- P1: Разработчик может подключить 1–N middleware через одну опцию и быть уверен, что они вызовутся для всех pipeline-операций в объявленном порядке.
- P2: Разработчик может написать простую middleware (5–10 строк), которая модифицирует запрос или ответ, не трогая остальной pipeline.
- P3: Разработчик может прервать pipeline из middleware (например, guardrail отклонил запрос) и получить понятную ошибку.

## MVP Slice

Одна опция `PipelineOptions.Middleware` типа `[]Middleware`, принимающая срез интерфейсных значений. Middleware цепочка выполняется в заданном порядке на стадиях Index, Query, Retrieve, Answer, AnswerStream. Без middleware pipeline идентичен текущему поведению. Минимальный интерфейс `Middleware` — один метод `Handle`. MVP закрывает AC-001–AC-005.

## First Deployable Outcome

После первого implementation pass: пример `examples/middleware/main.go`, где сконфигурирован pipeline с двумя middleware (например, логгер + PII-цензор), и `go run ./examples/middleware` показывает в stdout порядок вызова middleware и результат работы.

## Scope

- Интерфейс `Middleware` в `internal/domain/` с методом-обработчиком, принимающим контекст выполнения, стадию и данные.
- Middleware-цепочка как срез, выполняется последовательно (каждая middleware получает результат предыдущей).
- Интеграция в `PipelineOptions`: опция `Middleware []middleware.Middleware`.
- Интеграция в pipeline: middleware вызываются на всех HookStage-стадиях: pre/post для chunking, embed, search, generate.
- Существующие Hooks продолжают работать независимо (наблюдаемость, не модификация).
- Возможность для middleware прервать pipeline, вернув sentinel-ошибку.
- Middleware не имеет доступа к внутренностям pipeline (store, llm, embedder) — только к данным, проходящим через стадию.

## Контекст

- В репозитории уже есть разрозненные механизмы сквозных concern'ов: `Hooks` (observability), `PIIDetector` (PII-фильтр), кастомные проверки в pipeline. Нет единого расширяемого контракта.
- Middleware должны быть легковесными: без рефлексии, кодогенерации или длительной инициализации.
- `context.Context` пробрасывается во все вызовы middleware; middleware может унаследовать/отменить контекст.
- Все публичные операции pipeline (`context.Context`) остаются совместимыми — middleware невидима для вызывающего кода (opaque injection).

## Зависимости

- `internal/domain/` — старый интерфейс `Hooks` остаётся; ни одна команда не требует его изменения.
- `none` — внешних зависимостей не требуется; механизм строится на стандартных типах Go.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять интерфейс `Middleware` в `internal/domain/`.
- RQ-002 Middleware цепочка ДОЛЖНА выполняться последовательно в порядке объявления.
- RQ-003 Pipeline ДОЛЖЕН вызывать middleware на каждой стадии из HookStage: chunking, embed, search, generate. Для каждой стадии выполняются pre-middleware (до) и post-middleware (после). На Index-пути — pre_chunking, post_chunking, pre_embed, post_embed. На Query/Retrieve/Answer-путях — pre_search, post_search, pre_generate, post_generate.
- RQ-004 Ошибка из любой middleware ДОЛЖНА немедленно прерывать pipeline: downstream-стадии и последующие middleware не выполняются, ошибка возвращается вызывающему коду.
- RQ-005 Middleware ДОЛЖНА иметь доступ к модификации данных запроса/ответа (не только read-only).
- RQ-006 При отсутствии middleware pipeline работает идентично behaviour до введения фичи.

## Вне scope

- Конкретные реализации middleware (логирование, PII, guardrails, rate limiting) — это отдельные features, которые могут использовать middleware-chain, но не входят в эту spec.
- Замена или удаление интерфейса `Hooks` — Hooks остаются как параллельный механизм наблюдаемости.
- Замена или удаление `PIIDetector` — PIIDetector может быть мигрирован на middleware отдельно.
- Динамическая (runtime) переконфигурация middleware-цепочки — middleware фиксируется при создании pipeline.
- Скоуп middleware на отдельные методы (например, «только для Index») — middleware применяется ко всем pipeline-операциям без возможности выборочного подключения.

## Критерии приемки

### AC-001 Middleware цепочка выполняется в порядке объявления

- Почему это важно: предсказуемый порядок предотвращает race condition между фильтрами и гарантирует, что PII-фильтр выполнится до guardrail, если разработчик указал их в таком порядке.
- **Given** pipeline с тремя middleware: `A`, `B`, `C`, каждая из которых пишет уникальную метку в список вызовов
- **When** вызывается `pipeline.Index(ctx, docs)`
- **Then** результирующий порядок меток в списке — `["A", "B", "C"]`
- Evidence: unit-тест с collect-списком в middleware проверяет порядок для Index, Query, Retrieve, Answer.

### AC-002 Ошибка в middleware прерывает pipeline

- Почему это важно: guardrail не должен пропускать запрещённый контент; нарушение политики должно немедленно останавливать обработку.
- **Given** pipeline с middleware, возвращающей sentinel-ошибку `ErrGuardrailRejected` для любого запроса
- **When** вызывается `pipeline.Answer(ctx, query)`
- **Then** результат — ошибка `ErrGuardrailRejected`; ни одна downstream-стадия (search, generate) не была вызвана
- Evidence: mock-проверка (spy на store.Search и llm.Generate) подтверждает, что они не вызывались после ошибки middleware.

### AC-003 Middleware вызывается на всех стадиях pipeline

- Почему это важно: сквозные concern'ы (логирование, PII) должны применяться единообразно ко всем операциям, иначе появляются дыры в безопасности/наблюдаемости.
- **Given** pipeline с middleware, которая инкрементирует счётчик по имени стадии
- **When** для `Answer` вызывается `pipeline.Answer(ctx, query)`
- **Then** счётчик содержит вызовы для стадий: `pre_search`, `post_search`, `pre_generate`, `post_generate`
- Evidence: unit-тест с подсчётом вызовов для каждой операции (Index, Query, Retrieve, Answer, AnswerStream) подтверждает, что middleware вызывается на всех соответствующих HookStage-стадиях

### AC-004 Middleway модифицирует входящие данные

- Почему это важно: PII-фильтр должен заменять персональные данные перед отправкой в LLM; без модификации middleware бесполезна для content-aware filtering.
- **Given** pipeline с middleware, заменяющей текст вопроса с `"original"` на `"redacted"` на стадии pre-generate
- **When** вызывается `pipeline.Answer(ctx, "original text")`
- **Then** ответ LLM сформирован на основе `"redacted text"`, а не оригинального запроса
- Evidence: mock LLMProvider с записью полученного userMessage подтверждает модифицированный запрос.

### AC-005 Pipeline без middleware идентичен исходному

- Почему это важно: миграция/внедрение не должны ломать существующие системы; обратная совместимость — обязательное условие.
- **Given** pipeline без middleware (опция не задана)
- **When** выполняются все публичные методы (Index, Query, Retrieve, Answer, AnswerStream)
- **Then** поведение и результаты идентичны pipeline той же конфигурации до введения middleware-chain
- Evidence: identity-тест сравнивает вывод pipeline без middleware с эталонным behaviour (те же моки, те же входные данные).

## Допущения

- Middleware достаточно одного интерфейсного метода `Handle` — разделения на OnRequest/OnResponse не требуется; при необходимости middleware сама вызывает next.
- Порядок middleware статичен и задаётся только при конструировании pipeline — runtime-изменения не нужны.
- Все публичные методы pipeline (Index/Query/Retrieve/Answer/AnswerStream) проходят через общий механизм dispatch, куда middleware и встраивается.
- Контекст (`context.Context`) пробрасывается во все вызовы middleware, middleware может вернуть новый контекст (например, с values).

## Критерии успеха

- SC-001 Нагрузочный тест: pipeline с 3 no-op middleware добавляет не более 5% к latency по сравнению с pipeline без middleware.
- SC-002 Покрытие unit-тестами middleware-механизма > 85%.

## Краевые случаи

- Пустой срез middleware (nil или len==0) — pipeline работает без middleware (идентично AC-005).
- Middleware, не вызывающая next (short-circuit) — последующие middleware не выполняются.
- Middleware, возвращающая контекст с отменой — pipeline должен корректно завершиться с context.Canceled.
- Middleware, модифицирующая данные, влияет только на текущий вызов, не на последующие вызовы pipeline.
- Паника в middleware: pipeline должен восстанавливаться через panic recovery и возвращать ошибку.

## Открытые вопросы

- ~~Какие именно стадии (HookStage) должны иметь middleware-точки: все существующие (chunking, embed, search, generate) или только search+generate как границы внешних вызовов?~~ **Решено: все HookStage.**
- ~~Должна ли middleware получать raw-данные (domain.Document, domain.Chunk) или универсальный тип-обёртку?~~ **Решено: типизированные данные + next handler (шаблон net/http).**
- ~~Как быть со streaming-стадиями (AnswerStream): middleware применяется до начала стрима, к каждому токену или только к финальному результату?~~ **Решено: pre-stream + обёртка канала.**
- Нужен ли механизм guaranteed middleware (always-run, даже при ошибке предыдущей middleware)?
