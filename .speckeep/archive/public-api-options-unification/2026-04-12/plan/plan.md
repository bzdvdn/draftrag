# Единый options pattern в публичном API План

## Phase Contract

Inputs: `.speckeep/specs/public-api-options-unification/spec.md`, `.speckeep/specs/public-api-options-unification/inspect.md` и узкий контекст `pkg/draftrag/*`.
Outputs: `plan.md`, `data-model.md`. Contracts не требуются.
Stop if: выбранный паттерн требует массовых breaking changes без миграционного пути.

## Цель

Зафиксировать один канонический паттерн конфигурации публичных конструкторов `pkg/draftrag` и привести “особые случаи” к этому паттерну, сохранив дефолты и backward compatibility. Добавить guardrail (тест/проверка), чтобы новые конструкторы не ломали консистентность.

## Scope

- Зафиксировать canonical паттерн (DEC-001) и правила исключений.
- Устранить случаи, где публичный конструктор использует больше одного options-объекта (например, `PGVectorRuntimeOptions`) — через новый unified options struct + депрекацию старых API.
- Обновить документацию/примеры и CONTRIBUTING правилом “как писать options в public API”.
- Добавить guardrail unit-test, который валидирует сигнатуры экспортируемых `New*` в `pkg/draftrag`.

## Implementation Surfaces

- `pkg/draftrag/*.go`: публичные конструкторы `New*` и их `...Options` типы.
- `pkg/draftrag/pgvector.go`: унификация `PGVectorOptions` + `PGVectorRuntimeOptions` в один options контейнер.
- `README.md`, `docs/*`, `CONTRIBUTING.md`: правило и примеры единого паттерна.
- `pkg/draftrag/options_pattern_test.go` (новый): guardrail test.

## Влияние на архитектуру

- Это изменение затрагивает только публичный API слой (`pkg/draftrag`): цель — UX/консистентность, а не внутренние архитектурные слои.
- Внутренние functional options (internal) остаются допустимы, но публичный API становится предсказуемым.

## Acceptance Approach

- AC-001: правило canonical паттерна описано в `CONTRIBUTING.md` и/или `README.md` + примеры.
- AC-002: ключевые конструкторы следуют одному виду options; `go test ./...` проходит.
- AC-003: для любых API изменений добавлен миграционный блок (или депрекация старых функций с новым рекомендуемым API).
- AC-004: README/docs/examples обновлены и не содержат “старых” вариантов для затронутых компонентов.
- AC-005: guardrail test падает при нарушении паттерна.

## Данные и контракты

- Data model сведён к определению canonical “options contract” (см. `data-model.md`).
- Внешние API/event contracts отсутствуют.

## Стратегия реализации

- DEC-001 Канонический публичный паттерн: `...Options` struct
  Why: в текущем `pkg/draftrag` большинство конструкторов уже используют `...Options` structs; zero-values хорошо выражают “минимальную конфигурацию” и default’ы.
  Tradeoff: functional options иногда удобнее для расширения без роста struct; но их можно использовать internal или как исключение только при сильной необходимости.
  Affects: все публичные `New*` в `pkg/draftrag`, документация, guardrail.
  Validation: guardrail test подтверждает, что `New*` принимает не более одного `...Options` struct (и/или следует явно описанным исключениям).

- DEC-002 Миграция “двух options” (pgvector runtime) через unified options struct
  Why: два options-аргумента ломают консистентность и заставляют пользователя “знать” исключение.
  Tradeoff: потребуется новый публичный тип (например, `PGVectorStoreOptions`), плюс депрекация старого конструктора.
  Affects: `pkg/draftrag/pgvector.go`, README/docs/examples.
  Validation: новый конструктор и/или новый тип используются в docs; старый остаётся рабочим; `go test ./...` проходит.

- DEC-003 Guardrail как unit-test на AST
  Why: консистентность деградирует без автоматической проверки.
  Tradeoff: AST-парсер в тесте добавляет небольшой maintenance overhead; но он ограничен `pkg/draftrag`.
  Affects: новый тест-файл в `pkg/draftrag`.
  Validation: тест валидирует сигнатуры экспортируемых `New*` и список разрешённых исключений.

## Incremental Delivery

### MVP (Первая ценность)

- Зафиксировать правило в `CONTRIBUTING.md`.
- Добавить guardrail test для `pkg/draftrag`.
- Выровнять самый заметный “особый случай” (pgvector runtime options) через unified options и обновить docs.

Критерий готовности MVP: `go test ./...` проходит, guardrail test активен, README/docs используют единый паттерн.

### Итеративное расширение

- Пройтись по остальным конструкторам и убрать оставшиеся исключения (если появятся) без breaking changes: добавлять новые `New*WithOptions` и депрекейтить старые формы.

## Порядок реализации

- Сначала: определить список публичных `New*` и подтвердить текущее состояние (что уже соответствует паттерну, что нет).
- Затем: внедрить DEC-001/DEC-003 (документация + guardrail test).
- Затем: сделать pgvector unified options и обновить примеры.
- В конце: прогнать `go test ./...` и убедиться, что docs не содержат старых вызовов.

## Риски

- Риск: “ложноположительные” срабатывания guardrail test на нестандартных функциях.
  Mitigation: явно хранить allowlist исключений в тесте с объяснением; держать проверку простой (на уровне “не более одного Options struct”).
- Риск: migration вызывает путаницу.
  Mitigation: депрекация + короткий миграционный блок в README/docs с “старый → новый” примером.

## Rollout и compatibility

- Предпочитать additive изменения: новые типы/конструкторы + депрекация старых.
- Breaking changes избегать; если неизбежны — оформлять отдельно как major с миграционным гайдом (в рамках этой спеки — не планируется).

## Проверка

- `go test ./...`
- Guardrail test в `pkg/draftrag` (AST) валидирует паттерн.
- Пробежка по README/docs/examples на предмет “старых” сигнатур для затронутых конструкторов.

## Соответствие конституции

- Минимальная конфигурация: `Options` structs поддерживают zero-values как defaults.
- Интерфейсная абстракция: не добавляем внешние зависимости; работаем в публичном API слое.
- Тестируемость: guardrail в виде unit-test предотвращает регрессию.

