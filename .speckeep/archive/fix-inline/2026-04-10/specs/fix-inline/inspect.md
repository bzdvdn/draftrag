---
report_type: inspect
slug: fix-inline
status: pass
docs_language: ru
generated_at: 2026-04-10
---

# Inspect Report: fix-inline

## Scope

Проверка спецификации `fix-inline` — исправление пропущенного маппинга `ErrFiltersNotSupported` в `SearchBuilder.InlineCite`.
Артефакты: `spec.md`. `plan.md` и `tasks.md` отсутствуют — cross-artifact проверки пропущены.

## Verdict

**pass**

Структурная проверка (`check-inspect-ready.sh`, `inspect-spec.sh`): `errors=0 warnings=0`.
Ручная проверка по конституции и содержательному качеству: нарушений не найдено.

## Errors

none

## Warnings

none

## Questions

none

## Suggestions

- В AC-003 Evidence описывает передачу ошибки напрямую в тесте (`fmt.Errorf("wrap: %w", application.ErrFiltersNotSupported)`). При написании теста убедиться, что mock-store возвращает именно такую обёрнутую ошибку, а не внутренняя Application-логика проверяется напрямую — иначе тест проверяет только wrapper, а не реальный путь через `core`.

## Traceability

tasks.md отсутствует — traceability будет доступна после фазы tasks.

## Next Step

Спецификация готова к планированию. Следующая команда: `/draftspec.plan fix-inline`
