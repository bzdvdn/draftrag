---
slug: public-api-options-unification
generated_at: 2026-04-12T12:06:03+03:00
---

## Goal
Унифицировать options-паттерн публичного API.

## Acceptance Criteria
| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | Единый паттерн описан | Правило в docs |
| AC-002 | Конструкторы унифицированы | `go test ./...` ok |
| AC-003 | Миграция/совместимость ясны | Миграция описана |
| AC-004 | Docs/examples унифицированы | Примеры обновлены |
| AC-005 | Guardrail против дрейфа | Check падает в CI |

## Out of Scope
- Переписывание internal паттернов.
- Новые провайдеры/хранилища.
- Тотальная стандартизация имён `Options/Config` вне public API.
- Метрики/трейсинг/прочие unrelated улучшения.

