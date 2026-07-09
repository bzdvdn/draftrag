# Eval harness: базовая оценка качества RAG (v1) — Модель данных

## Scope

- Связанные `AC-*`: `AC-001`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`

## Сущности

### DM-001 EvalCase

- Назначение: один тестовый кейс для оценки retrieval.
- Инварианты:
  - `Question` не пустой.
  - `ExpectedParentIDs` — множество (может содержать 1..N элементов).
- Поля:
  - `ID` - `string`, optional, стабильный идентификатор кейса.
  - `Question` - `string`, required.
  - `ExpectedParentIDs` - `[]string`, required, множество “правильных” parent IDs.
  - `TopK` - `int`, optional (0 => дефолт harness).

### DM-002 EvalCaseResult

- Назначение: результат прогона одного кейса.
- Поля:
  - `CaseID` - `string`
  - `Found` - `bool` (hit@k)
  - `Rank` - `int` (1..K) или 0 если не найдено
  - `RetrievedParentIDs` - `[]string` (для дебага)

### DM-003 EvalReport

- Назначение: агрегированный отчёт.
- Поля:
  - `Metrics` - агрегаты (HitAtK, MRR, TotalCases)
  - `Cases` - `[]EvalCaseResult`

