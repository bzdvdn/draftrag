# ChromaDB: миграции коллекций — API Contracts

## HTTP API Boundary: ChromaDB REST API v1

Base URL: `http://localhost:8000` (default) или `ChromaDBOptions.BaseURL`

---

## CreateCollection

**Endpoint**: `POST /api/v1/collections`

**Request Body**:
```json
{
  "name": "{collection_name}",
  "metadata": {},
  "get_or_create": false
}
```

**Success Response (200)**:
```json
{
  "id": "uuid",
  "name": "{collection_name}",
  "metadata": {},
  "tenant": "default_tenant",
  "database": "default_database"
}
```

**Error Responses**:
- `409 Conflict` — коллекция уже существует (без `get_or_create: true`)
- `400 Bad Request` — невалидное имя или параметры

**Mapping to Go**:
```go
func CreateCollection(ctx context.Context, opts ChromaDBOptions) error
```

---

## DeleteCollection

**Endpoint**: `DELETE /api/v1/collections/{collection_name}`

**Success Response (200)**:
```json
{
  "id": "uuid",
  "name": "{collection_name}",
  "metadata": {},
  "tenant": "default_tenant",
  "database": "default_database"
}
```

**Error Responses**:
- `404 Not Found` — коллекция не существует

**Idempotency**: 404 возвращается как `nil` ошибки (коллекция уже удалена)

**Mapping to Go**:
```go
func DeleteCollection(ctx context.Context, opts ChromaDBOptions) error
```

---

## CollectionExists

**Endpoint**: `GET /api/v1/collections/{collection_name}`

**Success Response (200)**:
```json
{
  "id": "uuid",
  "name": "{collection_name}",
  "metadata": {},
  "tenant": "default_tenant",
  "database": "default_database"
}
```

**Error Responses**:
- `404 Not Found` — коллекция не существует

**Mapping to Go**:
```go
func CollectionExists(ctx context.Context, opts ChromaDBOptions) (bool, error)
```

**Semantics**:
- `200 OK` → `(true, nil)`
- `404 Not Found` → `(false, nil)`
- Другие ошибки → `(false, error)`

---

## Error Handling

Все HTTP-ошибки оборачиваются в `fmt.Errorf` с контекстом:
- Ошибка создания запроса: `"create request: %w"`
- Ошибка HTTP клиента: `"chromadb request: %w"`
- Ошибка декодирования: `"decode response: %w"`
- Ошибка ChromaDB: `"chromadb error: status=%d, body=%s"`

---

## AC Mapping

| AC | Endpoint | Contract Element |
|---|---|---|
| AC-001 | POST /api/v1/collections | Request body с `name`, Success 200 |
| AC-002 | DELETE /api/v1/collections/{name} | Success 200, 404 = idempotent |
| AC-003/004 | GET /api/v1/collections/{name} | 200 → true, 404 → false |
