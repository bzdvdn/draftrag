---
report_type: inspect
slug: arch-issues
status: concerns
docs_language: ru
generated_at: 2026-07-12
---

# Inspect Report: arch-issues

## Scope

- snapshot: проверка spec «Архитектурное hardening: PII, tool calling, роутинг, lifecycle» — 4 workstreams в одной фиче
- artifacts:
  - .speckeep/constitution.summary.md
  - docs/specs/arch-issues/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- ~~W-01 Multi-feature scope creep~~ — resolved by explicit user decision (4 workstreams in one spec, 4 separate tasks).
- ~~W-02 AC-003 multi-turn flow~~ — исправлено: AC-003 описывает multi-turn flow с execution callback.
- ~~W-03 AC-005 arbitrary threshold~~ — исправлено: убран порог «3 записи», заменён на `// Code generated` маркер.
- ~~W-04 RQ-006 неоднозначность~~ — исправлено: уточнено «в реализации handler для нового route — без правки 7× map-маппингов».
- ~~W-05 Streaming tools не покрыт~~ — исправлено: `.Stream(ctx)` с tools возвращает ErrToolsNotSupportedInStream; Out of Scope обновлён.

Все Warnings устранены.

## Questions

- Q1: Router generator — выбран формат (A) Go-таблица? Если нет — потребуется переписать AC-005/AC-006.
- Q2: ToolCallingLLMProvider возвращает tool calls синхронно (блокируя pipeline для execution) или асинхронно (канал)? От этого зависит интерфейс.

## Suggestions

- S-01: Для PII в application слое достаточно просто переместить вызов `piidetector.Detect()` из `pkg/draftrag/draftrag.go` в `internal/application/pipeline.go` и убрать дублирующий вызов из public API. Это минимальное изменение.
- S-02: Router generator проще всего реализовать через `go generate` + `text/template` + внутреннюю Go-таблицу (вариант A из открытых вопросов). Внешние зависимости не требуются.
- S-03: Health() можно реализовать как fan-out: запустить три горутины с общим контекстом и таймаутом, собрать ошибки. Для graceful shutdown ошибка одного компонента не маскирует остальные.
- S-04: Для `Close()` — завести `sync.Once` и `closed` флаг в Pipeline. Все операции проверяют флаг в начале.

## Traceability

| AC-* | Workstream | RQ | Проверяем |
|------|-----------|-----|-----------|
| AC-001, AC-002 | PII | RQ-001, RQ-002 | PII в application слое, без дублирования |
| AC-003, AC-004 | Tool calling | RQ-003, RQ-004 | Интерфейс + SearchBuilder интеграция |
| AC-005, AC-006 | Router gen | RQ-005, RQ-006 | Кодогенерация + extensibility |
| AC-007, AC-008 | Health/Shutdown | RQ-007, RQ-008 | Агрегированный статус + освобождение ресурсов |

## Next Step

- safe to continue to plan
