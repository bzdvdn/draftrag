# search-builder-routing-fix — Модель данных

## Scope

- **Связанные AC**: AC-001, AC-002, AC-003, AC-004, AC-005
- **Связанные DEC**: DEC-001, DEC-002, DEC-003

## Сущности

Эта фича **не вводит новых domain-сущностей**. Используются существующие:

- `RetrievalResult` — результат поиска
- `InlineCitation` — inline-цитаты
- `Chunk` — чанки документов (в тестах)

## Изменения в существующих структурах

### `mockBatchStore` (batch_test.go)

Добавляется поле `mu sync.Mutex` для защиты доступа к `chunks`.

```go
type mockBatchStore struct {
	chunks []domain.Chunk
	mu     sync.Mutex // добавляется для thread-safety
}
```

## Вне scope

- Новые domain-модели
- Изменения в `SearchBuilder` (только логика routing, не структура)
- Изменения в `Pipeline` (только приватные helpers)
