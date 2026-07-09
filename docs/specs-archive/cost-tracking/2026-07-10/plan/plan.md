# Cost Tracking — План

## Phase Contract

Inputs: spec (`docs/specs/cost-tracking/spec.md`), inspect (`pass`), repo-контекст.
Outputs: plan, data-model.
Stop if: нет.

## Цель

Добавить прозрачный счётчик токенов и стоимости LLM-вызовов. Подход — wrapper `CostTracker`, реализующий `LLMProvider`, плюс optional capability `UsageAwareLLMProvider` для провайдеров, которые могут возвращать token usage. Пользователь оборачивает LLM-провайдер в CostTracker на уровне создания pipeline.

## MVP Slice

- Типы `TokenUsage`, `ModelPricing`, `CostSnapshot` в `domain`.
- Optional capability `UsageAwareLLMProvider` в `domain`.
- Wrapper `CostTracker` в `internal/infrastructure/costtracker/`.
- Реализация `UsageAwareLLMProvider` для `OpenAICompatibleResponsesLLM`.
- Re-export типов в `pkg/draftrag/`.
- AC-001, AC-002, AC-003, AC-004, AC-006, AC-007.

## First Validation Path

Пример в `examples/` с `OpenAICompatibleResponsesLLM`, обёрнутым в `CostTracker`. После `Answer` печатается `Snapshot()`. Запуск `go run` показывает ненулевые токены и стоимость.

## Scope

- `internal/domain/interfaces.go` — новые типы и optional interface.
- `internal/infrastructure/costtracker/costtracker.go` — новый пакет, CostTracker.
- `internal/infrastructure/llm/openai_compatible_responses.go` — парсинг usage из API-ответа + реализация `UsageAwareLLMProvider`.
- `internal/infrastructure/llm/anthropic.go` — парсинг usage + реализация.
- `internal/infrastructure/llm/openai_chat.go` — парсинг usage + реализация.
- `internal/infrastructure/llm/mistral_llm.go` (через `pkg/draftrag/mistral_llm.go`) — парсинг usage.
- `internal/infrastructure/llm/deepseek_llm.go` — парсинг usage.
- `pkg/draftrag/` — re-export новых типов + CostTracker.
- `examples/` — демо-пример.
- `pkg/draftrag/ollama_llm.go` — **не меняется** (Ollama не возвращает usage; только calls_count).
- `internal/infrastructure/llm/ollama.go` — **не меняется** (аналогично).

## Performance Budget

- `none` — только атомарные инкременты int64 и опциональный парсинг usage из уже прочитанного JSON-ответа. Дополнительных HTTP-вызовов нет.

## Implementation Surfaces

| Surface | Статус | Почему |
|---------|--------|--------|
| `internal/domain/interfaces.go` | existing, меняется | Новые типы + optional interface |
| `internal/infrastructure/costtracker/` | **новая** | Изоляция CostTracker; не привязан к pipeline |
| `internal/infrastructure/llm/*.go` | existing, меняется | Парсинг usage из ответа + `UsageAwareLLMProvider` |
| `pkg/draftrag/draftrag.go` | existing, меняется | Re-export типов |
| `pkg/draftrag/costtracker.go` | **новая** | Re-export CostTracker |
| `pkg/draftrag/errors.go` | existing, меняется | Новые sentinel'ы при необходимости |
| `examples/` | existing, дополняется | Демонстрация |

## Bootstrapping Surfaces

- `internal/infrastructure/costtracker/` — создать директорию и `costtracker.go`.

## Влияние на архитектуру

- **Локальное**: новый пакет costtracker, изолированный от pipeline.
- **Интеграции**: ни одна граница системы не меняется; CostTracker — прозрачная обёртка.
- **Compatibility**: `LLMProvider` не меняется; optional interface не ломает существующие реализации. `CostTracker` не добавляется в `PipelineOptions` — пользователь оборачивает LLM до передачи в конструктор.

## Acceptance Approach

### AC-001 Базовый подсчёт токенов

- **Подход**: Mock LLM, реализующий `UsageAwareLLMProvider`, возвращает фиксированный TokenUsage. CostTracker.Snapshot() проверяется после Generate.
- **Surfaces**: costtracker, domain (TokenUsage, UsageAwareLLMProvider).
- **Наблюдение**: unit-тест.

### AC-002 Расчёт стоимости по модели

- **Подход**: CostTracker с `map[string]ModelPricing`. Mock возвращает usage. Проверка Snapshot().TotalCost по формуле.
- **Surfaces**: costtracker, domain (ModelPricing).
- **Наблюдение**: unit-тест.

### AC-003 Снапшот потокобезопасен

- **Подход**: `go test -race` с конкурентными Generate + Snapshot.
- **Surfaces**: costtracker.
- **Наблюдение**: `-race` флаг.

### AC-004 Сброс статистики

- **Подход**: После накопления статистики вызвать Reset(), проверить Snapshot() == zero.
- **Surfaces**: costtracker.
- **Наблюдение**: unit-тест.

### AC-005 Streaming usage

- **Подход**: Mock streaming provider, финальный chunk содержит usage. GenerateStream полностью прочитан → Snapshot() включает токены.
- **Surfaces**: costtracker, domain (StreamingLLMProvider + UsageAwareLLMProvider).
- **Наблюдение**: unit-тест с mock.
- **Отложено до**: итеративного расширения (не в MVP).

### AC-006 API без usage (graceful degradation)

- **Подход**: Обычный LLMProvider без `UsageAwareLLMProvider`. После Generate — calls_count++.
- **Surfaces**: costtracker.
- **Наблюдение**: unit-тест.

### AC-007 Расчёт стоимости за период через checkpoint

- **Подход**: Checkpoint → Generate → Checkpoint → Diff(cp1, cp2).TotalCost == стоимость одного вызова.
- **Surfaces**: costtracker.
- **Наблюдение**: unit-тест.

## Данные и контракты

- **data-model.md** прилагается (изменения типов в domain).
- API-контракты: `UsageAwareLLMProvider` — новый optional interface. `CostTracker` — новый публичный тип. `Generate` на `LLMProvider` не меняется.
- Совместимость: обратная, никакие существующие контракты не ломаются.

## Стратегия реализации

### DEC-001: Optional capability вместо изменения LLMProvider

- **Why**: Не ломать существующие реализации. `StreamingLLMProvider` уже задаёт паттерн.
- **Tradeoff**: Два интерфейса вместо одного; type assertion в CostTracker.
- **Affects**: `domain/interfaces.go`, `costtracker/costtracker.go`.
- **Validation**: `go build ./...` без ошибок.

### DEC-002: CostTracker — отдельный wrapper, не встроен в PipelineOptions

- **Why**: pipeline не должен знать про cost tracking; это ортогональная сквозная забота. Пользователь может использовать CostTracker с любым LLMProvider независимо от pipeline.
- **Tradeoff**: Дополнительный шаг при конфигурации: `CostTracker{LLM: llm, Pricing: prices}`.
- **Affects**: costtracker, пользовательский код.
- **Validation**: Пример в `examples/`.

### DEC-003: Потокобезопасность через sync.Mutex

- **Why**: Три счётчика (tokens_in, tokens_out, calls), атомарно читаемые в Snapshot(). sync.Mutex проще и безопаснее atomic-пакета для трёх связанных полей.
- **Tradeoff**: Небольшой contention при высококонкурентном Generate (но LLM-вызов на 3+ порядка медленнее мьютекса).
- **Affects**: costtracker/costtracker.go.
- **Validation**: `go test -race`.

### DEC-004: Snapshot() — value copy под блокировкой, Checkpoint() — алиас

- **Why**: Snapshot() должен возвращать консистентный срез состояния. Checkpoint() идентичен Snapshot() по смыслу; может быть алиасом или отдельным методом для ясности API.
- **Tradeoff**: Checkpoint() как отдельный метод — семантическая ясность ценой дублирования.
- **Affects**: costtracker/costtracker.go.
- **Validation**: AC-007.

### DEC-005: ModelName — метод на UsageAwareLLMProvider, не отдельный interface

- **Why**: ModelName логически связана с usage; объединение в один interface сокращает type assertions. Fallback — имя модели из конструктора CostTracker.
- **Tradeoff**: Если провайдер не знает модель — пользователь задаёт явно при создании CostTracker.
- **Affects**: domain/interfaces.go, costtracker/costtracker.go.
- **Validation**: Тест с UsageAwareLLMProvider.ModelName().

## Incremental Delivery

### MVP (первая ценность)

Задачи:
1. Типы и optional interface в domain + data-model.
2. CostTracker wrapper (core logic).
3. Реализация UsageAwareLLMProvider для OpenAICompatibleResponsesLLM.
4. Re-export + пример.
5. Unit-тесты на AC-001, AC-002, AC-003, AC-004, AC-006, AC-007.

Критерий: `go test ./internal/infrastructure/costtracker/... -v -race` — зелёный; пример в `examples/` компилируется и запускается.

### Итеративное расширение

- Шаг 2: Реализация UsageAwareLLMProvider для Anthropic, OpenAI Chat, Mistral, DeepSeek + тесты парсинга.
- Шаг 3: Streaming usage (AC-005) — модификация CostTracker для перехвата финального chunk из GenerateStream.
- Шаг 4: Примеры для каждого провайдера (docs).

## Порядок реализации

1. **Типы + optional interface** в domain — без этого ничего не работает.
2. **CostTracker** — ядро; тесты с mock (AC-001..004, 006, 007).
3. **OpenAICompatibleResponsesLLM** — первый реальный провайдер с usage + тест.
4. **Re-export + пример** — проверка API.
5. Остальные провайдеры (параллельно).
6. Streaming (опционально, шаг 3).

## Риски

- **Риск 1**: API провайдера меняет структуру usage-поля.
  - Mitigation: тесты парсинга с замороженными JSON-фикстурами; ошибка парсинга usage → calls_count++ без cost (graceful degradation).
- **Риск 2**: Streaming usage — финальный chunk может не содержать usage у некоторых провайдеров.
  - Mitigation: вызов не тарифицируется, только calls_count++ (по spec). AC-005 deferred.
- **Риск 3**: CostTracker.Snapshot() возвращает устаревшие данные при concurrent Generate + Snapshot.
  - Mitigation: sync.Mutex; Snapshot копирует struct под блокировкой.

## Rollout and compatibility

- Специальных rollout-действий не требуется — новый функционал, не меняет существующее поведение.
- `CostTracker` не ломает сигнатуру `NewPipeline*`.

## Проверка

- `go test ./internal/infrastructure/costtracker/... -v -race` — unit + race.
- `go test ./internal/domain/... -v` — domain типы.
- `go test ./pkg/draftrag/... -v` — re-export.
- `go vet ./...` — статический анализ.
- `go build ./examples/...` — пример компилируется.
- AC-001..004, 006, 007 — покрыты unit-тестами.

## Соответствие конституции

- Нет конфликтов.
