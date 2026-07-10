# Semantic Chunker — Задачи

## Phase Contract

Inputs: plan.md, data-model.md, spec.md.
Outputs: упорядоченные исполнимые задачи с покрытием всех 9 AC.
Stop if: нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/chunker/semantic.go` | T1.1, T2.1, T4.1 |
| `pkg/draftrag/semantic_chunker.go` | T2.2, T4.1 |
| `pkg/draftrag/config.go` | T3.1 |
| `internal/infrastructure/chunker/semantic_test.go` | T4.1 |
| `pkg/draftrag/config_test.go` | T4.2 |

## Implementation Context

- Цель MVP: реализовать `SemanticChunker` — разбиение документов на семантически связные чанки через косинусное сходство эмбеддингов накапливаемых кандидатов, с публичным конструктором и YAML-конфигурацией.
- Инварианты/семантика:
  - Sentence splitting по `.` `!` `?`; исключения: `Dr.`, `Mr.`, `Mrs.`, `Ms.`, `e.g.`, `i.e.`, `vs.`, `etc.` (после них точка не конец предложения)
  - Эмбеддинг вычисляется для накапливаемого кандидата, а не каждого предложения
  - Сравнение косинусного сходства с эмбеддингом предыдущего сформированного чанка
  - `MinChunkSize` — не завершать чанк раньше; `MaxChunkSize` — форсировать завершение на границе предложения
  - Чанки не режут предложения
- Ошибки/коды:
  - `ErrInvalidChunkerConfig` — невалидные параметры (порог вне [0,1], min < 0, max <= min)
- Контракты/протокол:
  - Внутренняя реализация `semanticChunker` в `internal/infrastructure/chunker/semantic.go` (stateless, не экспортируется)
  - Публичный конструктор `NewSemanticChunker` в `pkg/draftrag/semantic_chunker.go` (паттерн как basic_chunker.go)
  - YAML-тип `"semantic"`, структура `SemanticChunkerConfig` в `config.go`
- Границы scope:
  - Не делаем: overlap, async embedding, LLM-based chunking, BufferSimilarity, замену BasicChunker
- Proof signals:
  - AC-001–AC-008: юнит-тесты внутренней реализации с mock Embedder
  - AC-009: type assertion `*semanticChunker` после `NewPipelineFromConfig` с YAML `type: semantic`
- References: DEC-001, DEC-002, DEC-003, DEC-004, DM (no-change)

## Фаза 1: Sentence splitter

Цель: подготовить rule-based sentence splitter, от которого зависит весь алгоритм.

- [x] T1.1 Реализовать `splitSentences(content string) []string` — разбиение текста на предложения по `.` `!` `?` со списком исключений (`Dr.`, `Mr.`, `Mrs.`, `Ms.`, `e.g.`, `i.e.`, `vs.`, `etc.`).
  - Touches: `internal/infrastructure/chunker/semantic.go`
  - AC: AC-005 (границы не режут предложения)
  - Результат: функция splitSentences существует, возвращает корректные предложения для тестовых случаев

## Фаза 2: MVP — реализация чанкера

Цель: реализовать алгоритм семантического чанкинга и публичный конструктор.

- [x] T2.1 Реализовать `semanticChunker` (внутренний тип) с методом `Chunk(ctx, doc)` — алгоритм по основному сценарию из spec: разбить на предложения, накапливать кандидата, эмбеддить, сравнивать сходство с предыдущим чанком, создавать границу при падении ниже порога, уважать Min/MaxChunkSize, не резать предложения, пробрасывать ошибки Embedder, уважать отмену контекста.
- [x] T2.2 Реализовать публичный конструктор `NewSemanticChunker(opts SemanticChunkerOptions) Chunker` с валидацией опций (threshold в [0,1], min >= 0, max > min или 0) в `pkg/draftrag/semantic_chunker.go`.
- [x] T3.1 Добавить `SemanticChunkerConfig` структуру, расширить `ChunkerConfig.Semantic` полем, добавить ветку `case "semantic"` в `NewPipelineFromConfig` в `pkg/draftrag/config.go`.
  - Touches: `pkg/draftrag/config.go`
  - AC: AC-009 (YAML `type: semantic` → chunker корректно создаётся)
  - Результат: YAML с `chunker: { type: semantic, semantic: { threshold: ..., min_chunk_size: ..., max_chunk_size: ... } }` конфигурирует SemanticChunker

## Фаза 4: Проверка

Цель: доказать корректность фичи через тесты.

- [x] T4.1 Написать юнит-тесты для `semanticChunker` с mock Embedder: `TestSemanticChunker_TwoTopics` (AC-001), `TestSemanticChunker_ThresholdEffect` (AC-002), `TestSemanticChunker_MinChunkSize` (AC-003), `TestSemanticChunker_MaxChunkSize` (AC-004), `TestSemanticChunker_SentenceIntegrity` (AC-005), `TestSemanticChunker_ContextCancel` (AC-006), `TestSemanticChunker_EmptyDoc` (AC-008), `TestSemanticChunker_WhitespaceOnlyDoc` (AC-008), `TestNewSemanticChunker_InvalidConfig` (AC-007), `TestNewSemanticChunker_ValidConfig` (AC-007).
  - Touches: `internal/infrastructure/chunker/semantic_test.go`, `pkg/draftrag/semantic_chunker_test.go`
  - AC: AC-001–AC-008
  - Результат: все тесты проходят

- [x] T4.2 Написать тест для YAML-конфигурации: `TestPipelineFromConfig_SemanticChunker` (AC-009) в `pkg/draftrag/semantic_chunker_test.go`.
  - Touches: `pkg/draftrag/semantic_chunker_test.go`
  - AC: AC-009
  - Результат: тест проходит

## Покрытие критериев приемки

- AC-001 → T2.1, T4.1
- AC-002 → T2.1, T4.1
- AC-003 → T2.1, T4.1
- AC-004 → T2.1, T4.1
- AC-005 → T1.1, T2.1, T4.1
- AC-006 → T2.1, T4.1
- AC-007 → T2.2, T4.1
- AC-008 → T2.1, T4.1
- AC-009 → T3.1, T4.2
