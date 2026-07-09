# arch-generics: Generics-driven архитектурный рефакторинг

## Scope Snapshot

- In scope: замена дублирующегося routing boilerplate (49 handler closures) на generic router, замена `panic("nil context")` на error return, устранение ручного mapping типов между `internal/domain` и `pkg/draftrag`.
- Out of scope: изменение публичного API (SearchBuilder сигнатуры остаются), добавление новых провайдеров/хранилищ, изменение логики retrieval/answer/stream.

## Цель

Разработчик, использующий draftRAG как библиотеку, получает более поддерживаемый код без потери обратной совместимости. Поддерживающий разработчик сокращает время на добавление нового routing-маршрута с ~15 мин до ~2 мин благодаря generics-driven dispatcher. Пользователь API перестаёт встречать `panic` при передаче `nil` контекста.

## Основной сценарий

1. Разработчик обновляет draftRAG до новой минорной версии.
2. Существующий код, использующий `pipeline.Search(...).TopK(5).Retrieve(ctx)`, продолжает работать без изменений.
3. При вызове `pipeline.Index(nil, docs)` вместо `panic` возвращается `error`.
4. При добавлении нового маршрута (например, нового retrieval strategy) достаточно добавить одну handler-функцию в generic map, а не 7 идентичных closures для каждого output-формата.

## User Stories

- P1: Разработчик библиотеки добавляет новый retrieval strategy и правит 1 файл (generic router), а не 7 файлов (каждый output handler).
- P2: Пользователь API передаёт `nil` context и получает `error`, а не `panic`.
- P3: Поддерживающий разработчик удаляет ~300 строк дублирующегося кода в `search_routing.go`.

## MVP Slice

Минимальный срез: generic router + замена panic в `pkg/draftrag/draftrag.go`. Закрывает AC-001, AC-002, AC-003.

## First Deployable Outcome

После первого implementation pass:
- `go build ./...` проходит
- `go test ./...` проходит
- `go vet ./...` без ошибок
- `search_routing.go` сокращён: 7 handler maps вместо 49
- panic в публичном API заменены на error return

## Scope

1. Generic router в `pkg/draftrag/search_router.go` — единый тип `router[T any]` с методом `execute`.
2. Замена 7 x 7 = 49 handler closures на 7 handler maps с generic dispatcher.
3. Замена `panic("nil context")` на `return error` во всех публичных методах `pkg/draftrag/draftrag.go`, `pkg/draftrag/search.go`.
4. Опционально: устранение ручного копирования полей между `PipelineOptions` и `application.PipelineOptions` через вспомогательные функции.

## Контекст

- Проект использует Go 1.23 — generics доступны на уровне языка.
- `search_routing.go` уже имеет структуру `router[T any]` (см. `search_router.go`), но не используется в handler maps — каждая карта объявлена отдельно с полным типом.
- `pkg/draftrag/draftrag.go` содержит 8+ вызовов `panic("nil context")`.
- `PipelineOptions` mapping между `pkg/draftrag` и `internal/application` — ручное копирование 15+ полей.
- Все существующие `@sk-task` аннотации `searchbuilder-generics#*` в коде должны быть обновлены на `arch-generics#*`.

## Зависимости

- Go 1.23+ (generics support).
- Существующий тип `router[T any]` в `pkg/draftrag/search_router.go` — базовый строительный блок.
- `none` внешних сервисных зависимостей.

## Требования

- RQ-001 Система ДОЛЖНА использовать generic `router[T any]` для всех 7 output-форматов (Retrieve, Answer, Cite, InlineCite, Stream, StreamSources, StreamCite) вместо отдельных map с полными типами.
- RQ-002 Все публичные методы Pipeline (`Index`, `Query`, `Answer`, `Retrieve`, `DeleteDocument`, `UpdateDocument`, `Search.*`) ДОЛЖНЫ возвращать `error` при `nil` context вместо `panic`.
- RQ-003 System ДОЛЖНА сохранить 100% обратную совместимость публичного API: все существующие сигнатуры методов SearchBuilder остаются без изменений.
- RQ-004 При добавлении нового retrieval strategy (новый route) разработчик ДОЛЖЕН добавить одну handler-функцию в generic map, а не 7 дублирующих closures.
- RQ-005 Все `@sk-task` аннотации `searchbuilder-generics#*` в изменяемых файлах ДОЛЖНЫ быть обновлены на `arch-generics#*`.

## Вне scope

- Изменение сигнатур публичного API (SearchBuilder, Pipeline).
- Изменение логики retrieval/answer/stream — только dispatcher.
- Рефакторинг `internal/application` routing (QueryHyDE, QueryMulti и т.д.) — только публичный слой `pkg/draftrag`.
- Добавление новых LLM провайдеров или VectorStore.
- Изменение `internal/domain` интерфейсов.

## Критерии приемки

### AC-001 Generic router dispatcher

- Почему это важно: устраняет 42 строки дублирующегося кода, делает добавление нового маршрута атомарным.
- **Given** существующий тип `router[T any]` в `search_router.go`
- **When** все 7 output handler maps (`retrieveHandlers`, `answerHandlers`, `citeHandlers`, `inlineCiteHandlers`, `streamHandlers`, `streamSourcesHandlers`, `streamCiteHandlers`) переписаны через `router[ResultType]{handlers: map[route]func(...) -> ResultType}`
- **Then** handler maps используют общий generic тип, количество строк в `search_routing.go` сокращается с ~270 до ~100
- Evidence: `go build ./...`, `go test ./...`, `go vet ./...` проходят; `git diff --stat` показывает сокращение

### AC-002 No panic on nil context

- Почему это важно: Go-сообщество считает panic в библиотечном коде анти-паттерном; пользователь должен получать ошибку.
- **Given** публичный метод Pipeline (например, `Index(nil, docs)`, `Query(nil, "")`, `Search("").Retrieve(nil)`)
- **When** context == nil
- **Then** метод возвращает `error`, а не вызывает `panic`
- Evidence: unit test с `nil` context для каждого метода возвращает ошибку

### AC-003 Backward-compatible API

- Почему это важно: пользователи библиотеки не должны менять код при обновлении.
- **Given** существующий код, использующий `pipeline.Search("q").TopK(5).Retrieve(ctx)`
- **When** после рефакторинга
- **Then** все цепочки SearchBuilder компилируются и работают идентично
- Evidence: `go build ./...` + `go test ./...` без изменений в существующих тестах

### AC-004 Single-point route registration

- Почему это важно: при добавлении нового retrieval strategy разработчик должен изменить минимум кода.
- **Given** новый маршрут `routeCustom`
- **When** разработчик добавляет handler в generic map
- **Then** handler автоматически доступен для всех 7 output-форматов
- Evidence: код-ревью показывает 1 изменение в handler map, а не 7

### AC-005 Trace markers update

- Почему это важно: SpeckKeep traceability требует актуальных маркеров.
- **Given** файлы `pkg/draftrag/search_routing.go`, `pkg/draftrag/search.go`, `pkg/draftrag/search_router.go`
- **When** после рефакторинга
- **Then** все `@sk-task searchbuilder-generics#*` обновлены на `@sk-task arch-generics#*`
- Evidence: `grep -r "searchbuilder-generics" pkg/draftrag/` не находит совпадений

## Допущения

- Существующий тип `router[T any]` и `execResult` интерфейсы (`rRetrieve`, `rAnswer`, `rCite`, `rInlineCite`, `rStream`, `rStreamSources`, `rStreamCite`) остаются без изменений.
- Все output handler функции (`retrieveHandlers`, `answerHandlers` и т.д.) имеют одинаковую сигнатуру `func(ctx context.Context, q string, topK int, b *SearchBuilder) (T, error)`.
- Замена panic на error return не меняет контракт существующих тестов (тесты могут ожидать error вместо recovery).
- Go 1.23 generics достаточно для реализации единого dispatcher.

## Критерии успеха

- SC-001 Количество строк в `search_routing.go` сокращается на ≥60% (с ~270 до ≤110).
- SC-002 `go vet ./...` и `golangci-lint run ./...` проходят без предупреждений.
- SC-003 Все существующие тесты проходят без изменений (zero test modifications required).

## Краевые случаи

- **Пустой handler map**: если generic router вызван с пустой картой, возвращается ошибка.
- **Nil context в SearchBuilder методах**: все 7 методов (Retrieve, Answer, Cite, InlineCite, Stream, StreamSources, StreamCite) проверяют ctx == nil и возвращают error.
- **Zero-value router**: `router[T]{}` с nil handlers возвращает ошибку при execute.
- **Mixed route with unsupported format**: если handler для конкретного route отсутствует в map — ошибка, а не panic.

## Открытые вопросы

- `none` — все решения покрыты анализом существующего кода и шаблоном `router[T any]`.
