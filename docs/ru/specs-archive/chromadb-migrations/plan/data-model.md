# ChromaDB: миграции коллекций — Модель данных

## Scope

- **Связанные AC**: AC-001, AC-002, AC-003, AC-004 (управление коллекциями ChromaDB)
- **Связанные DEC**: DEC-001, DEC-002 (ChromaDB API endpoints)

## Сущности

Эта фича **не вводит новых domain-сущностей**. Коллекция ChromaDB — это external resource, управляемый через HTTP API, а не persisted entity внутри draftRAG.

Существующие сущности (`Chunk`, `Document`, `RetrievalResult`) не изменяются.

## Данные в ChromaDB API

### Request: CreateCollection (POST /api/v1/collections)

```json
{
  "name": "string",
  "metadata": {},
  "get_or_create": false
}
```

**Поля**:
- `name` — имя коллекции (обязательное), соответствует `ChromaDBOptions.Collection`
- `metadata` — опциональные метаданные коллекции
- `get_or_create` — если `true`, не возвращает ошибку при существовании

### Response: CreateCollection

- **200 OK** — коллекция создана или уже существовала (при `get_or_create: true`)
- **409 Conflict** — коллекция уже существует (при `get_or_create: false`)

### Request: DeleteCollection (DELETE /api/v1/collections/{name})

- **200 OK** — коллекция удалена
- **404 Not Found** — коллекция не существует (идемпотентность)

### Request: CollectionExists (GET /api/v1/collections/{name})

**Response (200)**:
```json
{
  "id": "...",
  "name": "...",
  "metadata": {},
  "tenant": "...",
  "database": "..."
}
```

**Response (404)** — коллекция не найдена

## Вне scope

- Внутренняя структура коллекций ChromaDB (embeddings, documents, метаданные чанков)
- Миграция данных между коллекциями
- Schema versioning для коллекций
