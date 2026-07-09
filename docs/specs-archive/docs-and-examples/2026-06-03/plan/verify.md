---
report_type: verify
slug: docs-and-examples
status: pass
docs_language: ru
generated_at: 2026-06-03
---

# Verify Report: docs-and-examples

## Scope

- snapshot: проверка 20 задач (T1.1–T4.4) для 6 per-backend examples, 10 tutorials, CI smoke matrix, обновлений README/docs/ROADMAP
- verification_mode: default
- artifacts:
  - docs/specs/docs-and-examples/spec.md
  - docs/specs/docs-and-examples/plan.md
  - docs/specs/docs-and-examples/tasks.md
  - .speckeep/constitution.summary.md
- inspected_surfaces:
  - examples/shared/{mock,print}.go + _test.go
  - examples/{memory,pgvector,qdrant,chromadb,weaviate,milvus}/main.go + docker-compose.yml + .env.example + README.md
  - .github/workflows/examples-smoke.yml
  - README.md, docs/vector-stores.md, ROADMAP.md
  - docs/tutorials/ru/01-quickstart.md .. 10-production-hardening.md
  - go build ./examples/... + go test ./... + docker-compose config (5 backends)

## Verdict

- status: pass
- archive_readiness: safe
- summary: все 20 задач закрыты observable proof; 17 AC покрыты; zero diff в pkg/ и internal/; тесты и сборка проходят

## Checks

- task_state: completed=20, open=0
- acceptance_evidence:
  - AC-001 -> T2.2: examples/pgvector/main.go — MigratePGVector + NewPGVectorStore, docker-compose.yml
  - AC-002 -> T2.3: examples/qdrant/main.go — CollectionExists/CreateCollection + NewQdrantStore, docker-compose.yml
  - AC-003 -> T3.1: examples/chromadb/main.go — CreateChromaCollection + NewChromaDBStore, docker-compose.yml
  - AC-004 -> T3.2: examples/weaviate/main.go — CreateWeaviateCollection + NewWeaviateStore, docker-compose.yml
  - AC-005 -> T3.3: examples/milvus/main.go — NewMilvusStore (internal API), multi-service docker-compose.yml
  - AC-006 -> T2.1: examples/memory/main.go — NewInMemoryStore, без Docker, 10 demo-документов
  - AC-007 -> T2.2/T2.3/T3.1-T3.3: каждая main.go имеет buildComponents switch по LLM_PROVIDER (mock|ollama|openai|anthropic), env-driven без правок Go-кода
  - AC-008 -> T1.2: examples/shared/mock.go — mockLLM + mockEmbedder, префикс "[mock] ", детерминированный хэш
  - AC-009 -> T2.4/T3.4: .github/workflows/examples-smoke.yml — compose-validate + examples-build + examples-smoke (6×matrix)
  - AC-010 -> T4.1/T4.2/T4.3: 10 tutorials созданы в docs/tutorials/ru/
  - AC-011 -> T4.1: каждая main.go содержит "set LLM_PROVIDER=mock to run without API key" для отсутствующих ключей
  - AC-012 -> T4.1/T4.2/T4.3: все 10 tutorials содержат ссылки examples/ и ссылки на следующий tutorial
  - AC-013 -> T3.5: README.md обновлён (Быстрый старт, таблица примеров, Tutorials, Провайдеры LLM)
  - AC-014 -> T3.4: examples-smoke CI job с 6 backend'ами в matrix, LLM_PROVIDER=mock
  - AC-015 -> T3.6: docs/vector-stores.md — все 6 бэкендов имеют ссылки examples/<backend>/ в capability-таблице
  - AC-016 -> T4.4: git diff --stat master -- pkg/ internal/ — пусто
  - AC-017 -> T1.5/T4.4: go test ./... exit 0; coverage: domain 100%, application 83.3%, vectorstore 60.7%
- implementation_alignment:
  - Surface Map в tasks.md обновлён: ссылки на config.go/build.go удалены, добавлена запись `examples/*/main.go (buildComponents inline switch)`

## Errors

- none

## Questions

- none

## Not Verified

- CI runtime не проверялся: workflow запускается только на push/PR в master
- docker-compose up -d не выполнялся (только `config` валидация)
- go run ./examples/<backend>/ не выполнялся (требует Docker + внешние сервисы)

## Next Step

- safe to archive
