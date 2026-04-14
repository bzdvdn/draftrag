---
slug: anthropic-claude-llm
status: completed
archived_at: 2026-04-09
---

## Goal

Нативный клиент для Anthropic Messages API с поддержкой `anthropic-version` заголовка.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Базовая генерация текста | Тест `TestClaudeLLM_Generate_Success` проходит |
| AC-002 | Корректный Anthropic-формат | Тест проверяет JSON-структуру запроса |
| AC-003 | Заголовок anthropic-version | Тест проверяет наличие и значение заголовка |
| AC-004 | Streaming поддержка | Тест `TestClaudeLLM_GenerateStream_Success` проходит |
| AC-005 | Обработка ошибок API | Тесты на 401/429 проходят, ключ редатирован |

## Out of Scope

- Tool use / function calling
- Extended thinking / reasoning
- Vision (image input)
- Computer use API
- Prompt caching, Batch API, Fine-tuning
