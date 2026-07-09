# Security redaction: секреты и PII в ошибках и логах — План

## Phase Contract

Inputs: `.speckeep/specs/security-redaction/spec.md`, `.speckeep/specs/security-redaction/inspect.md`, текущие провайдеры/логирование (LLM/Embedder/VectorStore + `domain.Logger` + `SafeLog`).
Outputs: `.speckeep/specs/security-redaction/plan/plan.md`, `.speckeep/specs/security-redaction/plan/data-model.md`.
Stop if: для выполнения требований потребуется общий PII/DLP анализ контента документов/запросов (это вне scope).

## Цель

Сделать редактирование секретов “по умолчанию” и единообразным: ошибки, которые возвращает библиотека, и structured logs, которые библиотека сама эмитирует, не должны содержать значений известных библиотеке секретов (API keys/Authorization bearer tokens). При этом сообщения остаются полезными для диагностики (статус/код/operation/stage).

## Scope

- Унифицировать редактирование секретов в местах формирования ошибок для HTTP-провайдеров (LLM/Embedder) и сторов, где есть `APIKey`.
- Обеспечить, что ошибки, попадающие в structured logs (через `err` field), уже отредактированы на источнике.
- Добавить unit-тесты: минимум один LLM + один Embedder сценарий (AC-001) и сценарий с logger-коллектором через resilience/cache (AC-002).
- Обновить README/доки: короткая секция про redaction и границы ответственности (AC-003).

## Implementation Surfaces

- `internal/domain/` (существующая поверхность): добавить общий helper для редактирования секретов (например, `redaction.go`) и небольшой контракт `<redacted>`/no-op.
- `internal/infrastructure/llm/` (существующая поверхность): привести редактирование к общему helper и закрыть места, где есть `APIKey` и формируется `body=%q` snippet (Anthropic, OpenAI-compatible, Ollama).
- `internal/infrastructure/embedder/` (существующая поверхность): привести редактирование к общему helper (OpenAI-compatible, Ollama).
- `pkg/draftrag/*` (существующая поверхность): при необходимости — тесты на уровне публичного API, которые подтверждают отсутствие секрета в `err.Error()`.
- `internal/infrastructure/resilience/*` и `internal/infrastructure/embedder/cache/*` (существующая поверхность): тест через logger-коллектор, чтобы доказать, что `err` в логах не несёт секрет.
- `README.md` и/или `docs/*` (существующая поверхность): добавить секцию “Redaction / безопасность логов”.

## Влияние на архитектуру

- Clean Architecture сохраняется: редактирование — утилита доменного слоя (`internal/domain`) и использование на границе инфраструктуры (HTTP провайдеры).
- Публичный API не ломается: изменения ограничены текстами ошибок, утилитами internal и тестами/доками.

## Acceptance Approach

- AC-001 -> выбрать минимальный набор поверхностей и явно покрыть:
  - LLM: OpenAI-compatible (Responses) **или** Anthropic;
  - Embedder: OpenAI-compatible;
  - дополнительно, если есть `APIKey` surface у store (например, Weaviate) — либо покрыть, либо явно зафиксировать как “вне MVP” на уровне tasks.
  Evidence: тесты на `pkg/draftrag` и/или `internal/infrastructure/*`, которые создают сервер, возвращающий body с секретом, и проверяют отсутствие секрета в `err.Error()`.

- AC-002 -> доказать через logger-коллектор:
  - собрать pipeline/provider с секретом;
  - инициировать ошибку, которая проходит через `Retry*`/cache logging;
  - проверить, что `msg` и `fields` не содержат секрет.
  Evidence: unit-тест собирает logger, который сохраняет `msg` и `fields` как строки, и assert’ит отсутствие секрета.

- AC-003 -> добавить секцию в README рядом с логированием:
  - “что редактируется”: известные библиотеке секреты (APIKey/Authorization);
  - “что не редактируется”: произвольный текст документов/запросов и пользовательские логи;
  - рекомендация: не логировать сырой контент без своей политики.
  Evidence: README содержит новый абзац/подсекцию.

## Данные и контракты

- Data model: не требуется.
- API/event contracts: не вводятся.
- Контракт редактирования (внутренний, но документируемый):
  - маркер: `<redacted>`;
  - пустой секрет → no-op;
  - replace-all: редактируются все вхождения.

## Стратегия реализации

- DEC-001 Редактировать “на источнике”, где библиотека знает секрет
  Why: компоненты вроде retry/cache не знают APIKey, но логируют `err`; если `err` редактирован при формировании — утечки не будет в downstream логах.
  Tradeoff: нужно аккуратно покрыть все места формирования ошибок с body snippet.
  Affects: `internal/infrastructure/llm/*`, `internal/infrastructure/embedder/*`, `internal/domain/redaction.go`.
  Validation: AC-001/AC-002 тесты.

- DEC-002 Общий helper в `internal/domain`, без попытки PII detection
  Why: единообразие и низкий риск; соответствует “best-effort” и out-of-scope PII/DLP.
  Tradeoff: не защитит от секретов/PII, которые библиотека не знала (например, пользователь вложил их в документ).
  Affects: `internal/domain/redaction.go`, README.
  Validation: unit-тесты на helper (опционально) и провайдерные тесты.

- DEC-003 Structured logging: не менять типы полей, если нет необходимости
  Why: пользователи могут ожидать `err` как `error`; лучше редактировать текст ошибки до логирования (через источник ошибки), чем превращать поле в строку повсеместно.
  Tradeoff: если где-то секрет попадёт в `err` не из provider’а, лог может утечь; это покрываем targeted tests и добавляем checklist в review.
  Affects: провайдерные error constructors + тест через logger-коллектор.
  Validation: AC-002.

## Incremental Delivery

### MVP (Первая ценность)

- Ввести общий helper редактирования.
- Закрыть 2 наиболее вероятных источника утечек: OpenAI-compatible LLM + OpenAI-compatible Embedder (или Anthropic вместо одного из них).
- Добавить logger-коллектор тест на отсутствие секрета в `err` поле через retry/cache.
- Добавить README секцию про redaction.

### Итеративное расширение

- Расширить coverage на Ollama `APIKey` (если используется) и store с `APIKey` (Weaviate).
- Добавить дополнительные тесты на wrapped errors (nested `%w`) и multiple occurrences.

## Порядок реализации

- Сначала: выбрать “secret sources” и минимальный coverage список (фиксирует границы и закрывает warning из inspect).
- Затем: вынести/унифицировать `redactSecret` в `internal/domain`.
- Затем: обновить провайдеры, где формируется error с `body=%q` snippet, чтобы использовать общий helper.
- После: добавить tests (AC-001/AC-002), затем docs (AC-003).

## Риски

- Риск: пропустить редкий error-path, который включает секрет в message.
  Mitigation: централизованный helper + grep по шаблонам формирования body snippet + targeted tests по ключевым провайдерам.
- Риск: ухудшить диагностику из-за чрезмерной редактции.
  Mitigation: редактировать только известные секреты; оставлять статус/код/operation/stage.

## Rollout и compatibility

- Breaking changes не ожидаются; меняются тексты ошибок (это может затронуть snapshot-based тесты пользователя, но считается приемлемым).
- Специальных rollout-действий нет.

## Проверка

- `go test ./...` (базовая регрессия).
- Тесты redaction:
  - AC-001: LLM + Embedder — секрет не присутствует в `err.Error()`.
  - AC-002: logger-коллектор — секрет не присутствует в `msg` и `fields` при logging путях.
  - AC-003: README содержит секцию про redaction.

## Соответствие конституции

- Контекстная безопасность: редактирование не меняет `context` поведение и не блокирует отмену/таймауты.
- Минимальная конфигурация: редактирование работает по умолчанию, без дополнительных настроек со стороны пользователя.
- Тестируемость: добавляются unit-тесты, покрывающие риск утечек секретов.
- Документация: изменения в README/доках на русском языке.

