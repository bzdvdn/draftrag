---
report_type: inspect
slug: docs-and-examples
status: pass
docs_language: ru
generated_at: 2026-06-03
---

# Inspect Report: docs-and-examples

## Scope

- snapshot: проверка `docs/specs/docs-and-examples/spec.md` (10 RQ, 17 AC) на соответствие конституции, полноту AC, неоднозначности и placeholders перед фазой plan. Проверены constitution (CONSTITUTION.md), spec.md и текущий `examples/`.
- artifacts:
  - CONSTITUTION.md
  - docs/specs/docs-and-examples/spec.md

## Verdict

- status: pass
- summary: 0 errors, 2 false-positive warnings на стандартном термине «Быстрый старт» (не прилагательное — название секции; в обоих случаях рядом стоит измеримое ограничение строк). Все 17 AC имеют Given/When/Then с observable proof. Конфликтов с конституцией не обнаружено. Scope строго одна фича (документация + examples, ноль правок `pkg/`/`internal/`). [NEEDS CLARIFICATION] маркеры устранены.

## Errors

- none

## Warnings

- W-1 «Быстрый старт» в RQ-005 (line 59) — false positive heuristic. «Быстрый старт» — стандартное название раздела в русскоязычной документации (калька с англ. «Quickstart»), не расплывчатое прилагательное. Измеримое ограничение рядом: «≤10 строк».
- W-2 «Быстрый старт» в AC-013 (line 172) — false positive heuristic (та же причина). Измеримое ограничение рядом: «5..10 строк» + «в первых 50 строках README».

Оба warning'а оставлены сознательно: замена «Быстрый старт» на «Введение» или «Начало работы» ухудшила бы discoverability для русскоязычной аудитории без реальной выгоды (измеримость сохранена через явные числовые ограничения на размер секции).

## Questions

- none (4 вопроса из spec.md перенесены в Open Questions и будут разрешены в `plan`-фазе: RQ-007 уже разрешён в этом inspect-pass; tutorial-09 scope, tutorial-10 split, нужен ли examples/cli — отложены)

## Suggestions

- S-1 В `plan`-фазе для tutorial 09-evaluation зафиксировать, что используется существующий `pkg/draftrag/eval` (harness + metrics) и демонстрируется 2-3 простых Case; без новых Go-файлов.
- S-2 В `plan`-фазе для tutorial 10-production-hardening — оставить единый tutorial, но в нём явно выделить секции 10.1 resilience (retry + circuit breaker), 10.2 observability (OTel), 10.3 redaction (Redactor).
- S-3 В `plan`-фазе решить, нужен ли `examples/cli/` (единый entrypoint `--store=qdrant --llm=ollama`); default recommendation = не нужен (6 отдельных директорий достаточно).

## Constitution ↔ Spec Alignment

- ✅ "Код ДОЛЖЕН следовать принципам Clean Architecture" — RQ-008 явно запрещает изменения `pkg/draftrag/` и `internal/`; examples используют только публичный API.
- ✅ "Все операции ДОЛЖНЫ принимать `context.Context`" — все примеры в `examples/{pgvector,qdrant}/main.go` уже следуют; новые примеры обязаны.
- ✅ "Каждый публичный интерфейс ДОЛЖЕН иметь мок-реализацию" — RQ-003 вводит mock-эмбеддер + mock-LLM в `examples/shared/`, что и является мок-реализацией.
- ✅ "Документация: каждый публичный тип и функция ДОЛЖНЫ иметь godoc-комментарий на русском языке" — не затрагивается этой спекой (нет новых публичных типов).
- ✅ "Все новые функции ДОЛЖНЫ иметь unit-тесты" — для examples = CI smoke-job (AC-014).
- ✅ "Язык документации: русский" — все README и tutorials на русском; код и команды на английском.
- ✅ "Каждая фича ДОЛЖНА разрабатываться в отдельной git-ветке с префиксом `feature/<slug>`" — ветка `feature/docs-and-examples` создана.
- ⚠️ "Время сборки: `go build ./...` ДОЛЖЕН завершаться <5 секунд" — не должно пострадать (6 новых `main.go` компилируются независимо и параллельно); проверить в plan/implement.

## Scope Analysis

- In scope: 6 директорий `examples/{memory,pgvector,qdrant,chromadb,weaviate,milvus}/` (pgvector и qdrant — рефакторинг под общий шаблон; memory/chromadb/weaviate/milvus — новые), 1 общий Go-пакет `examples/shared/`, 10 tutorials, обновления README + docs/vector-stores.md + ROADMAP.md, 1 новый CI workflow.
- Out of scope (явно): новые VectorStore-бэкенды, новые LLM-провайдеры, перевод на английский, web UI, бенчмарки, изменения `pkg/draftrag/`, изменения `chat/` и `index-dir/`.
- Single feature: да — всё сводится к «сделать библиотеку легко запускаемой новым разработчиком».

## Acceptance Criteria Quality

- 17 ACs, все имеют Given/When/Then с observable proof.
- AC-001..AC-005: по одному на каждый бэкенд (pgvector, qdrant, chromadb, weaviate, milvus) — симметрично и проверяемо через CI.
- AC-006: in-memory без Docker — закрывает edge case "разработчик без Docker".
- AC-007: переключение LLM через env — закрывает основной user story.
- AC-008: mock-LLM без API-ключей — закрывает edge case "разработчик без API-ключей" + CI smoke.
- AC-009: compose-validate — syntax gate.
- AC-010..AC-012: tutorial структура + cross-links.
- AC-013: README обновлён.
- AC-014: CI matrix examples-smoke.
- AC-015: capability-таблица линки.
- AC-016: zero-diff в pkg/internal — non-regression gate.
- AC-017: existing tests проходят — non-regression gate.

Каждый AC имеет Evidence (строка с observable artefact: file path, exit code, grep output, CI job).

## Ambiguity Check

- "Быстрый старт" × 2 — false positives (см. W-1, W-2).
- "минимальная конфигурация" (Допущения) — стандартная формулировка из конституции, не ambiguity.
- "разумные настройки по умолчанию" (Конституция) — стандартная формулировка, не из spec'а.
- "zero-friction" (Цель) — идиома; смысл раскрыт в SC-001 (≤10 минут).
- Все измеримые критерии (10 минут, 5..10 строк, 6 бэкендов, 10 tutorials) — конкретные числа.

## Technology Mentions

Упоминания бэкендов (`pgvector`, `qdrant`, `chromadb`, `weaviate`, `milvus`) и провайдеров (`ollama`, `openai`, `anthropic`) — не warnings, т.к. это:
1. Явное требование пользователя ("со всеми хранилищами и провайдерами").
2. Существующие repo-constraint'ы (библиотека уже поддерживает эти бэкенды; см. `internal/infrastructure/vectorstore/`).
3. Внешний contract: Docker images — публичные артефакты, к которым библиотека обязана предоставлять examples.

Версионные pins (`qdrant:v1.12.4`, `weaviate:1.27.x`, etc.) — в RQ-010 как требование, не как implementation choice; pin — это требование воспроизводимости, не "implementation preference".

## Placeholder Check

- TODO/???/<placeholder>/TKTK — не найдено.
- [NEEDS CLARIFICATION: ...] — устранены в этом inspect-pass (RQ-007).
- Незакрытые `[` brackets — не обнаружено.

## Traceability

- AC ↔ RQ:
  - AC-001..AC-005 → RQ-001 (по одному AC на бэкенд)
  - AC-006 → RQ-001 (memory, edge case)
  - AC-007 → RQ-002 (LLM provider selection)
  - AC-008 → RQ-003 (mock mode)
  - AC-009 → RQ-010 (compose syntax gate)
  - AC-010, AC-011, AC-012 → RQ-004 (tutorials)
  - AC-013 → RQ-005 (README)
  - AC-014 → RQ-006 (CI matrix)
  - AC-015 → RQ-007 (capability table links)
  - AC-016 → RQ-008 (zero pkg/internal changes)
  - AC-017 → RQ-009 (existing tests)
- 10 RQ, 17 AC, 0 gaps.

## Next Step

- safe to continue to plan
