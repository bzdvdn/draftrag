# Поддержка Milvus — Модель данных

## Scope

- Связанные `AC-*`: AC-001, AC-002, AC-003, AC-004, AC-005, AC-006
- Связанные `DEC-*`: DEC-001, DEC-003
- Фича не вводит новых Go-сущностей в domain-слое. Здесь зафиксирован mapping `domain.Chunk` → Milvus REST payload и ожидаемый формат ответа.

## Сущности

### DM-001 Milvus Entity (запись в коллекции)

- Назначение: физическое представление `domain.Chunk` в Milvus-коллекции; создаётся при `Upsert`, удаляется при `Delete` / `DeleteByParentID`
- Источник истины: Milvus-коллекция (внешняя система)
- Инварианты: `id` — непустой VARCHAR PK; `vector` — FLOAT_VECTOR ненулевой длины
- Связанные `AC-*`: AC-001, AC-002, AC-003, AC-004, AC-005, AC-006
- Связанные `DEC-*`: DEC-001, DEC-003
- Поля:
  - `id` — VARCHAR, required; маппится из `domain.Chunk.ID`
  - `text` — VARCHAR, required; маппится из `domain.Chunk.Content`
  - `parent_id` — VARCHAR, required; маппится из `domain.Chunk.ParentID`
  - `metadata` — JSON (Milvus JSON type), optional; маппится из `domain.Chunk.Metadata map[string]string` через `json.Marshal`; десериализуется обратно через `json.Unmarshal` при чтении
  - `vector` — FLOAT_VECTOR, required; маппится из `domain.Chunk.Embedding []float64`
- Жизненный цикл:
  - Создаётся/обновляется: `POST /v2/vectordb/entities/upsert`
  - Удаляется по ID: `POST /v2/vectordb/entities/delete` с фильтром `id == "<id>"`
  - Удаляется по ParentID: `POST /v2/vectordb/entities/delete` с фильтром `parent_id == "<parent_id>"`
- Замечания по консистентности: дубликаты по `id` обрабатываются на стороне Milvus (upsert-семантика); частичная запись при ошибке HTTP → метод возвращает ошибку, retry — на усмотрение клиента

### DM-002 Milvus Search Request (Upsert/Delete/Search тела запросов)

- Назначение: wire-формат JSON-тел для трёх типов операций; зафиксирован явно, чтобы tasks-агент не гадал о структуре
- Связанные `AC-*`: AC-001..AC-006
- Связанные `DEC-*`: DEC-001, DEC-004

**Upsert body:**
```json
{
  "collectionName": "<collection>",
  "data": [
    {
      "id": "<chunk.ID>",
      "text": "<chunk.Content>",
      "parent_id": "<chunk.ParentID>",
      "metadata": { "<key>": "<value>", ... },
      "vector": [0.1, 0.2, ...]
    }
  ]
}
```

**Delete body:**
```json
{
  "collectionName": "<collection>",
  "filter": "id == \"<id>\""
}
```
Для `DeleteByParentID`: `"filter": "parent_id == \"<parent_id>\""`

**Search body:**
```json
{
  "collectionName": "<collection>",
  "data": [[0.1, 0.2, ...]],
  "limit": <topK>,
  "outputFields": ["id", "text", "parent_id", "metadata"],
  "filter": "<expression>"
}
```
Поле `filter` опускается, если фильтр пуст.

**Фильтр-выражения Milvus:**
- `SearchWithFilter(ParentIDs: ["a","b"])` → `parent_id in ["a","b"]`
- `SearchWithFilter(ParentIDs: [])` → поле `filter` не добавляется
- `SearchWithMetadataFilter(Fields: {"source": "wiki", "lang": "ru"})` → `metadata["source"] == "wiki" && metadata["lang"] == "ru"`
- `SearchWithMetadataFilter(Fields: {})` → поле `filter` не добавляется

### DM-003 Milvus Search Response

- Назначение: wire-формат ответа на `/v2/vectordb/entities/search`; фиксирует поля, используемые при десериализации
- Связанные `AC-*`: AC-003, AC-004, AC-005
- Поля верхнего уровня:
  - `code` — int; 0 = успех, ненулевой = ошибка (см. DEC-004)
  - `message` — string; описание ошибки при `code != 0`
  - `data` — array of entity objects
- Поля элемента `data[]`:
  - `id` — string → `domain.Chunk.ID`
  - `text` — string → `domain.Chunk.Content`
  - `parent_id` — string → `domain.Chunk.ParentID`
  - `metadata` — object → `json.Unmarshal` → `map[string]string` → `domain.Chunk.Metadata`
  - `distance` — float64 → `domain.RetrievedChunk.Score`

## Связи

- `DM-001` создаётся/удаляется через `DM-002`; читается через `DM-003`
- Значимых межсущностных Go-связей нет — всё взаимодействие идёт через HTTP

## Производные правила

- `RetrievedChunk.Score` = `distance` из Milvus-ответа (Milvus возвращает cosine или L2 distance в зависимости от метрики коллекции; нормализация остаётся на стороне пользователя)
- Если `data` в ответе пустой массив — `RetrievalResult.Chunks` = `[]RetrievedChunk{}`, ошибки нет

## Переходы состояний

- Жизненный цикл достаточно прост: Chunk существует или не существует в Milvus. Отдельный список переходов не нужен.

## Вне scope

- Поля `position`, `embedding` в Milvus-ответе — `domain.Chunk.Position` и `domain.Chunk.Embedding` не сохраняются в Milvus и не возвращаются при поиске
- Управление схемой коллекции (DDL) — вне scope фичи
- Пагинация (offset/cursor) — вне scope фичи
