# search-builder-stream-sources: задачи

## Phase Contract

Inputs: plan.md, summary.md.
Outputs: упорядоченные исполнимые задачи с покрытием всех AC.
Stop if: задачи расплывчаты или coverage нельзя сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/application/pipeline.go | T1.1 |
| pkg/draftrag/search.go | T2.1 |
| pkg/draftrag/search_builder_test.go | T3.1, T3.2 |

## Фаза 1: Application layer — методы Answer*StreamWithSources

Цель: добавить 6 тонких wrapper-методов, каждый = `Query* + streamFromResult + return (chan, result, err)`.

- [x] T1.1 Добавить 6 методов `Answer*StreamWithSources` в pipeline.go после `AnswerStreamWithMetadataFilter` — `go build ./...` ok (DEC-001, AC-001, AC-002). Touches: internal/application/pipeline.go

## Фаза 2: Public API — метод StreamSources

Цель: добавить `StreamSources` в SearchBuilder с routing-структурой идентичной `Stream`.

- [x] T2.1 Добавить метод `StreamSources` в search.go — routing switch покрывает все 6 веток, `go build ./...` ok (DEC-002, AC-001, AC-002). Touches: pkg/draftrag/search.go

## Фаза 3: Тесты и валидация

Цель: подтвердить поведение `ErrStreamingNotSupported` и корректность возврата канала + источников.

- [x] T3.1 Добавить `TestSearchBuilder_StreamSources_StreamingNotSupported` — `errors.Is(err, ErrStreamingNotSupported)` (AC-003). Touches: pkg/draftrag/search_builder_test.go
- [x] T3.2 Прогнать `go test ./pkg/draftrag/... -run TestSearchBuilder_StreamSources` — все тесты зелёные (AC-001, AC-002, AC-003). Touches: pkg/draftrag/search_builder_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1, T3.2
- AC-002 -> T1.1, T2.1, T3.2
- AC-003 -> T3.1, T3.2
