# Ollama LLM + Embedder Модель данных

## Scope

- Связанные `AC-*`: AC-001, AC-002, AC-003, AC-004, AC-005
- Для этой фичи значимого persisted data model не требуется — только transient request/response структуры

## Сущности

### DM-001 Ollama Chat Request

- **Назначение**: Структура запроса к `POST /api/chat`
- **Источник истины**: Код конструирует перед HTTP-запросом
- **Инварианты**: `Model` не пустой; `Messages` содержит хотя бы одно сообщение; `Stream: false` для синхронного вызова
- **Связанные `AC-*`**: AC-001, AC-005
- **Поля**:
  - `Model` (string, required) — имя модели в Ollama (например, `llama3.2`)
  - `Messages` (array of `{Role, Content}`, required) — история сообщений
  - `Stream` (bool, required, default false) — для синхронного вызова всегда `false`
  - `Temperature` (float64, optional) — параметр сэмплирования
  - `MaxTokens` (int, optional) — максимальное количество токенов

### DM-002 Ollama Chat Response

- **Назначение**: Структура ответа от `POST /api/chat`
- **Источник истины**: Ollama API
- **Инварианты**: `Message.Content` не пустой при успешном ответе
- **Связанные `AC-*`**: AC-001
- **Поля**:
  - `Message` (object, required) — содержит `Role` и `Content`
  - `Message.Role` (string) — роль ответа (обычно `assistant`)
  - `Message.Content` (string, required) — сгенерированный текст

### DM-003 Ollama Embeddings Request

- **Назначение**: Структура запроса к `POST /api/embeddings`
- **Источник истины**: Код конструирует перед HTTP-запросом
- **Инварианты**: `Model` и `Prompt` не пустые
- **Связанные `AC-*`**: AC-002, AC-005
- **Поля**:
  - `Model` (string, required) — имя embedding-модели (например, `nomic-embed-text`)
  - `Prompt` (string, required) — текст для эмбеддинга (в Ollama это `prompt` вместо `input`)

### DM-004 Ollama Embeddings Response

- **Назначение**: Структура ответа от `POST /api/embeddings`
- **Источник истины**: Ollama API
- **Инварианты**: `Embedding` — массив float64, все значения finite (не NaN, не Inf)
- **Связанные `AC-*`**: AC-002
- **Поля**:
  - `Embedding` ([]float64, required) — векторное представление текста

## Связи

Нет межсущностных связей — структуры независимые и используются только для сериализации/десериализации HTTP payload.

## Производные правила

- Валидация embedding-вектора: все значения должны быть finite (проверка `math.IsNaN` и `math.IsInf` как в OpenAI-совместимой реализации)
- Normalization не требуется — Ollama возвращает уже нормализованные векторы для моделей, которые это поддерживают

## Переходы состояний

Не применимо — нет persisted состояния, только request-response цикл.

## Вне scope

- Streaming response структуры (Ollama поддерживает stream, но вне scope этой фичи)
- Multi-modal input (images в messages)
- Advanced options (format, keep_alive, options dict)
- Персистентное состояние сессии или кеширование
