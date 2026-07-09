---
slug: eval-harness-retrieval-only
generated_at: 2026-04-14
---

## Goal

Разработчики получают инструмент для оценки retrieval-компонента по классическим IR-метрикам без LLM-генерации.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Поддержка NDCG@K для ранжирования | Report.Metrics.NDCG содержит значение от 0 до 1 |
| AC-002 | Поддержка Precision@K и Recall@K | Report.Metrics.Precision и Recall содержат значения от 0 до 1 |
| AC-003 | Конфигурируемый набор метрик | Отключённая метрика имеет значение 0 или не вычисляется |
| AC-004 | Детализация результатов по кейсам | Report.Cases[i] содержит поля результатов для i-го кейса |
| AC-005 | Валидация входных данных | Run возвращает error с текстом о некорректном кейсе |
| AC-006 | Сериализация Report в JSON | Report реализует MarshalJSON, результат валиден по json.Unmarshal |

## Out of Scope

- Метрики качества генерации (faithfulness, answer relevance, factual correctness)
- LLM-based evaluation (асессор через LLM)
- End-to-end evaluation retrieval + generation
- Метрики на основе эмбеддингов (embedding similarity)
- Визуализация результатов (dashboards, графики)
