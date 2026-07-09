---
slug: structured-logger-hooks
generated_at: 2026-04-12T02:33:44+03:00
---

## Goal
Подключаемый структурированный логгер вместо `log.Printf`.

## Acceptance Criteria
| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | No-op и совместимость | `go test ./...` проходит |
| AC-002 | Redis L2 деградация логируется | Logger вызван с полями |
| AC-003 | Retry/CB события логируются | Logger вызван на retry/CB |
| AC-004 | Logger panic не ломает поток | Операция не падает |
| AC-005 | Есть пример подключения | Раздел в docs/README |

## Out of Scope
- Метрики/трейсинг (OTel/Prometheus).
- Глобальная “магическая” конфигурация логов.
- Изменение `domain.Hooks` / breaking change hooks.
- Семплирование/дедупликация логов.
