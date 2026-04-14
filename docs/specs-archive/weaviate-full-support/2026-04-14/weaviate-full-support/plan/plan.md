# Weaviate Full Support — План реализации

## Phase Contract

- **Inputs:** spec из `docs/specs/weaviate-full-support/spec.md`
- **Outputs:** план реализации с задачами
- **Stop if:** — (спецификация конкретна, но есть открытые вопросы Q1–Q3)

---

## Цель

Достичь production-ready статуса Weaviate путём:
1. **Консистентности API** (breaking change для унификации с другими хранилищами)
2. **Паритета возможностей** (hybrid search или явное документирование ограничения)
3. **Надёжности через тесты** (полное покрытие всех сценариев)
4. **Production документации** (checklist, performance, troubleshooting)

Изменения затрагивают публичный API (`pkg/draftrag`), тесты и документацию. Внутренняя реализация (`internal/infrastructure/vectorstore/weaviate.go`) может меняться только если требуется для hybrid search.

---

## Scope

### Входит
- `pkg/draftrag/weaviate.go` — переименование функций (breaking change)
- `pkg/draftrag/weaviate_test.go` — расширение тестового покрытия
- `docs/weaviate.md` — расширение до production уровня
- `docs/compatibility.md` — изменение статуса experimental → stable
- `ROADMAP.md` — актуализация статуса Weaviate
- `internal/infrastructure/vectorstore/weaviate.go` — только если требуется для hybrid search

### Не входит
- Weaviate-specific features (GraphQL mutations, custom modules)
- Интеграция с Weaviate Cloud специфичными API
- Performance tuning за пределами разумных defaults

---

## Implementation Surfaces

### Существующие поверхности (подлежат изменению)
- **`pkg/draftrag/weaviate.go`** — публичный API для управления коллекциями и store creation
  - Функции: `WeaviateCollectionExists`, `CreateWeaviateCollection`, `DeleteWeaviateCollection`
  - Причина: требуется переименование для консистентности с другими хранилищами

- **`pkg/draftrag/weaviate_test.go`** — существующие тесты
  - Причина: требуется расширение покрытия (auth, errors, edge cases)

- **`docs/weaviate.md`** — текущая документация
  - Причина: требуется расширение до production уровня (checklist, performance guidance)

- **`docs/compatibility.md`** — матрица backend статусов
  - Причина: требуется изменение статуса Weaviate experimental → stable

### Новые поверхности (требуются только при необходимости)
- **`internal/infrastructure/vectorstore/weaviate.go`** — внутренняя реализация
  - Причина: только если требуется реализовать hybrid search через Weaviate API
  - Если не требуется — изменений нет

---

## Влияние на архитектуру

### Локальное влияние
- **Пакет `pkg/draftrag`**: breaking change — переименование функций Weaviate для консистентности API
  - Пользовательский код, использующий старые имена, потребует миграции
  - Внутренняя логика не меняется, только имена функций

### Интеграционное влияние
- **Нет влияния на другие vector stores** — изменения изолированы в Weaviate
- **Нет влияния на embedders или LLM providers** — изменения только в vector store слое

### Migration последствия
- **Breaking change до v1.0** — допустим, т.к. продукт не зарелизился
- **Migration guide** требуется в `docs/weaviate.md` для пользователей с существующим кодом

---

## Acceptance Approach

### AC-001: API консистентность
- **Подход:** Переименование функций в `pkg/draftrag/weaviate.go` с префиксом `Weaviate*` (как для ChromaDB)
  - `WeaviateCollectionExists` → `WeaviateCollectionExists` (без изменений, если уже уникально)
  - `CreateWeaviateCollection` → `CreateWeaviateCollection` (без изменений, если уже уникально)
  - `DeleteWeaviateCollection` → `DeleteWeaviateCollection` (без изменений, если уже уникально)
- **Surfaces:** `pkg/draftrag/weaviate.go`, `pkg/draftrag/weaviate_test.go`
- **Наблюдение:** `go build ./...` проходит без ошибок, тесты используют новые имена
- **Зависимость:** Нет

### AC-002: Hybrid search или документирование ограничения
- **Подход:** Исследование Weaviate API на поддержку BM25
  - Если поддерживается → реализовать hybrid search через Weaviate API
  - Если НЕ поддерживается → явно документировать ограничение в `docs/weaviate.md` и `docs/compatibility.md`
- **Surfaces:** `internal/infrastructure/vectorstore/weaviate.go` (только если BM25 поддерживается), `docs/weaviate.md`, `docs/compatibility.md`
- **Наблюдение:** `docs/weaviate.md` содержит секцию о hybrid search с явным статусом
- **Зависимость:** Зависит от ответа на Q1 (поддерживает ли Weaviate BM25)

### AC-003: Тестовое покрытие
- **Подход:** Добавление тестов для auth (401/403), errors (404, 500, network), edge cases
- **Surfaces:** `pkg/draftrag/weaviate_test.go`
- **Наблюдение:** Coverage >= 90%, все тесты проходят
- **Зависимость:** Нет

### AC-004: Production документация
- **Подход:** Расширение `docs/weaviate.md` секциями: production checklist, performance guidance, migration guide, troubleshooting
- **Surfaces:** `docs/weaviate.md`
- **Наблюдение:** Документация содержит все секции из RQ-004
- **Зависимость:** Зависит от AC-001 (migration guide требует знания новых имён функций)

### AC-005: Статус backend'а
- **Подход:** Изменение статуса в `docs/compatibility.md` experimental → stable
- **Surfaces:** `docs/compatibility.md`, `ROADMAP.md`
- **Наблюдение:** `docs/compatibility.md` показывает Weaviate = stable
- **Зависимость:** Зависит от AC-001–AC-004 (все должны быть выполнены)

---

## Данные и контракты

### Data model
- **Изменения не требуются** — структура `WeaviateOptions` остаётся без изменений
- **Новые сущности не вводятся**

### API contracts
- **Breaking change:** Переименование функций в `pkg/draftrag/weaviate.go`
  - Старые имена: `WeaviateCollectionExists`, `CreateWeaviateCollection`, `DeleteWeaviateCollection`
  - Новые имена: зависят от решения Q3 (префикс или без)
  - Compatibility сохраняется через migration guide (breaking change до v1.0 допустим)

- **Новые contracts не вводятся** — только переименование существующих

---

## Стратегия реализации

### DEC-001: Переименование с коротким префиксом Weaviate*
- **Why:** В пакете `draftrag` уже есть `CreateCollection`, `DeleteCollection`, `CollectionExists` для Qdrant. Go требует уникальности имён в пакете. Короткий префикс `Weaviate*` короче и чище чем `WeaviateDB*`, но обеспечивает уникальность.
- **Tradeoff:** Не полностью убирает префикс (как хотелось бы для максимальной консистентности), но это ограничение Go. Альтернатива — разнести Qdrant и Weaviate в разные пакеты, но это избыточное усложнение.
- **Affects:** `pkg/draftrag/weaviate.go`, `pkg/draftrag/weaviate_test.go`
- **Validation:** `go build ./...` проходит без конфликтов имён, тесты используют новые имена

### DEC-002: Hybrid search через явное документирование ограничения
- **Why:** Weaviate НЕ поддерживает BM25 нативно (как ChromaDB). Реализация через external index избыточна и не соответствует философии draftRAG (минимализм > расширяемость). Явное документирование ограничения честно с пользователями и позволяет им выбирать pgvector/Qdrant для hybrid search.
- **Tradeoff:** Weaviate не будет иметь parity с pgvector по hybrid search, но это ограничение понятно и документировано. Альтернатива — сложная реализация external index, что увеличивает сложность без явной бизнес-ценности.
- **Affects:** `docs/weaviate.md`, `docs/compatibility.md` (только документация, без кода)
- **Validation:** `docs/weaviate.md` содержит секцию "Ограничения" с явным указанием "Hybrid search (BM25) не поддерживается"

---

## Incremental Delivery

### MVP (Первая ценность)

- **Задачи:**
  1. Переименование функций (DEC-001)
  2. Обновление тестов для новых имён
  3. Базовое расширение тестового покрытия (auth, errors)
  4. Документирование ограничения hybrid search (DEC-002)

- **Покрывает AC:** AC-001, AC-002 (частично — документирование), AC-003 (частично)

- **Критерий готовности:** `go build ./...` проходит, `go test ./pkg/draftrag -run Weaviate` проходит, документация содержит секцию о hybrid search

### Итеративное расширение

- **Шаг 1 (после MVP):**
  - Полное тестовое покрытие (edge cases, network errors)
  - Production документация (checklist, performance guidance)
  - **Покрывает AC:** AC-003 (полностью), AC-004

- **Шаг 2 (финальный):**
  - Migration guide
  - Изменение статуса experimental → stable
  - Обновление ROADMAP.md
  - **Покрывает AC:** AC-005

---

## Порядок реализации

```
1. Переименование функций (weaviate.go)
   → 2. Обновление тестов (weaviate_test.go)
      → 3. Исследование hybrid search (Weaviate docs)
         → 4. Документирование ограничения hybrid search (weaviate.md, compatibility.md)
            → 5. Расширение тестового покрытия (auth, errors, edge cases)
               → 6. Production документация (weaviate.md)
                  → 7. Migration guide (weaviate.md)
                     → 8. Изменение статуса experimental → stable (compatibility.md, ROADMAP.md)
                        → 9. Финальная проверка
```

**Параллельно:** Нельзя — каждая задача зависит от предыдущей (изменения API влияют на тесты, тесты должны проходить перед документацией).

---

## Риски

### Риск 1: Мы не можем однозначно ответить на Q1 (поддерживает ли Weaviate BM25)
- **Mitigation:** План использует допущение DEC-002 (Weaviate НЕ поддерживает BM25) на основе аналогии с ChromaDB. Если это неверно, план можно скорректировать после исследования.

### Риск 2: Пользователи с существующим кодом будут сломаны breaking change
- **Mitigation:** Breaking change допустим до v1.0. Migration guide в документации поможет пользователям перейти на новые имена.

### Риск 3: Тестовое покрытие не достигнет 90%
- **Mitigation:** План включает систематическое добавление тестов для всех сценариев (auth, errors, edge cases). Если coverage останется <90%, добавим дополнительные тесты в финальной фазе.

---

## Rollout и compatibility

### Breaking change
- **Тип:** Breaking change (переименование функций)
- **Обоснование:** Продукт не зарелизился (v0.x), изменения допустимы
- **Migration guide:** Требуется в `docs/weaviate.md` с таблицей старых/новых имён

### Feature flags
- **Не требуются** — изменения сразу применяются ко всем пользователям

### Monitoring
- **Не требуется** — это библиотека, не сервис. Пользователи сами мониторят свои приложения.

---

## Проверка

| Шаг | Проверка | Подтверждает |
|-----|----------|--------------|
| 1 | `go build ./...` | AC-001 (нет ошибок компиляции) |
| 2 | `go test ./pkg/draftrag -run Weaviate` | AC-001, AC-003 (тесты проходят) |
| 3 | `go test ./...` | AC-003 (все тесты зелёные) |
| 4 | `go test -cover ./pkg/draftrag` | AC-003 (coverage >= 90%) |
| 5 | Ручной просмотр docs/weaviate.md | AC-002, AC-004 (секции присутствуют) |
| 6 | Ручной просмотр docs/compatibility.md | AC-002, AC-005 (матрица обновлена, статус stable) |
| 7 | Ручной просмотр ROADMAP.md | AC-005 (статус актуализирован) |

---

## Соответствие конституции

- **Нет конфликтов** — план соответствует принципам конституции:
  - Clean Architecture: изменения только в pkg/draftrag (публичный API) и docs
  - Language policy: документация на русском, godoc на русском
  - Development workflow: feature branch создан, spec перед кодом
  - Decision priorities: простота > расширяемости (отказ от external index для hybrid search)

---

**Статус:** Готова к планированию  
**Создан:** 2026-04-14  
**На основе:** spec.md (v1)
