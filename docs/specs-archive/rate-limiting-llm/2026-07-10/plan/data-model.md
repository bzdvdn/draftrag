# Rate Limiting для LLM API — Data Model

## Status

no-change

## Причина

Ни одна существующая модель данных не меняется. Единственное изменение — новое значение константы `HookStageRateLimit` в `domain/hooks.go`, что является добавлением const, а не изменением data model. Новые типы (`tokenBucket`, `TokenBucketLLMProvider`, `TokenBucketOptions`) не влияют на существующие модели и контракты.
