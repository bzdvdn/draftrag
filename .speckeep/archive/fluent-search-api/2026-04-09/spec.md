# Fluent Search API (SearchBuilder)

## Scope Snapshot

- In scope: замена 15+ verbose методов Pipeline единым fluent builder'ом `Search(q).TopK(n).<terminal>(ctx)`.
- Out of scope: изменение внутренней логики поиска, retrieval стратегий, форматов ответа.

## Цель

Разработчики, использующие draftRAG, получают единый, composable API поиска вместо комбинаторного взрыва методов вида `QueryTopKWithParentIDsAndMetadataFilter`. Читаемость кода улучшается; добавление новых опций не множит сигнатуры.

## Основной сценарий

1. **Стартовая точка**: пользователь хочет выполнить поиск с фильтром, topK и получить ответ.
2. **Основное действие**: строит цепочку `pipeline.Search("вопрос").TopK(5).Filter(f).Answer(ctx)`.
3. **Результат**: возвращается ответ LLM, аналогичный старым методам; никаких изменений в поведении.
4. **Ошибка**: `ErrEmptyQuery` при пустом вопросе, `ErrInvalidTopK` при topK ≤ 0.

## Scope

- Тип `SearchBuilder` в `pkg/draftrag/search.go`
- Entry point `Pipeline.Search(question string) *SearchBuilder`
- Builder-методы: `TopK`, `Filter`, `ParentIDs`, `Hybrid`, `HyDE`, `MultiQuery`
- Terminal-методы: `Retrieve`, `Answer`, `Cite`, `InlineCite`, `Stream`, `StreamCite`
- Удаление 15+ verbose методов из `pkg/draftrag/draftrag.go`

## Контекст

- Исторически каждая новая комбинация опций приводила к новому методу (QueryTopK, AnswerTopK, QueryTopKWithParentIDs и т.д. — 15+ методов).
- Fluent builder — стандартный Go-паттерн для опциональных параметров без variadic-функций.
- `Retrieve` метод нужен как реализация интерфейса `eval.RetrievalRunner`.

## Требования

- **RQ-001** `Pipeline.Search(q)` возвращает `*SearchBuilder`; без вызова `TopK` терминальные методы возвращают ошибку.
- **RQ-002** `TopK(n)` с n ≤ 0 возвращает `ErrInvalidTopK` при вызове терминального метода.
- **RQ-003** Пустой или только-пробельный вопрос возвращает `ErrEmptyQuery`.
- **RQ-004** nil context вызывает panic (по контракту конституции).
- **RQ-005** `Retrieve(ctx)` реализует `eval.RetrievalRunner.Retrieve`.
- **RQ-006** Все старые verbose методы (QueryTopK, AnswerTopKWithParentIDs и т.д.) удалены из публичного API.
- **RQ-007** Builder-методы `HyDE()` и `MultiQuery(n)` проксируют соответствующие стратегии.

## Вне scope

- Изменение внутренних pipeline методов в `internal/application`.
- Добавление новых retrieval стратегий (HyDE, MultiQuery — отдельная спека).
- Async / batch API.

## Критерии приемки

### AC-001 Базовый Retrieve

- **Почему важно**: основной путь поиска без LLM.
- **Given** pipeline с заполненным store
- **When** `pipeline.Search("q").TopK(2).Retrieve(ctx)`
- **Then** возвращается `RetrievalResult` с непустыми Chunks без ошибки
- **Evidence**: `TestSearchBuilder_Retrieve` pass

### AC-002 Валидация

- **Почему важно**: fail-fast вместо непонятных ошибок из store/LLM.
- **Given** pipeline готов
- **When** пустой вопрос или topK ≤ 0
- **Then** `ErrEmptyQuery` / `ErrInvalidTopK` соответственно
- **Evidence**: `TestSearchBuilder_EmptyQuestion`, `TestSearchBuilder_InvalidTopK` pass

### AC-003 Composable options

- **Почему важно**: заменяет все комбинации старых методов.
- **Given** pipeline с несколькими документами
- **When** `Search("q").TopK(5).ParentIDs("doc-1").Retrieve(ctx)`
- **Then** возвращаются только чанки из doc-1
- **Evidence**: `TestSearchBuilder_ParentIDs` pass

### AC-004 Terminal Answer и Cite

- **Почему важно**: основные production use-cases.
- **Given** pipeline с LLM
- **When** `.Answer(ctx)` и `.Cite(ctx)`
- **Then** ответ непустой; Cite возвращает дополнительно sources
- **Evidence**: `TestSearchBuilder_Answer`, `TestSearchBuilder_Cite` pass

### AC-005 Stream cancellation

- **Почему важно**: Stream должен уважать ctx.
- **Given** cancelled context
- **When** `.Stream(ctx)`
- **Then** возвращается `context.Canceled`
- **Evidence**: `TestSearchBuilder_StreamContextCancel` pass

## Допущения

- Старые методы полностью удалены; нет deprecation-периода (breaking change).
- Порядок builder-вызовов не влияет на результат (каждый метод независимо устанавливает поле).

## Открытые вопросы

- none
