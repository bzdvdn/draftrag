# План: Кэширование векторов (Embedding Cache)

## Phase Contract

- **Inputs**: spec `embedding-cache`, inspect report (PASS)
- **Outputs**: plan.md, data-model.md
- **Contracts/API**: не требуются (внутренняя инфраструктура)

## Цель

Реализовать обёртку `EmbedderCache` в `internal/infrastructure/embedder/cache/`, которая прозрачно кэширует векторы по хэшу текста. Обёртка реализует `domain.Embedder`, делегируя вызовы к базовому embedder при miss. In-memory LRU — основной уровень, Redis — опциональный second-level.

## Scope

- Новый подпакет `internal/infrastructure/embedder/cache/`
- Структура `EmbedderCache` с конфигурацией размера и опциональным Redis
- LRU in-memory реализация через `container/list` (стандартная библиотека)
- Thread-safe операции через `sync.RWMutex`
- Redis клиент с сериализацией msgpack и TTL
- Статистика hits/misses через atomic counters
- Graceful fallback при ошибках Redis

## Implementation Surfaces

| Surface | Тип | Описание |
|---------|-----|----------|
| `internal/infrastructure/embedder/cache/cache.go` | Новый | Основная структура `EmbedderCache`, реализует `domain.Embedder` |
| `internal/infrastructure/embedder/cache/lru.go` | Новый | In-memory LRU реализация с `sync.RWMutex` |
| `internal/infrastructure/embedder/cache/redis.go` | Новый | Redis second-level cache с msgpack сериализацией |
| `internal/infrastructure/embedder/cache/stats.go` | Новый | Структура `CacheStats` с atomic counters |
| `internal/infrastructure/embedder/cache/options.go` | Новый | Functional options для конфигурации |
| `go.mod` | Модификация | Добавить `github.com/vmihailenco/msgpack/v5` для сериализации векторов в Redis |

## Влияние на архитектуру

- **Локальное**: Новый подпакет в infrastructure слое, не затрагивает domain или application
- **Интеграция**: `EmbedderCache` реализует `domain.Embedder`, полностью совместим с существующими use-cases
- **Compatibility**: Не ломает существующий API; обёртка используется явно через конструктор
- **Zero-downtime**: Не требует миграций или rollout-флагов

## Acceptance Approach

| AC | Реализация | Surfaces | Observable Proof |
|----|------------|----------|------------------|
| **AC-001** | Базовое кэширование | `cache.go`, `lru.go` | Счётчик вызовов мок-embedder == 1 после двух вызовов с одним текстом |
| **AC-002** | LRU eviction | `lru.go` | Размер кэша == capacity при добавлении capacity+1 элементов; первый элемент evict'ится |
| **AC-003** | Thread-safety | `cache.go`, `lru.go` | Тест с `go test -race` проходит без race conditions |
| **AC-004** | Redis fallback | `redis.go`, `cache.go` | При недоступном Redis лог содержит warning, вызов делегируется к базовому embedder |
| **AC-005** | Redis second-level | `redis.go` | Мок Redis с предустановленным ключом возвращает вектор без вызова базового embedder |
| **AC-006** | Хэш консистентности | `cache.go` | `sha256.Sum256([]byte(text))` даёт одинаковый результат для идентичных строк |
| **AC-007** | Статистика | `stats.go` | `Stats()` возвращает корректные значения Hits/Misses/HitRate после серии вызовов |

## Данные и контракты

См. `data-model.md`. Изменения:
- Нет новых persisted сущностей (in-memory кэш)
- Redis ключи: `embed:cache:<sha256_hash>` с msgpack payload
- TTL в Redis управляется через `EXPIRE`

## Стратегия реализации

### DEC-001 LRU через container/list + sync.RWMutex

**Why**: Стандартная библиотека Go предоставляет `container/list` для двусвязного списка — достаточно для LRU без внешних зависимостей. `sync.RWMutex` обеспечивает thread-safety с минимальным contention для чтений.

**Tradeoff**: Не самая производительная реализация (vs segmented LRU), но достаточна для типичных нагрузок. Если потребуется extreme performance — можно заменить на `github.com/hashicorp/golang-lru` без изменения публичного API.

**Affects**: `lru.go`

**Validation**: AC-002, AC-003

### DEC-002 Redis second-level как optional wrapper

**Why**: Redis не должен быть обязательным для работы кэша. Двухуровневая архитектура (in-memory L1, Redis L2) позволяет разделять кэш между инстансами без жёсткой зависимости от Redis availability.

**Tradeoff**: Дополнительная сложность (fallback логика), но повышенная resilience. Проверка Redis происходит после in-memory miss.

**Affects**: `redis.go`, `cache.go`

**Validation**: AC-004, AC-005

### DEC-003 SHA-256 для хэширования ключей

**Why**: SHA-256 обеспечивает равномерное распределение, стабильность между запусками, отсутствие коллизий на практике. Ключ кэша — строка хэша, не требует дополнительного encoding.

**Tradeoff**: 64 символа на ключ в Redis (vs 32 для MD5), но MD5 считается устаревшим.

**Affects**: `cache.go`

**Validation**: AC-006

### DEC-004 Msgpack для сериализации векторов

**Why**: Msgpack компактнее JSON (~50% для float64 массивов), быстрее парсится, поддерживает binary float64 без потери точности. `github.com/vmihailenco/msgpack/v5` — стабильная и популярная библиотека.

**Tradeoff**: Дополнительная зависимость (не стандартная библиотека).

**Affects**: `redis.go`, `go.mod`

**Validation**: AC-005

## Incremental Delivery

### MVP (Первая ценность)

**Scope**: In-memory LRU кэш без Redis

**Реализация**:
- `cache.go` — структура `EmbedderCache` с `Embed()` методом
- `lru.go` — LRU с `sync.RWMutex`
- `stats.go` — atomic counters
- `options.go` — `WithCacheSize(size int)`
- Тесты для AC-001, AC-002, AC-003, AC-006, AC-007

**Готовность MVP**: `go test -race ./...` проходит; мок-embedder подтверждает кэширование.

### Итерация 2: Redis second-level

**Scope**: Redis бэкенд, fallback, TTL

**Реализация**:
- `redis.go` — Redis client wrapper
- `options.go` — `WithRedis(client *redis.Client, ttl time.Duration)`
- Fallback логика в `cache.go`
- Тесты для AC-004, AC-005

**Готовность**: Интеграционный тест с testcontainers или мок Redis.

## Порядок реализации

1. **Базовые структуры** (`cache.go`, `stats.go`) — интерфейс и конфигурация
2. **LRU реализация** (`lru.go`) — можно параллельно с тестами
3. **In-memory тесты** — AC-001, AC-002, AC-006, AC-007
4. **Race detector** — AC-003 (`go test -race`)
5. **Redis реализация** (`redis.go`) — после стабилизации in-memory
6. **Redis тесты** — AC-004, AC-005
7. **Интеграционные тесты** — полный цикл с моками

**Параллельно**: Написание godoc, примеры использования в `pkg/draftrag/`.

## Риски

| Риск | Mitigation |
|------|------------|
| Рост памяти при большом размере кэша | LRU eviction, настраиваемый размер; пользователь контролирует capacity |
| Необнаруженные race conditions | Обязательный `go test -race` в CI; RWMutex на всех операциях |
| Redis downtime ломает кэш | Graceful fallback к in-memory + логирование (AC-004) |
| Утечка горутин при context cancellation | Проверка `ctx.Done()` в `Embed()`, корректное закрытие ресурсов |
| Несовместимость msgpack версий | Pin версии `github.com/vmihailenco/msgpack/v5` в go.mod |

## Rollout и compatibility

- **No migration required**: Новая функциональность, не ломает существующий код
- **Opt-in usage**: Пользователь явно создаёт `EmbedderCache` через конструктор
- **No feature flags**: Не требуется, т.к. это additive change
- **Operational**: Мониторинг через `Stats()` — hit rate, размер кэша

## Проверка

| Что проверить | Метод | Подтверждает |
|---------------|-------|--------------|
| Базовое кэширование | Unit test с мок-embedder | AC-001 |
| LRU eviction | Unit test с capacity=2, 3 элементами | AC-002 |
| Thread-safety | `go test -race -count=100` | AC-003 |
| Redis fallback | Интеграционный тест с остановленным Redis | AC-004 |
| Redis SLC | Интеграционный тест с предустановленным ключом | AC-005 |
| Хэш консистентности | Unit test с одинаковыми строками | AC-006 |
| Статистика | Unit test с проверкой counters | AC-007 |
| Производительность | Benchmark `BenchmarkCacheHit` | SC-002 |

## Соответствие конституции

| Принцип | Статус | Примечание |
|---------|--------|------------|
| Интерфейсная абстракция | ✅ | `EmbedderCache` реализует `domain.Embedder` |
| Чистая архитектура | ✅ | Infrastructure слой, зависит от domain, не наоборот |
| Минимальная конфигурация | ✅ | Работает из коробки с разумным default размером |
| Контекстная безопасность | ✅ | Все методы принимают `context.Context` |
| Тестируемость | ✅ | Зависимость от `Embedder` интерфейса позволяет моки |

**Конфликтов нет.**
