# Query Rewriting — Задачи

## Phase Contract

Inputs: plan, data-model, spec.
Outputs: упорядоченные исполнимые задачи с покрытием всех 7 AC.
Stop if: нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/domain/models.go` | T1.2 |
| `pkg/draftrag/draftrag.go` | T2.1 |
| `pkg/draftrag/search.go` | T2.1 |
| `pkg/draftrag/search_routing.go` | T2.2, T3.1 |
| `internal/infrastructure/rewriter/llm_rewriter.go` | T3.3 |
| `internal/infrastructure/rewriter/llm_rewriter_test.go` | T4.2 |

## Implementation Context

- Цель MVP: плагируемый QueryRewriter — интерфейс + pipeline-интеграция + LLMRewriter
- Инварианты/семантика:
  - Rewriter неблокирующий: ошибка → fallback на исходный запрос с логом
  - Per-request Rewriter приоритетнее pipeline-level
  - При установленном Rewriter флаги HyDE()/MultiQuery() игнорируются с warning
- Ошибки/коды: Rewriter.Rewrite error → log + исходный query (не fatal)
- Контракты/протокол:
  - `QueryRewriter.Rewrite(ctx, query, history)` → `[]RewrittenQuery` (1:N)
  - `QueryHistory` = `[]Message{Role, Content}`, caller управляет
- Границы scope: не делаем chain-of-thought rewriting, не храним историю в pipeline, не трогаем VectorStore/Embedder/LLMProvider
- Proof signals: каждый AC подтверждается unit-тестом; AC-06 дополнительно integration-тестом
- References: DEC-001–DEC-005, DM-001–DM-003

## Фаза 1: Domain foundation

Цель: подготовить интерфейс и модели данных.

- [x] T1.1 Добавить интерфейс `QueryRewriter` в `internal/domain/interfaces.go`:
  ```go
  type QueryRewriter interface {
      Rewrite(ctx context.Context, query string, history QueryHistory) ([]RewrittenQuery, error)
  }
  ```
  Touches: `internal/domain/interfaces.go`
  AC: AC-001
  DEC: DEC-001

- [x] T1.2 Добавить модели `RewrittenQuery` и `QueryHistory` в `internal/domain/models.go`:
  - `RewrittenQuery{Query string; Weight float64}` (Weight зарезервирован, default 1.0)
  - `QueryHistory{Entries []Message}` где `Message{Role string; Content string}`
  Touches: `internal/domain/models.go`
  AC: AC-001
  DEC: DEC-002, DEC-003

## Фаза 2: MVP Slice

Цель: интеграция QueryRewriter в PipelineOptions, SearchBuilder и routing.

- [x] T2.1 Добавить поле `QueryRewriter` в `PipelineOptions` в `pkg/draftrag/draftrag.go` и метод `.Rewriter(r QueryRewriter)` в `SearchBuilder` в `pkg/draftrag/search.go`. Per-request Rewriter имеет приоритет.
  Touches: `pkg/draftrag/draftrag.go`, `pkg/draftrag/search.go`
  AC: AC-002
  DEC: DEC-001

- [x] T2.2 Добавить маршрут `routeRewriter` в `search_routing.go`:
  - Приоритет: проверка Rewriter первой → routeRewriter
  - Если Rewriter установлен и включены HyDE/MultiQuery — warning через hooks + игнор флагов
  - При ошибке Rewriter.Rewrite → log + fallback на исходный запрос
  Touches: `pkg/draftrag/search_routing.go`
  AC: AC-005, AC-007
  DEC: DEC-004

## Фаза 3: Основная реализация

Цель: 1:N fusion, multi-turn поддержка, LLMRewriter.

- [x] T3.1 Реализовать 1:N multi-query fusion: при `len(rewritten) > 1` выполнить retrieval для каждой переформулировки и объединить через RRF (переиспользовать существующую логику).
  Touches: `pkg/draftrag/search_routing.go`
  AC: AC-003
  DEC: DEC-002

- [x] T3.2 Реализовать передачу `QueryHistory` в `Rewriter.Rewrite`: history передаётся из `SearchBuilder` (новое поле или параметр метода Rewriter). Если history не задана — передавать пустую `QueryHistory{}`.
  Touches: `pkg/draftrag/search.go`, `pkg/draftrag/search_routing.go`
  AC: AC-004
  DEC: DEC-003

- [x] T3.3 Реализовать `LLMRewriter` в `internal/infrastructure/rewriter/llm_rewriter.go`:
  - `NewLLMRewriter(llm domain.LLMProvider, promptTemplate string) *LLMRewriter`
  - При пустом promptTemplate — ошибка валидации
  - Использует LLMProvider.Generate для генерации переформулировок
  - Возвращает `[]RewrittenQuery` (одну или несколько в зависимости от prompt)
  Touches: `internal/infrastructure/rewriter/llm_rewriter.go`
  AC: AC-006
  DEC: DEC-005

## Фаза 4: Проверка

Цель: automated тесты подтверждают поведение.

- [x] T4.1 Добавить unit-тесты для AC-001–AC-005, AC-007:
  - AC-001: type assert кастомной структуры в `domain.QueryRewriter`
  - AC-002: два rewriter'а → разные результаты retrieval
  - AC-003: mock возвращает 3 переформулировки → RRF fusion
  - AC-004: LLMRewriter с history → контекстная переформулировка
  - AC-005: error-rewriter → fallback + log
  - AC-007: Rewriter + HyDE/MultiQuery → warning + игнор
  Touches: `pkg/draftrag/search_test.go` (или новый `_test.go`)
  AC: AC-001, AC-002, AC-003, AC-004, AC-005, AC-007

- [x] T4.2 Добавить unit-тест для LLMRewriter (AC-006) с mock LLM (Ollama integration требует running instance).
  Touches: `internal/infrastructure/rewriter/llm_rewriter_test.go`
  AC: AC-006

## Покрытие критериев приемки

- AC-001 → T1.1, T1.2, T4.1
- AC-002 → T2.1, T4.1
- AC-003 → T3.1, T4.1
- AC-004 → T3.2, T4.1
- AC-005 → T2.2, T4.1
- AC-006 → T3.3, T4.2
- AC-007 → T2.2, T4.1

## Заметки

- Фаза 1 и Фаза 2 можно выполнять последовательно (T1.x → T2.x).
- T3.1, T3.2, T3.3 независимы друг от друга после завершения T2.2.
- T4.x выполняются после завершения всех T3.x.
- Все новые публичные типы реэкспортируются через `pkg/draftrag/` для внешнего использования.
