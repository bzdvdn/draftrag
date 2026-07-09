# Ollama LLM: без streaming Задачи

## Phase Contract

Inputs: plan и существующий код
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/llm/ollama.go | T1.1, T2.1 |
| pkg/draftrag/ollama_llm.go | T1.2, T2.2 |
| internal/infrastructure/llm/ollama_test.go | T3.1 |

## Фаза 1: Основа

Цель: подтвердить существующую структуру реализации.

- [x] T1.1 Подтвердить реализацию LLMProvider интерфейса — OllamaLLM содержит метод Generate с корректной сигнатурой. Touches: internal/infrastructure/llm/ollama.go — AC-001, DEC-001
- [x] T1.2 Подтвердить публичный API — NewOllamaLLM фабрика и OllamaLLMOptions структура существуют. Touches: pkg/draftrag/ollama_llm.go — AC-001, DEC-002

## Фаза 2: Основная реализация

Цель: подтвердить основное поведение и валидацию.

- [x] T2.1 Подтвердить отключение streaming — ollamaChatRequest содержит Stream: false. Touches: internal/infrastructure/llm/ollama.go — AC-002, DEC-001
- [x] T2.2 Подтвердить валидацию конфигурации — validateOllamaLLMOptions проверяет параметры. Touches: pkg/draftrag/ollama_llm.go — AC-003, DEC-002

## Фаза 3: Проверка

Цель: подтвердить тестовое покрытие и обработку ошибок.

- [x] T3.1 Подтвердить unit-тесты — тесты покрывают Generate, валидацию URL и обработку ошибок. Touches: internal/infrastructure/llm/ollama_test.go — AC-004, AC-005, AC-006

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2
- AC-002 -> T2.1
- AC-003 -> T2.2
- AC-004 -> T3.1
- AC-005 -> T3.1
- AC-006 -> T3.1

## Заметки

- Реализация уже существует, задачи отмечены как выполненные
- Все задачи ориентированы на подтверждение существующего кода
- Unit-тесты уже покрывают основные сценарии
