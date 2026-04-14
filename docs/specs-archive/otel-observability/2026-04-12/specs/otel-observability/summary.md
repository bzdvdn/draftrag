---
slug: otel-observability
generated_at: 2026-04-12
---

## Goal

Дать опциональные OTel hooks для spans+metrics по стадиям pipeline и пример подключения в README.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | Публичный OTel hooks для `Hooks` | В godoc есть тип/конструктор, подключаемый в `PipelineOptions.Hooks` |
| AC-002 | Spans по стадиям с ошибками | В exporter видны stage-spans с `operation`/`stage` и error-status |
| AC-003 | Метрики длительности и ошибок | В метриках есть duration+errors с labels `operation`/`stage` |
| AC-004 | README пример подключения OTel | В README есть секция с примером и ограничениями синхронных hooks |

## Out of Scope

- Автоконфигурация OTel SDK/экспортеров
- Трассировка сетевых клиентов/БД и resilience событий
- Готовые дашборды/алерты и SLO значения

