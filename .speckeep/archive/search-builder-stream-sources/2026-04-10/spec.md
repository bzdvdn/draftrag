# SearchBuilder: метод StreamSources — стриминг ответа с источниками

## Scope Snapshot

- In scope: добавить метод `StreamSources` в `SearchBuilder` — потоковый аналог `Cite`, возвращающий `(<-chan string, RetrievalResult, error)`.
- Out of scope: изменение существующих методов `Stream`, `StreamCite`; добавление стриминга на уровне `application` за пределами нового маршрута.

## Цель

Пользователь библиотеки, который хочет стримить ответ и одновременно показывать источники (без inline-разметки), сейчас вынужден выбирать: либо `Stream` (нет источников), либо `StreamCite` (есть inline-разметка, избыточна). `StreamSources` закрывает этот gap — возвращает токен-канал и плоский список источников, как это делает `Cite` для нестриминговых ответов.

## Основной сценарий

1. Пользователь вызывает `pipeline.Search("вопрос").TopK(5).StreamSources(ctx)`.
2. Метод выполняет retrieval, формирует `RetrievalResult` и запускает LLM в стриминговом режиме.
3. Возвращает `(<-chan string, RetrievalResult, error)` немедленно; канал получает токены по мере генерации.
4. Пользователь читает токены из канала и одновременно использует `RetrievalResult` для отображения источников.
5. Если LLM не поддерживает стриминг — возвращается `ErrStreamingNotSupported` (канал `nil`, `RetrievalResult` пустой).

## Scope

- `pkg/draftrag/search.go` — новый метод `StreamSources` на `*SearchBuilder`, покрывающий все существующие routing-ветки: basic, HyDE, MultiQuery, Hybrid, ParentIDs, Filter
- `pkg/draftrag/search_builder_test.go` — unit-тест на возврат `ErrStreamingNotSupported` и структуру возвращаемых значений

## Контекст

- `SearchBuilder` содержит routing-логику по 6 веткам (basic, HyDE, MultiQuery, Hybrid, ParentIDs, Filter); `StreamSources` должен следовать той же структуре, что `Stream` и `StreamCite`.
- На уровне `application` уже существуют `AnswerStream*` методы, возвращающие `(<-chan string, error)` — routing в `StreamSources` вызывает их же, что и `Stream`, и возвращает `RetrievalResult` через параллельный `Retrieve`-вызов.
- `ErrStreamingNotSupported` уже экспортирован из `pkg/draftrag/errors.go`; `application.ErrStreamingNotSupported` уже существует и маппируется.
- `Cite` возвращает `(string, RetrievalResult, error)`; `StreamSources` — потоковый аналог с `<-chan string` вместо `string`.

## Требования

- RQ-001 `StreamSources` ДОЛЖЕН возвращать `(<-chan string, RetrievalResult, error)`; канал закрывается после последнего токена.
- RQ-002 `StreamSources` ДОЛЖЕН поддерживать все 6 routing-веток SearchBuilder: basic, HyDE, MultiQuery, Hybrid, ParentIDs, Filter — аналогично `Stream`.
- RQ-003 Если LLM не поддерживает стриминг, `StreamSources` ДОЛЖЕН возвращать `nil, RetrievalResult{}, ErrStreamingNotSupported`.
- RQ-004 `RetrievalResult` ДОЛЖЕН быть доступен до начала чтения канала (возвращается вместе с каналом, а не после).

## Вне scope

- Стриминговый аналог `InlineCite` — уже покрыт `StreamCite`.
- Стриминговый аналог `Retrieve` — retrieval не требует стриминга.
- Добавление новых `Answer*Stream*` методов в `application` layer — используются существующие.
- Изменение сигнатуры или поведения `Stream`, `StreamCite`, `Cite`.

## Критерии приемки

### AC-001 Метод существует и возвращает правильные типы

- Почему это важно: пользователь должен получить источники синхронно и токены асинхронно — именно этот паттерн закрывает gap между `Stream` и `StreamCite`.
- **Given** pipeline собран с LLM, поддерживающим стриминг
- **When** вызван `builder.StreamSources(ctx)`
- **Then** возвращается `(<-chan string, RetrievalResult, error)` где error == nil, канал содержит токены, `RetrievalResult.Chunks` непуст
- Evidence: тест читает канал до закрытия и проверяет `len(result.Chunks) > 0`

### AC-002 Покрытие всех routing-веток

- Почему это важно: непокрытая ветка означает молчаливый fallback на неверный путь.
- **Given** SearchBuilder настроен с модификаторами HyDE / MultiQuery / Hybrid / ParentIDs / Filter соответственно
- **When** вызван `StreamSources(ctx)` для каждой конфигурации
- **Then** метод не возвращает ошибку маршрутизации; канал открывается
- Evidence: code review routing switch в `StreamSources` содержит все 6 веток; `go build ./...` проходит

### AC-003 ErrStreamingNotSupported корректно маппируется

- Почему это важно: пользователь должен получить ожидаемую публичную ошибку, а не внутреннюю.
- **Given** pipeline собран с LLM, не поддерживающим стриминг
- **When** вызван `builder.StreamSources(ctx)`
- **Then** возвращается `nil, RetrievalResult{}, ErrStreamingNotSupported`
- Evidence: тест с `noStreamLLM` mock проверяет `errors.Is(err, ErrStreamingNotSupported)`

## Допущения

- На уровне `application` уже есть `AnswerStream*` с полным покрытием 6 маршрутов — `StreamSources` оборачивает их без добавления новых application-методов.
- `RetrievalResult` формируется через тот же retrieve-вызов внутри application-методов; он доступен до начала стриминга (application возвращает его синхронно вместе с каналом).
- Существующий mock `mockLLM` в тестах поддерживает стриминг; `noStreamLLM` mock уже существует или тривиально создаётся.

## Краевые случаи

- LLM не поддерживает стриминг → `ErrStreamingNotSupported`, канал `nil`.
- Контекст отменён до начала чтения канала → канал закрывается, ошибка контекста распространяется через канал или возвращается из `StreamSources`.
- `Filter.Fields` непуст, но VectorStore не поддерживает фильтрацию → `ErrFiltersNotSupported` (существующее поведение, не меняется).

## Открытые вопросы

- none
