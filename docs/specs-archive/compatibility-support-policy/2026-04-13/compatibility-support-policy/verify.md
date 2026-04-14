---
report_type: verify
slug: compatibility-support-policy
status: pass
docs_language: russian
generated_at: 2026-04-13
---

# Verify Report: compatibility-support-policy

## Scope

- **Mode**: standard
- **Surfaces checked**:
  - `docs/compatibility.md`
  - `README.md`
  - `.speckeep/specs/compatibility-support-policy/plan/tasks.md`
- **Task list**: all tasks completed (T1.1-T4.1)
- **Acceptance criteria**: AC-001..AC-003 verified

## Verdict

**PASS** — фича готова к архивированию.

## Acceptance Evidence

| AC | Verification | Evidence |
|----|--------------|----------|
| **AC-001** Документ добавлен и доступен из README | ✅ PASS | `docs/compatibility.md` создан; ссылка добавлена в `README.md` в секцию “Документация” |
| **AC-002** Go/semver/депрекации без двусмысленностей | ✅ PASS | `docs/compatibility.md`: минимум Go **1.23**; окно поддержки **N=2** minor; депрекации: **2 minor или 6 месяцев (что дольше)**; breaking только major |
| **AC-003** Матрицы совместимости и возможностей присутствуют | ✅ PASS | `docs/compatibility.md`: таблицы “Backends vs Status” и “Матрица возможностей”; есть пометка best-effort без SLA |

## Consistency Notes

- Weaviate присутствует в публичном API; в политике отмечен как **experimental** и включён в матрицу, при этом отдельного дока пока нет.

## Test Results

```
$ go test ./...
ok
```

## Errors

None.

## Warnings

None.

## Next Step

```
/speckeep.archive compatibility-support-policy
```

