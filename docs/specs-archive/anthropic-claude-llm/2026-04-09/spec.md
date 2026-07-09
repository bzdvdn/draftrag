# Нативная поддержка Anthropic Claude LLM

## Scope Snapshot

- In scope: Реализация нативного клиента для Anthropic Messages API с поддержкой `anthropic-version` заголовка
- Out of scope: OpenAI-совместимый режим Anthropic, адаптеры для других LLM-провайдеров

## Цель

Разработчики получают возможность использовать Claude модели через нативный Anthropic API (не через OpenAI-совместимый эндпоинт). Это обеспечивает доступ к специфичным возможностям Anthropic (extended thinking, tool use, computer use) и корректную обработку `anthropic-version` заголовка для стабильности API.

## Основной сценарий

1. Разработчик создаёт клиент `anthropic.NewClaudeLLM()` с API ключом и моделью
2. Клиент использует `https://api.anthropic.com/v1/messages` эндпоинт с `anthropic-version` заголовком
3. При вызове `Generate()` клиент отправляет нативный Anthropic-формат запроса
4. При вызове `GenerateStream()` клиент получает SSE-стрим в Anthropic-формате
5. Реализация удовлетворяет интерфейсам `LLMProvider` и `StreamingLLMProvider`

## Scope

- Клиент для `https://api.anthropic.com/v1/messages`
- Поддержка `anthropic-version` заголовка (значение по умолчанию: `2023-06-01`)
- Реализация `LLMProvider.Generate()` через нативный Anthropic API
- Реализация `StreamingLLMProvider.GenerateStream()` с SSE парсингом
- Структуры запроса/ответа для Anthropic Messages API
- Unit-тесты с мок-сервером

## Контекст

- Существующий `OpenAICompatibleResponsesLLM` использует OpenAI-формат (`/v1/responses`)
- Anthropic Messages API имеет отличную структуру запроса/ответа от OpenAI
- `anthropic-version` заголовок обязателен для стабильной работы API
- Интерфейс `LLMProvider` определён в `internal/domain/interfaces.go:44-48`
- Интерфейс `StreamingLLMProvider` определён в `internal/domain/interfaces.go:56-62`
- Реализации размещаются в `internal/infrastructure/llm/`

## Требования

- **RQ-001** Клиент ДОЛЖЕН использовать эндпоинт `https://api.anthropic.com/v1/messages`
- **RQ-002** Каждый запрос ДОЛЖЕН содержать заголовок `anthropic-version` с валидным значением
- **RQ-003** Клиент ДОЛЖЕН реализовывать интерфейс `domain.LLMProvider`
- **RQ-004** Клиент ДОЛЖЕН реализовывать интерфейс `domain.StreamingLLMProvider`
- **RQ-005** Запрос ДОЛЖЕН поддерживать `system` сообщение через поле `system` (не через массив messages)
- **RQ-006** Запрос ДОЛЖЕН поддерживать `max_tokens` с разумным значением по умолчанию (например, 1024)

## Вне scope

- Поддержка tool use / function calling
- Поддержка extended thinking / reasoning
- Поддержка vision (image input)
- Computer use API
- Prompt caching
- Batch API
- Fine-tuning

## Критерии приемки

### AC-001 Базовая генерация текста

- **Почему важно:** Core capability для ответов в RAG-системе
- **Given:** Клиент инициализирован с валидным API ключом и моделью `claude-3-haiku-20240307`
- **When:** Вызывается `Generate(ctx, "System prompt", "User question")`
- **Then:** Возвращается непустой текстовый ответ без ошибок
- **Evidence:** Тест `TestClaudeLLM_Generate_Success` проходит

### AC-002 Корректный Anthropic-формат запроса

- **Почему важно:** Нативный API требует специфичной структуры запроса
- **Given:** Перехвачен HTTP-запрос к API
- **When:** Клиент вызывает `Generate()`
- **Then:** Тело запроса содержит поля `model`, `max_tokens`, `system`, `messages` в Anthropic-формате
- **Evidence:** Тест проверяет JSON-структуру запроса

### AC-003 Заголовок anthropic-version

- **Почему важно:** API требует этот заголовок для версионирования
- **Given:** Перехвачен HTTP-запрос к API
- **When:** Клиент выполняет любой запрос
- **Then:** Заголовок `anthropic-version` присутствует и содержит валидное значение
- **Evidence:** Тест проверяет наличие и значение заголовка

### AC-004 Streaming поддержка

- **Почему важно:** Потоковая генерация улучшает UX для длинных ответов
- **Given:** Клиент инициализирован с поддержкой streaming
- **When:** Вызывается `GenerateStream(ctx, "System", "User")`
- **Then:** Возвращается канал, который получает текстовые чанки по мере генерации
- **Evidence:** Тест `TestClaudeLLM_GenerateStream_Success` проходит, канал получает >1 чанка

### AC-005 Обработка ошибок API

- **Почему важно:** Надёжность при сбоях внешнего сервиса
- **Given:** API возвращает HTTP 401 или 429
- **When:** Клиент выполняет запрос
- **Then:** Возвращается ошибка с деталями из ответа API, API ключ редатирован в ошибке
- **Evidence:** Тесты на ошибочные статусы проходят

## Допущения

- Пользователь предоставляет валидный Anthropic API ключ
- Модель по умолчанию: `claude-3-haiku-20240307` (быстрая и экономичная)
- Значение `anthropic-version` по умолчанию: `2023-06-01`
- `max_tokens` по умолчанию: `1024`
- Таймаут HTTP-клиента управляется через `context.Context`
- SSE формат Anthropic совместим с базовым SSE парсингом (data: {...})
- API доступен и стабилен (нет retry-логики в первой версии)

## Краевые случаи

- Пустой `userMessage`: возвращается ошибка
- Пустой `systemPrompt`: поле `system` не включается в запрос
- Пустой ответ от API: возвращается ошибка "empty response"
- `anthropic-version` не задан: используется значение по умолчанию
- Контекст отменён во время streaming: канал закрывается корректно

## Открытые вопросы

none
