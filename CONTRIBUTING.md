# Contributing to draftRAG

Спасибо за интерес к проекту! Этот документ описывает workflow разработки через SpecKeep (`.speckeep/`).

## Быстрый старт

```bash
# 1. Запустите оболочку SpecKeep (покажет справку и доступные команды)
./.speckeep/scripts/run-speckeep.sh

# 2. Быстрый health check
./.speckeep/scripts/run-speckeep.sh doctor .

# 3. Или используйте IDE slash-команды (Claude Code / Windsurf / Cursor / Codex)
/speckeep.constitution  # обновить constitution
/speckeep.spec          # создать/уточнить спецификацию (branch-first)
/speckeep.inspect       # проверить spec
/speckeep.plan          # создать/уточнить plan package
/speckeep.tasks         # разложить на задачи
/speckeep.implement     # реализовать задачи
/speckeep.verify        # подтвердить AC
/speckeep.archive       # архивировать фичу (move-based)
```

## Структура `.speckeep/`

```
.speckeep/
├── constitution.md          # Принципы и стандарты проекта
├── constitution.summary.md  # Краткая версия
├── specs/                   # Спецификации фич
│   └── <slug>/
│       ├── spec.md
│       ├── inspect.md
│       ├── summary.md
│       └── plan/
│           ├── plan.md
│           ├── data-model.md
│           ├── tasks.md
│           └── verify.md
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

Создаётся через `/speckeep.spec --name <slug>`:

- Декомпозиция фичи
- Acceptance Criteria (AC-\*)
- Зависимости и риски
- Структура кода

Пример slug: `streaming-llm`, `qdrant-store`, `hyde-retrieval`

Важно: SpecKeep требует **branch-first**. До записи `.speckeep/specs/<slug>/spec.md` переключитесь/создайте ветку `feature/<slug>`.

### 3. Inspect — проверка спецификации

`/speckeep.inspect <slug>` — проверяет:

- Согласованность с constitution
- Полноту acceptance criteria
- Корректность структуры

### 4. Plan — план реализации

Создаётся через `/speckeep.plan <slug>`:

- Архитектурные решения (DEC-\*)
- Дизайн компонентов
- Surfaces и порядок реализации
- Data model и contracts (если нужны)

### 5. Tasks — список задач

`/speckeep.tasks <slug>` — создаёт/обновляет `.speckeep/specs/<slug>/plan/tasks.md`:

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

`/speckeep.archive <slug>` — перемещает feature package в `archive/`.

Примечание: CLI-архивация требует наличия `.speckeep/specs/<slug>/plan/verify.md` (если вы делали verify только “в чат”, сохраните отчёт через `--persist`).

## Скрипты

### run-speckeep.sh

Главный wrapper для CLI-работы с SpecKeep:

```bash
./.speckeep/scripts/run-speckeep.sh
# или с командой:
./.speckeep/scripts/run-speckeep.sh check redis-cache-backend .
```

### Ready-check скрипты

```bash
./.speckeep/scripts/check-spec-ready.sh <slug>    # проверка спецификации (в т.ч. feature branch)
./.speckeep/scripts/check-inspect-ready.sh <slug> # проверка readiness для inspect
./.speckeep/scripts/check-plan-ready.sh <slug>    # проверка readiness для plan
./.speckeep/scripts/check-tasks-ready.sh <slug>   # проверка readiness для tasks
./.speckeep/scripts/check-verify-ready.sh <slug>  # проверка readiness для verify
./.speckeep/scripts/check-archive-ready.sh ...    # проверка readiness для archive (см. использование в CLI)
```

## Для контрибьюторов

### Добавление новой фичи

1. Прочитайте `constitution.md`
2. Создайте спецификацию: `/speckeep.spec --name <slug>`
3. Пройдите inspect: `/speckeep.inspect <slug>`
4. Создайте план: `/speckeep.plan <slug>`
5. Создайте список задач: `/speckeep.tasks <slug>`
6. Выполните: `/speckeep.implement <slug>`
7. Проверьте: `/speckeep.verify <slug>` (при необходимости сохраните отчёт через `--persist`)
8. Архивируйте: `/speckeep.archive <slug>`

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

Используйте `@sk-task` (и при необходимости `@sk-test`) для traceability:

```go
// @sk-task T1.1: Создать структуру QdrantStore с HTTP клиентом (RQ-001, RQ-002)
type QdrantStore struct { ... }
```

### Логирование

- В продакшен-коде: без `fmt.Println` для дебага
- В тестах: можно использовать `t.Logf`
- Для observability: предпочитайте hook-интерфейсы/опции, не `log.Printf` напрямую (если есть выбор)

## Тестирование

### Unit-тесты

```bash
go test ./pkg/draftrag/...
go test ./internal/...
go test ./...
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
