# Weaviate Full Support — Задачи

## Phase Contract

- **Inputs:** plan из `docs/specs/weaviate-full-support/plan/plan.md`
- **Outputs:** упорядоченные исполнимые задачи
- **Stop if:** — (задачи конкретны и покрывают все AC)

---

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/weaviate.go` | T1.1 |
| `pkg/draftrag/weaviate_test.go` | T2.1, T2.2, T2.3, T2.4 |
| `docs/weaviate.md` | T3.1, T3.2, T3.3, T3.4, T3.5 |
| `docs/compatibility.md` | T4.1 |
| `ROADMAP.md` | T4.2 |

---

## Фаза 1: Переименование API

**Цель:** Выполнить breaking change переименования для консистентности с другими хранилищами (DEC-001).

- [ ] **T1.1** Проверить текущие имена функций в `pkg/draftrag/weaviate.go`:
  - Проверить, есть ли конфликты имён с Qdrant (CreateCollection, DeleteCollection, CollectionExists)
  - Если конфликты есть — оставить префикс Weaviate* (как для ChromaDB)
  - Если конфликтов нет — убрать префикс для максимальной консистентности
  - Проверить, что `go build ./...` проходит
  - **Touches:** `pkg/draftrag/weaviate.go`
  - **Покрывает:** AC-001

---

## Фаза 2: Тесты

**Цель:** Обеспечить полное тестовое покрытие для production-ready статуса.

- [ ] **T2.1** Обновить `pkg/draftrag/weaviate_test.go` для использования новых имён:
  - Заменить все вызовы старых функций на новые
  - Проверить, что `go test ./pkg/draftrag -run Weaviate` проходит
  - **Touches:** `pkg/draftrag/weaviate_test.go`
  - **Покрывает:** AC-001, AC-003

- [ ] **T2.2** Добавить тесты для authentication (401/403 errors):
  - Тест с неверным API key
  - Тест с отсутствием auth header
  - Проверить, что errors возвращаются явно, не panic
  - **Touches:** `pkg/draftrag/weaviate_test.go`
  - **Покрывает:** AC-003

- [ ] **T2.3** Добавить тесты для error handling:
  - Тест 404 (collection not found)
  - Тест 500 (server error)
  - Тест network errors (connection refused, timeout)
  - **Touches:** `pkg/draftrag/weaviate_test.go`
  - **Покрывает:** AC-003

- [ ] **T2.4** Добавить тесты для edge cases:
  - Тест пустой коллекции
  - Тест дубликатов при индексации
  - Тест context cancellation во всех операциях
  - Проверить coverage >= 90%
  - **Touches:** `pkg/draftrag/weaviate_test.go`
  - **Покрывает:** AC-003

---

## Фаза 3: Документация

**Цель:** Обновить документацию до production уровня.

- [ ] **T3.1** Документировать ограничение hybrid search в `docs/weaviate.md` (DEC-002):
  - Добавить секцию "Ограничения"
  - Явно указать: "Hybrid search (BM25) не поддерживается"
  - Объяснить причину: Weaviate не имеет нативной реализации BM25
  - Рекомендовать pgvector/Qdrant для hybrid search
  - **Touches:** `docs/weaviate.md`
  - **Покрывает:** AC-002

- [ ] **T3.2** Добавить production checklist в `docs/weaviate.md`:
  - Deployment steps (schema setup, init containers)
  - Monitoring requirements (latency, error rates)
  - Best practices (timeouts, batch sizes)
  - **Touches:** `docs/weaviate.md`
  - **Покрывает:** AC-004

- [ ] **T3.3** Добавить performance guidance в `docs/weaviate.md`:
  - Рекомендации по batch size для индексации
  - Рекомендации по timeouts для разных операций
  - Индексирование и performance tuning
  - **Touches:** `docs/weaviate.md`
  - **Покрывает:** AC-004

- [ ] **T3.4** Добавить migration guide в `docs/weaviate.md`:
  - Таблица старых/новых имён функций (если были изменения)
  - Примеры миграции кода
  - Breaking changes note (допустимы до v1.0)
  - **Touches:** `docs/weaviate.md`
  - **Покрывает:** AC-004

- [ ] **T3.5** Добавить troubleshooting guide в `docs/weaviate.md`:
  - Расширенные сценарии ошибок (auth, network, timeouts)
  - Common issues и решения
  - Debugging tips
  - **Touches:** `docs/weaviate.md`
  - **Покрывает:** AC-004

---

## Фаза 4: Статус и ROADMAP

**Цель:** Актуализировать статус Weaviate как production-ready.

- [ ] **T4.1** Обновить `docs/compatibility.md`:
  - Изменить статус Weaviate experimental → stable
  - Обновить матрицу возможностей: Hybrid search (BM25) для Weaviate = —
  - Обновить notes для Weaviate
  - **Touches:** `docs/compatibility.md`
  - **Покрывает:** AC-002, AC-005

- [ ] **T4.2** Обновить `ROADMAP.md`:
  - Отметить Weaviate как production-ready в разделе "Additional vector stores"
  - Обновить приоритет (снизить, если выполнено)
  - **Touches:** `ROADMAP.md`
  - **Покрывает:** AC-005

- [ ] **T4.3** Выполнить финальную проверку:
  - `go build ./...` — без ошибок
  - `go test ./...` — все зелёные
  - `go test -cover ./pkg/draftrag` — coverage >= 90%
  - Ручной просмотр docs/weaviate.md — все секции присутствуют
  - Ручной просмотр docs/compatibility.md — статус обновлён
  - **Touches:** —
  - **Покрывает:** AC-001, AC-002, AC-003, AC-004, AC-005

---

## Покрытие критериев приемки

| AC | Задачи |
|----|--------|
| AC-001 | T1.1, T2.1 |
| AC-002 | T3.1, T4.1 |
| AC-003 | T2.1, T2.2, T2.3, T2.4, T4.3 |
| AC-004 | T3.2, T3.3, T3.4, T3.5 |
| AC-005 | T4.1, T4.2, T4.3 |

---

## Порядок выполнения

```
T1.1 → T2.1 → T2.2 → T2.3 → T2.4 → T3.1 → T3.2 → T3.3 → T3.4 → T3.5 → T4.1 → T4.2 → T4.3
```

**Параллельно:**
- T2.2, T2.3, T2.4 можно выполнять параллельно после T2.1
- T3.2, T3.3, T3.4, T3.5 можно выполнять параллельно после T3.1
- T4.1 и T4.2 можно выполнять параллельно после фазы 3

---

## Заметки

- Фаза 1 (T1.1) — breaking change, выполняется первой
- Фаза 2 (T2.1) — блокируется T1.1, иначе тесты не скомпилируются
- Фаза 3 (T3.4) — migration guide требует знания новых имён из T1.1
- Фаза 4 (T4.1, T4.2) — блокируется всеми предыдущими фазами
- Нет необходимости в feature flags или gradual rollout — продукт не в production

---

**Статус:** Готов к реализации  
**Создан:** 2026-04-14  
**На основе:** plan.md (v1)
