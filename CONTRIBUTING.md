# Contributing to draftRAG

Спасибо за интерес к проекту! Этот документ описывает workflow разработки через speckeep.

## Быстрый старт

```bash
# 1. Убедитесь что speckeep workflow доступен
ls .speckeep/workflows/

# 2. Запустите readiness check
./.speckeep/scripts/run-speckeep.sh

# 3. Или используйте IDE-интеграцию (Claude Code / Windsurf / Cursor)
/speckeep.constitution  # обновить constitution
/speckeep.spec          # создать спецификацию
/speckeep.plan          # создать план
/speckeep.implement     # выполнить задачи
```

## Структура speckeep

```
.speckeep/
├── constitution.md          # Принципы и стандарты проекта
├── constitution.summary.md  # Краткая версия
├── specs/                   # Спецификации фич
│   └── <slug>/
│       └── spec.md
├── plans/                   # Планы реализации
│   └── <slug>/
│       ├── architecture.md
│       ├── design.md
│       └── tasks.md
├── archive/                 # Архивированные specs/plans
└── scripts/                 # Вспомогательные скрипты
```

## Workflow разработки

Цепочка: `constitution → spec → inspect → plan → tasks → implement → verify → archive`

### 1. Constitution — фундамент проекта

Файл `.speckeep/constitution.md` содержит:

- Цели и принципы проекта
- Архитектурные решения (DEC-\*)
- Требования (RQ-\*) и ограничения
- Кодстайл и соглашения

**Когда обновлять:** при изменении архитектуры или добавлении новых стандартов.

### 2. Spec — спецификация фичи

Создаётся через `/speckeep.spec <slug>`:

- Декомпозиция фичи
- Acceptance Criteria (AC-\*)
- Зависимости и риски
- Структура кода

Пример slug: `streaming-llm`, `qdrant-store`, `hyde-retrieval`

### 3. Inspect — проверка спецификации

`/speckeep.inspect <slug>` — проверяет:

- Согласованность с constitution
- Полноту acceptance criteria
- Корректность структуры

### 4. Plan — план реализации

Создаётся через `/speckeep.plan <slug>`:

- Архитектурные решения (DEC-\*)
- Дизайн компонентов
- Задачи (таск-лист)

### 5. Tasks — список задач

`/speckeep.tasks <slug>` — создаёт/обновляет `tasks.md`:

- Конкретные шаги реализации
- Приоритеты и зависимости
- Definition of Done для каждой задачи

### 6. Implement — выполнение

`/speckeep.implement <slug>` — последовательно выполняет задачи из `tasks.md`.

### 7. Verify — проверка

`/speckeep.verify <slug>` — проверяет:

- Все acceptance criteria выполнены
- Код соответствует спецификации
- Тесты проходят

### 8. Archive — архивация

`/speckeep.archive <slug>` — перемещает `spec/` и `plan/` в `archive/`.

## Скрипты

### run-speckeep.sh

Главный скрипт для CLI-работы с speckeep:

```bash
./.speckeep/scripts/run-speckeep.sh
# или с командой:
./.speckeep/scripts/run-speckeep.sh spec streaming-llm
```

### Ready-check скрипты

```bash
./.speckeep/scripts/check-constitution.sh      # проверка constitution
./.speckeep/scripts/check-spec-ready.sh <slug> # проверка спецификации
./.speckeep/scripts/check-plan-ready.sh <slug> # проверка плана
```

## Для контрибьюторов

### Добавление новой фичи

1. Прочитайте `constitution.md`
2. Создайте спецификацию: `/speckeep.spec <slug>`
3. Пройдите inspect: `/speckeep.inspect <slug>`
4. Создайте план: `/speckeep.plan <slug>`
5. Выполните: `/speckeep.implement <slug>`
6. Проверьте: `/speckeep.verify <slug>`
7. Архивируйте: `/speckeep.archive <slug>`

### Горячие исправления (hotfix)

Для мелких исправлений (≤3 файлов) без полного workflow:

```bash
/speckeep.hotfix
```

### Adversarial review

Перед завершением фичи рекомендуется:

```bash
/speckeep.challenge --spec <slug>   # проверка спецификации
/speckeep.challenge --plan <slug>   # проверка плана
```

## Code Style

### Комментарии

Кодовые комментарии на **русском** (согласно constitution). Только технические термины на английском:

```go
// НЕБУФЕРИЗОВАННЫЙ канал блокирует отправителя до готовности получателя.
// Буферизованный канал блокирует только при заполнении буфера.
```

### Task annotations

Используйте `@ds-task` для отслеживания требований:

```go
// @ds-task T1.1: Создать структуру QdrantStore с HTTP клиентом (RQ-001, RQ-002)
type QdrantStore struct { ... }
```

### Логирование

- В продакшен-коде: без `fmt.Println` для дебага
- В тестах: можно использовать `t.Logf`
- Для observability: используйте Hooks

## Тестирование

### Unit-тесты

```bash
go test ./pkg/draftrag/...
go test ./internal/...
```

### Интеграционные тесты

Требуют внешних сервисов (PostgreSQL, Qdrant):

```bash
PGVECTOR_TEST_DSN="postgres://..." go test ./pkg/draftrag/... -run PGVector
```

### Eval

```bash
go test ./pkg/draftrag/eval/...
```

## Pull Requests

1. Фича должна быть завершена через `speckeep.verify`
2. Все тесты проходят
3. Код соответствует constitution
4. Документация обновлена (при необходимости)

## Вопросы

- Технические вопросы: создайте issue с меткой `question`
- Баги: issue с меткой `bug` + минимальный reproduction
- Фичи: обсудите в issue перед созданием spec

---

Лицензия: MIT
