---
report_type: inspect
slug: query-rewriting
status: concerns
docs_language: ru
generated_at: 2026-07-10
---

# Inspect Report: query-rewriting

## Scope

- snapshot: проверка spec плагируемого QueryRewriter для pipeline draftRAG
- artifacts:
  - .speckeep/constitution.summary.md
  - docs/specs/query-rewriting/spec.md

## Verdict

- status: concerns — два мелких дефекта, не блокирующих планирование

## Errors

- none

## Warnings

1. **AC-006 (строка 137)**: китайские иероглифы `最常见的` в описании важности. Заменить на русский эквивалент, например: «из коробки working solution для наиболее частого случая».
2. **Основной сценарий (строка 16)**: `re writer.Rewrite` — лишний пробел, должно быть `rewriter.Rewrite`.

## Questions

- none

## Suggestions

- SC-002: «не требует модификации pipeline или SearchBuilder» — корректно для pipeline-level, но per-request метод `.Rewriter(r)` добавляется именно в SearchBuilder. Стоит уточнить формулировку в SC-002: «не требует модификации внутреннего кода pipeline».
- В MVP Slice перечислены AC-001–005, но AC-006 (LLMRewriter) и AC-007 (HyDE/MultiQuery override) тоже часть MVP. Рекомендуется добавить их в список или явно указать, что они отложены.

## Traceability

- 7 RQ, 7 AC — полное покрытие 1:1
- Plan/tasks ещё не созданы, проверка покрытия задачами не проводилась

## Next Step

- Исправить Warnings перед планированием (опционально: можно на этапе plan, но AC-006 содержит не-RU текст)
- safe to continue to plan after minor cleanup
