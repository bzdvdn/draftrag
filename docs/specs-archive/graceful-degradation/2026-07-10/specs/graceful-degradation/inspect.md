---
report_type: inspect
slug: graceful-degradation
status: pass
docs_language: ru
generated_at: 2026-07-10
---

# Inspect Report: graceful-degradation

## Scope

- snapshot: механизм chain fallback для LLM-провайдеров (Primary → Secondary → Local) с graceful degradation и наблюдаемостью
- artifacts:
  - CONSTITUTION.md
  - docs/specs/graceful-degradation/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- none

## Questions

- AC-006: детекция "преждевременного закрытия канала с ошибкой" зависит от контракта GenerateStream (канал закрывается с ошибкой vs ошибка через канал). Рекомендуется уточнить на уровне plan: использовать sentinel-ошибку или отдельный метод для ошибки потока.

## Suggestions

- Рассмотреть возможность единого `FallbackLLMProvider` с type assertion на `StreamingLLMProvider` и `UsageAwareLLMProvider` (как это делает Pipeline), вместо трёх отдельных типов. Это уменьшит дублирование кода. Предложение зафиксировано в spec как отдельные обёртки — acceptable для первого прохода.
- Для `FallbackStats` рассмотреть атомарные счётчики (`sync/atomic.Int64`) — thread-safety потребуется, если FallbackLLMProvider используется из нескольких goroutine.

## Traceability

- AC-001–AC-005: FallbackLLMProvider (Generate + Health + Hooks + Stats)
- AC-006: FallbackStreamingLLMProvider (GenerateStream)
- AC-007: FallbackUsageAwareLLMProvider (GenerateWithUsage + TokenUsage)
- AC-008: FallbackStats на всех Fallback-типах
- AC-009: валидация пустой цепи

Каждый AC покрыт отдельным RQ (RQ-001–RQ-012).

## Next Step

- safe to continue to plan