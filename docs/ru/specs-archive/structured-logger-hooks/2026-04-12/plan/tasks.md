# Structured logger hooks (замена log.Printf) Задачи

## Phase Contract

Inputs: `.speckeep/specs/structured-logger-hooks/plan/plan.md`, `.speckeep/specs/structured-logger-hooks/plan/data-model.md`, `.speckeep/specs/structured-logger-hooks/spec.md`.
Outputs: упорядоченные исполнимые задачи с покрытием критериев `AC-*`.
Stop if: хотя бы один `AC-*` нельзя сопоставить с выполнимой задачей.

## Surface Map

| Surface | Tasks |
|---------|-------|
| domain.Logger | `T1.1, T1.2` |
| log.Printf | `T2.1` |
| logger.Log | `T2.1, T2.4` |
| internal/domain/logger.go | `T1.1` |
| pkg/draftrag/draftrag.go | `T1.2` |
| internal/infrastructure/embedder/cache/cache.go | `T2.1` |
| internal/infrastructure/embedder/cache/options.go | `T2.2` |
| pkg/draftrag/cached_embedder.go | `T2.3` |
| internal/infrastructure/resilience/embedder.go | `T2.4` |
| internal/infrastructure/resilience/llm.go | `T2.4` |
| pkg/draftrag/resilience.go | `T2.5` |
| internal/infrastructure/embedder/cache/redis_test.go | `T3.1` |
| internal/infrastructure/resilience/embedder_test.go | `T3.2` |
| internal/infrastructure/resilience/llm_test.go | `T3.2` |
| README.md | `T3.3` |
| docs/embedders.md | `T3.3` |

## Фаза 1: Основа (интерфейс логгера)

Цель: определить минимальный интерфейс логгера и безопасный способ вызова (recover), чтобы implement не “угадывал” форму.

- [x] T1.1 Добавить `domain.Logger`/`LogLevel`/`LogField` и safe wrapper (recover, nil-check). Touches: internal/domain/logger.go (AC-004)
- [x] T1.2 Переэкспортировать публичные типы логгера в `pkg/draftrag`. Touches: pkg/draftrag/draftrag.go (AC-001)

## Фаза 2: Интеграция в компоненты

Цель: убрать прямые `log.Printf` и добавить структурированные события в кэш и retry/CB.

- [x] T2.1 Заменить `log.Printf` в `EmbedderCache` на safe logger вызовы с полями (`component`, `operation`, `err`, `key_prefix?`). Touches: internal/infrastructure/embedder/cache/cache.go (AC-001, AC-002)
- [x] T2.2 Добавить internal опцию `WithLogger(...)` и хранение логгера в `EmbedderCache` (no-op при nil). Touches: internal/infrastructure/embedder/cache/options.go (AC-001)
- [x] T2.3 Добавить логгер в публичные опции `CachedEmbedder` и пробросить в internal cache. Touches: pkg/draftrag/cached_embedder.go (AC-001, AC-002)
- [x] T2.4 Добавить логгер в internal resilience и логировать retry attempt/CB rejection структурированно (best-effort). Touches: internal/infrastructure/resilience/embedder.go, internal/infrastructure/resilience/llm.go (AC-003, AC-004)
- [x] T2.5 Добавить логгер в публичные `RetryOptions` и пробросить в internal resilience. Touches: pkg/draftrag/resilience.go (AC-001, AC-003)

## Фаза 3: Доказательства и документация

Цель: подтвердить AC тестами и показать пользователю, как подключить логгер.

- [x] T3.1 Добавить unit-тесты логирования для Redis деградации/битых данных (fake logger + проверка полей). Touches: internal/infrastructure/embedder/cache/redis_test.go (AC-002, AC-004)
- [x] T3.2 Добавить unit-тесты логирования для retry/CB (fake logger; retry attempt; rejection) и safety при panic логгера. Touches: internal/infrastructure/resilience/embedder_test.go, internal/infrastructure/resilience/llm_test.go (AC-003, AC-004)
- [x] T3.3 Обновить docs с минимальным примером подключения (например, адаптер под `log/slog`). Touches: README.md, docs/embedders.md (AC-005)

## Покрытие критериев приемки

- AC-001 -> T1.2, T2.1, T2.2, T2.3, T2.5
- AC-002 -> T2.1, T2.3, T3.1
- AC-003 -> T2.4, T2.5, T3.2
- AC-004 -> T1.1, T2.1, T2.4, T3.1, T3.2
- AC-005 -> T3.3
