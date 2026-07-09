# Ollama LLM: без streaming План

## Phase Contract

Inputs: spec и существующий код в internal/infrastructure/llm/ollama.go и pkg/draftrag/ollama_llm.go
Outputs: plan, data model
Stop if: spec слишком расплывчата для безопасного планирования

## Цель

Документировать существующую реализацию Ollama LLM без streaming, которая уже реализует интерфейс LLMProvider для локального Ollama API. Реализация предоставляет базовый интерфейс с методом Generate без потоковой передачи ответов.

## Scope

- Инфраструктурный слой: internal/infrastructure/llm/ollama.go
- Публичный API: pkg/draftrag/ollama_llm.go
- Конфигурация через OllamaLLMOptions
- Валидация параметров и обработка ошибок
- Поддержка context.Context и timeout
- Явно остаётся нетронутым: streaming-режим (GenerateStream), другие параметры Ollama API

## Implementation Surfaces

- internal/infrastructure/llm/ollama.go — существующая поверхность, содержит реализацию OllamaLLM с методом Generate
- pkg/draftrag/ollama_llm.go — существующая поверхность, содержит публичный API с фабрикой NewOllamaLLM и валидацией
- domain/interfaces.go — существующая поверхность, содержит интерфейс LLMProvider (не меняется)

## Влияние на архитектуру

- Локальное влияние: код уже реализован в infrastructure слое, соответствует Clean Architecture
- Влияние на интеграции: OllamaLLM реализует LLMProvider интерфейс, совместим с существующей архитектурой
- Migration/compatibility: не требуется, код уже существует и используется

## Acceptance Approach

- AC-001 -> реализация уже существует в ollama.go, метод Generate соответствует сигнатуре LLMProvider
- AC-002 -> код содержит `Stream: false` в ollamaChatRequest, streaming отключён
- AC-003 -> validateOllamaLLMOptions в ollama_llm.go проверяет Model, Timeout, Temperature, MaxTokens
- AC-004 -> Generate использует context.WithTimeout при opts.Timeout > 0
- AC-005 -> Generate проверяет resp.StatusCode и возвращает ошибку с body snippet
- AC-006 -> NewOllamaLLM использует ollamaDefaultBaseURL при пустом baseURL

## Данные и контракты

- Эта фича не вводит новых persisted сущностей или state transitions
- Эта фича не вводит новых API или event boundaries
- Использует существующий интерфейс LLMProvider из domain/interfaces.go
- Использует HTTP для коммуникации с Ollama API (внешний контракт)

## Стратегия реализации

### DEC-001 Использование стандартного HTTP клиента для Ollama API

- Why: стандартный http.Client обеспечивает совместимость с прокси, timeouts и cancellation через context
- Tradeoff: нет дополнительной абстракции, но требуется явная обработка ошибок HTTP
- Affects: internal/infrastructure/llm/ollama.go
- Validation: unit-тесты в ollama_test.go покрывают успешные и ошибочные сценарии

### DEC-002 Валидация конфигурации в публичном API

- Why: валидация на уровне публичного API обеспечивает более понятные ошибки для пользователя
- Tradeoff: дублирование части валидации, но улучшает DX
- Affects: pkg/draftrag/ollama_llm.go
- Validation: validateOllamaLLMOptions проверяет все обязательные и ограниченные параметры

## Incremental Delivery

Реализация уже существует, инкрементальная доставка не требуется.

## Порядок реализации

Реализация уже завершена. Порядок был:
1. Реализация OllamaLLM в infrastructure слое
2. Реализация публичного API с валидацией
3. Добавление unit-тестов

## Риски

- Риск 1: Ollama API недоступен или отвечает некорректно
  Mitigation: код возвращает информативные ошибки с фрагментом тела ответа
- Риск 2: Таймауты не настроены для медленных моделей
  Mitigation: пользователь может настроить Timeout через OllamaLLMOptions

## Rollout и compatibility

- Специальных rollout-действий не требуется, код уже существует
- Обратная совместимость сохраняется, интерфейс LLMProvider не меняется
- Monitoring: пользователь может использовать context.WithTimeout для контроля времени выполнения

## Проверка

- Unit-тесты в internal/infrastructure/llm/ollama_test.go покрывают Generate
- Unit-тесты в internal/infrastructure/llm/ollama_test.go покрывают валидацию URL
- Тесты подтверждают AC-001, AC-002, AC-005, AC-006

## Соответствие конституции

- [CONST-ARCH] Чистая архитектура: реализация в infrastructure слое, зависит только от domain интерфейсов
- [CONST-LANG] Язык Go 1.23+: код использует стандартную библиотеку и http.Client
- [CONST-INTERFACE] Интерфейсная абстракция: OllamaLLM реализует LLMProvider интерфейс
- [CONST-CONTEXT] Контекстная безопасность: Generate принимает context.Context и поддерживает отмену
- [CONST-TEST] Тестируемость: есть mock-реализация в mock_streaming.go для тестирования
