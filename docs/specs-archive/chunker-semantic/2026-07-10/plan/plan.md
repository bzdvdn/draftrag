# Semantic Chunker — план

## Phase Contract

Inputs: spec docs/specs/chunker-semantic/spec.md, inspect (concerns, исправлены), конституция.
Outputs: plan.md, data-model.md.
Stop if: нет — spec стабильна, inspect concerns исправлены.

## Цель

Реализовать `SemanticChunker` — новую реализацию `domain.Chunker`, которая разбивает документ на семантически связные чанки на основе косинусного сходства эмбеддингов предложений, с возможностью настройки через `PipelineOptions.Chunker` и YAML-конфигурацию.

## MVP Slice

Базовая реализация: sentence splitting по `.`/`!`/`?` со списком исключений, эмбеддинг каждого накопленного кандидата через переданный `Embedder`, сравнение с предыдущим чанком по порогу, `MinChunkSize`/`MaxChunkSize`, публичный конструктор, YAML-конфигурация. Покрывает AC-001–AC-009.

## First Validation Path

Юнит-тест: рукописный документ из двух тематических блоков → `Chunk()` возвращает >= 2 чанков с корректными границами. Тот же тест с mock Embedder, возвращающим предсказуемые векторы.

## Scope

- `internal/infrastructure/chunker/semantic.go` — реализация алгоритма (статлесс, без публичного экспорта)
- `pkg/draftrag/semantic_chunker.go` — публичный конструктор + опции
- `pkg/draftrag/config.go` — расширение `ChunkerConfig` и `NewPipelineFromConfig`
- `internal/infrastructure/chunker/semantic_test.go` + `pkg/draftrag/semantic_chunker_test.go`
- `pkg/draftrag/config.go` тесты для YAML-ветки `"semantic"`

## Performance Budget

- `none` — semantic chunker по природе медленнее BasicChunker (вызовы Embedder), performance не is a constraint на MVP

## Implementation Surfaces

- `internal/infrastructure/chunker/semantic.go` — новая внутренняя реализация
- `pkg/draftrag/semantic_chunker.go` — новая публичная обёртка (паттерн как basic_chunker.go)
- `pkg/draftrag/config.go` — существующий, расширение ChunkerConfig (`SemanticChunkerConfig`, ветка `"semantic"`)
- `pkg/draftrag/errors.go` — существующий, reuse `ErrInvalidChunkerConfig` (новая валидация)
- `pkg/draftrag/draftrag.go` — существующий, `PipelineOptions.Chunker` уже есть, изменений не требует

## Bootstrapping Surfaces

- `none` — все нужные пакеты уже существуют

## Влияние на архитектуру

- `Chunker` впервые зависит от `Embedder` — усиливается связность, но через интерфейс, что допустимо по конституции
- Нет изменений публичного API `Pipeline` (только подстановка через `PipelineOptions.Chunker`)
- Нет изменений domain-слоя

## Acceptance Approach

| AC | Подход | Поверхности |
|----|--------|-------------|
| AC-001 | mock Embedder с предсказуемыми векторами → проверить 2+ чанка | semantic.go |
| AC-002 | два chunker с разными порогами → `len(high) >= len(low)` | semantic.go |
| AC-003 | MinChunkSize > 1 предложения → второй чанк не создаётся | semantic.go |
| AC-004 | MaxChunkSize меньше суммы предложений → каждый чанк <= MaxChunkSize | semantic.go |
| AC-005 | произвольный документ → каждый чанк заканчивается на `.`/`!`/`?` | semantic.go |
| AC-006 | отменённый ctx → `ctx.Err()` | semantic.go |
| AC-007 | невалидные опции → `ErrInvalidChunkerConfig` | semantic_chunker.go |
| AC-008 | пустой Content → пустой слайс | semantic.go |
| AC-009 | YAML `type: semantic` → type assertion в `*semanticChunker` | config.go |

## Данные и контракты

- `domain.Chunker` — не меняется (только новая реализация)
- `domain.Embedder` — не меняется
- `domain.Chunk` — не меняется
- Новые типы: `SemanticChunkerOptions` в `pkg/draftrag`, `SemanticChunkerConfig` в `pkg/draftrag/config.go`, внутренний `semanticChunker`
- `data-model.md`: stub `no-change`

## Стратегия реализации

### DEC-001 Rule-based sentence splitting без внешней библиотеки

- **Why**: spec требует только английский; список исключений (Dr., Mr., Ms., e.g., i.e., vs., etc.) покрывает common cases; нулевая зависимость.
- **Tradeoff**: не обрабатывает сложные случаи (вложенные кавычки, "e.g." в середине предложения), но для RAG-чанкинга это приемлемо.
- **Affects**: `internal/infrastructure/chunker/semantic.go`
- **Validation**: AC-005 (границы на `.`/`!`/`?`)

### DEC-002 Embedding накапливаемого кандидата, не каждого предложения

- **Why**: K эмбеддингов на документ вместо N предложений (K << N) — меньше API-вызовов к Embedder. Один запрос на финальный чанк + один на кандидат при проверке границы.
- **Tradeoff**: менее точное определение границы (эмбеддинг кандидата «размазан»), но в MVP это приемлемо.
- **Affects**: `semantic.go`
- **Validation**: AC-001, AC-002

### DEC-003 Сравнение с эмбеддингом предыдущего сформированного чанка (не скользящее окно)

- **Why**: простота реализации; не требует буфера и настройки размера окна.
- **Tradeoff**: чувствителен к первому чанку (если он «шумный», границы смещаются). При необходимости в будущем — `BufferSimilarity` (открытый вопрос в spec).
- **Affects**: `semantic.go`
- **Validation**: AC-001

### DEC-004 Internal + public wrapper (паттерн basic_chunker)

- **Why**: единообразие с BasicChunker; валидация опций в публичном слое, алгоритм — во внутреннем.
- **Tradeoff**: два файла вместо одного, но следует установленной convention.
- **Affects**: `semantic.go`, `semantic_chunker.go`
- **Validation**: AC-007 (валидация в публичном слое)

## Incremental Delivery

### MVP (Первая ценность)

- Внутренняя реализация `semanticChunker` + sentence splitting
- Публичный конструктор `NewSemanticChunker` + валидация
- Юнит-тесты на AC-001–AC-008
- YAML-конфигурация + тест AC-009
- Покрытие: AC-001–AC-009

### Итеративное расширение

- P2 из User Stories укладывается в MVP (MinChunkSize/MaxChunkSize уже включены)
- Буферное сглаживание (`BufferSimilarity`) — отложено (открытый вопрос)

## Порядок реализации

1. Sentence splitter (утилита внутри `semantic.go`)
2. Алгоритм семантического чанкинга
3. Публичный конструктор + валидация
4. Юнит-тесты (mock Embedder)
5. YAML-конфигурация + тест
6. `data-model.md` stub

Параллельно: ничего (один разработчик, один скоуп).

## Риски

- **Embedder нестабилен на коротких текстах** (1–2 предложения)
  - Mitigation: в допущениях spec указано; пользователь выбирает Embedder осознанно
- **Косинусное сходство не различает темы для коротких предложений**
  - Mitigation: MinChunkSize уменьшает влияние; в будущем — `BufferSimilarity`
- **Список исключений sentence splitting недостаточен**
  - Mitigation: легко расширяется константой в коде; тесты AC-005

## Rollout и compatibility

- Специальных rollout-действий не требуется — новая реализация не меняет существующее поведение
- `ChunkerConfig` расширяется полем `Semantic` (optional pointer), `Basic` остаётся без изменений
- `PipelineOptions.Chunker` уже nil-совместим: отсутствие chunker = no chunking, как и раньше

## Проверка

- `go test ./internal/infrastructure/chunker/` — все AC-001–AC-008
- `go test ./pkg/draftrag/ -run SemanticChunker` — AC-009
- `go test ./pkg/draftrag/ -run Config` — YAML-конфигурация
- `go vet ./...` — без ошибок
- `golangci-lint run ./...` — без ошибок

## Соответствие конституции

- Нет конфликтов: semantic chunker имплементирует существующий `domain.Chunker`, зависит от `Embedder` через интерфейс, следует Clean Architecture, принимает `context.Context`.
