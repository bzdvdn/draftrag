---
status: extended
---

# Data Model: eval-ragas-metrics

## Изменения

### `pkg/draftrag/eval.Case`

| Поле | Тип | Изменение | Назначение |
|------|-----|-----------|------------|
| `ExpectedAnswer` | `string` | добавлено | Опциональный эталонный ответ для RAGAS-метрик. Если пустая строка — метрики, требующие ответа, дают 0. |

### `pkg/draftrag/eval.Metrics`

| Поле | Тип | Изменение | Назначение |
|------|-----|-----------|------------|
| `Faithfulness` | `float64` | добавлено | Средний Faithfulness score по всем кейсам [0,1] |
| `AnswerRelevance` | `float64` | добавлено | Средний Answer Relevance score по всем кейсам [0,1] |
| `ContextRelevance` | `float64` | добавлено | Средний Context Relevance score по всем кейсам [0,1] |

### `pkg/draftrag/eval.CaseResult`

Без изменений.

### `pkg/draftrag/eval.Options`

| Поле | Тип | Изменение | Назначение |
|------|-----|-----------|------------|
| `EnableFaithfulness` | `bool` | добавлено | Включить вычисление Faithfulness |
| `EnableAnswerRelevance` | `bool` | добавлено | Включить вычисление Answer Relevance |
| `EnableContextRelevance` | `bool` | добавлено | Включить вычисление Context Relevance |

### `internal/domain`

Без изменений. Используются существующие интерфейсы `LLMProvider` и `Embedder`.

## Инварианты

- Zero value для всех новых bool-полей = false (метрики отключены).
- Zero value для `Case.ExpectedAnswer` = "" (метрики, требующие ответа, не вычисляются).
- Zero value для новых float64-полей в `Metrics` = 0.0 (отсутствие метрики).
- Размерность эмбеддингов должна совпадать внутри одного вызова `ComputeContextRelevance` / `ComputeAnswerRelevance`.
