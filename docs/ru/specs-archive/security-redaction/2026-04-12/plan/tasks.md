# Security redaction: секреты и PII в ошибках и логах — Задачи

## Phase Contract

Inputs: `.speckeep/specs/security-redaction/plan/plan.md`, текущие провайдеры/логирование.
Outputs: единый helper redaction + обновлённые провайдеры/логи + тесты + docs.
Stop if: требуется PII/DLP для произвольного текста (вне scope).

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/ | T1.1 |
| internal/infrastructure/llm/ | T2.1 |
| internal/infrastructure/embedder/ | T2.2 |
| pkg/draftrag/weaviate.go | T2.3 |
| README.md | T2.4 |
| pkg/draftrag/*_test.go | T3.1 |
| internal/infrastructure/resilience/ | T3.2 |
| internal/infrastructure/embedder/cache/ | T3.2 |

## Фаза 1: Основа

Цель: зафиксировать единый контракт redaction и убрать дублирование.

- [x] T1.1 Добавить общий helper redaction в `internal/domain`. Touches: internal/domain/
  - Outcome: единый helper редактирует секреты (`<redacted>`, replace-all, no-op для пустых).
  - Links: RQ-001, RQ-002, RQ-004, DEC-002

## Фаза 2: Основная реализация

Цель: применить redaction в местах формирования ошибок/логов и задокументировать границы.

- [x] T2.1 Унифицировать redaction в HTTP LLM провайдерах. Touches: internal/infrastructure/llm/
  - Outcome: ошибки `... status=%d body=%q` редактируют известный секрет через общий helper.
  - Links: AC-001, RQ-001, DEC-001

- [x] T2.2 Унифицировать redaction в HTTP Embedder провайдерах. Touches: internal/infrastructure/embedder/
  - Outcome: ошибки `... status=%d body=%q` редактируют известный секрет через общий helper.
  - Links: AC-001, RQ-001, DEC-001

- [x] T2.3 Редактировать Weaviate `APIKey` в ошибках коллекции (если body содержит секрет). Touches: pkg/draftrag/weaviate.go
  - Outcome: ошибки Weaviate не содержат `APIKey` (замена на `<redacted>`).
  - Links: RQ-001

- [x] T2.4 Добавить секцию в README про redaction и границы ответственности. Touches: README.md
  - Outcome: README описывает “что редактируется/что нет” и предупреждает про пользовательские логи/сырой контент.
  - Links: AC-003, RQ-005

## Фаза 3: Проверка

Цель: доказать отсутствие утечек секретов в ошибках и structured logs.

- [x] T3.1 Добавить тесты redaction для LLM и Embedder на уровне публичного API. Touches: pkg/draftrag/*_test.go
  - Outcome: минимум 1 LLM и 1 Embedder тест гарантируют отсутствие секрета в `err.Error()`.
  - Links: AC-001

- [x] T3.2 Добавить тест logger-коллектора: секрет не попадает в `msg`/`fields`. Touches: internal/infrastructure/resilience/, internal/infrastructure/embedder/cache/
  - Outcome: лог-события от retry/cache не содержат секретов при искусственно созданной ошибке.
  - Links: AC-002, RQ-002

- [x] T3.3 Прогнать `go test ./...` и проверить отсутствие регрессий. Touches: internal/domain/
  - Outcome: `go test ./...` зелёный.
  - Links: AC-001, AC-002

## Покрытие критериев приемки

- AC-001 -> T2.1, T2.2, T2.3, T3.1
- AC-002 -> T3.2, T3.3
- AC-003 -> T2.4

## Заметки

- Минимальный coverage для AC-001: OpenAI-compatible LLM + OpenAI-compatible Embedder; Weaviate — как явный store с `APIKey`.
- Structured logging: целевой путь — чтобы `err` был редактирован на источнике; не превращать поле `err` в строку без необходимости.
