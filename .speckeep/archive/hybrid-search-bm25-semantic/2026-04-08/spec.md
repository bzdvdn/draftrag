# Спецификация: Hybrid search (BM25 + semantic)

## Feature slug

`hybrid-search-bm25-semantic`

## Цель

Реализовать гибридный поиск, комбинирующий семантический (векторный) поиск с ключевым (BM25) поиском для улучшения recall на коротких и точных запросах.

## Контекст

Семантический поиск (cosine similarity) хорошо находит похожий смысл, но плохо справляется с:
- Точными названиями, идентификаторами, кодами
- Короткими запросами (1-2 слова)
- Специфической терминологией, не представленной в обучающих данных эмбеддинг-модели

Ключевой поиск (BM25 через PostgreSQL full-text search) компенсирует эти недостатки, обеспечивая точное совпадение терминов.

## Требования

### RQ-001: Интерфейс гибридного поиска

В `internal/domain/interfaces.go` добавить:

```go
// HybridSearcher определяет capability для хранилищ, поддерживающих гибридный поиск.
type HybridSearcher interface {
	// SearchHybrid выполняет гибридный поиск: семантический + BM25.
	// Возвращает объединённые результаты с скором от fusion-стратегии.
	SearchHybrid(ctx context.Context, query string, embedding []float64, topK int, config HybridConfig) (RetrievalResult, error)
}

// HybridConfig задаёт параметры гибридного поиска.
type HybridConfig struct {
	// SemanticWeight вес семантического скора (0.0 - 1.0).
	// BM25Weight вычисляется как 1.0 - SemanticWeight.
	// При значении 0.0 используется только BM25, при 1.0 — только семантический.
	// Default: 0.7
	SemanticWeight float64

	// UseRRF если true, используется Reciprocal Rank Fusion вместо weighted score.
	// При UseRRF=true поле SemanticWeight игнорируется.
	// Default: true
	UseRRF bool

	// RRFK константа для RRF-формулы: score = 1/(k + rank).
	// Default: 60
	RRFK int

	// BMFinalK количество результатов, возвращаемых после fusion.
	// Должно быть <= topK.
	// Default: равно topK
	BMFinalK int
}
```

### RQ-002: BM25-поиск в PostgreSQL

В `internal/infrastructure/vectorstore/pgvector.go` добавить:

```go
// SearchBM25 выполняет полнотекстовый поиск через PostgreSQL tsvector/tsquery.
// Требует наличия колонки content_tsv с GIN-индексом.
func (s *PGVectorStore) SearchBM25(ctx context.Context, query string, topK int) (domain.RetrievalResult, error)
```

**SQL-реализация:**
- Токенизация запроса через `plainto_tsquery('english', $1)` (или язык из конфигурации)
- Поиск: `WHERE content_tsv @@ query`
- Ранжирование: `ts_rank_cd(content_tsv, query, 32)` (covers density, нормализованный 0-1)
- Limit: `topK * 2` (берём с запасом для fusion)

### RQ-003: Гибридный поиск с RRF

**Reciprocal Rank Fusion (RRF):**

```
RRFScore(d) = Σ 1/(k + rank_i(d))
```

gде:
- `k` = RRFK (обычно 60)
- `rank_i(d)` — позиция документа d в результатах метода i (начиная с 1)
- Документы без rank в методе i не получают score от этого метода

**Алгоритм:**
1. Получить TopN семантических результатов (N = topK * 2)
2. Получить TopN BM25 результатов (N = topK * 2)
3. Для каждого уникального чанка вычислить RRFScore
4. Отсортировать по RRFScore (убывание)
5. Вернуть topK результатов

### RQ-004: Гибридный поиск с weighted score

**Weighted Score (альтернатива RRF):**

```
FinalScore(d) = w_semantic * normalize(semantic_score) + w_bm25 * normalize(bm25_score)
```

**Нормализация:**
- Semantic score уже в [0, 1] (cosine similarity)
- BM25 score: `ts_rank_cd` возвращает [0, 1]

### RQ-005: Миграция для BM25

В `pkg/draftrag/migrations/` добавить миграцию `0003_add_bm25.sql`:

```sql
-- Добавляем tsvector колонку для полнотекстового поиска
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS content_tsv tsvector;

-- Создаём GIN-индекс для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_chunks_content_tsv ON chunks USING GIN (content_tsv);

-- Функция для автоматического обновления tsvector
CREATE OR REPLACE FUNCTION chunks_content_tsv_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.content_tsv := to_tsvector('english', NEW.content);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер для автоматического обновления при insert/update
DROP TRIGGER IF EXISTS trigger_chunks_content_tsv ON chunks;
CREATE TRIGGER trigger_chunks_content_tsv
    BEFORE INSERT OR UPDATE ON chunks
    FOR EACH ROW
    EXECUTE FUNCTION chunks_content_tsv_update();

-- Обновляем существующие записи
UPDATE chunks SET content_tsv = to_tsvector('english', content) WHERE content_tsv IS NULL;
```

### RQ-006: Upsert с поддержкой BM25

Модифицировать `PGVectorStore.Upsert`:
- При наличии колонки `content_tsv` в схеме — она обновляется автоматически через триггер (см. RQ-005)
- Поддержать fallback: если `content_tsv` отсутствует (pre-migration), upsert продолжает работать

### RQ-007: Фильтрация в гибридном поиске

Поддержать `ParentIDFilter` и `MetadataFilter` в гибридном поиске:

```go
// HybridSearcherWithFilters расширяет HybridSearcher фильтрами.
type HybridSearcherWithFilters interface {
	HybridSearcher
	
	SearchHybridWithParentIDFilter(ctx context.Context, query string, embedding []float64, topK int, config HybridConfig, filter ParentIDFilter) (RetrievalResult, error)
	SearchHybridWithMetadataFilter(ctx context.Context, query string, embedding []float64, topK int, config HybridConfig, filter MetadataFilter) (RetrievalResult, error)
}
```

### RQ-008: Публичный API

В `pkg/draftrag/pipeline.go` добавить:

```go
// PipelineConfig расширить полем:
type PipelineConfig struct {
	// ... существующие поля ...
	
	// EnableHybridSearch включает гибридный поиск (BM25 + semantic).
	// Требует VectorStore, реализующего HybridSearcher.
	EnableHybridSearch bool
	
	// HybridConfig параметры гибридного поиска.
	// Игнорируется если EnableHybridSearch=false.
	HybridConfig domain.HybridConfig
}

// RetrieveContextHybrid выполняет гибридный поиск.
// Доступен только если config.EnableHybridSearch=true и store реализует HybridSearcher.
func (p *Pipeline) RetrieveContextHybrid(ctx context.Context, question string, topK int) (domain.RetrievalResult, error)
```

## Архитектурные решения

### DEC-001: Расположение BM25-логики

BM25-логика размещается в `pgvector.go`, не выносится в отдельный файл:
- Это capability существующего PGVectorStore
- Другие VectorStore (Qdrant, ChromaDB) могут иметь свои реализации HybridSearcher
- Интерфейс остаётся в domain-слое

### DEC-002: Язык полнотекстового поиска

По умолчанию используется `'english'` для `to_tsquery/plainto_tsquery`.

В будущем можно добавить:
- `WithLanguage(lang string)` опцию в RuntimeOptions
- Автоопределение языка запроса
- Мультиязычные конфигурации

### DEC-003: Стратегия fusion по умолчанию

По умолчанию `UseRRF: true`:
- RRF более устойчив к разным масштабам скоров
- Не требует tuning весов
- Проверенная практикой (Elasticsearch, Weaviate)

### DEC-004: Совместимость с MetadataFilter

Гибридный поиск с MetadataFilter работает так:
1. Применить MetadataFilter к обоим поискам (семантический и BM25)
2. Fusion только по результатам, прошедшим фильтр
3. Если один из поисков не поддерживает MetadataFilter — fallback на семантический с фильтром

## Границы (out of scope)

- Поддержка других языков полнотекстового поиска (возможно в будущем)
- Поддержка BM25 в Qdrant/ChromaDB (зависит от их capabilities)
- Query expansion (synonyms, stemming на уровне приложения)
- Re-ranking после fusion (отдельная фича)

## Зависимости

- Metadata filtering (фича должна быть реализована до или вместе с гибридным поиском)
- Migration system (уже реализовано)

## Критерии приёмки

- [ ] AC-001: `HybridSearcher` интерфейс определён в domain
- [ ] AC-002: `SearchBM25` реализован в pgvector
- [ ] AC-003: RRF fusion корректно объединяет результаты
- [ ] AC-004: Weighted fusion работает с заданными весами
- [ ] AC-005: Миграция 0003_add_bm25.sql создаёт необходимые колонки/индексы
- [ ] AC-006: Upsert работает с и без BM25-колонки (backward compatibility)
- [ ] AC-007: `RetrieveContextHybrid` доступен в публичном API
- [ ] AC-008: Фильтрация по ParentID и Metadata работает с гибридным поиском
- [ ] AC-009: Unit-тесты покрывают ≥80% нового кода
- [ ] AC-010: Benchmark сравнивает pure semantic vs hybrid на тестовом наборе

## Нефункциональные требования

- Производительность: гибридный поиск ≤ 2x времени чистого семантического
- Откатываемость: миграция обратима (down-migration удаляет колонку и триггер)
- Совместимость: существующий код без EnableHybridSearch работает без изменений

## Связанные файлы

- `internal/domain/interfaces.go`
- `internal/domain/models.go` (HybridConfig)
- `internal/infrastructure/vectorstore/pgvector.go`
- `pkg/draftrag/pipeline.go`
- `pkg/draftrag/migrations/0003_add_bm25.sql` (новый)

## Дата создания спецификации

2026-04-08
