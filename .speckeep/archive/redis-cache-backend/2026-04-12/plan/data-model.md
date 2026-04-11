# Redis backend для кэша эмбеддингов — Data Model

## Redis Key

- Формат: `<prefix><sha256_hex(text)>`
- `sha256_hex(text)`: текущий алгоритм ключа в L1 (SHA-256 от исходного текста, hex-строка).
- `prefix`: строковый префикс keyspace для избежания коллизий в shared Redis.
  - Дефолт: `draftrag:embedder:`
  - Если пользователь задаёт prefix — используется он.

## Redis Value

- Тип: `[]byte`
- Содержимое: msgpack-encoded `[]float64`
- Требование совместимости: формат стабилен в рамках версии библиотеки; ошибка decode трактуется как cache miss (treat-as-miss).

## TTL

- TTL задаётся как `time.Duration`.
- Семантика:
  - `ttl == 0`: запись без TTL (персистентно до eviction/удаления в Redis).
  - `ttl > 0`: запись с TTL.

## Invariants

- Redis не является источником истины: любые ошибки Redis (Get/Set) и ошибки decode не должны ломать основной путь `Embed`.
- При L2 hit значение прогревает L1 (warming), чтобы повторный запрос не делал `Redis GET`.

