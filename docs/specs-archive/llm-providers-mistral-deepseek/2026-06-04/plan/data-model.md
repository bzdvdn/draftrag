# LLM-провайдеры Mistral и DeepSeek — Модель данных

## Scope

- Связанные `AC-*`: все — ни один не требует изменения data model.
- Связанные `DEC-*`: DEC-001, DEC-002.
- Статус: `no-change`
- Причина: фича добавляет configuration objects (`MistralLLMOptions`, `DeepSeekLLMOptions`, `MistralEmbedderOptions`) и реализацию существующих интерфейсов (`LLMProvider`/`StreamingLLMProvider`/`Embedder`). Ни одна domain-сущность, state transition или API/event contract не создаётся и не модифицируется.

## No-Change Stub

- Статус: `no-change`
- Причина: фича не добавляет и не меняет persisted entities, value objects, state transitions или contract-relevant payload shapes. Все изменения локальны для infrastructure + public API слоёв.
- Revisit triggers:
  - появляется новое сохраняемое состояние (напр., persisted credentials)
  - появляются новые инварианты или lifecycle states в `domain`
  - API/event payload shape нужно отслеживать
