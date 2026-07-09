# Спецификация: ChromaDB Parity

## Назначение

Достижение parity между ChromaDB и другими vector store (pgvector, Qdrant) по функциональности управления коллекциями и документирование ограничений гибридного поиска.

---

## Контекст

### Текущее состояние

| Функция | Статус | Примечание |
|---------|--------|------------|
| `ChromaDBCollectionExists` | ✅ Реализовано | `pkg/draftrag/chromadb.go:149` |
| `CreateChromaDBCollection` | ✅ Реализовано | `pkg/draftrag/chromadb.go:51` |
| `DeleteChromaDBCollection` | ✅ Реализовано | `pkg/draftrag/chromadb.go:105` |
| Внутренний `CollectionManager` | ✅ Реализовано | `internal/.../chromadb.go:39` |
| Hybrid search (BM25) | ❌ Невозможно | ChromaDB не поддерживает нативно |

### Проблемы

1. **Inconsistent API naming**: ChromaDB использует префикс `ChromaDB` в именах (`CreateChromaDBCollection`), в то время как Qdrant — нет (`CreateCollection`). Это нарушает консистентность публичного API.

2. **Недокументированное ограничение**: Отсутствие гибридного поиска не отражено в compatibility matrix и может вводить пользователей в заблуждение.

---

## Требования

### RQ-001: Консистентность именования API

Переименовать функции ChromaDB для consistency с Qdrant и pgvector API:

```go
// Было (с префиксом ChromaDB) → Станет (без префикса)
ChromaDBCollectionExists → CollectionExists
CreateChromaDBCollection → CreateCollection
DeleteChromaDBCollection → DeleteCollection
```

Сигнатуры функций:

```go
func CollectionExists(ctx context.Context, opts ChromaDBOptions) (bool, error)
func CreateCollection(ctx context.Context, opts ChromaDBOptions) error
func DeleteCollection(ctx context.Context, opts ChromaDBOptions) error
```

**Обоснование:** Продукт еще не зарелизился, breaking changes допустимы. Единообразие API важнее временной совместимости.

### RQ-002: Документирование ограничений

Обновить `docs/compatibility.md` для явного указания:
- ChromaDB **не поддерживает** гибридный поиск (BM25)
- Обходной путь: использовать только семантический поиск или переключиться на pgvector/Qdrant

### RQ-003: Feature parity matrix

Обновить матрицу возможностей в `docs/compatibility.md` для отражения реального состояния:

| Feature | In-memory | pgvector | Qdrant | ChromaDB |
|---------|-----------|----------|--------|----------|
| Постоянное хранение | — | ✓ | ✓ | ✓ |
| Metadata filters | ✓ | ✓ | ✓ | ✓ |
| Hybrid search (BM25) | ✓ | ✓ | — | **— (не поддерживается)** |
| Управление коллекцией | n/a | n/a | ✓ | **✓** |

---

## Решения

### DEC-001: Чистое переименование (breaking change)

Переименовать существующие функции для consistency с другими хранилищами:
- `ChromaDBCollectionExists` → `CollectionExists`
- `CreateChromaDBCollection` → `CreateCollection`
- `DeleteChromaDBCollection` → `DeleteCollection`

Так как продукт не зарелизился, breaking changes допустимы. Цель — единообразный API без исключений.

### DEC-002: Явное документирование ограничений

Вместо попытки эмулировать BM25 (что требует внешнего индекса и значительной сложности), **явно документировать** ограничение и предложить альтернативы.

---

## Acceptance Criteria

### AC-001: API консистентность

- [ ] Функции переименованы в `pkg/draftrag/chromadb.go`:
  - `ChromaDBCollectionExists` → `CollectionExists`
  - `CreateChromaDBCollection` → `CreateCollection`
  - `DeleteChromaDBCollection` → `DeleteCollection`
- [ ] Все функции имеют полный godoc на русском языке
- [ ] Тесты обновлены для использования новых имен

### AC-002: Документация

- [ ] `docs/compatibility.md` обновлён с явным указанием отсутствия hybrid search для ChromaDB
- [ ] Матрица возможностей отражает статус управления коллекциями для ChromaDB
- [ ] Добавлена секция "Ограничения ChromaDB" с пояснением

### AC-003: Тестирование

- [ ] Тесты обновлены для использования новых имен функций (`pkg/draftrag/chromadb_test.go`)
- [ ] `go test ./...` проходит без ошибок

### AC-004: ROADMAP update

- [ ] `ROADMAP.md` обновлён — пункт "ChromaDB: гибридный поиск и миграции" отмечен как выполненный (миграции) или скорректированный (гибридный поиск)

---

## Out of Scope

- **Гибридный поиск (BM25)** для ChromaDB — не реализуется из-за отсутствия нативной поддержки в ChromaDB. Пользователи должны использовать pgvector или Qdrant для этой функции.
- **Weaviate parity** — отдельная фича, не входит в эту спецификацию.

---

## Зависимости

- Отсутствуют. Спецификация работает с существующим кодом.

---

## Примечания

- Согласно конституции, публичный API должен быть консистентным (принцип "Минимальная конфигурация")
- Язык документации и godoc: русский
- Breaking changes допустимы только с major version bump (SemVer)

---

**Статус:** Готова к планированию  
**Создан:** 2026-04-14  
**Автор:** speckeep  
**Последнее обновление:** 2026-04-14 (breaking change подход)
