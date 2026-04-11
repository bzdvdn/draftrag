# fix-inline: задачи

## Phase Contract

Inputs: plan.md, summary.md.
Outputs: упорядоченные исполнимые задачи с покрытием всех AC.
Stop if: задачи расплывчаты или coverage нельзя сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/search.go | T1.1 |
| pkg/draftrag/search_builder_test.go | T2.1 |

## Фаза 1: Основная реализация

Цель: исправить пропущенный маппинг ошибки в InlineCite — одна ветка, два изменения строк.

- [x] T1.1 Исправить ветку filter.Fields в InlineCite — errors.Is(err, application.ErrFiltersNotSupported) маппится в ErrFiltersNotSupported перед возвратом (AC-001, AC-003, DEC-001). Touches: pkg/draftrag/search.go

## Фаза 2: Проверка

Цель: добавить тест, воспроизводящий баг, и подтвердить что регрессий нет.

- [x] T2.1 Добавить TestSearchBuilder_InlineCite_FilterNotSupported — тест проверяет AC-001 и AC-003 через errors.Is. Touches: pkg/draftrag/search_builder_test.go
- [x] T2.2 Прогнать go test ./pkg/draftrag/... и go vet ./pkg/draftrag/... — все тесты зелёные, vet без ошибок (AC-002). Touches: pkg/draftrag/search_builder_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1
- AC-002 -> T2.2
- AC-003 -> T1.1, T2.1

## Заметки

- Фаза "Основа" осознанно пропущена: фича не вводит новых моделей, migrations или flag-изменений.
- Mock для T2.1: использовать inline-тип без метода SearchWithMetadataFilter, чтобы не реализовывать VectorStoreWithFilters случайно. Для AC-003 mock возвращает fmt.Errorf("wrap: %w", application.ErrFiltersNotSupported) напрямую.
