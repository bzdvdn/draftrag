# Единый options pattern — Data Model

## Canonical публичный контракт

Канонический паттерн для `pkg/draftrag`:

- Конструктор имеет форму `NewX(...required, opts XOptions) ...`
- `XOptions` — struct с zero-values, означающими “использовать дефолты”.
- Если опции не нужны — конструктор не принимает options вовсе (`NewX(...)` без `opts`).

## Правила исключений

Исключения допускаются только при явной необходимости и должны быть:

- описаны в `CONTRIBUTING.md`
- внесены в allowlist guardrail теста
- сопровождены примером в документации

## Unified Options

Если ранее существовало несколько options-структур для одного компонента (пример: base options + runtime options),
то новый canonical вариант должен быть представлен одним “контейнером” options:

- `XOptions` (или `XStoreOptions`) содержит вложенные struct’и для подсекций (например, `Runtime`).
- Старые формы остаются совместимыми (deprecate + migration).

