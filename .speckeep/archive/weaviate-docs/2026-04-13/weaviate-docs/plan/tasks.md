# Weaviate docs — Задачи

## Phase Contract

Inputs: `.speckeep/specs/weaviate-docs/spec.md`, `.speckeep/specs/weaviate-docs/plan/plan.md`, текущие `docs/vector-stores.md` и `docs/compatibility.md`.
Outputs: `docs/weaviate.md` + обновления ссылок в docs.
Stop if: требуется менять код/публичный API для выполнения AC — это вне scope; ограничиться корректными формулировками “best-effort” и документировать текущие возможности.

## Surface Map

| Surface | Tasks |
|---------|-------|
| docs/weaviate.md | T1.1, T1.2, T1.3, T2.1, T2.2, T4.1 |
| docs/vector-stores.md | T3.1, T4.1 |
| docs/compatibility.md | T3.2, T4.1 |

## Фаза 1: Документ Weaviate (каркас + quickstart)

Цель: дать пользователю копипастабельный “production-minded” старт.

- [x] T1.1 Создать `docs/weaviate.md` со структурой и кратким введением (best-effort, без SLA). Touches: docs/weaviate.md
  - Outcome: документ на русском с разделами: “Быстрый старт”, “Управление коллекцией”, “Возможности/ограничения”, “Типовые ошибки”, “Ссылки” (AC-001).
  - Links: AC-001, RQ-001

- [x] T1.2 Добавить пример создания store через публичный API `NewWeaviateStore(WeaviateOptions{...})`. Touches: docs/weaviate.md
  - Outcome: пример показывает необходимые поля (`Host`, `Collection`, опц. `APIKey`, `Timeout`), ошибки валидации и рекомендуемые таймауты через `context.WithTimeout` (AC-002).
  - Links: AC-002, RQ-002

- [x] T1.3 Добавить блок “подготовка коллекции” (deploy job/init) с функциями `WeaviateCollectionExists/CreateWeaviateCollection/DeleteWeaviateCollection`. Touches: docs/weaviate.md
  - Outcome: документ явно рекомендует запускать schema/DDL как отдельный шаг деплоя; показан идемпотентный flow `exists -> create` (AC-002).
  - Links: AC-002, RQ-003

## Фаза 2: Возможности и troubleshooting

Цель: зафиксировать, что поддерживается и как дебажить типовые ошибки.

- [x] T2.1 Секция “Возможности/ограничения” (filters/metadata, no hybrid). Touches: docs/weaviate.md
  - Outcome: перечислены поддерживаемые возможности (по публичному API и docs) и явное ограничение “нет hybrid search BM25” (AC-003).
  - Links: AC-003, RQ-004

- [x] T2.2 Секция “Типовые ошибки” (collection missing, auth, timeout). Touches: docs/weaviate.md
  - Outcome: краткие checks/actions для 404/collection missing, 401/403 auth, `context deadline exceeded` (AC-003).
  - Links: AC-003

## Фаза 3: Discoverability и согласованность docs

Цель: сделать документ обнаруживаемым и выровнять текущие матрицы.

- [x] T3.1 Обновить `docs/vector-stores.md`: добавить Weaviate в список store + ссылку на `docs/weaviate.md`; расширить таблицу “Сравнение” колонкой Weaviate. Touches: docs/vector-stores.md
  - Outcome: пользователь находит Weaviate в обзорном документе и переходит в подробную страницу (AC-001).
  - Links: AC-001, RQ-005

- [x] T3.2 Обновить `docs/compatibility.md`: заменить примечание “нет дока” на ссылку на `docs/weaviate.md` и выровнять формулировки статуса. Touches: docs/compatibility.md
  - Outcome: политика совместимости не противоречит docs; заметка про Weaviate указывает на `docs/weaviate.md`.
  - Links: RQ-005

## Фаза 4: Самопроверка по AC

Цель: подтвердить критерии приемки и отсутствие ссылок на `internal/`.

- [x] T4.1 Self-review: пройтись по AC-001..AC-003 и проверить consistency ссылок. Touches: docs/weaviate.md, docs/vector-stores.md, docs/compatibility.md
  - Outcome: (1) ссылки ведут на `docs/weaviate.md`; (2) quickstart использует `context.WithTimeout`; (3) нет ссылок на `internal/`; (4) формулировки best-effort/без SLA (AC-001..AC-003).
  - Links: AC-001, AC-002, AC-003

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1, T4.1
- AC-002 -> T1.2, T1.3, T4.1
- AC-003 -> T2.1, T2.2, T4.1
