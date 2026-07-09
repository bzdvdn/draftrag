# go-version-target — Задачи

## Phase Contract

Inputs: `.speckeep/specs/go-version-target/plan/plan.md`, `.speckeep/specs/go-version-target/plan/data-model.md`  
Outputs: упорядоченные исполнимые задачи с покрытием критериев приёмки  
Stop if: `go test ./...` не удаётся запустить/пройти на Go 1.23 без поднятия минимальной версии

## Surface Map

| Surface | Tasks |
|---------|-------|
| go.mod | T1.1 |
| README.md | T1.2 |
| docs/getting-started.md | T1.2 |
| .speckeep/constitution.md | T1.3 |
| .github/workflows/ci.yml | T2.1 |

## Фаза 1: Выровнять “источник правды”

Цель: один канонический минимум Go (DEC-001) во всех пользовательских артефактах.

- [x] T1.1 Обновить `go.mod`: выставить `go 1.23.x` (и не добавлять `toolchain`, либо оставить его не повышающим минимум). После изменения прогнать `go mod tidy` (если требуется) и убедиться, что `go test ./...` по-прежнему проходит локально. Touches: go.mod — AC-001, RQ-001, RQ-002, DEC-001, DEC-002
- [x] T1.2 Обновить документацию: привести требование “минимальная версия Go” к `1.23` в `README.md` и `docs/getting-started.md`. Touches: README.md, docs/getting-started.md — AC-002, RQ-003
- [x] T1.3 В `/.speckeep/constitution.md`: проверить, что минимум Go совпадает с DEC-001 (1.23+). Если отличается — обновить формулировку, чтобы конституция не противоречила `go.mod` и docs. Touches: .speckeep/constitution.md — AC-002, RQ-001

## Фаза 2: Guardrail через CI

Цель: автоматическая проверка минимальной версии Go (DEC-003).

- [x] T2.1 Добавить GitHub Actions workflow `/.github/workflows/ci.yml`: запуск `go test ./...` как минимум на Go `1.23` (опционально — второй job на latest stable). Touches: .github/workflows/ci.yml — AC-003, RQ-004, DEC-003

## Покрытие критериев приемки

- AC-001 -> T1.1 (`go.mod` выровнен под Go 1.23, локальные тесты проходят)
- AC-002 -> T1.2 (README + getting-started едины), T1.3 (constitution не противоречит)
- AC-003 -> T2.1 (CI проверяет `go test ./...` на Go 1.23)
