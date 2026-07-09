# Единый options pattern в публичном API — Задачи

## Phase Contract

Inputs: `.speckeep/specs/public-api-options-unification/plan/plan.md`, `.speckeep/specs/public-api-options-unification/plan/data-model.md`  
Outputs: упорядоченные исполнимые задачи с покрытием критериев приёмки  
Stop if: задачи получаются расплывчатыми или coverage не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| CONTRIBUTING.md | T1.1 |
| README.md | T1.1 |
| pkg/draftrag/pgvector.go | T2.1 |
| pkg/draftrag/pgvector_runtime_test.go | T2.2 |
| pkg/draftrag/options_pattern_test.go | T3.1 |
| docs/vector-stores.md | T3.2 |

## Фаза 1: Правило публичного options pattern

Цель: зафиксировать канонический публичный контракт и правила исключений, чтобы консистентность не “съезжала”.

- [x] T1.1 Обновить `CONTRIBUTING.md` и `README.md`: описать канонический паттерн для `pkg/draftrag` (struct `XOptions` как последний параметр, zero-values = defaults; если опции не нужны — options параметра нет), правила для “двух наборов опций” (вложенный `Runtime`/`Limits` внутри одного контейнера), и как оформлять исключения (только с явным обоснованием + allowlist в guardrail тесте). Touches: CONTRIBUTING.md, README.md — AC-001, RQ-001

## Фаза 2: Унификация pgvector runtime options

Цель: убрать самый заметный “особый случай” (две options-структуры в одном конструкторе), сохранив backward compatibility.

- [x] T2.1 В `pkg/draftrag/pgvector.go`: добавить единый options-контейнер для pgvector store (например, `PGVectorStoreOptions` с вложенным `Runtime PGVectorRuntimeOptions`), добавить новый canonical конструктор (например, `NewPGVectorStoreWithOptions(db, opts PGVectorStoreOptions)`), и перевести существующие `NewPGVectorStore`/`NewPGVectorStoreWithRuntimeOptions` на него. `NewPGVectorStoreWithRuntimeOptions` пометить `// Deprecated:` и оставить рабочим как thin-wrapper (миграция “старый → новый” будет в docs). Touches: pkg/draftrag/pgvector.go — AC-002, AC-003, DEC-001, DEC-002
- [x] T2.2 Обновить pgvector-тесты под новый API: перенести проверки и интеграционный тест в `pkg/draftrag/pgvector_runtime_test.go` на `NewPGVectorStoreWithOptions` (или эквивалент), и добавить маленькую проверку, что deprecated `NewPGVectorStoreWithRuntimeOptions` всё ещё работает (compile/runtime smoke). Touches: pkg/draftrag/pgvector_runtime_test.go — AC-002, AC-003

## Фаза 3: Guardrail + документация

Цель: закрепить правило автоматической проверкой и обновить пользовательские примеры.

- [x] T3.1 Добавить guardrail unit-test `pkg/draftrag/options_pattern_test.go`: через `go/parser`/`go/ast` найти экспортируемые `New*` в `pkg/draftrag` и проваливать тест, если функция принимает больше одного `...Options` struct (или нарушает правило “0/1 options, options — последний параметр”). Для исключений — явная allowlist (с кратким объяснением) и требование `// Deprecated:` для legacy-форм. Touches: pkg/draftrag/options_pattern_test.go — AC-005, DEC-003, RQ-005
- [x] T3.2 Обновить `docs/vector-stores.md`: заменить пример `NewPGVectorStoreWithRuntimeOptions` на новый canonical вариант с единым options-контейнером (включая runtime-подраздел), и добавить короткий миграционный блок “старый → новый” (с указанием депрекации). Пробежать `rg` по `README.md`, `docs/*`, `examples/*` на предмет старого вызова и обновить найденное. Touches: docs/vector-stores.md — AC-004, AC-003, RQ-004

## Покрытие критериев приёмки

- AC-001 -> T1.1 (правило и примеры в документации проекта)
- AC-002 -> T2.1 (унификация API), T2.2 (тесты)
- AC-003 -> T2.1 (депрекация + thin-wrapper), T3.2 (миграция в docs)
- AC-004 -> T3.2 (обновление docs/examples)
- AC-005 -> T3.1 (guardrail тест на дрейф паттерна)
