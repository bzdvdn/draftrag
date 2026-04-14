# ChromaDB Parity — Задачи

## Phase Contract

- **Inputs:** plan из `docs/specs/chromadb-parity/plan/plan.md`
- **Outputs:** упорядоченные исполнимые задачи
- **Stop if:** — (задачи конкретны и покрывают все AC)

---

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/chromadb.go` | T1.1 |
| `pkg/draftrag/chromadb_test.go` | T2.1 |
| `docs/compatibility.md` | T3.1 |
| `ROADMAP.md` | T3.2 |

---

## Фаза 1: Переименование API

**Цель:** Выполнить breaking change переименования для консистентности с другими хранилищами.

- [x] **T1.1** Переименовать функции в `pkg/draftrag/chromadb.go` согласно RQ-001:
  - `ChromaDBCollectionExists` → `CollectionExists`
  - `CreateChromaDBCollection` → `CreateCollection`
  - `DeleteChromaDBCollection` → `DeleteCollection`
  - Обновить все godoc комментарии (ссылки на старые имена)
  - Проверить, что `go build ./...` проходит
  - **Touches:** `pkg/draftrag/chromadb.go`
  - **Покрывает:** AC-001

---

## Фаза 2: Обновление тестов

**Цель:** Адаптировать тесты к новым именам функций.

- [x] **T2.1** Обновить `pkg/draftrag/chromadb_test.go` для использования новых имён:
  - Заменить все вызовы `ChromaDBCollectionExists` → `CollectionExists`
  - Заменить все вызовы `CreateChromaDBCollection` → `CreateCollection`
  - Заменить все вызовы `DeleteChromaDBCollection` → `DeleteCollection`
  - Проверить, что `go test ./...` проходит
  - **Touches:** `pkg/draftrag/chromadb_test.go`
  - **Покрывает:** AC-003

---

## Фаза 3: Документация и ROADMAP

**Цель:** Документировать ограничения и актуализировать статус.

- [x] **T3.1** Обновить `docs/compatibility.md` согласно RQ-002 и RQ-003:
  - Добавить явное указание в раздел Vector stores: "Hybrid search (BM25) — не поддерживается"
  - Обновить матрицу: ChromaDB в строке "Управление коллекцией" = ✓
  - Добавить секцию "Ограничения ChromaDB" с пояснением
  - **Touches:** `docs/compatibility.md`
  - **Покрывает:** AC-002

- [x] **T3.2** Обновить `ROADMAP.md` — раздел "ChromaDB: гибридный поиск и миграции":
  - Отметить миграции как ✅ выполненные
  - Явно указать, что гибридный поиск не планируется (ограничение ChromaDB)
  - **Touches:** `ROADMAP.md`
  - **Покрывает:** AC-004

---

## Фаза 4: Финальная проверка

**Цель:** Убедиться в готовности фичи.

- [x] **T4.1** Выполнить финальную проверку:
  - `go build ./...` — без ошибок
  - `go test ./...` — все зелёные
  - `grep -r "ChromaDBCollection\|CreateChromaDB\|DeleteChromaDB" pkg/` — нет результатов
  - Визуальная проверка compatibility.md — матрица корректна
  - **Touches:** —
  - **Покрывает:** AC-001, AC-002, AC-003, AC-004

---

## Покрытие критериев приемки

| AC | Задачи |
|----|--------|
| AC-001 | T1.1, T4.1 |
| AC-002 | T3.1, T4.1 |
| AC-003 | T2.1, T4.1 |
| AC-004 | T3.2, T4.1 |

---

## Порядок выполнения

```
T1.1 → T2.1 → T3.1 → T3.2 → T4.1
```

**Параллельно:** Нельзя — каждая задача зависит от предыдущей (изменения API влияют на тесты, тесты должны проходить перед документацией).

---

## Заметки

- Фаза 1 (T1.1) — breaking change, выполняется первой
- Фаза 2 (T2.1) — блокируется T1.1, иначе тесты не скомпилируются
- Фазы 3-4 можно объединить в один коммит с фазой 2, но задачи разделены для ясности
- Нет необходимости в feature flags или gradual rollout — продукт не в production

---

**Статус:** Все задачи выполнены  
**Создан:** 2026-04-14  
**На основе:** plan.md (v1)
