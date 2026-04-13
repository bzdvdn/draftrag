---
report_type: inspect
slug: weaviate-docs
status: pass
docs_language: ru
generated_at: 2026-04-13
---

# Inspect Report: weaviate-docs

## Scope

- snapshot: проверка спека на документационную фичу `docs/weaviate.md` + ссылки из docs
- artifacts:
  - `.speckeep/constitution.md`
  - `.speckeep/specs/weaviate-docs/spec.md`
  - текущие docs: `docs/vector-stores.md`, `docs/compatibility.md`, `docs/production.md`

## Verdict

- status: pass

Спек в scope, не требует изменений кода, соответствует конституции (docs на русском, публичный API в `pkg/draftrag`, best-effort без SLA).

## Errors

- none

## Warnings

- W-001 Язык/оформление: заголовок и часть секций спека на английском (`# Weaviate documentation`, `Scope Snapshot`). По политике проекта “docs=русский” — лучше унифицировать в русскую форму, чтобы все speckeep-артефакты были однородны.
- W-002 Discoverability: спек требует ссылку из `docs/vector-stores.md`, но не фиксирует, где именно она должна быть (отдельный раздел vs. таблица сравнения). В plan стоит выбрать один конкретный путь, чтобы не расползтись.
- W-003 Consistency: `docs/compatibility.md` уже содержит Weaviate (как experimental) с примечанием “нет дока”. В реализации нужно заменить примечание на ссылку на `docs/weaviate.md`, чтобы матрица не противоречила docs.

## Questions

- none

## Suggestions

- В `docs/weaviate.md` держать пример “production-minded”: `context.WithTimeout`, подготовка коллекции отдельным шагом (deploy job/init), и краткий troubleshooting (404/collection missing, auth, timeout).
- В `docs/vector-stores.md` добавить Weaviate в список и в таблицу сравнения, но оставить подробности в отдельной странице `docs/weaviate.md` (чтобы `vector-stores.md` не разрастался).

## Traceability

| AC | Summary |
|----|---------|
| AC-001 | `docs/weaviate.md` существует + ссылка из `docs/vector-stores.md` |
| AC-002 | Quickstart: подготовка коллекции → store → index → retrieve (с таймаутами) |
| AC-003 | Возможности/ограничения + типовые ошибки |

## Next Step

Ошибок нет, можно переходить к планированию:

```
/speckeep.plan weaviate-docs
```

