# Embedder OpenAI-compatible для draftRAG — Модель данных

## Scope

- Связанные `AC-*`: AC-001, AC-004
- Связанные `DEC-*`: DEC-002, DEC-003, DEC-005
- Значимого persisted data model нет: добавляется только конфигурационная структура options и sentinel-ошибка.

## Сущности

### DM-001 OpenAICompatibleEmbedderOptions

- Назначение: конфигурация HTTP embedder’а (endpoint, авторизация, модель, timeouts).
- Источник истины: создаётся клиентом библиотеки и передаётся в `draftrag.NewOpenAICompatibleEmbedder`.
- Инварианты:
  - `BaseURL` не пустой и валидный URL.
  - `APIKey` не пустой (если выбранный провайдер требует ключ).
  - `Model` не пустой.
  - `Timeout` (если есть) > 0.
- Связанные `AC-*`: AC-001, AC-004
- Связанные `DEC-*`: DEC-002, DEC-003, DEC-005
- Поля (ожидаемые в v1):
  - `BaseURL` — string, required
  - `APIKey` — string, required
  - `Model` — string, required
  - `HTTPClient` — *http.Client, optional (если nil — используется дефолтный клиент/transport)
  - `Timeout` — time.Duration, optional (если задан — применяется к клиенту или к контексту)
- Жизненный цикл:
  - создаётся пользователем
  - передаётся в фабрику
  - используется при каждом вызове `Embed`

## Связи

- Значимых межсущностных связей нет.

## Производные правила

- URL запроса строится как `{BaseURL}/v1/embeddings` (с аккуратным join слэшей).

## Переходы состояний

- отсутствуют (конфигурация неизменяема после создания embedder’а в рамках v1).

## Вне scope

- Persisted кэш embeddings.
- Ротация ключей, multi-tenant ключи, advanced rate-limit state.

