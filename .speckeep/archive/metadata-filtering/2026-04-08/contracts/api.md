# Metadata filtering — API Contracts

Граница: публичный пакет `pkg/draftrag`, тип `Pipeline`.

## Новые публичные методы

### QueryWithMetadataFilter

```go
func (p *Pipeline) QueryWithMetadataFilter(
    ctx context.Context,
    question string,
    topK int,
    filter MetadataFilter,
) (RetrievalResult, error)
```

- Связанные AC: AC-003
- Предусловия: `ctx != nil`, `question` непустой после TrimSpace, `topK > 0`
- Поведение при пустом `filter.Fields`: эквивалентно `QueryTopK(ctx, question, topK)`
- Поведение при бэкенде без `VectorStoreWithFilters`: возвращает `ErrFiltersNotSupported`
- Error cases:
  - `ErrEmptyQuery` — пустой question
  - `ErrInvalidTopK` — topK <= 0
  - `ErrFiltersNotSupported` — store не реализует `VectorStoreWithFilters`
  - прочие ошибки от store/embedder — прокидываются как есть

### AnswerWithMetadataFilter

```go
func (p *Pipeline) AnswerWithMetadataFilter(
    ctx context.Context,
    question string,
    topK int,
    filter MetadataFilter,
) (string, error)
```

- Связанные AC: AC-003
- Предусловия: аналогично `QueryWithMetadataFilter`
- Поведение при пустом `filter.Fields`: эквивалентно `AnswerTopK(ctx, question, topK)`
- Поведение при бэкенде без `VectorStoreWithFilters`: возвращает `ErrFiltersNotSupported`
- Error cases: аналогично `QueryWithMetadataFilter`

## Переэкспортируемые типы

```go
// MetadataFilter задаёт условие точного совпадения по метаданным документа.
type MetadataFilter = domain.MetadataFilter
```

Добавляется в `pkg/draftrag/draftrag.go` рядом с `ParentIDFilter`.

## Совместимость

- Аддитивное изменение: существующие методы `Pipeline` не изменяются.
- `ErrFiltersNotSupported` уже экспортирован — новые методы следуют тому же error contract.
- Semver: minor bump (новые exported symbols, нет breaking changes).
