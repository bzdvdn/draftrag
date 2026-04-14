# План: Hybrid search (BM25 + semantic)

## Feature slug

`hybrid-search-bm25-semantic`

## Статус плана

**READY** — готов к реализации (implement)

## Основная цель

Реализовать гибридный поиск, комбинирующий семантический (векторный) поиск с ключевым (BM25) поиском через PostgreSQL full-text search для улучшения recall на коротких и точных запросах.

## Архитектура решения

### Компоненты

```
┌─────────────────────────────────────────────────────────────────┐
│                        Domain Layer                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐     │
│  │ HybridSearcher │  │ HybridConfig    │  │ HybridSearcher  │     │
│  │   interface    │  │   struct        │  │ WithFilters     │     │
│  └─────────────────┘  └─────────────────┘  │   interface     │     │
│                                            └─────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                         │
│                     (pgvector.go)                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │
│  │  SearchBM25()   │  │ SearchHybrid()  │  │ RRF/Weighted    │   │
│  │                 │  │                 │  │   Fusion        │   │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           Hybrid Search with Filters                      │   │
│  │  SearchHybridWithParentIDFilter / SearchHybridWithMetadata│   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API Layer (pkg/draftrag)                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │               Pipeline Extensions                         │   │
│  │  - PipelineConfig.EnableHybridSearch                      │   │
│  │  - PipelineConfig.HybridConfig                            │   │
│  │  - Pipeline.RetrieveContextHybrid()                       │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Стратегия Fusion

1. **RRF (Reciprocal Rank Fusion)** — default
   - `score = 1/(k + rank)`
   - k = 60 (default)
   - Устойчив к разным масштабам скоров

2. **Weighted Score** — альтернатива
   - `score = w_sem * norm(sem_score) + w_bm25 * norm(bm25_score)`
   - `w_sem` = SemanticWeight (default 0.7)
   - Требует tuning

### Структура данных

```go
// HybridConfig — domain/models.go
type HybridConfig struct {
    SemanticWeight float64  // 0.0-1.0, default 0.7
    UseRRF         bool     // default true
    RRFK           int      // default 60
    BMFinaKK       int      // default = topK
}
```

## Этапы реализации

### Этап 1: Domain Layer (interfaces, models)
**Приоритет:** P0 — блокирует остальные задачи
**Файлы:** `internal/domain/interfaces.go`, `internal/domain/models.go`

- Добавить `HybridSearcher` интерфейс
- Добавить `HybridSearcherWithFilters` интерфейс
- Добавить `HybridConfig` структуру с методом `Validate()`
- Добавить `DefaultHybridConfig()` factory

### Этап 2: BM25 Infrastructure
**Приоритет:** P0 — фундамент гибридного поиска
**Файлы:** `internal/infrastructure/vectorstore/pgvector.go`

- Реализовать `SearchBM25()` метод
- SQL: `plainto_tsquery('english', $query)` + `ts_rank_cd()`
- Обработка ошибок, таймауты
- Unit-тесты с mock БД

### Этап 3: Fusion Algorithms
**Приоритет:** P0 — core logic
**Файлы:** `internal/infrastructure/vectorstore/pgvector.go`, новый `hybrid.go`

- Реализовать RRF алгоритм
- Реализовать weighted score алгоритм
- Нормализация скоров
- Мержинг результатов (deduplication по Chunk.ID)

### Этап 4: Hybrid Search Core
**Приоритет:** P0 — основной метод
**Файлы:** `internal/infrastructure/vectorstore/pgvector.go`

- Реализовать `SearchHybrid()`
- Параллельное выполнение semantic + BM25 (goroutines)
- Применение Fusion стратегии
- Сортировка и обрезка до topK

### Этап 5: Hybrid Search with Filters
**Приоритет:** P1 — расширенная функциональность
**Файлы:** `internal/infrastructure/vectorstore/pgvector.go`

- `SearchHybridWithParentIDFilter()`
- `SearchHybridWithMetadataFilter()`
- Fallback: если BM25 с фильтром не поддерживается → только semantic

### Этап 6: Migration
**Приоритет:** P0 — требуется для работы BM25
**Файлы:** `pkg/draftrag/migrations/0003_add_bm25.sql`

- Колонка `content_tsv tsvector`
- GIN индекс
- Триггер для автоматического обновления
- Backfill существующих записей
- Down-migration (drop column, trigger, function)

### Этап 7: Upsert Compatibility
**Приоритет:** P1 — backward compatibility
**Файлы:** `internal/infrastructure/vectorstore/pgvector.go`

- Проверка наличия колонки `content_tsv` при upsert
- Fallback на legacy upsert если колонки нет
- Graceful degradation

### Этап 8: Public API
**Приоритет:** P0 — пользовательский интерфейс
**Файлы:** `pkg/draftrag/pipeline.go`

- Расширить `PipelineConfig` полями `EnableHybridSearch`, `HybridConfig`
- Реализовать `RetrieveContextHybrid()`
- Интеграция с embedder для получения embedding query
- Валидация конфигурации

### Этап 9: Testing & Benchmarks
**Приоритет:** P1 — качество
**Файлы:** `*_test.go`, `benchmark/` (новый)

- Unit-тесты для всех новых методов (≥80% coverage)
- Интеграционные тесты с тестовым PostgreSQL
- Benchmark: pure semantic vs hybrid search
- Тестовые случаи: короткие запросы, точные термины, код

## Зависимости

| Зависимость | Статус | Блокирует |
|-------------|--------|-----------|
| MetadataFilter в domain | ✅ Реализовано | Этап 5 |
| Migration system | ✅ Реализовано | Этап 6 |
| pgvector store | ✅ Реализовано | Этап 2-5 |
| ParentIDFilter | ✅ Реализовано | Этап 5 |

## Риски и митигация

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Производительность BM25 | Средняя | Высокое | GIN индекс, ограничение topK*2 |
| RRF качество хуже pure semantic | Низкая | Среднее | Benchmark, fallback на UseRRF=false |
| Сложность параллельного кода | Средняя | Среднее | Пул горутин, context cancellation |
| Backward compatibility | Низкая | Высокое | Fallback на legacy schema, feature flag |

## Критерии завершения

- [ ] Все задачи из `tasks.md` выполнены
- [ ] Unit-тесты ≥80% для нового кода
- [ ] Интеграционные тесты проходят
- [ ] Benchmark показывает улучшение recall на коротких запросах
- [ ] `go vet`, `go fmt`, `golangci-lint` без ошибок
- [ ] Миграция тестирована (up/down)
- [ ] Документация (godoc) на русском языке

## Связанные артефакты

- Spec: `.speckeep/specs/hybrid-search-bm25-semantic/spec.md`
- Inspect: `.speckeep/specs/hybrid-search-bm25-semantic/inspect.md`
- Tasks: `.speckeep/plans/hybrid-search-bm25-semantic/tasks.md`

## Дата создания плана

2026-04-08
