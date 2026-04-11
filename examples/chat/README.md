# chat — Базовый RAG-чат (CLI)

Интерактивный RAG-чат с in-memory хранилищем. Загружает набор документов, после чего принимает вопросы через stdin и отвечает с inline-цитатами `[1]`, `[2]`.

Хранилище живёт в памяти — данные не сохраняются между запусками.

## Быстрый старт

```bash
EMBEDDER_API_KEY=sk-... \
LLM_API_KEY=sk-... \
go run ./examples/chat/
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `EMBEDDER_API_KEY` | — | **Обязательно.** Ключ API для embedder |
| `EMBEDDER_BASE_URL` | `https://api.openai.com` | Базовый URL embedder API |
| `EMBEDDER_MODEL` | `text-embedding-ada-002` | Модель эмбеддингов |
| `LLM_API_KEY` | — | **Обязательно.** Ключ API для LLM |
| `LLM_BASE_URL` | `https://api.openai.com` | Базовый URL LLM API |
| `LLM_MODEL` | `gpt-4o-mini` | Языковая модель |

## Локальный режим (Ollama)

```bash
# Запустите Ollama и скачайте нужные модели:
ollama pull nomic-embed-text
ollama pull llama3.2

EMBEDDER_BASE_URL=http://localhost:11434 \
EMBEDDER_API_KEY=ollama \
EMBEDDER_MODEL=nomic-embed-text \
LLM_BASE_URL=http://localhost:11434 \
LLM_API_KEY=ollama \
LLM_MODEL=llama3.2 \
go run ./examples/chat/
```

## Пример сессии

```
Индексируем базу знаний...
Проиндексировано 8 документов.

RAG-чат готов. Введите вопрос (Ctrl+C для выхода):
────────────────────────────────────────────────────────────

> Как добавить Zigbee-устройство?

Чтобы добавить Zigbee-устройство, откройте приложение SmartHome,
выберите «Добавить устройство» → «Zigbee» и переведите устройство
в режим сопряжения [1]. Хаб обнаружит его в течение 30 секунд [1].

Источники:
  [1] smarthome-zigbee (score=0.921)
────────────────────────────────────────────────────────────
```

## База знаний

Пример использует встроенную базу знаний о продукте SmartHome Hub (8 документов). Чтобы использовать собственные документы — замените срез `knowledgeBase` в `main.go`.
