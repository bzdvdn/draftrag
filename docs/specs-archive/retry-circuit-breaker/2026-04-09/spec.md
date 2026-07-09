# Retry / circuit breaker

## Scope Snapshot

- In scope: обёртки `RetryEmbedder` и `RetryLLMProvider` с exponential backoff и circuit breaker для отказоустойчивости внешних провайдеров
- Out of scope: изменения самих интерфейсов `Embedder`/`LLMProvider`, retry для других компонентов (VectorStore, Chunker)

## Цель

Пользователи библиотеки draftRAG получают стабильную работу с внешними API (эмбеддинги и LLM) даже при временных сбоях сети или перегрузке провайдеров. Обёртки автоматически повторяют неудачные запросы с экспоненциальной задержкой и разрывают цепь при критическом количестве ошибок, предотвращая каскадные отказы.

## Основной сценарий

1. Пользователь создаёт базовый `Embedder` или `LLMProvider` (например, OpenAI клиент)
2. Оборачивает его в `RetryEmbedder(embedder, maxRetries, backoff)` или `RetryLLMProvider(llm, maxRetries, backoff)`
3. При вызове `Embed()` или `Generate()` с временной ошибкой (timeout, 5xx, network error) происходит автоматический retry
4. После превышения `maxRetries` возвращается последняя ошибка
5. При множестве последовательных ошибок circuit breaker переходит в состояние "open" и быстро отклоняет запросы
6. По истечении timeout circuit breaker пробует восстановление (half-open) и закрывается при успехе

## Scope

- Реализация `RetryEmbedder` — декоратор для `Embedder` с retry-логикой
- Реализация `RetryLLMProvider` — декоратор для `LLMProvider` с retry-логикой
- Exponential backoff с настраиваемыми параметрами (base delay, max delay, jitter)
- Circuit breaker с состояниями (closed, open, half-open) и настраиваемыми порогами
- Интеграция с `Hooks` для observability (события retry, circuit breaker transitions)
- Поддержка `context.Context` для cancellation и timeout
- Различение retryable и non-retryable ошибок

## Контекст

- Интерфейсы `Embedder` и `LLMProvider` определены в `internal/domain/interfaces.go`
- Интерфейс `Hooks` доступен для интеграции observability
- Go 1.21+ с поддержкой `context`, `sync/atomic` для thread-safe реализации
- Существующие реализации (OpenAI, Anthropic) используют эти интерфейсы — обёртки должны быть прозрачны
- Clean Architecture: обёртки находятся в infrastructure-слое или как отдельные utility-компоненты

## Требования

- RQ-001 `RetryEmbedder` ДОЛЖЕН реализовывать интерфейс `Embedder` и быть прозрачной обёрткой
- RQ-002 `RetryLLMProvider` ДОЛЖЕН реализовывать интерфейс `LLMProvider` и быть прозрачной обёрткой
- RQ-003 Retry ДОЛЖЕН использовать exponential backoff с jitter для предотвращения thundering herd
- RQ-004 Retry ДОЛЖЕН уважать `context.Context` — отмена контекста прерывает retry-цикл
- RQ-005 Circuit breaker ДОЛЖЕН иметь три состояния: closed (нормальная работа), open (блокировка), half-open (пробное восстановление)
- RQ-006 Circuit breaker ДОЛЖЕН переходить в open при достижении порога ошибок за окно времени
- RQ-007 Circuit breaker ДОЛЖЕН автоматически переходить в half-open по таймауту восстановления
- RQ-008 При наличии `Hooks` обёртки ДОЛЖНЫ вызывать `StageStart`/`StageEnd` с информацией о попытках и состоянии circuit breaker
- RQ-009 Настройки ДОЛЖНЫ иметь разумные значения по умолчанию (maxRetries=3, baseDelay=100ms)

## Вне scope

- Retry для `VectorStore` операций (отдельная фича)
- Retry для `StreamingLLMProvider` (может быть добавлено позже)
- Persistence состояния circuit breaker между перезапусками
- Distributed circuit breaker (для множества инстансов)
- Retry на уровне HTTP-клиента (это ответственность конкретных провайдеров)
- Метрики и алертинг (базовая observability через Hooks — в scope, продвинутые метрики — нет)

## Критерии приемки

### AC-001 RetryEmbedder успешный retry

- Почему это важно: пользователь не должен получать ошибки из-за временных сбоев сети
- **Given** базовый `Embedder` возвращает ошибку на первой попытке, но успех на второй
- **When** вызывается `RetryEmbedder.Embed(ctx, text)` с `maxRetries >= 2`
- **Then** метод возвращает успешный результат без ошибки пользователю
- Evidence: тест показывает 2 вызова базового embedder, 1 успешный результат

### AC-002 RetryLLMProvider исчерпание попыток

- Почему это важно: при постоянном сбое система должна честно сообщить об ошибке
- **Given** базовый `LLMProvider` возвращает ошибку на всех попытках
- **When** вызывается `RetryLLMProvider.Generate(ctx, system, user)` с `maxRetries=3`
- **Then** метод возвращает последнюю ошибку после 3 попыток
- Evidence: тест подтверждает ровно 3 вызова базового провайдера и возврат ошибки

### AC-003 Circuit breaker блокировка

- Почему это важно: предотвращает каскадные отказы при длительной недоступности провайдера
- **Given** circuit breaker в состоянии closed, порог ошибок = 5 за 10 секунд
- **When** происходит 5 последовательных ошибок от базового провайдера
- **Then** circuit breaker переходит в open и последующие вызовы немедленно возвращают ошибку без обращения к базовому провайдеру
- Evidence: тест показывает переход состояния и подсчёт запросов к базовому провайдеру (ровно 5)

### AC-004 Circuit breaker восстановление

- Почему это важно: система должна автоматически восстанавливаться после временного сбоя
- **Given** circuit breaker в состоянии open, timeout восстановления = 5 секунд
- **When** прошло 5 секунд и пришёл новый запрос
- **Then** circuit breaker переходит в half-open, выполняет пробный запрос, и при успехе — возвращается в closed
- Evidence: тест с mock clock показывает переходы состояний и успешный запрос в half-open

### AC-005 Context cancellation

- Почему это важно: пользователь должен иметь контроль над выполнением через timeout/cancellation
- **Given** активный `context.Context` с заданным timeout
- **When** timeout истекает во время retry-задержки между попытками
- **Then** retry-цикл прерывается и возвращается `context.DeadlineExceeded`
- Evidence: тест показывает отмену retry-цикла без лишних попыток после cancellation

### AC-006 Observability через Hooks

- Почему это важно: операторы должны видеть retry-попытки и состояние системы
- **Given** настроенный `Hooks` и обёртка с retry
- **When** происходит retry или переход circuit breaker
- **Then** hooks получают события с информацией о количестве попыток, задержках и состояниях
- Evidence: тест с mock hooks подтверждает вызовы с правильными аргументами

## Допущения

- Базовые провайдеры возвращают retryable ошибки через стандартные Go error patterns (можно определить интерфейс `RetryableError`)
- HTTP-статусы 5xx, timeouts, temporary network errors — retryable; 4xx клиентские ошибки — не retryable
- Один circuit breaker на один экземпляр обёртки (не shared state)
- Thread-safe доступ к circuit breaker обеспечивается через `sync.RWMutex` или `sync/atomic`
- Jitter для backoff — случайная составляющая до 25% от задержки
- Восстановление circuit breaker — один пробный запрос в half-open, при успехе — closed, при ошибке — снова open

## Критерии успеха

- SC-001 Время ответа при ошибке (с retry) не превышает `maxRetries * (maxBackoff + jitter)` + базовое время запроса
- SC-002 Circuit breaker в open состоянии отклоняет запросы за <1ms (без обращения к базовому провайдеру)
- SC-003 Покрытие unit-тестами ≥80% для retry-логики и circuit breaker

## Краевые случаи

- Первый запрос после старта — всегда через базовый провайдер
- Отрицательное или нулевое `maxRetries` — обработать как "без retry"
- Context cancelled до первой попытки — немедленный возврат без вызова базового провайдера
- Circuit breaker в open — вызов hook `StageStart` с флагом rejected, затем быстрый return
- Concurrent запросы при half-open — только один должен быть пробным, остальные ждут или отклоняются

## Открытые вопросы

- none
