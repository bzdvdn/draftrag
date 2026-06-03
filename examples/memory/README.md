# Memory example

In-memory RAG-пример на базе Go-документации. Не требует Docker или внешних сервисов.

## Быстрый старт

```bash
cd examples/memory && cp .env.example .env && go run .
```

## Переменные окружения

Базовые (из `.env.example`):

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `LLM_PROVIDER` | `mock` | LLM провайдер (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Размерность эмбеддингов |

Для реального LLM (ollama/openai/anthropic) потребуются дополнительные переменные — см. [examples/shared/config.go](../shared/config.go).

## Что делает пример

1. Создаёт in-memory векторное хранилище
2. Индексирует 10 документов по языку Go (горутины, каналы, контекст, интерфейсы и др.)
3. Задаёт вопрос "Что такое goroutine?"
4. Выводит ответ с источниками

## Требования

- Go 1.21+
- Для `LLM_PROVIDER=mock` — ничего дополнительно
- Для `LLM_PROVIDER=ollama` — запущенный [Ollama](https://ollama.ai) с моделями
- Для `LLM_PROVIDER=openai` — `OPENAI_API_KEY`
- Для `LLM_PROVIDER=anthropic` — `ANTHROPIC_API_KEY`
