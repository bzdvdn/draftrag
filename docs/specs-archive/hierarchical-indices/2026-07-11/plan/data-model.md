# Hierarchical Indices — Parent Document Retrieval: Модель данных

## Scope

- Связанные `AC-*`: `AC-001`, `AC-002`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`
- Статус: `changed`

## Сущности

### DM-001 RetrievedChunk.ParentContent

- Назначение: перенос текста родительского документа от retrieval до генерации ответа LLM
- Источник истины: `VectorStore.GetParentDocument()` (или zero value, если store не поддерживает)
- Инварианты: `ParentContent` равен полному тексту родительского документа; пустая строка означает «parent недоступен»
- Связанные `AC-*`: `AC-001`, `AC-002`, `AC-003`, `AC-004`
- Связанные `DEC-*`: `DEC-002`
- Поля: не отдельная сущность, а поле `ParentContent string` в существующей `RetrievedChunk`
- Жизненный цикл:
  - создаётся в `maybeAttachParentContent()` после поиска чанков
  - не кэшируется, не персистится отдельно
- Замечания по консистентности: stale parent невозможен — parent-документ загружается в момент retrieval

### DM-002 ParentDocumentStore (optional capability)

- Назначение: интерфейс для хранения и загрузки parent-документов в VectorStore
- Источник истины: реализация VectorStore
- Инварианты: parent-документ хранится по ключу `parentID` (совпадает с `doc.ID`)
- Связанные `AC-*`: `AC-001`, `AC-003`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`
- Поля: не сущность данных, а контракт интерфейса
  - `UpsertParent(ctx, doc Document, embedding []float64) error`
  - `GetParentDocument(ctx, parentID string) (*Document, error)`
  - `DeleteParent(ctx, parentID string) error` — неявно через `DeleteByParentID`
- Жизненный цикл:
  - parent создаётся при `Index` (если `ParentContextEnabled=true` и store поддерживает)
  - parent удаляется при `DeleteDocument` (через `DeleteByParentID`)
  - parent перезаписывается при `UpdateDocument`
- Замечания по консистентности: parent-документ и чанки — независимые сущности; удаление чанка не удаляет parent

## Связи

- `DM-002 -> Document`: ParentDocumentStore оперирует `domain.Document` — родительским документом целиком

## Производные правила

- `ParentContent` в `RetrievedChunk` вычисляется как результат `GetParentDocument(ctx, chunk.ParentID)` при `ParentContextEnabled=true` и наличии capability

## Переходы состояний

Жизненный цикл parent-документа:

| Событие | До | После |
|---|---|---|
| `Index(doc)` | parent не существует | parent создан с embedding |
| `UpdateDocument(doc)` | старый parent | parent перезаписан |
| `DeleteDocument(docID)` | parent существует | parent удалён |
| `ParentContextEnabled=false` | parent мог бы быть создан | parent не создаётся |

## Вне scope

- Версионирование parent-документов
- Хранение parent в отдельной коллекции/таблице (зависит от реализации store)

## No-Change Stub

Не применимо — модель данных меняется.
