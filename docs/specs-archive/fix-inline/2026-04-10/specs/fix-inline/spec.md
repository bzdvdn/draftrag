# SearchBuilder.InlineCite: фикс пропущенного маппинга ErrFiltersNotSupported

## Scope Snapshot

- In scope: исправить ветку `filter.Fields` в `SearchBuilder.InlineCite` — добавить маппинг `application.ErrFiltersNotSupported` → `ErrFiltersNotSupported`.
- Out of scope: другие методы SearchBuilder, другие типы ошибок, рефакторинг routing-логики.

## Цель

Пользователь библиотеки, вызывающий `pipeline.Search(q).Filter(f).InlineCite(ctx)` на хранилище без поддержки фильтров, должен получить `draftrag.ErrFiltersNotSupported` — то же поведение, что уже есть в `Retrieve`, `Answer`, `Cite`, `Stream` и `StreamCite`. Сейчас `InlineCite` пробрасывает внутреннюю `application.ErrFiltersNotSupported` напрямую, нарушая изоляцию публичного API.

## Основной сценарий

1. Пользователь создаёт `Pipeline` с хранилищем, не реализующим `VectorStoreWithFilters`.
2. Вызывает `pipeline.Search(question).Filter(MetadataFilter{...}).InlineCite(ctx)`.
3. Хранилище не поддерживает фильтры → application возвращает `application.ErrFiltersNotSupported`.
4. **До фикса**: `InlineCite` пробрасывает ошибку без маппинга → `errors.Is(err, draftrag.ErrFiltersNotSupported)` возвращает `false`.
5. **После фикса**: `InlineCite` маппирует ошибку → `errors.Is(err, draftrag.ErrFiltersNotSupported)` возвращает `true`, поведение идентично остальным методам SearchBuilder.

## Scope

- `pkg/draftrag/search.go` — ветка `filter.Fields` в методе `InlineCite`
- Добавление unit-теста, воспроизводящего баг и покрывающего фикс

## Контекст

- Все другие методы `SearchBuilder` (`Retrieve`, `Answer`, `Cite`, `Stream`, `StreamCite`) уже содержат `errors.Is(err, application.ErrFiltersNotSupported)` → `ErrFiltersNotSupported` в ветке `filter.Fields`.
- `InlineCite` — единственный метод, где эта проверка отсутствует (строки 269-271 `search.go`).
- `ErrFiltersNotSupported` в `pkg/draftrag/errors.go` — публичная переменная; `application.ErrFiltersNotSupported` — внутренняя. Пользователь не должен зависеть от internal-пакета.
- Публичный API стабилен: изменения только в одном внутреннем `if`-блоке, сигнатура метода не меняется.

## Требования

- RQ-001 `SearchBuilder.InlineCite` при ветке `filter.Fields > 0` ДОЛЖЕН маппировать `application.ErrFiltersNotSupported` в `draftrag.ErrFiltersNotSupported` перед возвратом.
- RQ-002 Маппинг ДОЛЖЕН использовать `errors.Is`, чтобы корректно обрабатывать обёрнутые ошибки.
- RQ-003 При успешном поиске (фильтры поддерживаются) поведение `InlineCite` ДОЛЖНО оставаться идентичным текущему.

## Вне scope

- Другие методы `SearchBuilder` — они уже корректны.
- Изменение сигнатуры `InlineCite` или добавление новых параметров.
- Рефакторинг или унификация routing-кода в `SearchBuilder` (отдельная задача).
- Другие недостающие маппинги ошибок (circuit breaker, streaming) — за пределами этой фичи.

## Критерии приемки

### AC-001 InlineCite возвращает публичный ErrFiltersNotSupported

- Почему это важно: пользователи не должны зависеть от internal-пакета при обработке ошибок.
- **Given** Pipeline создан с хранилищем, не реализующим `VectorStoreWithFilters`, и задан ненулевой `MetadataFilter`
- **When** вызывается `SearchBuilder.InlineCite(ctx)`
- **Then** возвращённая ошибка удовлетворяет `errors.Is(err, draftrag.ErrFiltersNotSupported) == true`
- Evidence: unit-тест проходит `require.ErrorIs(t, err, draftrag.ErrFiltersNotSupported)`.

### AC-002 При успехе InlineCite работает без изменений

- Почему это важно: фикс не должен ломать happy path.
- **Given** Pipeline создан с хранилищем, реализующим `VectorStoreWithFilters`, задан `MetadataFilter`
- **When** вызывается `SearchBuilder.InlineCite(ctx)` и поиск возвращает результаты
- **Then** метод возвращает ответ, источники и цитаты без ошибки
- Evidence: существующие тесты `InlineCite` на поддерживающем хранилище продолжают проходить.

### AC-003 Маппинг использует errors.Is (обёрнутые ошибки)

- Почему это важно: application может обернуть ошибку в `fmt.Errorf("...: %w", application.ErrFiltersNotSupported)`.
- **Given** application возвращает ошибку, обёртывающую `application.ErrFiltersNotSupported` через `%w`
- **When** `InlineCite` проверяет ошибку
- **Then** маппинг срабатывает корректно; публичный `ErrFiltersNotSupported` возвращается пользователю
- Evidence: тест передаёт `fmt.Errorf("wrap: %w", application.ErrFiltersNotSupported)` и проверяет маппинг.

## Допущения

- Исправление состоит ровно из одного добавленного `if errors.Is(...)` блока по аналогии с `Cite` (строки 219-221) и `StreamCite` (строки 391-394).
- Тест размещается в `pkg/draftrag/search_builder_test.go` или новом файле в том же пакете.
- Нет необходимости менять internal/application — баг исключительно в слое публичного API.

## Открытые вопросы

- none
