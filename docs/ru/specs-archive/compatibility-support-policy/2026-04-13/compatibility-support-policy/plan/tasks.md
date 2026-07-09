# Compatibility & Support Policy — Задачи

## Phase Contract

Inputs: `.speckeep/specs/compatibility-support-policy/spec.md`, `.speckeep/specs/compatibility-support-policy/plan/plan.md`, текущие `docs/` и `README.md`.
Outputs: `docs/compatibility.md` + ссылка в `README.md`.
Stop if: для честного заполнения матриц требуется менять код или делать широкий аудит реализации (это вне scope) — тогда ограничиться “best-effort по docs/README” и явно пометить границы.

## Surface Map

| Surface | Tasks |
|---------|-------|
| docs/compatibility.md | T1.1, T1.2, T1.3, T2.1, T2.2, T4.1 |
| README.md | T3.1, T4.1 |

## Фаза 1: Документ политики (каркас + правила)

Цель: создать “источник истины” для поддержки/совместимости без двусмысленностей.

- [x] T1.1 Создать `docs/compatibility.md` со структурой и оглавлением. Touches: docs/compatibility.md
  - Outcome: документ на русском с разделами: `Go support`, `SemVer & Deprecation`, `Backends`, `Features`, `Update policy` (AC-001).
  - Links: AC-001, RQ-001

- [x] T1.2 Зафиксировать политику версий Go (минимум + окно поддержки). Touches: docs/compatibility.md
  - Outcome: явно указано: минимум Go **1.23**; правило поддержки “N последних minor Go, N=2”; когда пересматривается минимум и как об этом сообщается (AC-002).
  - Links: AC-002, RQ-002, DEC-001

- [x] T1.3 Зафиксировать SemVer и депрекации публичного API `pkg/draftrag`. Touches: docs/compatibility.md
  - Outcome: правила SemVer + конкретное окно поддержки deprecated API: **минимум 2 minor релиза или 6 месяцев (что дольше)**; каналы коммуникации: godoc (`Deprecated:`), release notes/CHANGELOG (AC-002).
  - Links: AC-002, RQ-005

## Фаза 2: Матрицы совместимости и возможностей

Цель: дать пользователю таблицы для выбора backend’а без чтения кода.

- [x] T2.1 Добавить таблицу “Backends vs Status” (stable/experimental). Touches: docs/compatibility.md
  - Outcome: таблица по группам (vector store / embedder / LLM); список backend’ов не содержит несуществующих в проекте; статусы согласованы с текущими docs/README (AC-003).
  - Links: AC-003, RQ-003, DEC-002, DEC-003

- [x] T2.2 Добавить таблицу “Features vs Backends” (best-effort по docs/README). Touches: docs/compatibility.md
  - Outcome: отдельная таблица возможностей (filters/hybrid/streaming/cache/retry+CB/hooks/OTel/migrations и т.п.) с отметками `✓/—/n/a` и короткими примечаниями, где нужно (AC-003).
  - Links: AC-003, RQ-004, DEC-003

## Фаза 3: README ссылка

Цель: сделать политику обнаруживаемой.

- [x] T3.1 Добавить ссылку на `docs/compatibility.md` в секцию “Документация” в `README.md`. Touches: README.md
  - Outcome: в README есть явная ссылка с кратким описанием (“политика совместимости и поддержки”) (AC-001).
  - Links: AC-001, RQ-006

## Фаза 4: Самопроверка на критерии приемки

Цель: убедиться, что документ пригоден для enterprise планирования и не обещает лишнего.

- [x] T4.1 Пройтись по AC-001..AC-003 и убрать двусмысленности. Touches: docs/compatibility.md, README.md
  - Outcome: (1) все числа/окна поддержки конкретны; (2) матрицы читаемы; (3) есть явная пометка “best-effort, без SLA”; (4) Go минимум совпадает с конституцией (AC-001..AC-003).
  - Links: AC-001, AC-002, AC-003

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1, T4.1
- AC-002 -> T1.2, T1.3, T4.1
- AC-003 -> T2.1, T2.2, T4.1
