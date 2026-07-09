# LLMProvider OpenAI-compatible (Responses API) для draftRAG — Модель данных

## Scope

- Связанные `AC-*`: AC-001, AC-004
- Связанные `DEC-*`: DEC-002, DEC-003, DEC-005
- Значимого persisted data model нет: добавляется только конфигурационная структура options и sentinel-ошибка.

## Сущности

### DM-001 OpenAICompatibleLLMOptions

- Назначение: конфигурация HTTP LLMProvider’а (endpoint, авторизация, модель, параметры генерации).
- Источник истины: создаётся клиентом библиотеки и передаётся в `draftrag.NewOpenAICompatibleLLM`.
- Инварианты:
  - `BaseURL` валидный URL (scheme + host).
  - `APIKey` не пустой.
  - `Model` не пустой.
  - `Temperature` (если задан) >= 0.
  - `MaxOutputTokens` (если задан) > 0.
- Связанные `AC-*`: AC-001, AC-004
- Связанные `DEC-*`: DEC-002, DEC-003, DEC-005
- Поля (ожидаемые в v1):
  - `BaseURL` — string, required
  - `APIKey` — string, required
  - `Model` — string, required
  - `Temperature` — *float64, optional
  - `MaxOutputTokens` — *int, optional
  - `HTTPClient` — *http.Client, optional
  - `Timeout` — time.Duration, optional (таймаут на один `Generate`)
- Жизненный цикл:
  - создаётся пользователем
  - передаётся в фабрику
  - используется при каждом вызове `Generate`

## Связи

- Значимых межсущностных связей нет.

## Производные правила

- URL запроса: `{BaseURL}/v1/responses` (join со срезанием лишних слэшей).

## Переходы состояний

- отсутствуют

## Вне scope

- Persisted история сообщений, кэш ответов, key rotation, rate-limit state.

