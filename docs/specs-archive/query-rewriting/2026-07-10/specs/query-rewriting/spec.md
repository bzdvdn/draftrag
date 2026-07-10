# Query Rewriting — плагируемый рерайтер запросов

## Scope Snapshot

- In scope: добавить интерфейс `QueryRewriter` в domain и интегрировать его в pipeline: плагируемое переписывание запроса перед retrieval с поддержкой 1:1 и 1:N режимов, multi-turn контекста и встроенных стратегий HyDE / MultiQuery.
- Out of scope: рекурсивный chain-of-thought rewriting, routing между несколькими rewritter'ами, персистентное хранение истории диалога.

## Цель

Разработчик RAG-системы получает возможность подключать кастомные стратегии переписывания запросов (LLM-based, rule-based, гибридные) через единый интерфейс, без модификации pipeline. Встроенные стратегии (HyDE, MultiQuery) становятся реализациями этого интерфейса; multi-turn контекст (история предыдущих сообщений) передаётся в rewriter для разрешения ко-референтности. Успех фичи измеряется тем, что новый `SearchBuilder` метод `.Rewriter(r)` заменяет `.HyDE()` / `.MultiQuery(n)`, а реализация кастомного rewriter'а не требует копирования внутреннего кода pipeline.

## Основной сценарий

1. Пользователь создаёт структуру, реализующую `domain.QueryRewriter`, и передаёт её в `PipelineOptions.QueryRewriter` (или в новый метод `SearchBuilder.Rewriter(r)`).
2. При вызове `SearchBuilder.{Retrieve,Answer,Stream,...}`:
   - pipeline вызывает `rewriter.Rewrite(ctx, query, history)`.
   - Rewriter возвращает `[]RewrittenQuery` (одну или несколько переформулировок).
   - Для каждой переформулировки pipeline выполняет embedding + retrieval.
   - При 1:N результаты объединяются через RRF (как в текущем MultiQuery).
3. Если rewriter не установлен — pipeline работает как сегодня (без переписывания).
4. При ошибке в `Rewriter.Rewrite` pipeline логирует и использует исходный запрос (fallback, не fatal).

## User Stories

none — brownfield feature, достаточно scope-секций.

## MVP Slice

- Интерфейс `domain.QueryRewriter` с методом `Rewrite(ctx, query, history) -> ([]RewrittenQuery, error)`.
- Интеграция в `PipelineOptions` и `SearchBuilder` (метод `.Rewriter(r)`).
- Встроенная стратегия `LLMRewriter` (замена HyDE): LLM генерирует одну или N переформулировок по prompt.
- Multi-turn: `QueryHistory` структура (список пар user/assistant) передаётся в `Rewrite`.
- AC-001, AC-002, AC-003, AC-004, AC-005.

## First Deployable Outcome

После первого implementation pass можно:
- Создать кастомный rewriter, реализующий `domain.QueryRewriter`.
- Подключить его через `pipeline.Search("...").Rewriter(myRewriter).Retrieve(ctx)`.
- Убедиться, что переписанный запрос (или несколько переформулировок) влияют на retrieval.
- Использовать `QueryHistory` для multi-turn сценариев.

## Scope

- Интерфейс `domain.QueryRewriter` в `internal/domain/interfaces.go`.
- Модель `RewrittenQuery` (query string + optional score/weight) в `internal/domain/models.go`.
- Модель `QueryHistory` (список пар user/assistant) в `internal/domain/models.go`.
- Встроенная `LLMRewriter` реализация в `internal/infrastructure/rewriter/llm_rewriter.go`.
- Интеграция в `PipelineOptions` (поле `QueryRewriter`) и `SearchBuilder` (метод `.Rewriter(r)`).
- Изменение routing в `search_routing.go`: новый маршрут `routeRewriter`, приоритет выше HyDE/MultiQuery; при установленном `Rewriter` — HyDE/MultiQuery через `SearchBuilder` не используются (предупреждение в документации).
- Все output-типы (Retrieve/Answer/Cite/InlineCite/Stream/StreamSources/StreamCite) поддерживают новый маршрут.

## Контекст

- `SearchBuilder` уже имеет зашитые HyDE и MultiQuery как built-in флаги, без общего интерфейса.
- pipeline.Query уже умеет multi-query fusion (RRF) — новый rewriter 1:N переиспользует эту логику.
- История диалога (QueryHistory) передаётся только в Rewrite, не хранится в pipeline — caller отвечает за её сбор и поддержание.
- Никаких изменений в VectorStore, Embedder, LLMProvider интерфейсы не вносится.

## Зависимости

- `LLMProvider` (уже есть) — для встроенной `LLMRewriter`.
- Переиспользование RRF-логики из `internal/application/retrieval.go` (или `internal/application/`).
- Нет внешних зависимостей.

## Требования

- RQ-001 Система ДОЛЖНА предоставлять интерфейс `QueryRewriter` с методом `Rewrite(ctx, query, history) ([]RewrittenQuery, error)` в domain.
- RQ-002 Система ДОЛЖНА позволять подключать реализацию `QueryRewriter` через `PipelineOptions.QueryRewriter` и через `SearchBuilder.Rewriter(r)`.
- RQ-003 Система ДОЛЖНА поддерживать два режима: 1:1 (один переписанный запрос) и 1:N (несколько переформулировок); при 1:N результаты fusion через RRF.
- RQ-004 Система ДОЛЖНА передавать `QueryHistory` (список предыдущих пар user/assistant) в метод `Rewrite` для multi-turn контекста.
- RQ-005 Встроенная `LLMRewriter` ДОЛЖНА принимать prompt-шаблон для кастомизации и через LLMProvider генерировать переформулировки.
- RQ-006 При ошибке `Rewriter.Rewrite` pipeline НЕ ДОЛЖЕН прерывать операцию: исходный запрос используется как fallback с логированием.
- RQ-007 Если в `SearchBuilder` установлен `Rewriter`, флаги `.HyDE()` и `.MultiQuery(n)` игнорируются (предупреждение через hooks или логирование).

## Вне scope

- Хранение истории диалога вне pipeline — caller управляет `QueryHistory`.
- Выбор/роутинг между несколькими rewriter'ами (chain, conditional routing).
- Персистентный кэш переформулировок.
- Рекурсивное/итеративное переписывание (chain-of-thought).
- Изменение интерфейсов VectorStore, Embedder, LLMProvider для поддержки rewriting.
- Web/CLI интерфейс для отладки rewriting.

## Критерии приемки

### AC-001 QueryRewriter interface в domain

- Почему это важно: единый контракт для всех стратегий переписывания.
- **Given** clean codebase
- **When** компиляция пакета `internal/domain`
- **Then** существует экспортируемый (на уровне пакета) интерфейс `QueryRewriter` с методом `Rewrite(ctx, query, history) ([]RewrittenQuery, error)`
- Evidence: `go build ./...` проходит; type-чекер может присвоить кастомную структуру переменной типа `domain.QueryRewriter`.

### AC-002 Подключение через PipelineOptions и SearchBuilder

- Почему это важно: API-гибкость — rewriter можно задать на уровне pipeline или per-request.
- **Given** pipeline, созданный с `PipelineOptions{QueryRewriter: myRewriter}` или без него
- **When** вызывается `pipeline.Search("q").Rewriter(myRewriter2).Retrieve(ctx)`
- **Then** per-request `Rewriter` имеет приоритет над pipeline-level
- Evidence: в тесте при разных rewriter'ах возвращаются разные retrieval-результаты для одного исходного запроса.

### AC-003 1:N multi-query fusion

- Почему это важно: поддержка multi-query улучшает recall.
- **Given** `Rewriter.Rewrite` возвращает 3 переформулировки
- **When** вызывается `SearchBuilder.Rewriter(r).Retrieve(ctx)`
- **Then** результаты retrieval для каждой переформулировки объединяются через RRF (как в текущем MultiQuery)
- Evidence: retrieval содержит чанки, найденные по разным переформулировкам; дубликаты удалены.

### AC-004 Multi-turn: QueryHistory передаётся в Rewrite

- Почему это важно: для диалоговых сценариев запрос "а что насчёт второго?" должен разрешаться в контексте истории.
- **Given** `QueryHistory{Entries: []{User: "как работает RAG?", Assistant: "RAG это..."}}`
- **When** вызывается `Rewriter.Rewrite(ctx, "а какие минусы?", history)`
- **Then** rewriter получает history и может сгенерировать самодостаточный запрос "какие минусы у RAG?"
- Evidence: в тесте с `LLMRewriter` и соответствующим prompt переформулировка включает контекст из history.

### AC-005 Fallback при ошибке Rewriter

- Почему это важно: отказ rewriter'а не должен блокировать поиск.
- **Given** rewriter, который всегда возвращает ошибку
- **When** вызывается `SearchBuilder.Rewriter(r).Retrieve(ctx)`
- **Then** pipeline логирует ошибку и выполняет retrieval с исходным (непереписанным) запросом
- Evidence: retrieval успешно возвращает результаты; лог содержит сообщение об ошибке rewriter'а.

### AC-007 Rewriter отключает HyDE/MultiQuery флаги

- Почему это важно: избежать двойного переписывания и неоднозначного поведения.
- **Given** `SearchBuilder` с установленным `Rewriter` и включёнными `.HyDE()` или `.MultiQuery(3)`
- **When** вызывается `Retrieve(ctx)`
- **Then** pipeline игнорирует флаги HyDE/MultiQuery, использует только установленный `Rewriter`, и эмитит предупреждение через hooks (или лог)
- Evidence: при `Hooks.OnEnd` проверяется, что stage имеет warning о конфликте; retrieval-результат соответствует только стратегии из `Rewriter`.

### AC-006 LLMRewriter базовая реализация

- Почему это важно: из коробки working solution для наиболее частого случая.
- **Given** `LLMRewriter`, созданный через `NewLLMRewriter(llm, defaultPrompt)`
- **When** вызывается `Rewrite(ctx, "query", nil)`
- **Then** LLM генерирует одну или несколько переформулировок согласно prompt-шаблону
- Evidence: возвращённый `[]RewrittenQuery` содержит непустые переформулировки, отличные от исходного запроса.

## Допущения

- Caller управляет жизненным циклом `QueryHistory` и его размером (pipeline не усекает историю).
- При 1:N все переформулировки имеют равный вес при fusion (как в текущем MultiQuery).
- Встроенный `LLMRewriter` использует `LLMProvider` для генерации; стоимость LLM-вызова ложится на пользователя.
- При установленном кастомном `Rewriter` встроенные HyDE и MultiQuery через `SearchBuilder` отключаются — пользователь сам решает, какую стратегию реализовать.

## Критерии успеха

- SC-001 Переключение с `SearchBuilder.HyDE()` на `SearchBuilder.Rewriter(myRewriter)` не требует изменений в логике приложения, кроме замены одной строки.
- SC-002 Добавление нового rewriter'а не требует модификации pipeline или SearchBuilder — только реализация `domain.QueryRewriter`.

## Краевые случаи

- `Rewriter` возвращает пустой `[]RewrittenQuery{}` — pipeline использует исходный запрос.
- `Rewriter` возвращает `nil` без ошибки — pipeline использует исходный запрос.
- `QueryHistory` пуст — rewriter обрабатывает запрос без контекста (одношаговый режим).
- `LLMRewriter` с пустым prompt — возвращает ошибку при валидации.
- Overlap между кастомным rewriter и флагами `.HyDE()` / `.MultiQuery(n)` — флаги игнорируются, emitted warning через hooks.

## Открытые вопросы

- Нужна ли возможность указывать weight для каждого `RewrittenQuery` при fusion? Пока нет — равные веса.
- Должен ли `LLMRewriter` поддерживать streaming-генерацию переформулировок? Пока нет — синхронный вызов.
- Какова сигнатура `QueryHistory` — гибкий `[]Message{User, Assistant}` или строгая структура? Выбран гибкий вариант: `[]struct{User, Content string}`.
