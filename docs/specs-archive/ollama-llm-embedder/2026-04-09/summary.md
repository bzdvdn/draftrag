---
slug: ollama-llm-embedder
generated_at: 2026-04-09T01:25:00+03:00
---

## Goal
Поддержка локальных LLM и embedding-моделей через Ollama API для оффлайн и приватных сценариев.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | LLM-генерация через Ollama | Unit-test с мок-сервером проверяет POST /api/chat и парсинг message.content |
| AC-002 | Эмбеддинги через Ollama | Unit-test с мок-сервером проверяет POST /api/embeddings и парсинг embedding |
| AC-003 | Обработка ошибок Ollama | Тесты с мок 404/500, проверка содержимого ошибки |
| AC-004 | Контекстная безопасность | Unit-test с таймаутом, проверка корректного поведения при отмене |
| AC-005 | Валидация входных данных | Unit-тесты проверяют ошибку для пустых строк и nil context |

## Out of Scope

- Streaming-генерация для OllamaLLM
- Multi-modal модели (image input)
- Автоматический pull моделей при отсутствии
- Управление Ollama сервером (запуск/остановка)
- Advanced параметры Ollama (format, options, keep_alive)
