# Compatibility & Support Policy — План

## Phase Contract

Inputs: `.speckeep/specs/compatibility-support-policy/spec.md`, `.speckeep/specs/compatibility-support-policy/inspect.md`, текущие docs/README.
Outputs: `.speckeep/specs/compatibility-support-policy/plan/plan.md`, `.speckeep/specs/compatibility-support-policy/plan/data-model.md`.
Stop if: матрицы нельзя составить без чтения широких областей кода (в этом случае ограничить их по публичной документации/README).

## Цель

Добавить документ `docs/compatibility.md`, который явно фиксирует: поддержку версий Go, правила semver/депрекации публичного API `pkg/draftrag`, и 2 матрицы (backend’ы и возможности) для выбора production-конфигурации. Документ должен быть “источником истины” для статусов поддержки и обновляться вместе с релизами.

## Scope

- Новый документ: `docs/compatibility.md` (русский).
- Правка `README.md`: явная ссылка на `docs/compatibility.md` в блоке “Документация”.
- Без изменений кода библиотеки (документационная фича).

## Implementation Surfaces

- `docs/compatibility.md` (новая поверхность): политика поддержки и матрицы.
- `README.md` (существующая поверхность): ссылка на документ.

## Влияние на архитектуру

- Архитектура не меняется; фиксируется контракт поддержки и ожиданий пользователей.
- Rollout: нет миграций/флагов; изменения только в docs.

## Acceptance Approach

- AC-001 -> `docs/compatibility.md` создан + ссылка в `README.md`.
- AC-002 -> в документе есть секции:
  - `Go support`: минимум Go=1.23 и правило “поддерживаем N последних minor Go (N=2)”; пересмотр минимума — в minor релизах с notice.
  - `SemVer & Deprecation`: breaking changes только в major; deprecated API поддерживается минимум 2 minor релиза или 6 месяцев (что дольше); депрекации помечаются в godoc + changelog/release notes.
- AC-003 -> в документе есть 2 таблицы:
  - “Backends vs Status”: vector stores (in-memory, pgvector, Qdrant, Chroma, Weaviate), embedders (OpenAI-compatible, Ollama, CachedEmbedder), LLM (OpenAI-compatible, Anthropic, Ollama), статусы (stable/experimental).
  - “Features vs Backends”: streaming, filters, hybrid, cache, retry/CB, hooks/OTel, migrations; где применимо — отметить поддержку.

## Данные и контракты

- Data model: не требуется.
- API/event contracts: не вводятся.
- Контракт docs: документ является best-effort отражением текущего публичного API и docs, без обещаний SLA.

## Стратегия реализации

- DEC-001 Явное правило поддержки Go: “минимум + N последних minor”
  Why: снимает двусмысленность и соответствует enterprise ожиданиям планирования апгрейдов.
  Tradeoff: требует регулярного обновления при выходе новых Go minor.
  Affects: `docs/compatibility.md`.
  Validation: AC-002.

- DEC-002 Статусы backend’ов: stable/experimental (без “deprecated” в MVP)
  Why: проще поддерживать; deprecated добавлять только при реальной депрекации.
  Tradeoff: меньше гранулярности.
  Affects: `docs/compatibility.md`.
  Validation: AC-003.

- DEC-003 Матрица возможностей основана на docs/README, а не на полном аудите кода
  Why: удерживает scope и снижает необходимость широкого сканирования репозитория.
  Tradeoff: возможны пробелы; документ помечает “best-effort”.
  Affects: `docs/compatibility.md`.
  Validation: AC-003 (таблица пригодна для выбора без чтения кода).

## Incremental Delivery

### MVP (Первая ценность)

- `docs/compatibility.md` с Go/semver/deprecation правилами и 2 таблицами.
- Ссылка в `README.md`.

### Итеративное расширение

- Добавить “Support window” раздел (например, “поддерживаем последнюю major + N minor”).
- Добавить “Known gaps”/“Limitations” по backend’ам, если появятся.

## Порядок реализации

- Сначала: создать `docs/compatibility.md` (структура → правила → таблицы).
- Затем: добавить ссылку в `README.md`.
- В конце: self-review на двусмысленности и соответствие конституции (Go>=1.23).

## Риски

- Риск: документ устареет и введёт пользователей в заблуждение.
  Mitigation: указать “обновлять при релизах”; держать матрицы компактными.
- Риск: статусы окажутся спорными.
  Mitigation: в MVP ограничиться stable/experimental и опираться на текущий README “production-ready”.

## Rollout и compatibility

- Специальных действий не требуется.

## Проверка

- Manual:
  - AC-001: README ссылка ведёт на `docs/compatibility.md`.
  - AC-002: правила Go/semver/deprecation имеют конкретные значения (N=2, 6 месяцев/2 minor).
  - AC-003: обе таблицы присутствуют и читаемы.

## Соответствие конституции

- Язык docs: русский.
- Go минимум: соответствует конституции (1.23+).
- Публичный API: документ закрепляет semver и дисциплину депрекаций, не меняя код.

