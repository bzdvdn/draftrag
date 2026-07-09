# Cost Tracking — Счётчик токенов и стоимости LLM-вызовов

## Scope Snapshot

- In scope: сквозной подсчёт токенов (prompt + completion) и их стоимости в $ для каждого LLM-вызова в pipeline.
- Out of scope: трекинг стоимости embedder-вызовов, постоянное хранение статистики между рестартами приложения, лимиты/бюджеты.

## Цель

Разработчик, использующий draftRAG, не видит финансовых последствий LLM-вызовов — нет инструментария для оценки затрат на инференс. Без этого невозможно контролировать бюджет и выявлять аномальный расход. Фича даёт прозрачный счётчик токенов и стоимости, встроенный в pipeline, с возможностью читать накопленную статистику и сбрасывать её между сессиями.

## Основной сценарий

1. Пользователь создаёт pipeline с опциональным `CostTracker`.
2. При каждом LLM-вызове (`Answer*`, `QueryHyDE`, `QueryMulti` — везде, где вызывается `LLMProvider.Generate`) автоматически фиксируются prompt_tokens, completion_tokens и модель.
3. Стоимость вычисляется по предварительно настроенным ценам за 1K токенов для используемой модели.
4. Пользователь в любой момент может получить снапшот накопленной статистики (total_tokens, total_cost, количество вызовов) через публичный API.
5. При ошибке LLM-вызова токены не засчитываются (stateless: вызов считается успешным только при `err == nil`).

## User Stories

- P1: Как разработчик, я хочу видеть, сколько токенов и денег потрачено на LLM-вызовы за сессию или произвольный период (например, за последние N запросов или между двумя метками времени), чтобы оценить стоимость эксплуатации.
- P2: Как разработчик, я хочу указывать цены за модель (input cost + output cost per 1K tokens), чтобы расчёт соответствовал моему тарифу.

## MVP Slice

Обёртка `CostTracker`, реализующая `LLMProvider`, которая:
- перехватывает `Generate`, получает token usage через optional capability;
- накапливает статистику;
- предоставляет `Snapshot() (CostSnapshot, error)`, `Reset()` и `Checkpoint()`;
- экспортирует функцию `Diff(prev, curr CostSnapshot) CostSnapshot` для расчёта дельты между двумя точками.
- AC-001, AC-003, AC-004, AC-007

## First Deployable Outcome

Один пример в `examples/`, где pipeline обёрнут в CostTracker, после `Answer*` печатается снапшот. Можно запустить и увидеть в stdout токены + $.

## Scope

- Реализация провайдеров, возвращающих token usage (optional capability `UsageAwareLLMProvider` / изменение `LLMProvider.Generate`).
- Wrapper `CostTracker`, реализующий `LLMProvider`, накапливающий статистику.
- Модель ценообразования: настраиваемая таблица `map[string]ModelPricing` с ценами за input и output 1K токенов.
- Публичный API: `CostTracker.Snapshot()`, `CostTracker.Reset()`, `CostTracker.Checkpoint()`, и свободная функция `CostTracker.Diff(prev, curr) CostSnapshot`.
- Интеграция с существующими concrete provider'ами (OpenAI-compatible, Anthropic, Mistral, DeepSeek, Ollama — если API возвращает usage).
- Поддержка streaming: отдельный подсчёт для `StreamingLLMProvider.GenerateStream`.
- Trace-маркеры: `@sk-task cost-tracking`.

## Контекст

- Все LLM-вызовы проходят через `domain.LLMProvider.Generate` — единственная точка перехвата.
- Большинство коммерческих API (OpenAI, Anthropic, Mistral, DeepSeek) возвращают `usage` в JSON-ответе; Ollama — не всегда.
- `StreamingLLMProvider` — optional capability, stream не даёт суммарного usage до закрытия; часть API присылает usage в финальном chunk.
- Существующий hooks-механизм (`domain.Hooks`) не передаёт данные о токенах — недостаточен для cost tracking без расширения.

## Зависимости

- `LLMProvider` / `StreamingLLMProvider` — единственные точки расширения.
- Для каждого concrete провайдера нужно прочитать usage из API-ответа (изменение в `internal/infrastructure/llm/`).
- `none` внешних зависимостей (чисто Go).

## Требования

- RQ-001 Система ДОЛЖНА подсчитывать prompt_tokens, completion_tokens и total_tokens для каждого успешного LLM-вызова.
- RQ-002 Система ДОЛЖНА вычислять стоимость вызова в USD на основе настраиваемых цен за input и output 1K токенов.
- RQ-003 Система ДОЛЖНА предоставлять атомарный снапшот накопленной статистики: total_tokens, total_cost, calls_count. Значения — от создания CostTracker или последнего Reset (absolute).
- RQ-007 Система ДОЛЖНА предоставлять возможность зафиксировать checkpoint (текущий absolute снапшот) и вычислить дельту между двумя checkpoint'ами для анализа стоимости за произвольный период.
- RQ-004 Система ДОЛЖНА поддерживать сброс статистики (reset) без создания нового pipeline.
- RQ-005 Система ДОЛЖНА корректно обрабатывать API, не возвращающие usage (Ollama): пропускать cost-расчёт, токены не считать — только инкремент calls_count. Поведение документируется.
- RQ-006 Для streaming-режима token usage ДОЛЖЕН извлекаться из финального chunk ответа (если API его присылает) либо не учитываться с документированным ограничением.

## Вне scope

- Трекинг embedder-вызовов (только LLM).
- Персистентное хранение статистики (БД, файл).
- Автоматические бюджетные лимиты и алерты.
- UI/дашборд — только программный API.
- Расчёт стоимости для ошибок и partial failures.

## Критерии приемки

### AC-001 Базовый подсчёт токенов через CostTracker

- Почему это важно: пользователь должен видеть токены, потраченные на генерацию.
- **Given** pipeline, обёрнутый в `CostTracker`, и настроенный provider, возвращающий token usage
- **When** выполнен `Answer(ctx, question)`
- **Then** `CostTracker.Snapshot().TotalTokens > 0`, `Snapshot().CallsCount == 1`
- Evidence: unit-тест с mock LLM, возвращающим фиксированный usage.

### AC-002 Расчёт стоимости по модели

- Почему это важно: пользователь должен видеть не только токены, но и деньги.
- **Given** `CostTracker` с `ModelPricing{InputCostPer1K: 0.01, OutputCostPer1K: 0.03}` для модели `gpt-4o`
- **When** `Generate` возвращает `{prompt_tokens: 100, completion_tokens: 50}`
- **Then** `Snapshot().TotalCost == 0.01*(100/1000) + 0.03*(50/1000) == 0.0025`
- Evidence: unit-тест с верификацией расчёта.

### AC-003 Снапшот потокобезопасен

- Почему это важно: pipeline может вызываться конкурентно, гонки недопустимы.
- **Given** `CostTracker` используется в многопоточной среде
- **When** несколько goroutine одновременно вызывают `Generate` и `Snapshot()`
- **Then** данные консистентны, нет `data race`
- Evidence: `go test -race` не детектит race condition.

### AC-004 Сброс статистики

- Почему это важно: возможность начать "с чистого листа" для новой сессии.
- **Given** `CostTracker` с накопленной статистикой > 0
- **When** вызван `Reset()`
- **Then** `Snapshot().TotalTokens == 0`, `Snapshot().TotalCost == 0`, `Snapshot().CallsCount == 0`
- Evidence: unit-тест.

### AC-005 Streaming usage (опционально)

- Почему это важно: streaming-режим — распространённый сценарий, затраты в нём не должны быть невидимыми.
- **Given** `CostTracker` и `StreamingLLMProvider`, чей финальный chunk содержит usage
- **When** вызван `GenerateStream` и канал полностью прочитан
- **Then** `Snapshot()` включает токены streaming-вызова
- Evidence: unit-тест с mock streaming provider.

### AC-007 Расчёт стоимости за период через checkpoint

- Почему это важно: пользователь может замерить стоимость отдельного запроса или блока работы, не сбрасывая весь счётчик.
- **Given** `CostTracker` с накопленной статистикой от нескольких вызовов
- **When** взят `checkpoint1 = Checkpoint()`, сделан ещё один `Generate`, взят `checkpoint2 = Checkpoint()`
- **Then** `Diff(checkpoint1, checkpoint2).TotalCost` равен стоимости одного последнего вызова, а `Diff(checkpoint1, checkpoint2).CallsCount == 1`
- Evidence: unit-тест.

### AC-006 API без usage (graceful degradation)

- Почему это важно: Ollama и некоторые локальные модели не возвращают usage.
- **Given** `CostTracker`, оборачивающий provider, не возвращающий usage через optional capability
- **When** выполнен `Generate`
- **Then** вызов засчитывается как 1 вызов (calls_count++), но `TotalTokens` и `TotalCost` не увеличиваются (явно документировано)
- Evidence: unit-тест с provider, не реализующим optional capability.

## Допущения

- Цены в `ModelPricing` задаются за 1K токенов (input и output раздельно) — де-факто стандарт индустрии.
- Провайдеры OpenAI-compatible возвращают `usage` в теле ответа — доминирующий случай.
- Ollama не возвращает usage — fallback на calls_count-only.
- Stream-режим: финальный chunk может содержать usage; если нет — вызов не тарифицируется (calls_count++).
- `CostTracker` не меняет поведение LLM — прозрачная обёртка.
- Периоды считаются через разницу двух checkpoint'ов: `Diff(cp1, cp2)` — свободная функция; хранения истории внутри CostTracker не требуется.

## Критерии успеха

- SC-001 Нулевой оверхед на benchmark: CostTracker не добавляет >1% latency к Generate (только атомарный инкремент счётчиков).
- SC-002 `go test -race` на всём пакете: 0 races.
- SC-003 Покрытие новых строк кода: >= 80% (определяется `go test -cover`).

## Краевые случаи

- Provider возвращает usage с нулевыми токенами — не засчитывать (calls_count++, cost=0).
- `Generate` возвращает ошибку — вызов не учитывать ни в одном счётчике.
- Concurrent `Generate` + `Reset` — сброс не теряет конкурентно завершённые вызовы (snapshot до/после).
- Бесконечный стрим без финального usage — вызов не тарифицируется, calls_count++.

## Открытые вопросы

1. Оценка токенов по длине строки для Ollama: нужно ли? — Без оценки, только calls_count; оценка — P2.
2. Как быть с retry (resilience): при retry вызвавшем Generate повторно — каждый вызов считается отдельно? — Да, каждый вызов Generate считается независимо (retry — часть стоимости).
