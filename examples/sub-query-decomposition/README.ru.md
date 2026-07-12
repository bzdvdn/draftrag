# Sub-Query Decomposition Example

Демонстрирует разбиение сложного запроса на под-вопросы
через `SearchBuilder.SubDecompose()` для улучшения recall.

## Запуск

```bash
# С mock-провайдером (без внешних зависимостей)
LLM_PROVIDER=mock go run .
```

Для использования реальных LLM/embedder провайдеров установите
`LLM_PROVIDER=ollama` или `LLM_PROVIDER=openai` с соответствующими
переменными окружения (см. примеры `examples/memory/`).
