# Data Model: vectorstore-weaviate

## Weaviate Collection Schema

Используется для `CreateWeaviateCollection`. Коллекция создаётся с `vectorizer: "none"` (внешний embedder) и явно объявленными property.

### Фиксированные свойства объекта (AC-001, RQ-002)

| Property | Weaviate dataType | Источник в Chunk | Назначение |
|----------|-------------------|------------------|------------|
| `chunkId` | `["text"]` | `chunk.ID` | Хранит оригинальный строковый ID для восстановления при Search |
| `content` | `["text"]` | `chunk.Content` | Текстовое содержимое чанка |
| `parentId` | `["text"]` | `chunk.ParentID` | Фильтрация в SearchWithFilter (AC-002, RQ-004) |
| `position` | `["int"]` | `chunk.Position` | Порядковый номер чанка в документе |
| `chunkMetadata` | `["text"]` | JSON(chunk.Metadata) | JSON-сериализация полного `map[string]string` для чтения при Search |

### Динамические свойства (auto-schema, AC-003)

При Upsert дополнительно записываются `meta_{key}` для каждого ключа `chunk.Metadata`. Пример: `chunk.Metadata{"category":"go"}` → свойство `meta_category = "go"`.

Weaviate auto-schema (включён по умолчанию) добавляет эти свойства автоматически при первом Upsert с новым ключом.

Назначение: server-side WHERE-фильтр в SearchWithMetadataFilter. GraphQL-запросы при Search НЕ перечисляют `meta_*`-свойства — для чтения используется `chunkMetadata` (JSON).

### Объект Weaviate

```json
{
  "class": "{CollectionName}",
  "id": "{UUID v5 от chunk.ID}",
  "vector": [0.1, 0.2, ...],
  "properties": {
    "chunkId":      "original-chunk-string-id",
    "content":      "текст чанка",
    "parentId":     "doc-1",
    "position":     0,
    "chunkMetadata": "{\"category\":\"go\"}",
    "meta_category": "go"
  }
}
```

## UUID v5 маппинг (DEC-002)

`chunk.ID` (string) → UUID v5 (RFC 4122) через `crypto/sha1`:

```
namespace = UUID{0x6b, 0xa7, 0xb8, 0x10, ...}  // DNS namespace UUID
digest    = sha1(namespace_bytes + []byte(chunk.ID))
uuid      = format(digest[:16] с version=5, variant=RFC4122)
```

Детерминированность: один и тот же `chunk.ID` всегда даёт один и тот же UUID. Это делает `Upsert` идемпотентным и `Delete` возможным без хранения маппинга.

## Weaviate API Endpoints (DEC-001)

| Метод | Endpoint | Назначение |
|-------|----------|------------|
| `PUT /v1/objects/{class}/{id}` | Upsert (replace if exists) | Update path |
| `POST /v1/objects` | Upsert (create if not found) | Create path (если PUT вернул 404) |
| `DELETE /v1/objects/{class}/{id}` | Delete | 204=success, 404=success |
| `POST /v1/graphql` | Search / SearchWithFilter / SearchWithMetadataFilter | Near-vector query |
| `POST /v1/schema` | CreateWeaviateCollection | Создание коллекции |
| `DELETE /v1/schema/{class}` | DeleteWeaviateCollection | Удаление коллекции |
| `GET /v1/schema/{class}` | WeaviateCollectionExists | Проверка существования |

## GraphQL Search Query (шаблон)

```graphql
{
  Get {
    {Collection}(
      nearVector: { vector: [...] }
      limit: {topK}
      where: { ... }   // опционально для Filter-методов
    ) {
      chunkId
      content
      parentId
      position
      chunkMetadata
      _additional { id certainty }
    }
  }
}
```

### WHERE-блок для SearchWithFilter (ParentID, AC-002)

Один parentID:
```json
{ "path": ["parentId"], "operator": "Equal", "valueText": "doc-A" }
```

Несколько parentID:
```json
{ "path": ["parentId"], "operator": "ContainsAny", "valueTextArray": ["doc-A", "doc-B"] }
```

### WHERE-блок для SearchWithMetadataFilter (AC-003)

```json
{
  "operator": "And",
  "operands": [
    { "path": ["meta_category"], "operator": "Equal", "valueText": "go" }
  ]
}
```

## Score нормализация (DEC-004)

`RetrievedChunk.Score = _additional.certainty` (Weaviate float64, диапазон 0–1 для cosine).

Пустая коллекция (нет результатов): `RetrievalResult{Chunks: []domain.RetrievedChunk{}, TotalFound: 0}`, без ошибки.
