# Публичные production-ready примеры в README — План

## Phase Contract

Inputs: `.speckeep/specs/public-examples/spec.md`, `.speckeep/specs/public-examples/inspect.md` и минимальный контекст текущего `README.md`.
Outputs: `.speckeep/specs/public-examples/plan/plan.md`, `.speckeep/specs/public-examples/plan/data-model.md`.
Stop if: невозможно выбрать конкретные API-символы для примеров без расширения scope.

## Цель

Добавить в `README.md` новый “production-ready” подраздел с двумя короткими end-to-end примерами (pgvector и Qdrant), которые демонстрируют рекомендуемое wiring пайплайна (store + embedder + кеш + retry/CB) и конкретные таймауты через `context.WithTimeout`, оставаясь копипастабельными и не меняя публичный API библиотеки.

## Scope

- Обновление только `README.md`: добавление 1–2 code-block примеров + краткие пояснения по таймаутам/ретраям/кешу.
- Использование только существующих экспортируемых API из `pkg/draftrag`.
- Явная фиксация конкретных значений таймаутов и minimal retry/CB конфигурации в примерах.

## Implementation Surfaces

- `README.md` (существующая поверхность): добавить подраздел “Production-ready” и 2 code-block’а.
- `pkg/draftrag/*` и `examples/*`: НЕ изменяются; используются только как источник существующих публичных символов и ориентир по корректному wiring.

## Влияние на архитектуру

- Архитектура и runtime поведение библиотеки не меняются; меняется только документация.
- Compatibility/rollout: нет миграций, флагов и breaking changes.

## Acceptance Approach

- AC-001 -> добавить в `README.md` pgvector пример на `NewPGVectorStoreWithOptions` + `NewCachedEmbedder` + `NewRetryEmbedder`/`NewRetryLLMProvider` + `NewPipelineWithOptions`; observable proof: наличие code-block и используемых экспортируемых символов.
- AC-002 -> добавить в `README.md` Qdrant пример на `NewQdrantStore` (+ при необходимости `CollectionExists`/`CreateCollection`) + те же кеш/ретраи/таймауты; observable proof: наличие второго code-block и корректный wiring.
- AC-003 -> в обоих примерах задать конкретные таймауты и корректный паттерн `defer cancel()` для (a) индексации и (b) запроса/ответа; observable proof: `context.WithTimeout` присутствует и значения указаны числами.

## Данные и контракты

- Data model: не изменяется (документационная фича).
- API/event contracts: не вводятся и не меняются.

## Стратегия реализации

- DEC-001 Два примера: pgvector + Qdrant
  Why: покрывает два production store, сохраняя размер README разумным и демонстрируя переносимый wiring-паттерн.
  Tradeoff: README растёт; нужно удерживать примеры короткими и устойчивыми к API-изменениям.
  Affects: `README.md`.
  Validation: reviewer видит 2 отдельных code-block’а, каждый использует публичный API и соответствует AC-001/AC-002.

- DEC-002 Конкретные “рекомендуемые” таймауты задаются в примерах
  Why: снимает warning из inspect и делает “production-ready” недвусмысленным.
  Tradeoff: значения могут быть неидеальны для всех; в README дать краткую оговорку “стартовые ориентиры”.
  Affects: `README.md`.
  Validation: в коде есть `indexCtx` и `queryCtx` с `context.WithTimeout` и числами рядом.

- DEC-003 Redis L2 в README показываем как опциональную вставку (коротко)
  Why: `NewCachedEmbedder` уже поддерживает Redis через интерфейс `RedisCacheClient`; короткий snippet закрывает RQ-004 без привязки к конкретной Redis-библиотеке.
  Tradeoff: пример становится чуть длиннее; нужно удержать snippet минимальным (1–3 строки) и без “внедрения” конкретного клиента.
  Affects: `README.md`.
  Validation: в README есть небольшой фрагмент, показывающий `CacheOptions.Redis` и `TTL/KeyPrefix`, без дополнительных зависимостей.

## Incremental Delivery

### MVP (Первая ценность)

- Добавить подраздел “Production-ready” и pgvector пример (AC-001 + AC-003).
- Включить кеширование и ретраи/CB (RQ-003/RQ-004).
- Критерий готовности MVP: пример читается как end-to-end, содержит конкретные таймауты и ключевые обёртки.

### Итеративное расширение

- Добавить Qdrant пример (AC-002 + AC-003).
- Добавить короткий optional snippet для Redis L2 (DEC-003 / RQ-004).

## Порядок реализации

- Сначала: выбрать финальные экспортируемые символы и собрать pgvector пример вокруг них (минимизирует риск “не компилится”).
- Затем: добавить Qdrant пример (используя те же embedder/LLM/resilience/timeout паттерны).
- В конце: выровнять текстовые пояснения и убедиться, что примеры не дублируют “Быстрый старт”.

## Риски

- Риск: примеры расходятся с актуальным API (ошибочные имена типов/опций).
  Mitigation: перед final review проверить сигнатуры публичных конструкторов и собрать примеры только из экспортируемых символов.
- Риск: “production-ready” воспринимается как гарантия SLO.
  Mitigation: добавить короткую оговорку, что таймауты/ретраи — стартовые ориентиры и должны калиброваться под окружение.

## Rollout и compatibility

- Специальные rollout-действия не требуются: документационное изменение.

## Проверка

- Manual review: сверить, что оба code-block’а соответствуют AC-001..AC-003 (наличие wiring, таймаутов, кеша, ретраев/CB).
- Быстрый sanity-check репозитория: `go test ./...` (подтверждает, что упоминаемые символы реально существуют и проект в зелени).

## Соответствие конституции

- Контекстная безопасность: примеры используют `context.WithTimeout` и передают `ctx` в операции (`Index`, `Retrieve/Answer/Cite`) согласно принципу “context первым параметром”.
- Минимальная конфигурация: примеры показывают безопасные defaults (и только минимально необходимые опции), без усложнения wiring.
- Интерфейсная абстракция: используются публичные интерфейсы `VectorStore`/`Embedder`/`LLMProvider` и их фабрики без жесткой привязки к инфраструктурной реализации.
- Документация на русском: новый раздел `README.md` и пояснения — на русском.

