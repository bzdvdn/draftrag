# Сводка архива

## Спецификация

- snapshot: добавлен опциональный MMR rerank/selection для диверсификации retrieval источников в Answer* путях
- slug: retrieval-reranker-mmr
- archived_at: 2026-04-08
- status: completed

## Причина

TopK retrieval часто возвращает почти одинаковые чанки. MMR уменьшает overlap контекста и повышает разнообразие источников без внешних моделей.

## Результат

- Реализован MMR selection (lambda/candidate pool) на embeddings чанков.
- Интегрировано в Answer/AnswerWithCitations/AnswerWithInlineCitations и вариант с ParentIDs.
- Добавлены детерминированные unit-тесты (диверсификация, no-op при выключении, guard на отсутствие embeddings).

## Продолжение

- Возможное расширение: поддержка MMR, когда VectorStore не возвращает embeddings (например через отдельный fetch), и дополнительные метрики для оценки эффекта (через eval harness).

