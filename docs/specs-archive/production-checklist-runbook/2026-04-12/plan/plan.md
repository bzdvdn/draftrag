# Production checklist + runbook — План

## Phase Contract

Inputs: `.speckeep/specs/production-checklist-runbook/spec.md`, `.speckeep/specs/production-checklist-runbook/inspect.md`, текущий `README.md` и существующие docs в `docs/`.
Outputs: `.speckeep/specs/production-checklist-runbook/plan/plan.md`, `.speckeep/specs/production-checklist-runbook/plan/data-model.md`.
Stop if: потребуется менять публичный API или добавлять новый backend ради документа (вне scope).

## Цель

Добавить один практичный документ в `docs/production.md` (checklist + runbook) и ссылку на него из `README.md`, чтобы пользователи могли быстро подготовить сервис к production и иметь пошаговые инструкции на инциденты. “Быстро” трактуется как “коротко и по шагам”, без обещаний latency/SLO.

## Scope

- Новый документ: `docs/production.md` (русский), структура: `Checklist` + `Runbook` + `Backend notes` + `Security/Redaction`.
- Правка `README.md`: добавить явную ссылку на `docs/production.md` в секции документации/примеров.
- Без изменений кода библиотеки (кроме README ссылки).

## Implementation Surfaces

- `docs/production.md` (новая поверхность): основной документ.
- `README.md` (существующая поверхность): ссылка на документ и короткое пояснение “production entrypoint”.

## Влияние на архитектуру

- Архитектура/публичный API не меняются; фича документационная.
- Compatibility/rollout: нет миграций, флагов и breaking changes.

## Acceptance Approach

- AC-001 -> добавить `docs/production.md` + ссылку в `README.md`; observable proof: ссылка кликабельна и файл существует.
- AC-002 -> в `docs/production.md` сделать checklist 5–15 пунктов, каждый — проверяемое действие, с ссылками на существующие секции README/docs вместо дублирования больших блоков текста.
- AC-003 -> в `docs/production.md` сделать runbook минимум 4 инцидента по одному шаблону `Symptoms / Checks / Actions` (короткие шаги).
- AC-004 -> в `docs/production.md` добавить раздел про безопасность логов: redaction best-effort + границы ответственности.

## Данные и контракты

- Data model: не требуется.
- API/event contracts: не вводятся.

## Стратегия реализации

- DEC-001 Документ живёт в `docs/production.md`
  Why: `docs/` уже содержит русскоязычные “how-to”; `production.md` — узнаваемый entrypoint.
  Tradeoff: нужен stable URL/путь; в дальнейшем избегать переименований без редиректа (для ссылок).
  Affects: `docs/production.md`, `README.md`.
  Validation: AC-001.

- DEC-002 Checklist как “index” на существующие детали
  Why: снижает расхождение и дублирование с README и профильными docs.
  Tradeoff: чеклист зависит от актуальности ссылок; нужно следить за ними при изменениях docs.
  Affects: `docs/production.md`.
  Validation: AC-002 (каждый пункт — действие + ссылка).

- DEC-003 Runbook минимум 4 инцидента, единый шаблон
  Why: в инциденте важна повторяемая структура и быстрый “next action”.
  Tradeoff: часть инцидентов будет “best-effort” без гарантии причин; важно не обещать SLO.
  Affects: `docs/production.md`.
  Validation: AC-003.

## Incremental Delivery

### MVP (Первая ценность)

- `docs/production.md` с checklist (5–15 пунктов) и runbook (>=4 инцидента).
- Ссылка из `README.md`.
- Раздел про security/redaction.

### Итеративное расширение

- Добавить короткие “backend notes” таблицей различий (pgvector/Qdrant/Weaviate) и отдельные runbook-ветки для них.
- Добавить ссылки на конкретные метрики/атрибуты OTel hooks и рекомендации по кардинальности.

## Порядок реализации

- Сначала: создать `docs/production.md` со структурой и минимальным содержимым по AC-002..AC-004.
- Затем: добавить ссылку в `README.md` (AC-001).
- В конце: быстрый markdown sanity-check (линки/якоря) и review на “без двусмысленности”.

## Риски

- Риск: документ быстро устареет при изменениях README/docs.
  Mitigation: делать документ “index”-формата и ссылаться на existing sections; избегать копипаста.
- Риск: checklist воспринимают как “гарантию прод-готовности”.
  Mitigation: в начале документа коротко указать, что это стартовые практики, не гарантия SLO.

## Rollout и compatibility

- Специальные rollout-действия не требуются.

## Проверка

- Manual review:
  - AC-001: ссылка из `README.md` ведёт на `docs/production.md`.
  - AC-002: checklist 5–15 пунктов, каждый пункт проверяем и не двусмысленен.
  - AC-003: 4+ runbook инцидента с шаблоном `Symptoms/Checks/Actions`.
  - AC-004: есть раздел про redaction и границы ответственности.

## Соответствие конституции

- Контекстная безопасность: документ подчёркивает использование `context.Context` и таймаутов как базовую практику.
- Минимальная конфигурация: рекомендации основаны на текущих defaults и существующих опциях, без навязывания новых обязательных компонентов.
- Документация на русском: `docs/production.md` и README-ссылка — на русском языке.

