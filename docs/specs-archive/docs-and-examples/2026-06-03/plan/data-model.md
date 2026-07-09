---
report_type: data_model
slug: docs-and-examples
status: no-change
generated_at: 2026-06-03
---

# docs-and-examples Модель данных

## Scope

- Связанные `AC-*`: AC-001..AC-017 (все)
- Связанные `DEC-*`: DEC-001..DEC-008 (все)
- Статус: `no-change`
- Причина: фича не добавляет и не меняет persisted entities, value objects, state transitions или contract-relevant payload shapes. `pkg/draftrag/` (публичный API) и `internal/` (домен) остаются нетронутыми. Все артефакты фичи — examples (Go `main.go` пакеты, не часть публичного API), `docs/tutorials/ru/*.md` (документация), `.github/workflows/examples-smoke.yml` (CI), и точечные обновления README / docs/vector-stores.md / ROADMAP.md (markdown-линки, не schema).

## No-Change Stub

- Статус: `no-change`
- Причина: фича = документация + examples + CI; ноль изменений в `pkg/draftrag/*.go` (verified by AC-016: `git diff --stat main -- pkg/ internal/` пустой) и ноль изменений в `internal/**/*.go` (тот же gate). Никакие публичные типы, сигнатуры, sentinel'ы, payload shapes, состояния или переходы не вводятся и не модифицируются.
- Сущности draftRAG, которые остаются стабильными (для контекста, не для моделирования):
  - `domain.Document`, `domain.Chunk`, `domain.RetrievalResult`, `domain.HybridConfig` — без изменений.
  - `domain.VectorStore`, `domain.DocumentStore`, `domain.TransactionalDocumentStore`, `domain.Embedder`, `domain.LLMProvider`, `domain.Chunker`, `domain.Hooks`, `domain.Redactor` — без изменений.
  - `application.Pipeline` + 7 публичных методов SearchBuilder (Retrieve / Answer / Cite / InlineCite / Stream / StreamSources / StreamCite) — без изменений.
- Revisit triggers:
  - Появляется новая persisted entity в `internal/infrastructure/vectorstore/<backend>.go` (например, новый VectorStore) → нужен DM-001.
  - Меняется сигнатура `domain.VectorStore.Search` или `domain.LLMProvider.Generate` → нужен пересмотр.
  - Добавляется новое поле в `draftrag.PipelineOptions` → нужен DM-002.
  - Появляются новые tutorial-управляемые поля (например, `LLM_PROVIDER=mock` как опция) — НЕ требует data-model.md, т.к. это env-конфиг, не persisted state.
- Вне scope: описание `examples/shared/mock.go` mock-структур (mockLLM, mockEmbedder) как data-entities — это in-memory test doubles, не persisted state, не документируется в data-model.md.

## Entities (no-change confirmation)

Не вводятся новые DM-001+ entities. Существующие сущности draftRAG см. в `internal/domain/models.go` и `internal/domain/interfaces.go` (стабильны по AC-016 и AC-017).
