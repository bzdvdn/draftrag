---
report_type: inspect
slug: api-consistency-pass
status: concerns
docs_language: ru
generated_at: 2026-06-03
---

# Inspect Report: api-consistency-pass

## Scope

- snapshot: проверка спецификации `api-consistency-pass` перед планированием; 8 RQ, 16 AC; alignment с конституцией draftRAG, completeness, ambiguity, placeholder check.
- artifacts:
  - CONSTITUTION.md
  - docs/specs/api-consistency-pass/spec.md

## Verdict

- status: concerns

## Errors

- none

## Warnings

- **W-001 AC-014 арифметика в Evidence**: текст AC говорит "Все 6 stores присутствуют как строки" и "5 stores × 6 capabilities = 30" в Evidence. Корректный счёт для шести stores: 6 stores × 6 capabilities = 36 cells (с учётом N/A для in-memory в колонке Collection mgmt). Сейчас Evidence `grep -c "..." ≥ 30` занижен. Не блокер — поправить в `/speckeep.spec --amend` или принять ≥ 30 как floor (реальное значение будет выше, в диапазоне 30–36). Рекомендация: поднять до `≥ 30` floor и явно указать, что N/A считается.

- **W-002 OQ-2 (StreamBufferSize=0) и OQ-3 (best-effort rollback policy) открыты перед планированием**: AC-009 и AC-010 уже содержат разумные defaults (best-effort + `ErrUpdateNotAtomic`; default buffer 8), но эти defaults — рекомендация, а не зафиксированное решение. При переходе к plan без amend spec рискуем, что реализатор примет другую интерпретацию. Рекомендация: перед `/speckeep.plan` сделать `/speckeep.spec --amend` для фиксации defaults ("StreamBufferSize=0 → unbuffered (backward-compatible), значение 1..N — буферизованный"; "для не-транзакционных store: best-effort re-insert + `ErrUpdateNotAtomic`").

- **W-003 RQ-005 вводит новый публичный тип `TransactionalDocumentStore` capability-интерфейс**: конституция требует "Каждый публичный интерфейс ДОЛЖЕН иметь мок-реализацию для тестирования". В Acceptance это покрыто через "unit-тест с моком `TransactionalDocumentStore`" в AC-008, но явного требования "мок-реализация должна быть доступна пользователю библиотеки" нет. Если интерфейс публичный — мок должен идти в комплекте (например, в `pkg/draftrag/` или в testutil-пакете). Уточнить в plan, требуется ли public mock или достаточно internal test-double.

- **W-004 SC-001 содержит примерную цифру "≤ 14"**: это soft-target с оговоркой "плюс HyDE/MultiQuery branch внутри helper'ов". Для критерия успеха лучше иметь единственное измеримое значение (например, "≤ 7 публичных веток роутинга" или "≤ 14, считая helper'ы"). Сейчас формулировка оставляет интерпретацию. Рекомендация: уточнить в `/speckeep.spec --amend`.

- **W-005 Цель содержит "неуправляемой" (нечёткое прилагательное)**: "сделает 7×6 матрицу роутинга `SearchBuilder` неуправляемой". Контекст quantitative (42 if-блока, 7×6), но само слово "неуправляемой" — оценка, не критерий. Минорная редакторская правка; не блокер.

- **W-006 AC-008 ссылается на integration-тест с docker-compose**: pgvector-тесты в репозитории уже используют этот паттерн (`pgvector_test.go`), но новый файл `pgvector_atomic_update_test.go` не упомянут в существующем test-bootstrap механизме. Уточнить в plan, как новый integration-тест будет подключён к существующему CI-gate (`RUN_INTEGRATION_TESTS=1`).

## Questions

- Q-1: Подтвердить default для `StreamBufferSize=0` (см. W-002) — unbuffered / unbounded-with-warning / reject? Дефолт влияет на backward-compatibility.
- Q-2: Подтвердить policy rollback для не-транзакционных store (см. W-002) — best-effort re-insert + `ErrUpdateNotAtomic`, или просто delete-then-error без попытки восстановления?
- Q-3: `TransactionalDocumentStore` — должен ли mock быть публичным (см. W-003)?

## Suggestions

- S-1: Добавить в RQ-002 явное требование про `errors.New(fmt.Sprintf(...))` — на всякий случай, чтобы grep покрывал и форматированные inline-ошибки. Сейчас `errors.New("...")` без `fmt.Sprintf` — узкий паттерн.
- S-2: Рассмотреть в plan шаг "rename `mapValidationErr` → `mapAppError` или `wrapAppError`" как часть RQ-003, чтобы AC-006 был trivially выполнен (а не требовал отдельного решения об именовании).
- S-3: Capability-таблица (AC-014) — рассмотреть формат, где колонка "Hybrid" помечена footnote для каждой строки, где ❌, со ссылкой на issue/spec (например, "планируется в `milvus-hybrid-search`"). Это сделает таблицу самодокументирующейся.

## Traceability

- 8 RQ → 16 AC, покрытие 1:1+ (RQ-005 → 2 AC, RQ-007 → 2 AC, RQ-008 → 3 AC, остальные RQ → 1 AC; AC-016 — гейт для всех RQ).
- Все 16 AC имеют Given/When/Then/Evidence (verified by readiness-check: 16/16).
- Все 16 AC имеют observable proof signal (grep-count, exit-code, file size, memory cap, integration test outcome).
- Open Questions: 4 шт. (OQ-1..OQ-4). OQ-2 и OQ-3 блокирующие для plan (см. W-002); OQ-1, OQ-4 — soft.
- План ещё не существует, поэтому проверки `spec <-> plan` и `plan <-> tasks` не выполнялись.
- `Touches:` — не применимо (tasks отсутствуют).
- Маркеры `@sk-task`/`@sk-test` — не применимо на inspect-фазе.

## Next Step

- spec проходит inspect со статусом concerns. Warnings W-001..W-006 некритичны, но W-002 (OQ-2, OQ-3) лучше закрыть перед plan, чтобы plan не раздвоился.
- Рекомендуемый путь: `/speckeep.spec api-consistency-pass --amend` для фиксации defaults в AC-009/AC-010 (Q-1, Q-2) и подтверждения политики мока (Q-3), затем `/speckeep.plan api-consistency-pass`.
- Альтернативный (быстрый) путь: перейти к `/speckeep.plan api-consistency-pass` сейчас; defaults уже разумные, plan может пометить OQ-2/OQ-3 как design-time decisions и зафиксировать их в plan.md, без amend spec.
