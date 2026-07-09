---
report_type: verify
slug: cost-tracking
status: pass
docs_language: ru
generated_at: 2026-07-10
---

# Verify Report: cost-tracking

## Scope

- snapshot: сквозной подсчёт токенов и стоимости LLM-вызовов через CostTracker + поддержка streaming usage
- verification_mode: deep
- artifacts:
  - docs/specs/cost-tracking/spec.md
  - docs/specs/cost-tracking/plan.md
  - docs/specs/cost-tracking/tasks.md
  - docs/specs/cost-tracking/data-model.md
- inspected_surfaces:
  - internal/domain/interfaces.go — UsageAwareLLMProvider, UsageAwareStreamingLLMProvider
  - internal/domain/models.go — TokenUsage, ModelPricing, CostSnapshot, Diff
  - internal/infrastructure/costtracker/costtracker.go — CostTracker core
  - internal/infrastructure/llm/openai_chat.go — GenerateWithUsage, StreamUsage
  - internal/infrastructure/llm/anthropic.go — GenerateWithUsage, StreamUsage
  - internal/infrastructure/llm/openai_compatible_responses.go — GenerateWithUsage, StreamUsage
  - internal/infrastructure/llm/mistral.go — StreamUsage delegation
  - internal/infrastructure/llm/deepseek.go — StreamUsage delegation
  - pkg/draftrag/draftrag.go — re-exports
  - pkg/draftrag/costtracker.go — public CostTracker
  - internal/infrastructure/costtracker/costtracker_test.go — 10 unit tests
  - examples/cost-tracking/main.go — demo

## Verdict

- status: pass
- archive_readiness: safe
- summary: все 11 задач выполнены, unit-тесты (10), race test, build, vet — pass

## Checks

- task_state: completed=11, open=0
- acceptance_evidence:
  - AC-001 -> T1.2, T2.1, T4.1: TestCostTracker_AccumulatesUsage, TestCostTracker_ConcurrentSafety, TestGenerateWithUsage_OpenAIChat, TestGenerateWithUsage_Anthropic, TestGenerateWithUsage_OpenAICompatibleResponses
  - AC-002 -> T1.2, T2.1, T4.1: TestCostTracker_CalculatesCost
  - AC-003 -> T1.2, T4.1: TestCostTracker_ConcurrentSafety (go test -race)
  - AC-004 -> T1.2, T4.1: TestCostTracker_ResetClearsStats
  - AC-005 -> T3.4, T4.1: TestCostTracker_GenerateStreamWithUsage, TestGenerateStreamWithUsage_OpenAIChat, TestGenerateStreamWithUsage_Anthropic, TestGenerateStreamWithUsage_OpenAICompatibleResponses
  - AC-006 -> T1.2, T4.1: TestCostTracker_NonUsageProvider
  - AC-007 -> T1.2, T4.1: TestCostTracker_CheckpointAndDiff
- implementation_alignment:
  - CostTracker.Generate вызывает GenerateWithUsage и accumulate — T1.2
  - CostTracker.GenerateStream извлекает usage после закрытия канала — T3.4
  - OpenAIChatLLM.GenerateWithUsage парсит ChatResponse.Usage — T3.2
  - ClaudeLLM.GenerateWithUsage парсит Message.Usage — T3.1
  - OpenAICompatibleResponsesLLM.GenerateWithUsage парсит Response.Usage — T2.1
  - Mistral/DeepSeek делегируют StreamUsage() через OpenAIChatLLM.impl — T3.3
  - Snapshot/Checkpoint/Reset под sync.Mutex — T1.2
  - Diff — свободная функция, value copy — T1.1

## Errors

- none

## Warnings

- none

## Questions

- none

## Not Verified

- Ollama (локальный, не возвращает usage) — graceful degradation через fallback (non-UsageAware) покрыто тестом TestCostTracker_NonUsageProvider
- Интеграционные тесты с реальными API — только unit с mock, что соответствует плану

## Next Step

- safe to archive
