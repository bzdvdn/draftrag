---
slug: streaming-responses
generated_at: 2026-04-08T21:36:00+03:00
---

## Goal

Добавить streaming-генерацию ответов токен за токеном через канал для улучшения UX.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Streaming через канал | Канал `<-chan string` возвращает токены до закрытия |
| AC-002 | Streaming с цитатами | Канал + `[]InlineCitation` с валидными цитатами |
| AC-003 | Отмена контекста | Канал закрывается < 100мс при `ctx.Cancel()` |
| AC-004 | Backward compatibility | `AnswerStream*` возвращает "streaming not supported" для legacy |
| AC-005 | SSE парсинг | Корректная обработка `data:` линий из OpenAI SSE |

## Out of Scope

- WebSocket/SSE сервер (draftRAG — библиотека)
- Streaming для Anthropic/Ollama (только OpenAI-compatible)
- Frontend-фреймворк интеграции
- Streaming без retrieval
