---
report_type: inspect
slug: observability-hooks
status: pass
docs_language: ru
generated_at: 2026-04-08
---

# Inspect Report: observability-hooks

## Scope

- snapshot: проверена спецификация hooks для стадий pipeline на аддитивность и тестируемость без сети
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/observability-hooks/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- none

## Questions

- Хотим ли различать hook-события для Index/Query/Answer, или достаточно “stage events” (embed/search/generate/chunking) с опциональным operation name? (предложение: v1 добавить `Operation` строкой для аналитики)

## Suggestions

- Делать hooks sync и максимально лёгкими: вызывать только if-not-nil, не делать allocation-heavy payload.
- Не тащить зависимости observability в domain; интерфейсы/типы — чистые, без сторонних импортов.

## Traceability

- AC-001: тесты проверяют порядок и количество вызовов hooks на Answer (embed/search/generate) и chunking при наличии.
- AC-002: nil hooks не меняет поведение (тест на отсутствие паник + существующие тесты зелёные).

## Next Step

- safe to continue to plan

