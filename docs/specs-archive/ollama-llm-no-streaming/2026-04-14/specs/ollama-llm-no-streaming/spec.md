# Ollama LLM: без streaming

## Scope Snapshot

- In scope: реализация LLMProvider для локального Ollama API без поддержки streaming
- Out of scope: streaming-режим для Ollama, поддержка других LLM-провайдеров

## Цель

Разработчики получают возможность использовать локальные LLM-модели через Ollama API для генерации текста в RAG-системах. Реализация предоставляет базовый интерфейс LLMProvider с методом Generate без потоковой передачи ответов.

## Основной сценарий

1. Разработчик создаёт OllamaLLM через NewOllamaLLM с опциями (BaseURL, Model, Temperature, MaxTokens)
2. Разработчик вызывает Generate с systemPrompt и userMessage
3. OllamaLLM отправляет HTTP-запрос к Ollama Chat API с stream=false
4. Ответ возвращается как полная строка текста

## Scope

- Реализация LLMProvider для Ollama Chat API в internal/infrastructure/llm/ollama.go
- Публичный API в pkg/draftrag/ollama_llm.go с фабрикой NewOllamaLLM
- Конфигурация через OllamaLLMOptions (BaseURL, Model, Temperature, MaxTokens, Timeout)
- Валидация входных параметров и обработка ошибок
- Поддержка context.Context для отмены и таймаутов

## Контекст

- Ollama API работает локально по умолчанию на http://localhost:11434
- LLMProvider — базовый интерфейс без streaming, StreamingLLMProvider — опциональная capability
- Реализация использует HTTP POST к /api/chat endpoint
- Ответ Ollama содержит полный текст в одном сообщении (stream=false)

## Требования

- RQ-001 OllamaLLM ДОЛЖЕН реализовывать интерфейс LLMProvider с методом Generate
- RQ-002 Generate ДОЛЖЕН принимать systemPrompt и userMessage как отдельные аргументы
- RQ-003 Generate ДОЛЖЕН отправлять HTTP-запрос к Ollama Chat API с stream=false
- RQ-004 Generate ДОЛЖЕН возвращать полный текст ответа как строку
- RQ-005 Конфигурация ДОЛЖНА поддерживать BaseURL, Model, Temperature, MaxTokens, Timeout
- RQ-006 Generate ДОЛЖЕН поддерживать context.Context для отмены и таймаутов
- RQ-007 Generate ДОЛЖЕН возвращать ошибку при пустом userMessage или неверной конфигурации

## Вне scope

- Streaming-режим для Ollama (GenerateStream)
- Поддержка других параметров Ollama API (top_p, repeat_penalty и т.д.)
- Автоматический retry при сетевых ошибках
- Кеширование ответов

## Критерии приемки

### AC-001 Реализация LLMProvider интерфейса

- Почему это важно: обеспечивает совместимость с существующей архитектурой draftRAG
- **Given** разработчик создаёт OllamaLLM через NewOllamaLLM с валидными параметрами
- **When** разработчик вызывает Generate с systemPrompt и userMessage
- **Then** метод возвращает строку с генерированным текстом без ошибок
- Evidence: OllamaLLM реализует метод Generate, соответствующий сигнатуре LLMProvider

### AC-002 Отключение streaming

- Почему это важно: обеспечивает детерминированное поведение для базового сценария
- **Given** OllamaLLM отправляет запрос к Ollama API
- **When** запрос формируется в Generate
- **Then** поле stream в теле запроса установлено в false
- Evidence: код ollama.go содержит `Stream: false` в ollamaChatRequest

### AC-003 Валидация конфигурации

- Почему это важно: предотвращает ошибки конфигурации на ранней стадии
- **Given** разработчик создаёт OllamaLLM с некорректными параметрами
- **When** разработчик вызывает Generate
- **Then** метод возвращает ошибку ErrInvalidLLMConfig с описанием проблемы
- Evidence: validateOllamaLLMOptions проверяет Model, Timeout, Temperature, MaxTokens

### AC-004 Поддержка context и timeout

- Почему это важно: обеспечивает корректную работу с отменой и таймаутами
- **Given** разработчик вызывает Generate с context и настроенным Timeout
- **When** операция превышает таймаут или context отменяется
- **Then** метод возвращает context error или timeout error
- Evidence: Generate использует context.WithTimeout при opts.Timeout > 0

### AC-005 Обработка ошибок Ollama API

- Почему это важно: обеспечивает информативные сообщения при сбоях
- **Given** Ollama API возвращает HTTP статус вне диапазона 200-299
- **When** Generate обрабатывает ответ
- **Then** метод возвращает ошибку с кодом статуса и фрагментом тела ответа
- Evidence: код проверяет resp.StatusCode и формирует ошибку с body snippet

### AC-006 Дефолтные значения конфигурации

- Почему это важно: упрощает базовое использование без явной конфигурации
- **Given** разработчик создаёт OllamaLLM с пустым BaseURL
- **When** NewOllamaLLM инициализирует структуру
- **Then** BaseURL устанавливается в http://localhost:11434
- Evidence: код использует ollamaDefaultBaseURL при пустом baseURL

## Допущения

- Ollama API доступен по указанному BaseURL и отвечает корректно
- Модель указана корректно и существует в Ollama
- HTTP-клиент работает без прокси или с корректной конфигурацией прокси
- Время ответа Ollama API укладывается в разумные пределы для RAG-сценариев

## Краевые случаи

- Пустой userMessage: возвращается ошибка "userMessage is empty"
- Пустой ответ от Ollama: возвращается ошибка "invalid ollama response: empty message content"
- Network timeout: возвращается context error или timeout error
- Неверный BaseURL (без scheme/host): возвращается ошибка "invalid BaseURL"
- Нулевой context: вызывается panic (защита от misuse)

## Открытые вопросы

none
