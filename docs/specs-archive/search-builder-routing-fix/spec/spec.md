# search-builder-routing-fix

## Scope Snapshot

- In scope: три бага — (1) routing `Cite`/`InlineCite`/`Stream`/`StreamCite` игнорирует HyDE/MultiQuery/Hybrid/ParentIDs/Filter; (2) race condition в `mockBatchStore`; (3) устаревшие ссылки на удалённые методы в README.
- Out of scope: новые retrieval стратегии, рефакторинг application layer, новые провайдеры.

## Цель

Устранить три конкретных дефекта: тихое игнорирование retrieval-стратегий в citation/stream методах SearchBuilder, нестабильный тест при `-race`, и несоответствие README текущему API.

## Основной сценарий

1. `pipeline.Search("q").TopK(5).HyDE().Cite(ctx)` → HyDE retrieval + cited answer (сейчас HyDE тихо игнорируется).
2. `go test -race ./internal/application/...` → PASS (сейчас DATA RACE в mockBatchStore).
3. Пользователь копирует код из README → компилируется (сейчас ссылки на удалённые методы).

## Scope

- `internal/application/pipeline.go` — новые приватные helpers + методы с Citations/Stream для HyDE/Multi/Hybrid
- `pkg/draftrag/search.go` — routing в `Cite`, `InlineCite`, `Stream`, `StreamCite`
- `internal/application/batch_test.go` — mutex в `mockBatchStore`
- `README.md` — обновление feature list и code example

## Контекст

- `Retrieve` и `Answer` в SearchBuilder имеют полный routing (HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic).
- `Cite`, `InlineCite`, `Stream`, `StreamCite` не получили аналогичного routing при реализации.
- В application layer есть `QueryHyDE`, `QueryMulti`, `QueryHybrid` — retrieval часть уже готова. Не хватает приватных helpers для citation/stream генерации поверх готового result.
- `mockBatchStore.Upsert` пишет в `[]chunk` без mutex; `IndexBatch` параллелен → DATA RACE.

## Требования

- **RQ-001** `Cite` ДОЛЖЕН поддерживать тот же routing что `Answer`: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
- **RQ-002** `InlineCite` ДОЛЖЕН поддерживать тот же routing что `Answer`.
- **RQ-003** `Stream` ДОЛЖЕН поддерживать routing: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
- **RQ-004** `StreamCite` ДОЛЖЕН поддерживать тот же routing.
- **RQ-005** `go test -race ./internal/application/...` ДОЛЖЕН завершаться без DATA RACE.
- **RQ-006** README ДОЛЖЕН содержать только существующие методы и компилируемые примеры.
- **RQ-007** Новые application-методы ДОЛЖНЫ принимать `context.Context` первым параметром (конституция §Контекстная безопасность).

## Вне scope

- Новые методы `AnswerHyDEStream`, `AnswerMultiStream` как самостоятельная feature.
- Streaming с HyDE/MultiQuery (технически: retrieval синхронный, streaming generation — нет зависимости).
- Тестирование всех новых routing путей (достаточно compile-time check + существующих тестов).

## Критерии приемки

### AC-001 HyDE routing в Cite

- **Given** pipeline с `reverseReranker` или `fixedEmbedder`
- **When** `Search("q").TopK(2).HyDE().Cite(ctx)`
- **Then** возвращается непустой answer и непустые sources без ошибки
- **Evidence**: тест `TestSearchBuilder_HyDE_Cite` pass

### AC-002 MultiQuery routing в Cite

- **When** `Search("q").TopK(2).MultiQuery(2).Cite(ctx)`
- **Then** возвращается непустой answer и источники
- **Evidence**: тест `TestSearchBuilder_MultiQuery_Cite` pass

### AC-003 Полный routing в InlineCite

- **When** `Search("q").TopK(2).HyDE().InlineCite(ctx)` и `.MultiQuery(2).InlineCite(ctx)`
- **Then** возвращается answer, sources, citations без ошибки
- **Evidence**: тесты pass

### AC-004 Race-free batch тест

- **When** `go test -race ./internal/application/...`
- **Then** exit 0, no DATA RACE
- **Evidence**: вывод `go test -race`

### AC-005 README компилируется

- **When** пример из README встраивается в main.go
- **Then** `go build` без ошибок
- **Evidence**: ручная проверка / grep удалённых методов в README → 0

## Допущения

- Приватные helpers `generateCited` и `generateInlineCited` достаточны; отдельные публичные методы для каждой комбинации не нужны.
- Stream+HyDE/MultiQuery технически возможен (retrieval синхронный); реализуется в этом же fix.

## Открытые вопросы

- none
