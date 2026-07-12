# Semantic Chunking Example

Демонстрирует использование `SemanticChunker` для интеллектуального
разбиения документов на чанки на основе семантической близости предложений.

## Запуск

```bash
# С mock-провайдером (без внешних зависимостей)
LLM_PROVIDER=mock go run .
```

Для использования реальных LLM/embedder провайдеров установите
`LLM_PROVIDER=ollama` или `LLM_PROVIDER=openai` с соответствующими
переменными окружения (см. примеры `examples/memory/`).
