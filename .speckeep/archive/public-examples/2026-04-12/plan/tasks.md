# Публичные production-ready примеры в README — Задачи

## Phase Contract

Inputs: `.speckeep/specs/public-examples/plan/plan.md` и текущий `README.md`.
Outputs: обновлённый `README.md` с production-ready примерами и проверкой консистентности.
Stop if: нельзя собрать примеры только на публичном API без расширения scope.

## Surface Map

| Surface | Tasks |
|---------|-------|
| README.md | T1.1, T2.1, T2.2, T3.1 |

## Фаза 1: Основа

Цель: подготовить структуру README так, чтобы примеры легко читались и не конфликтовали с существующим “Быстрым стартом”.

- [x] T1.1 Добавить раздел “Production-ready” и зафиксировать ориентиры. Touches: README.md
  - Outcome: новый раздел с короткой оговоркой и списком таймаутов/ретраев/кеша.
  - Links: RQ-001, RQ-002, RQ-003, RQ-004, DEC-002

## Фаза 2: Основная реализация

Цель: добавить 2 end-to-end code-block примера с корректным wiring и конкретными таймаутами.

- [x] T2.1 Добавить pgvector end-to-end пример (cache+retry+timeouts). Touches: README.md
  - Outcome: code-block с `NewPGVectorStoreWithOptions` + `NewCachedEmbedder` + retry/CB + pipeline.
  - Links: AC-001, AC-003, RQ-001, RQ-002, RQ-003, RQ-004

- [x] T2.2 Добавить Qdrant end-to-end пример (cache+retry+timeouts). Touches: README.md
  - Outcome: code-block с `NewQdrantStore` (+ опционально create/exists) + cache + retry/CB + pipeline.
  - Links: AC-002, AC-003, RQ-001, RQ-002, RQ-003, RQ-004

## Фаза 3: Проверка

Цель: доказать, что примеры соответствуют публичному API и acceptance criteria.

- [x] T3.1 Проверить консистентность примеров с публичным API. Touches: README.md
  - Outcome: `go test ./...` проходит; примеры не используют неэкспортируемые символы.
  - Links: AC-001, AC-002, AC-003, DEC-001

## Покрытие критериев приемки

- AC-001 -> T2.1, T3.1
- AC-002 -> T2.2, T3.1
- AC-003 -> T1.1, T2.1, T2.2

## Заметки

- Таймауты в примерах задать числами (например, индексация дольше, запрос/ответ короче) и использовать `defer cancel()`.
- Redis L2 показать как опциональный snippet через `CacheOptions.Redis` без добавления зависимостей и без конкретного клиента.
