---
report_type: archive_summary
slug: fix-inline
status: completed
reason: баг в SearchBuilder.InlineCite исправлен и верифицирован — маппинг ErrFiltersNotSupported добавлен
docs_language: ru
archived_at: 2026-04-10
---

# Archive Summary: fix-inline

## Status

- status: completed
- reason: баг исправлен — ветка `filter.Fields` в `InlineCite` теперь маппирует `application.ErrFiltersNotSupported` → публичный `ErrFiltersNotSupported` через `errors.Is`, аналогично всем остальным методам `SearchBuilder`

## Snapshot

- path: `.speckeep/archive/fix-inline/2026-04-10/`
- mode: move-based (активные `.speckeep/specs/fix-inline/` и `.speckeep/plans/fix-inline/` удалены после переноса)

## Contents

- specs: `.speckeep/archive/fix-inline/2026-04-10/specs/fix-inline/` (spec + inspect + summary)
- plans: `.speckeep/archive/fix-inline/2026-04-10/plans/fix-inline/` (plan + data-model + tasks)

## Evidence

- tasks: 3/3 выполнено на момент архивации (`verify-task-state.sh`)
- implementation: `pkg/draftrag/search.go:272` — `errors.Is(err, application.ErrFiltersNotSupported)` + ранний возврат
- test: `pkg/draftrag/search_builder_test.go` — `TestSearchBuilder_InlineCite_FilterNotSupported` PASS
- go test: `go test ./pkg/draftrag/...` — ok, без регрессий
