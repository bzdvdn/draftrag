---
report_type: verify
slug: public-examples
status: pass
docs_language: ru
generated_at: 2026-04-12
---

# Verify Report: public-examples

## Scope

- snapshot: проверил, что `README.md` содержит 2 production-ready end-to-end примера (pgvector + Qdrant) с таймаутами/контекстом, кешом и retry/CB; задачи закрыты
- verification_mode: default
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/public-examples/plan/tasks.md
- inspected_surfaces:
  - README.md (разделы “Production-ready end-to-end (pgvector)” и “Production-ready end-to-end (Qdrant)”)
  - `go test ./...`

## Verdict

- status: pass
- archive_readiness: safe
- summary: все задачи закрыты, примеры присутствуют и используют публичный API; тесты проходят

## Checks

- task_state: completed=4, open=0
- acceptance_evidence:
  - AC-001 -> `README.md`: pgvector пример включает `NewPGVectorStoreWithOptions` + `MigratePGVector` + `NewCachedEmbedder` + `NewRetryEmbedder`/`NewRetryLLMProvider`
  - AC-002 -> `README.md`: Qdrant пример включает `NewQdrantStore` + `CollectionExists`/`CreateCollection` + `NewCachedEmbedder` + `NewRetryEmbedder`/`NewRetryLLMProvider`
  - AC-003 -> `README.md`: оба примера используют `context.WithTimeout` для setup/index/query и `defer cancel()`, таймауты заданы числами
- implementation_alignment:
  - T1.1/T2.1/T2.2 -> `README.md`: добавлен раздел “Production-ready” + 2 end-to-end code-block’а
  - T3.1 -> `go test ./...` завершился успешно

## Errors

- none

## Warnings

- none

## Questions

- none

## Not Verified

- Реальный запуск примеров против живых сервисов (PostgreSQL+pgvector, Qdrant, OpenAI-compatible API) не выполнялся; верификация ограничена наличием примеров и консистентностью публичных символов через `go test`.

## Next Step

- safe to archive: `/speckeep.archive public-examples`

