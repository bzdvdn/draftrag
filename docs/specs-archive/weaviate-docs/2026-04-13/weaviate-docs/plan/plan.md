# Weaviate docs — План

## Phase Contract

Inputs: `.speckeep/specs/weaviate-docs/spec.md`, `.speckeep/specs/weaviate-docs/inspect.md`, текущие `docs/vector-stores.md`, `docs/compatibility.md`.
Outputs: `.speckeep/specs/weaviate-docs/plan/plan.md`, `.speckeep/specs/weaviate-docs/plan/data-model.md`.
Stop if: для выполнения AC требуется менять реализацию Weaviate store или публичный API (вне scope) — тогда скорректировать ожидания/формулировки в docs, но не трогать код.

## Цель

Добавить обнаруживаемую и “production-minded” документацию `docs/weaviate.md`, которая описывает только публичный API `pkg/draftrag` и закрывает практические вопросы: подготовка коллекции, подключение store, индексация/поиск, возможности/ограничения и troubleshooting.

## Scope

- Новый документ: `docs/weaviate.md` (русский).
- Обновить `docs/vector-stores.md`: добавить Weaviate в список хранилищ и в таблицу сравнения, при этом подробности вынести в `docs/weaviate.md`.
- Обновить `docs/compatibility.md`: убрать/заменить примечание “отдельного дока нет” на ссылку на `docs/weaviate.md` и выровнять формулировки статуса.
- Без изменений кода библиотеки.

## Implementation Surfaces

- `docs/weaviate.md` (новая поверхность): quickstart, управление коллекциями, возможности, troubleshooting.
- `docs/vector-stores.md` (существующая): короткий блок про Weaviate + ссылка, обновление таблицы сравнения.
- `docs/compatibility.md` (существующая): исправить notes для Weaviate и, при необходимости, таблицу возможностей (consistent с `docs/weaviate.md`).

## Acceptance Approach

- AC-001 -> `docs/weaviate.md` создан + ссылка добавлена в `docs/vector-stores.md`.
- AC-002 -> в `docs/weaviate.md` есть quickstart, который покрывает:
  - подготовку коллекции (`WeaviateCollectionExists/CreateWeaviateCollection/DeleteWeaviateCollection`) с рекомендацией запускать как deploy job/init,
  - создание store через `NewWeaviateStore(WeaviateOptions{...})`,
  - `Pipeline.Index` и `Search(...).Retrieve(...)` с `context.WithTimeout`.
- AC-003 -> в `docs/weaviate.md` есть секции “Возможности/ограничения” и “Типовые ошибки”, и формулировки не обещают SLA.

## Дизайн-решения

- DEC-001 Отдельная страница `docs/weaviate.md`, а в `docs/vector-stores.md` — только короткий обзор + ссылка.
  Why: `docs/vector-stores.md` уже содержит подробные страницы для других store; Weaviate добавляем симметрично и не раздуваем обзорный документ.
  Affects: `docs/weaviate.md`, `docs/vector-stores.md`.

- DEC-002 Управление коллекцией документируется как “deployment concern”.
  Why: в production DDL/schema обычно не выполняют при старте сервиса.
  Affects: `docs/weaviate.md`.

- DEC-003 Статус Weaviate остаётся `experimental`, пока нет отдельной страницы и/или больше production сигналов.
  Why: соответствует текущей политике совместимости и снижает риск завышенных ожиданий.
  Affects: `docs/compatibility.md`.

## Порядок реализации

1. Скелет `docs/weaviate.md` (структура → quickstart → управление коллекциями).
2. Заполнить “Возможности/ограничения” и “Типовые ошибки”.
3. Обновить `docs/vector-stores.md`: добавить Weaviate + ссылку + колонку в таблице сравнения.
4. Обновить `docs/compatibility.md`: заменить примечание на ссылку и выровнять матрицу.
5. Самопроверка: AC-001..AC-003, отсутствие ссылок на `internal/`, язык = русский.

## Риски

- Риск: docs будет расходиться с реальным поведением.
  Mitigation: явно пометить “best-effort”, опираться на публичный API и существующие тесты/доки; избегать неподтверждённых обещаний.

## Проверка

- Manual:
  - ссылки кликабельны и ведут на `docs/weaviate.md`,
  - quickstart читабелен и использует `context.WithTimeout`,
  - `docs/compatibility.md` больше не содержит “нет дока” для Weaviate.

