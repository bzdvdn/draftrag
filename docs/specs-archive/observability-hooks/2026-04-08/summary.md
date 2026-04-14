# Сводка архива

## Спецификация

- snapshot: добавлены hooks наблюдаемости для стадий pipeline (chunking/embed/search/generate) с duration и error
- slug: observability-hooks
- archived_at: 2026-04-08
- status: completed

## Причина

Нужна минимальная наблюдаемость для production: измерять latency и ошибки по стадиям RAG без привязки к Prometheus/OTel и без форка библиотеки.

## Результат

- Добавлен domain-интерфейс hooks и типы событий.
- Pipeline инструментирован вокруг embed/search/generate и chunking (если включён).
- Unit-тесты фиксируют порядок вызовов и отсутствие влияния nil hooks.

## Продолжение

- Возможное расширение: более детальные операции, дополнительные стадии (например Upsert), интеграционные адаптеры под конкретные стеки (в отдельных пакетах).

