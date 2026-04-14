# Summary: embedding-cache

## Статус

✅ **Inspected** — готово к планированию

## Scope Snapshot

- **In scope**: Интерфейс-обёртка `EmbedderCache` над `Embedder` с LRU in-memory кэшем и опциональным Redis-бэкендом
- **Out of scope**: Persistent storage, distributed locking, cache warming

## Ключевые требования

- RQ-001..RQ-009: Интерфейс, LRU, thread-safety, Redis, статистика, fallback

## Критерии приемки

- AC-001..AC-007: Базовое кэширование, LRU eviction, thread-safety, Redis fallback/SLC, хэш консистентности, статистика

## Артефакты

- `spec.md` — полная спецификация
- `inspect.md` — отчёт проверки
- `summary.md` — краткое резюме (этот файл)

## Блокеры

Нет

## Следующая команда

```
/speckeep.plan embedding-cache
```
