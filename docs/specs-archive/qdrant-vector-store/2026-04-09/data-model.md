# Qdrant vector store Модель данных

## Scope

- Связанные `AC-*`: AC-001, AC-002, AC-003, AC-004, AC-005
- Связанные `DEC-*`: DEC-002 (плоский payload)

Для данной фичи **новых persisted сущностей в коде не создаётся** — данные хранятся в Qdrant. Этот документ описывает маппинг доменных сущностей на Qdrant points и payload.

## Сущности

### DM-001 Qdrant Point (маппинг доменного Chunk)

- **Назначение**: Представление чанка в Qdrant как point (вектор + payload)
- **Источник истины**: `domain.Chunk`, преобразуется при Upsert
- **Инварианты**:
  - ID точки === Chunk.ID (string)
  - Vector размерности === заданной при создании коллекции
  - Payload содержит обязательные поля: parent_id, content
- **Связанные `AC-*`**: AC-004 (Upsert/Delete), AC-006 (валидация)
- **Связанные `DEC-*`**: DEC-002 (плоский payload), DEC-003 (UUID как ID)

**Поля точки:**

| Поле | Тип в Qdrant | Источник | Смысл |
|------|--------------|----------|-------|
| `id` | string (point ID) | `Chunk.ID` | Уникальный идентификатор чанка |
| `vector` | array of float | `Chunk.Embedding` | Векторное представление |
| `payload["id"]` | string | `Chunk.ID` | Дублирование ID для фильтрации |
| `payload["content"]` | string | `Chunk.Content` | Текстовое содержимое чанка |
| `payload["parent_id"]` | string | `Chunk.ParentID` | ID родительского документа |
| `payload["position"]` | integer | `Chunk.Position` | Позиция чанка в документе |
| `payload["metadata.<key>"]` | string | `Chunk.Metadata[key]` | Метаданные с плоским ключом |

**Жизненный цикл:**
- **Создание**: HTTP PUT /collections/{name}/points с телом `{"points": [{"id": "...", "vector": [...], "payload": {...}}]}`
- **Обновление**: Тот же Upsert — Qdrant перезаписывает существующий point по ID
- **Удаление**: HTTP POST /collections/{name}/points/delete с телом `{"points": ["id1", "id2"]}`

**Замечания по консистентности:**
- Qdrant не поддерживает транзакций между несколькими операциями
- При ошибке сети точка может быть частично записана (вектор записан, payload нет) — mitigated через retry на уровне приложения
- Дублирование Chunk.ID в payload["id"] позволяет делать фильтры по ID без использования point ID

### DM-002 Qdrant Collection

- **Назначение**: Контейнер для points с фиксированной размерностью векторов
- **Источник истины**: Создаётся через миграцию в `pkg/draftrag`
- **Инварианты**:
  - Имя коллекции === заданное при создании store
  - Размерность векторов === заданная при создании
  - Distance metric === Cosine (для совместимости с другими реализациями)
- **Связанные `AC-*`**: AC-005
- **Связанные `DEC-*`**: нет

**Поля коллекции (Qdrant config):**

| Поле | Значение | Смысл |
|------|----------|-------|
| `name` | задаётся пользователем | Идентификатор коллекции |
| `vectors.size` | задаётся при создании | Размерность векторов |
| `vectors.distance` | "Cosine" | Метрика расстояния |

**Жизненный цикл:**
- **Создание**: HTTP PUT /collections/{name} с конфигурацией векторов
- **Удаление**: HTTP DELETE /collections/{name}
- **Обновление**: Вне scope — коллекция создаётся один раз

## Связи

- **DM-001 → DM-002**: many-to-one (points принадлежат коллекции)
- **DM-001 → domain.Chunk**: one-to-one (bijective маппинг)

## Производные правила

### Маппинг MetadataFilter на Qdrant payload filter

```json
{
  "filter": {
    "must": [
      {"key": "metadata.author", "match": {"value": "John"}},
      {"key": "metadata.tag", "match": {"value": "important"}}
    ]
  }
}
```

**Правило**: Каждая пара `(k, v)` из `MetadataFilter.Fields` становится элементом `must` массива с `key: "metadata.k"` и `match.value: v`. Все условия объединяются логическим AND.

### Маппинг ParentIDFilter на Qdrant payload filter

```json
{
  "filter": {
    "should": [
      {"key": "parent_id", "match": {"value": "doc1"}},
      {"key": "parent_id", "match": {"value": "doc2"}}
    ]
  }
}
```

**Правило**: Каждый ParentID становится элементом `should` массива с `key: "parent_id"`. Условия объединяются логическим OR.

### Пустой фильтр

**Правило**: Если `MetadataFilter.Fields` пуст (nil или len==0) — фильтр не применяется, поиск выполняется по всей коллекции. Эквивалентно базовому `Search`.

## Переходы состояний

Для данной фичи **сложные переходы состояний не требуются**:
- Point создаётся/обновляется атомарно через Upsert
- Point удаляется атомарно через Delete
- Коллекция создаётся/удаляется атомарно через миграции

## Вне scope

- Nested payload objects (payload["metadata"] = {"author": "John"}) — используем плоскую структуру
- Qdrant aliases — не используем
- Qdrant snapshots — не используем
- Sparse vectors — не используем
- Payload индексы — создаются Qdrant автоматически при первом использовании фильтра
