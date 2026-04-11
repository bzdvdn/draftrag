---
report_type: inspect
slug: metadata-filtering
status: pass
docs_language: ru
generated_at: 2026-04-08
---

# Inspect Report: metadata-filtering

## Scope

Проверены: `constitution.md`, `specs/metadata-filtering/spec.md`. Файлы `plan.md` и `tasks.md` отсутствуют — cross-artifact checks не применяются.

## Verdict

**pass**

Структурных ошибок нет. Все 5 AC имеют формат Given/When/Then и стабильные ID. Маркеры `[NEEDS CLARIFICATION]` отсутствуют. Открытые вопросы закрыты. Spec соответствует конституции.

## Errors

none

## Warnings

none

## Suggestions

- Упоминание `Query.Filter map[string]string` в секции «Контекст» как вытесняемого поля — допущение, которое стоит верифицировать при планировании: если поле уже используется в каком-либо вызове публичного API, потребуется явная migration note в плане.
- `## Критерии успеха` опущены обоснованно — фича поведенческая. Если в будущем появятся требования к latency pgvector-запросов с JSONB-фильтром, стоит добавить `SC-001`.

## Traceability

`tasks.md` не существует — traceability AC → tasks будет проверена на фазе `tasks`.

## Next Step

Spec готов к планированию. Следующая команда: `/draftspec.plan metadata-filtering`
