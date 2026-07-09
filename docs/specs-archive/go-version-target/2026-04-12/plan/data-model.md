# go-version-target — Data Model

## Канонические значения

- `MinGoVersion`: минимально поддерживаемая версия Go для пользователей библиотеки (канонический минимум).
- `RecommendedDevGoVersion`: рекомендуемая версия Go для разработки (может быть выше минимума), но не должна становиться обязательной для downstream-пользователей.

## Источники правды

Минимальная версия Go ДОЛЖНА быть согласована в:

- `go.mod` (`go <MinGoVersion>`)
- `.speckeep/constitution.md` (если конституция содержит версию языка)
- `README.md` и `docs/getting-started.md`
- CI (проверка `go test ./...` на `MinGoVersion`)
