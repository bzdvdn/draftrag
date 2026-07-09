---
report_type: inspect
slug: arch-quality-pass
status: concerns
docs_language: ru
generated_at: 2026-06-04
---

# Inspect Report: arch-quality-pass

## Scope

- snapshot: проверка spec на три архитектурных улучшения — panic→error, Hooks contract с возвратом context, устранение дублирования PipelineConfig/PipelineOptions
- artifacts:
  - .speckeep/constitution.summary.md
  - docs/specs/arch-quality-pass/spec.md

## Verdict

- status: concerns (not blocked)

## Errors

- none

## Warnings

1. **AC-004 evidence неточен**: evidence говорит "grep -r 'PipelineConfig' internal/ не находит упоминаний (кроме тестов, которые мигрированы)". Фактически grep показывает 55 совпадений во всех internal-файлах, из которых ~50 в тестах и 5 в production-коде. После миграции тестов grep должен чисто проходить. Стоит уточнить evidence: "grep -r 'PipelineConfig' internal/ находит 0 ссылок в production-коде (после рефакторинга) и 0 ссылок в тестах (после миграции тестов на PipelineOptions)".

2. **AC-005 нетривиально тестировать**: проверка "экспортированный span имеет StartTime ≈ времени вызова StageStart" требует OTel SpanExporter mock. В текущей кодовой базе нет такого мока. Нужно либо добавить тестовый exporter (рекомендуется: `go.opentelemetry.io/otel/exporters/stdout/stdouttrace`), либо ослабить evidence до "тест использует `tracesynctest` или кастомный `SpanExporter`". Рекомендуется добавить в spec ссылку на подход к тестированию.

3. **RQ-003 vs RQ-004 tension**: RQ-003 (единый struct) и RQ-004 (обратная совместимость) частично конфликтуют — удаление `application.PipelineConfig` ломает импорты для кода, который использует этот тип напрямую. Spec документирует это в Open Questions #2, но impact на пользователей не оценён. Рекомендуется явно указать, что это breaking change для importers of `PipelineConfig`.

## Questions

- Q1 (из spec): имя единого struct — `PipelineOptions` остаётся как есть, т.к. уже экспортирован. Меньше breaking change. Решение принято, можно закрыть.
- Q2 (из spec): тесты, использующие `PipelineConfig`, мигрируются на `PipelineOptions` или оставляют внутренний alias. Рекомендация: оставить `type PipelineConfig = PipelineOptions` как internal alias в `internal/application` на время миграции тестов, затем удалить.
- Q3 (из spec): nil store/llm/embedder — оставить panic. Решение принято, соответствует Go-идиоме.

## Suggestions

- AC-005: добавить в spec секцию "Подход к тестированию" или ссылку на использование `go.opentelemetry.io/otel/exporters/stdout/stdouttrace` или кастомного `SpanExporter`.
- RQ-002: добавить проверку, что `nil Hooks` в PipelineOptions НЕ вызывает error — это уже есть в "Краевые случаи".
- AC-004: определить точный grep-запрос для evidence (production code only vs including tests).

## Traceability

- 5 AC покрывают 4 RQ:
  - AC-001 → RQ-001 (Hooks contract)
  - AC-002 → RQ-002 (panic→error)
  - AC-003 → RQ-004 (backward compat)
  - AC-004 → RQ-003 (единый struct)
  - AC-005 → RQ-001 (OTel span creation)
- Каждый RQ покрыт ≥ 1 AC (RQ-004 покрыт AC-003).

## Next Step

- safe to continue to plan with warnings addressed
