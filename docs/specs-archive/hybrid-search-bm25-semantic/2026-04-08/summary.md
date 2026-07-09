# Archive Summary: hybrid-search-bm25-semantic

**Status**: `completed`  
**Archive Date**: 2026-04-08  
**Reason**: Feature fully implemented, verified, and ready for production use.

## Scope Completed

Hybrid search (BM25 + semantic) для draftRAG — объединение полнотекстового поиска PostgreSQL с семантическим векторным поиском.

### Deliverables

- **Domain Layer**: `HybridConfig`, `HybridSearcher`, `HybridSearcherWithFilters` интерфейсы
- **BM25 Infrastructure**: `SearchBM25()` с `ts_rank_cd`, SQL `tsvector`/`tsquery`
- **Fusion Algorithms**: RRF (Reciprocal Rank Fusion) и Weighted Score
- **Hybrid Search Core**: `SearchHybrid()` с fallback и частичными ошибками
- **Filtered Search**: `SearchHybridWithParentIDFilter()`, `SearchHybridWithMetadataFilter()`
- **Migration**: `0003_add_bm25.sql` — tsvector колонка, GIN индекс, триггер
- **Public API**: `QueryHybrid()`, `AnswerHybrid()` в `pkg/draftrag`
- **Tests**: Unit-тесты + бенчмарки (~13-16 мкс/op для fusion)

### Task Statistics

- **Total**: 23 tasks
- **Completed**: 20 ✅
- **Skipped/Integration**: 3 (require CI/PostgreSQL)

### Key Files

| Component | Path |
|-----------|------|
| Domain | `internal/domain/models.go`, `interfaces.go` |
| Infrastructure | `internal/infrastructure/vectorstore/pgvector.go`, `hybrid.go` |
| Migration | `pkg/draftrag/migrations/0003_add_bm25.sql` |
| Public API | `pkg/draftrag/draftrag.go` |
| Application | `internal/application/pipeline.go` |
| Tests | `*_test.go` в соответствующих пакетах |

### Notable Outcomes

- Полная обратная совместимость — работает без миграции BM25
- Graceful fallback при недоступности BM25
- Поддержка фильтров (ParentID, Metadata) в гибридном поиске
- Все проверки качества пройдены (`go vet`, `go fmt`, `golangci-lint`)

### Next Steps

Фича готова к использованию. Для активации BM25 в существующей базе:
```bash
psql -d your_db -f pkg/draftrag/migrations/0003_add_bm25.sql
```
