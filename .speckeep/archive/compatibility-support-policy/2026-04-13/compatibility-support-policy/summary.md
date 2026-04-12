---
slug: compatibility-support-policy
generated_at: 2026-04-12
---

## Goal

Добавить документ политики совместимости/поддержки (Go, semver/deprecation, матрицы backend’ов/фич) и ссылку на него из README.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | Doc доступен из README | В README есть ссылка, файл в `docs/` существует |
| AC-002 | Go/semver/deprecation без двусмысленностей | В doc есть секция с конкретными правилами/сроками |
| AC-003 | Матрицы backend’ов и фич присутствуют | В doc есть 2 таблицы для выбора backend’а |

## Out of Scope

- Изменение backend’ов и добавление новых провайдеров
- CI-матрица тестов по всем backend’ам
- Юридические SLA/контрактная поддержка

