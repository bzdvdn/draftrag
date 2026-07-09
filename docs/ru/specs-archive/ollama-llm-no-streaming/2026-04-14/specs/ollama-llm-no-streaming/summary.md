---
slug: ollama-llm-no-streaming
generated_at: 2026-04-14
---

## Goal

Разработчики получают возможность использовать локальные LLM-модели через Ollama API для генерации текста в RAG-системах без streaming.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Реализация LLMProvider интерфейса | OllamaLLM реализует метод Generate |
| AC-002 | Отключение streaming | Код содержит `Stream: false` в запросе |
| AC-003 | Валидация конфигурации | validateOllamaLLMOptions проверяет параметры |
| AC-004 | Поддержка context и timeout | Generate использует context.WithTimeout |
| AC-005 | Обработка ошибок Ollama API | Код проверяет resp.StatusCode и возвращает ошибку |
| AC-006 | Дефолтные значения конфигурации | BaseURL устанавливается в http://localhost:11434 |

## Out of Scope

- Streaming-режим для Ollama (GenerateStream)
- Поддержка других параметров Ollama API (top_p, repeat_penalty и т.д.)
- Автоматический retry при сетевых ошибках
- Кеширование ответов
