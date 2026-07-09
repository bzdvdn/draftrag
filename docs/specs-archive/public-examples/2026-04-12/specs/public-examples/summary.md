---
slug: public-examples
generated_at: 2026-04-12
---

## Goal

Добавить в `README.md` 1–2 коротких production-ready end-to-end примера с таймаутами, кешом эмбеддингов и retry/CB.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | Pgvector production-ready пример | В README есть pgvector код-блок с cache+retry+timeouts |
| AC-002 | Qdrant production-ready пример | В README есть Qdrant код-блок с cache+retry+timeouts |
| AC-003 | Явные таймауты и контекст | В примерах есть `context.WithTimeout` и `defer cancel()` |

## Out of Scope

- Изменение публичного API draftRAG
- Новые провайдеры/хранилища
- Редизайн `examples/` и CI

