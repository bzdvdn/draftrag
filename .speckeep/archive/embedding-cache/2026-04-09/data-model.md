# Модель данных: Кэширование векторов (Embedding Cache)

## Scope

- **Связанные AC**: AC-001, AC-002, AC-005, AC-007
- **Связанные DEC**: DEC-001, DEC-002, DEC-004
- **Примечание**: Нет persisted state в традиционном смысле; вся data model — runtime in-memory структуры и временное хранилище Redis

## Сущности

### DM-001 LRU Cache Entry (in-memory)

- **Назначение**: Хранение пары ключ-значение в in-memory LRU кэше с поддержкой порядка использования
- **Источник истины**: Runtime-only, создаётся при cache miss, удаляется при eviction или перезапуске
- **Инварианты**:
  - Key — SHA-256 хэш текста, строка фиксированной длины (64 hex chars)
  - Value — `[]float64` вектор, размерность соответствует базовому embedder
  - Каждый entry связан с элементом двусвязного списка для LRU ordering
- **Связанные AC**: AC-001, AC-002, AC-006
- **Связанные DEC**: DEC-001
- **Поля**:
  - `key string` — SHA-256 хэш текста, hex encoding
  - `value []float64` — embedding вектор
  - `listElement *list.Element` — ссылка на элемент в LRU list (для O(1) promotion/eviction)
- **Жизненный цикл**:
  - **Создание**: При cache miss, после вызова базового embedder
  - **Обновление**: При доступе — элемент перемещается в front LRU list (promotion)
  - **Удаление**: При превышении capacity (eviction LRU), при явной очистке, при shutdown
- **Замечания по консистентности**:
  - Нет stale состояний — entry immutable после создания
  - Нет partially-written состояний — запись в cache происходит после получения полного вектора
  - Race condition исключён через `sync.RWMutex`

### DM-002 Cache Stats (runtime metrics)

- **Назначение**: Атомарные счётчики для наблюдаемости эффективности кэширования
- **Источник истины**: Runtime-only, atomic counters
- **Инварианты**:
  - `Hits + Misses >= 0` (монотонно растущие counters)
  - `HitRate = Hits / (Hits + Misses)` (float64, NaN если сумма == 0)
- **Связанные AC**: AC-007
- **Связанные DEC**: —
- **Поля**:
  - `Hits uint64` — atomic, количество cache hits
  - `Misses uint64` — atomic, количество cache misses
  - `Evictions uint64` — atomic, количество LRU evictions (опционально)
- **Жизненный цикл**:
  - **Создание**: При инициализации `EmbedderCache`, zero values
  - **Обновление**: Increment при каждом hit/miss/eviction
  - **Удаление**: N/A — сбрасывается при создании нового экземпляра кэша
- **Замечания по консистентности**:
  - Counters eventual consistent — не критичны для correctness
  - Atomic operations гарантируют отсутствие data races

### DM-003 Redis Cache Entry (second-level)

- **Назначение**: Разделяемое хранилище векторов между инстансами приложения
- **Источник истины**: Redis (external), ключи с TTL
- **Инварианты**:
  - Key — `embed:cache:<sha256_hash>` (prefix + hex hash)
  - Value — msgpack сериализованный `[]float64`
  - TTL — положительное число секунд, после истечения ключ удаляется Redis
- **Связанные AC**: AC-004, AC-005
- **Связанные DEC**: DEC-002, DEC-004
- **Поля** (в Redis):
  - `Key string` — `embed:cache:` + SHA-256 hex hash
  - `Value []byte` — msgpack encoded `[]float64`
  - `TTL time.Duration` — время жизни записи (Redis EXPIRE)
- **Жизненный цикл**:
  - **Создание**: При cache miss in-memory + miss в Redis, после вызова базового embedder
  - **Обновление**: N/A — immutable entry, перезапись только если ключ expired и recreated
  - **Удаление**: Автоматически по TTL или при явной инвалидации (не в MVP)
- **Замечания по консистентности**:
  - Stale reads возможны при конкурентной записи из другого инстанса — acceptable (векторы детерминированы для одного текста)
  - Partial writes исключены — Redis ATOMIC для single-key операций
  - Expired keys возвращают miss, fallback к базовому embedder

## Связи

- `DM-001` (in-memory) ↔ `DM-003` (Redis): Two-level cache hierarchy
  - In-memory L1 — быстрый, local, bounded
  - Redis L2 — медленнее, shared, TTL-based
  - Flow: Check L1 → if miss, check L2 → if miss, call embedder → write to L1 and L2

- `DM-002` (stats) → `DM-001`, `DM-003`: Stats отражают операции над кэшем
  - Hit: найдено в L1 или L2
  - Miss: не найдено ни в L1, ни в L2

## Производные правила

1. **Ключ кэша**: `sha256(text) -> hex string`
   - Используется для DM-001 (in-memory map key) и DM-003 (Redis key suffix)
   - Детерминированный, collision-resistant

2. **LRU Promotion**: При чтении существующего entry в L1 — элемент перемещается в front списка
   - Гарантирует, что при eviction удаляется least recently used

3. **Redis Key Prefix**: Все ключи кэша имеют префикс `embed:cache:`
   - Позволяет избежать коллизий с другими данными в Redis
   - Упрощает инвалидацию по паттерну (если понадобится в будущем)

4. **TTL Policy**: Redis entries имеют фиксированный TTL (например, 24 часа)
   - Предотвращает бесконечный рост Redis
   - Пользователь может настроить через `WithRedis(..., ttl)`

## Переходы состояний

### Cache Entry Lifecycle (L1 in-memory)

```
[Cache Miss]
    ↓
[Call Base Embedder] ──→ [Error] ──→ [Return Error]
    ↓
[Create Entry]
    ↓
[Store in L1] ──→ [If L1 full] ──→ [Evict LRU entry]
    ↓
[Return Vector]
```

### Two-Level Lookup

```
[Embed Call]
    ↓
[Compute Hash]
    ↓
[Check L1] ──→ [Hit] ──→ [Increment Hits] ──→ [Promote in LRU] ──→ [Return Vector]
    ↓ (Miss)
[Check L2 (Redis)] ──→ [Hit] ──→ [Increment Hits] ──→ [Store in L1] ──→ [Return Vector]
    ↓ (Miss)
[Call Base Embedder] ──→ [Increment Misses]
    ↓
[Store in L1 and L2] ──→ [Return Vector]
```

## Вне scope

- Persistent storage для L1 (survive restart) — явно вне scope спецификации
- Distributed locking для L2 (concurrent writes) — вне scope, acceptable race
- Cache warming / prefetch — вне scope
- Invalidation by pattern/tag — вне scope
- Compression for L1 — вне scope (tradeoff memory vs CPU)
