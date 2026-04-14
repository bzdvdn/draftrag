# Retry / circuit breaker Модель данных

## Scope

- Связанные `AC-*`: AC-003 (Circuit breaker блокировка), AC-004 (Circuit breaker восстановление)
- Связанные `DEC-*`: DEC-002 (Circuit breaker state machine в памяти)
- Эта фича не вводит persisted сущностей — только runtime state в circuit breaker.

## Сущности

### DM-001 CircuitBreaker (runtime state)

- Назначение: отслеживание ошибок и управление состоянием доступности внешнего провайдера
- Источник истины: in-memory структура, создаётся с обёрткой
- Инварианты:
  - Только одно активное состояние в каждый момент: closed, open, half-open
  - Переходы состояний атомарны (защищены mutex)
  - В open — запросы немедленно отклоняются без вызова базового провайдера
- Связанные `AC-*`: AC-003, AC-004
- Связанные `DEC-*`: DEC-002
- Поля:
  - `state` — enum: closed, open, half-open
  - `failureCount` — int, счётчик ошибок в текущем окне
  - `lastFailureTime` — time.Time, timestamp последней ошибки
  - `threshold` — int, порог ошибок для перехода в open
  - `timeout` — time.Duration, время восстановления для перехода в half-open
  - `mu` — sync.RWMutex, защита состояния
- Жизненный цикл:
  - Создаётся в closed при инициализации обёртки
  - Переходит в open при threshold ошибок
  - Автоматически в half-open по timeout
  - Возвращается в closed при успешном пробном запросе
  - Уничтожается вместе с обёрткой (no persistence)
- Замечания по консистентности:
  - Недопустимо: частичный переход состояния (например, изменение failureCount без проверки threshold)
  - Недопустимо: concurrent изменение state без mutex

### DM-002 Backoff (runtime config)

- Назначение: расчёт задержки между retry-попытками
- Источник истины: конфигурация, передаётся в конструктор
- Инварианты:
  - Задержка не превышает maxDelay
  - Jitter добавляется к base задержке
- Связанные `AC-*`: AC-001, AC-002, AC-005
- Связанные `DEC-*`: DEC-003
- Поля:
  - `baseDelay` — time.Duration, начальная задержка
  - `maxDelay` — time.Duration, максимальная задержка
  - `multiplier` — float64, множитель для exponential (обычно 2)
  - `jitterFactor` — float64, доля jitter (0.25 = 25%)
- Жизненный цикл:
  - Создаётся при инициализации обёртки
  - Используется для расчёта задержки на каждом retry
  - Неизменяем после создания (immutable config)
- Замечания по консистентности:
  - Нет shared mutable state — каждый вызов CalculateDelay использует только входные параметры

## Связи

- `RetryEmbedder` -> `CircuitBreaker`: композиция, 1:1 на экземпляр
- `RetryEmbedder` -> `Backoff`: композиция, 1:1 на экземпляр
- `RetryLLMProvider` -> `CircuitBreaker`: композиция, 1:1 на экземпляр
- `RetryLLMProvider` -> `Backoff`: композиция, 1:1 на экземпляр
- Нет межсущностных связей между разными обёртками (isolation by design)

## Производные правила

- Расчёт задержки: `delay = min(baseDelay * multiplier^attempt, maxDelay) * (1 + random(0, jitterFactor))`
- Проверка перехода в open: `failureCount >= threshold`
- Проверка перехода в half-open: `time.Since(lastFailureTime) >= timeout`

## Переходы состояний CircuitBreaker

| Trigger | From | To | Guard |
|---------|------|-----|-------|
| Ошибка при closed | closed | closed (или open) | failureCount++ < threshold: stay closed; >= threshold: open |
| Запрос при open | open | open (или half-open) | time.Since(lastFailureTime) < timeout: stay open; >= timeout: half-open |
| Успех при half-open | half-open | closed | Успешный пробный запрос |
| Ошибка при half-open | half-open | open | Неудачный пробный запрос |

## Вне scope

- Persistence circuit breaker state между перезапусками
- Distributed/shared circuit breaker между инстансами
- История retry attempts (только текущие счётчики)
