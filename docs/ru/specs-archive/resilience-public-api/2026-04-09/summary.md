# Сводка архива

## Спецификация

- snapshot: экспортированы RetryEmbedder, RetryLLMProvider, CircuitBreakerStats и утилиты в pkg/draftrag
- slug: resilience-public-api
- archived_at: 2026-04-09
- status: completed

## Причина

Внутренняя resilience-логика была недоступна пользователям библиотеки из-за Go `internal/` ограничения. Тонкий фасад в `pkg/draftrag` открывает её без дублирования кода.

## Результат

- `pkg/draftrag/resilience.go` с `RetryEmbedder`, `RetryLLMProvider`, `RetryOptions`.
- Re-export `CircuitState`, `ErrCircuitOpen`, `CircuitBreakerStats`, `IsRetryable`, `WrapRetryable`, `WrapNonRetryable`.
- Документация в `docs/advanced.md` с таблицей RetryOptions и примерами.

## Продолжение

- RetryVectorStore для store-level resilience.
- Интеграция CB stats с Observability Hooks (onCircuitOpen/onCircuitHalfOpen события).
