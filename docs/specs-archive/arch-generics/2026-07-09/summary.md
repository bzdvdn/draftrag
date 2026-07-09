# arch-generics

**Status**: completed
**Archived**: 2026-07-09

## Summary

Generic handler factory for SearchBuilder + nil context guard + trace markers.

- Generic `router[T]` with 7 `mk*` and 7 `wrap*` handler factory helpers
- All `panic("nil context")` eliminated across the codebase (22+ spots)
- `panic("nil db")` replaced with error returns in pgvector constructors (breaking change: `NewPGVectorStore*` returns `(VectorStore, error)`)
- 7 handler maps rewritten through generic factory
- All trace markers renamed from `searchbuilder-generics#*` to `arch-generics#*`

## Artifacts

- `plan/plan.md` — implementation plan
- `plan/tasks.md` — task breakdown
- `plan/data-model.md` — data model
- `plan/verify.md` — verification report (status: pass)
- `specs/spec.md` — specification
