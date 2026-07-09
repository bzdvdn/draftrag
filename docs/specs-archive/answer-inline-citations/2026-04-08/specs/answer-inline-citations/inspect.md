---
report_type: inspect
slug: answer-inline-citations
status: pass
docs_language: ru
generated_at: 2026-04-08
---

# Inspect Report: answer-inline-citations

## Scope

- snapshot: проверена спецификация на полноту для планирования и соответствие конституции (аддитивное изменение публичного API, без breaking changes)
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/answer-inline-citations/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- none

## Questions

- Нужно ли в v1 валидировать номера цитат, которые вернула LLM (например, `[999]`), и как сигнализировать об ошибке? (предложение: v1 без строгой валидации; можно добавить опциональный строгий режим позже)

## Suggestions

- Держать формат prompt контрактом v1: нумеровать источники в prompt, чтобы “citations mapping” был детерминированным и не зависел от парсинга текста.
- Ограничивать доступные для цитирования источники минимумом из `topK` и `MaxContextChunks` (если задан).

## Traceability

- AC-001: покрывается добавлением нового метода Answer*WithInlineCitations + возвращаемой структуры `citation number → retrieved chunk`.
- AC-002: обеспечивается тем, что существующие Answer/AnswerWithCitations не меняют поведение; добавление только новых методов/типов.

## Next Step

- safe to continue to plan
