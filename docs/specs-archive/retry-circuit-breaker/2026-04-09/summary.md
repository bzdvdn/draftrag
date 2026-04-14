---
slug: retry-circuit-breaker
generated_at: 2026-04-09T12:30:00+03:00
---

## Goal

Обёртки для `Embedder` и `LLMProvider` с exponential backoff и circuit breaker для отказоустойчивости production.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | RetryEmbedder retry | 2 вызова базового, 1 результат |
| AC-002 | RetryLLMProvider исчерпание | 3 попытки, возврат ошибки |
| AC-003 | CB блокировка | Переход в open, 5 запросов max |
| AC-004 | CB восстановление | Half-open → closed при успехе |
| AC-005 | Context cancellation | Прерывание по DeadlineExceeded |
| AC-006 | Hooks observability | События retry и CB transitions |

## Out of Scope

- VectorStore retry
- StreamingLLMProvider retry
- Persistence CB state
- Distributed circuit breaker
- HTTP-level retry
