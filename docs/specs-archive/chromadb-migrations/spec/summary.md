# Summary: chromadb-migrations

**Slug**: chromadb-migrations  
**Название**: ChromaDB: миграции коллекций  
**Статус**: ✅ Спецификация готова к планированию

---

## Цель (one-liner)

Добавить функции управления коллекциями ChromaDB (`CreateCollection`, `DeleteCollection`, `CollectionExists`) в публичный API для консистентности с pgvector и Qdrant.

---

## Scope

**Входит**:
- CreateCollection, DeleteCollection, CollectionExists
- ChromaDBOptions.Validate()
- Тесты с HTTP mock
- Обновление NewChromaDBStore

**Не входит**:
- Гибридный поиск для ChromaDB
- Управление индексами
- Batch-операции

---

## Ключевые AC

1. **AC-001**: Создание коллекции через POST `/api/v1/collections`
2. **AC-002**: Удаление коллекции (404 = idempotent success)
3. **AC-003/004**: Проверка существования (200=true, 404=false)
4. **AC-005**: Валидация опций до HTTP запроса
5. **AC-006**: Поддержка context cancellation

---

## Артефакты

| Файл | Описание |
|---|---|
| `spec.md` | Полная спецификация |
| `inspect.md` | Отчет проверки |
| `summary.md` | Этот файл |

---

## Статус проверки

| Проверка | Результат |
|---|---|
| Конституция | ✅ PASS |
| Структура | ✅ PASS |
| AC формат | ✅ PASS (6/6) |
| Язык | ✅ PASS |
| Открытые вопросы | ✅ none |

---

## Блокеры

- none

---

## Следующий шаг

**Команда**: `/speckeep.plan chromadb-migrations`

**Действие**: Создание плана реализации (architecture.md, design.md, tasks.md)
