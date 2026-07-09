# Production checklist + runbook — Задачи

## Phase Contract

Inputs: `.speckeep/specs/production-checklist-runbook/plan/plan.md`, текущий `README.md` и `docs/`.
Outputs: `docs/production.md` + ссылка в `README.md`.
Stop if: появляется необходимость менять код библиотеки (вне scope).

## Surface Map

| Surface | Tasks |
|---------|-------|
| docs/production.md | T1.1, T2.1, T2.2, T2.3 |
| README.md | T1.2 |

## Фаза 1: Основа

Цель: создать документ и сделать его доступным как entrypoint из README.

- [x] T1.1 Создать `docs/production.md` со структурой документа. Touches: docs/production.md
  - Outcome: документ содержит разделы `Checklist`, `Runbook`, `Backend notes`, `Security/Redaction`.
  - Links: AC-001, AC-002, AC-003, AC-004, DEC-001

- [x] T1.2 Добавить ссылку на `docs/production.md` в `README.md`. Touches: README.md
  - Outcome: в README есть явная ссылка “Production checklist + runbook”.
  - Links: AC-001, RQ-005

## Фаза 2: Основная реализация

Цель: наполнить checklist и runbook конкретными действиями и сценариями.

- [x] T2.1 Написать checklist из 5–15 проверяемых пунктов. Touches: docs/production.md
  - Outcome: каждый пункт = действие + критерий проверки + ссылка на существующий README/docs (без больших копипаст).
  - Links: AC-002, DEC-002

- [x] T2.2 Добавить runbook минимум для 4 инцидентов по шаблону. Touches: docs/production.md
  - Outcome: 4+ раздела `Symptoms / Checks / Actions` (“быстро” = коротко и по шагам, без SLO).
  - Links: AC-003

- [x] T2.3 Добавить раздел Security/Redaction и границы ответственности. Touches: docs/production.md
  - Outcome: описано, что redaction best-effort для известных секретов; пользователь отвечает за свои логи и контент.
  - Links: AC-004

## Фаза 3: Проверка

Цель: убедиться, что документ соответствует AC и не содержит двусмысленностей.

- [x] T3.1 Провести self-review документа на чек-лист/инциденты/ссылки. Touches: docs/production.md
  - Outcome: checklist 5–15 пунктов; runbook 4+; ссылки из README работают.
  - Links: AC-001, AC-002, AC-003, AC-004

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2, T3.1
- AC-002 -> T2.1, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T2.3, T3.1

## Заметки

- Документ должен быть “index” формата: ссылки на существующие разделы README/docs вместо дублирования.
- Избегать обещаний SLO/latency; “быстро” = короткие инструкции.
