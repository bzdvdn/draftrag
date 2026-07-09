---
slug: redis-cache-backend
generated_at: 2026-04-11T23:33:54+03:00
---

## Goal
Добавить опциональный Redis (L2) к кэшу эмбеддингов.

## Acceptance Criteria
| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | Redis опция без breaking changes | Тесты проходят без Redis |
| AC-002 | L2 hit не вызывает embedder | Счётчик embedder остаётся 0 |
| AC-003 | L2 hit прогревает L1 | Второй вызов без Redis Get |
| AC-004 | Redis ошибки не ломают Embed | Embed успешен при ошибке Redis |
| AC-005 | Учитываются TTL и prefix ключей | Set с TTL и prefix |
| AC-006 | Битые данные Redis = miss | Embedder вызван, кэш заполнен |

## Out of Scope
- Кэширование других компонентов (LLM/vector store).
- Интеграционные тесты с реальным Redis.
- Singleflight/anti-stampede между инстансами.
- Распределённые блокировки и координация.
