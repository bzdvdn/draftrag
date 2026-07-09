# ChromaDB vector store — Модель данных

## Scope

- Связанные `AC-*`: AC-001, AC-002, AC-003, AC-007
- Связанные `DEC-*`: DEC-002, DEC-004

Эта фича не вводит новых domain-сущностей — использует существующие типы из `internal/domain`. ChromaDB является external state (внешнее хранилище).

## External State (ChromaDB)

### ES-001 ChromaDB Collection

- **Назначение**: Контейнер для embeddings с фиксированной размерностью и метрикой расстояния
- **Источник истины**: ChromaDB сервер (external)
- **Инварианты**: 
  - Размерность (dimension) фиксируется при создании и не меняется
  - Метрика расстояния определяется при создании (cosine по умолчанию)
- **Связанные `AC-*`**: AC-007 (автосоздание)
- **Связанные `DEC-*`**: DEC-003 (autocreate)
- **Поля**:
  - `name` — string, имя коллекции (из конфигурации ChromaStore)
  - `dimension` — int, размерность embedding-векторов
  - `metadata` — map[string]interface{}, опциональные метаданные коллекции

### ES-002 ChromaDB Point (Record)

- **Назначение**: Хранение одного embedding-вектора с метаданными
- **Источник истины**: ChromaDB сервер (external)
- **Инварианты**:
  - ID уникален в пределах коллекции
  - Длина вектора == dimension коллекции
- **Связанные `AC-*`**: AC-001 (upsert), AC-004 (delete)
- **Связанные `DEC-*`**: DEC-004 (ID как строка), DEC-002 (плоские метаданные)
- **Поля**:
  - `id` — string, идентификатор (== `Chunk.ID`)
  - `embedding` — []float64, векторное представление (== `Chunk.Embedding`)
  - `metadata` — map[string]string, плоские метаданные (== `Chunk.Metadata` + служебные поля)
  - `document` — string, опционально текстовое содержимое (== `Chunk.Content`)

## Маппинг domain.Chunk на ChromaDB

```
domain.Chunk -> ChromaDB Point
- Chunk.ID          -> id (string)
- Chunk.Content     -> document (string) или metadata.content
- Chunk.ParentID    -> metadata.parent_id (string)
- Chunk.Position    -> metadata.position (int)
- Chunk.Embedding   -> embedding ([]float64)
- Chunk.Metadata    -> metadata.* (плоское разворачивание)
```

## ChromaDB API Contracts

### Request: Create Collection
```json
POST /api/v1/collections
{
  "name": "collection_name",
  "metadata": {"hnsw:space": "cosine"},
  "get_or_create": true
}
```

### Request: Upsert Points
```json
POST /api/v1/collections/{name}/upsert
{
  "ids": ["chunk-id-1", "chunk-id-2"],
  "embeddings": [[0.1, 0.2, ...], [0.3, 0.4, ...]],
  "metadatas": [{"parent_id": "doc1", "source": "file1"}, ...],
  "documents": ["content1", "content2"]
}
```

### Request: Query (Search)
```json
POST /api/v1/collections/{name}/query
{
  "query_embeddings": [[0.1, 0.2, ...]],
  "n_results": 10,
  "where": {"source": "file1"},
  "include": ["metadatas", "documents", "distances"]
}
```

### Response: Query Results
```json
{
  "ids": [["chunk-id-1", "chunk-id-2"]],
  "distances": [[0.123, 0.456]],
  "metadatas": [[{"parent_id": "doc1"}, {"parent_id": "doc2"}]],
  "documents": [["content1", "content2"]]
}
```

### Request: Delete
```json
POST /api/v1/collections/{name}/delete
{
  "ids": ["chunk-id-1"]
}
```

## Связи

- `ChromaStore` → `ES-001 Collection`: 1:1 (store привязан к одной коллекции)
- `ES-001 Collection` → `ES-002 Point`: 1:N (коллекция содержит множество точек)
- `domain.Chunk` → `ES-002 Point`: 1:1 при upsert

## Производные правила

- **Score calculation**: ChromaDB возвращает `distances` (cosine distance); score = 1 - distance
- **Where-фильтр**: JSON-объект с exact match условиями для metadata полей
- **ID encoding**: ChromaDB принимает любые string IDs без дополнительного encoding

## Переходы состояний

| Trigger | Предыдущее | Следующее | Guard |
|---------|------------|-----------|-------|
| First Upsert/Search | Collection не существует | Collection создана | `get_or_create: true` в API |
| Upsert | Point может существовать или нет | Point обновлён/создан | — |
| Delete | Point существует | Point удалена | Idempotent — нет ошибки если ID не существует |

## Вне scope

- ChromaDB multi-tenancy (tenants API)
- ChromaDB embedding functions (встроенные эмбеддеры)
- ChromaDB persistent vs in-memory mode (определяется deployment, не кодом)
- Vector index tuning (hnsw параметры) — используем defaults
