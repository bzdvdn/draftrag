# Задачи: Кэширование векторов (Embedding Cache)

## Phase Contract

- **Inputs**: plan.md, data-model.md
- **Outputs**: исполнимые задачи с покрытием AC
- **Stop if**: задачи расплывчаты или AC не покрыты

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/embedder/cache/cache.go` | T1.1, T2.1, T2.4, T3.2 |
| `internal/infrastructure/embedder/cache/lru.go` | T1.2, T2.2, T2.3 |
| `internal/infrastructure/embedder/cache/stats.go` | T1.3, T2.5 |
| `internal/infrastructure/embedder/cache/options.go` | T1.4, T2.4, T3.2 |
| `internal/infrastructure/embedder/cache/redis.go` | T3.1, T3.2 |
| `internal/infrastructure/embedder/cache/*_test.go` | T2.6, T3.3, T3.4 |
| `go.mod` | T1.5 |

## Фаза 1: Основа

Цель: создать каркас пакета, базовые структуры и конфигурацию для последующей реализации LRU и Redis.

- [x] **T1.1** Создать пакет и основную структуру `EmbedderCache` — `internal/infrastructure/embedder/cache/cache.go` существует, содержит структуру с полями: базовый `Embedder`, LRU кэш, опциональный Redis, stats; метод `Embed(ctx, text)` вычисляет SHA-256 хэш и делегирует к LRU — AC-001, DEC-003. Touches: `internal/infrastructure/embedder/cache/cache.go`

- [x] **T1.2** Создать структуру LRU кэша — `internal/infrastructure/embedder/cache/lru.go` существует, содержит: map[string]*list.Element для O(1) lookup, `container/list.List` для LRU ordering, `sync.RWMutex` для thread-safety, методы `Get(key)`, `Set(key, value)`, `Len()` — DEC-001. Touches: `internal/infrastructure/embedder/cache/lru.go`

- [x] **T1.3** Создать структуру статистики — `internal/infrastructure/embedder/cache/stats.go` существует, содержит: `CacheStats` с `Hits`, `Misses`, `Evictions` (uint64 atomic), методы `RecordHit()`, `RecordMiss()`, `RecordEviction()`, `Stats() CacheStats` — AC-007. Touches: `internal/infrastructure/embedder/cache/stats.go`

- [x] **T1.4** Создать functional options — `internal/infrastructure/embedder/cache/options.go` существует, содержит: `WithCacheSize(size int)`, конструктор `NewEmbedderCache(embedder Embedder, opts ...Option)` — AC-001. Touches: `internal/infrastructure/embedder/cache/options.go`

- [x] **T1.5** Добавить зависимость msgpack — `go.mod` содержит `github.com/vmihailenco/msgpack/v5`, `go.sum` обновлён — DEC-004. Touches: `go.mod`, `go.sum`

## Фаза 2: In-memory LRU реализация и тесты

Цель: реализовать полноценный in-memory LRU кэш с тестами, покрывающими основные AC.

- [x] **T2.1** Реализовать двухуровневый lookup в `EmbedderCache.Embed` — метод сначала проверяет LRU (L1), при hit — возвращает вектор и инкрементит hits; при miss — вызывает базовый embedder, сохраняет в LRU, инкрементит misses, возвращает вектор — AC-001, RQ-001, RQ-002, RQ-003, RQ-005. Touches: `internal/infrastructure/embedder/cache/cache.go`

- [x] **T2.2** Реализовать LRU eviction логику — метод `Set(key, value)` в LRU: если capacity достигнут, удаляет tail элемента (LRU), инкрементит `Evictions`; добавляет новый элемент в front — AC-002, RQ-004. Touches: `internal/infrastructure/embedder/cache/lru.go`

- [x] **T2.3** Реализовать LRU promotion логику — метод `Get(key)` в LRU: при hit перемещает элемент в front списка (promotion), возвращает значение — DEC-001. Touches: `internal/infrastructure/embedder/cache/lru.go`

- [x] **T2.4** Реализовать дефолтные опции и валидацию — `WithCacheSize` устанавливает размер (default 1000, min 1); конструктор валидирует base embedder != nil — RQ-002, RQ-004. Touches: `internal/infrastructure/embedder/cache/options.go`, `internal/infrastructure/embedder/cache/cache.go`

- [x] **T2.5** Интегрировать stats в cache flow — `EmbedderCache.Embed` вызывает `stats.RecordHit()` или `stats.RecordMiss()`; LRU вызывает `stats.RecordEviction()` при eviction; `Stats()` метод возвращает актуальную структуру с `HitRate` вычислением — AC-007, RQ-008. Touches: `internal/infrastructure/embedder/cache/cache.go`, `internal/infrastructure/embedder/cache/stats.go`

- [x] **T2.6** Добавить unit-тесты для in-memory кэша — `cache_test.go` содержит тесты: `TestBasicCaching` (AC-001), `TestLRUEviction` (AC-002), `TestHashConsistency` (AC-006), `TestCacheStats` (AC-007); `go test -race` проходит — AC-003. Touches: `internal/infrastructure/embedder/cache/cache_test.go`, `internal/infrastructure/embedder/cache/lru_test.go`

## Фаза 3: Redis second-level и интеграция

Цель: добавить Redis как second-level cache с graceful fallback и покрыть оставшиеся AC.

- [x] **T3.1** Реализовать Redis client wrapper — `redis.go` содержит: структуру `redisCache` с `*redis.Client` и `ttl`, методы `Get(ctx, key) ([]float64, bool, error)`, `Set(ctx, key, value) error`, сериализация/десериализация через msgpack — RQ-006, RQ-007, DEC-004. Touches: `internal/infrastructure/embedder/cache/redis.go`

- [x] **T3.2** Интегрировать Redis в двухуровневый lookup — `EmbedderCache.Embed` обновлён: при miss L1 проверяет L2 (Redis), при hit L2 — сохраняет в L1, возвращает; при miss L2 — вызывает embedder, сохраняет в L1 и L2; при ошибке Redis — логирует warning, fallback к embedder без записи в L2 — AC-004, AC-005, RQ-009. Touches: `internal/infrastructure/embedder/cache/cache.go`, `internal/infrastructure/embedder/cache/options.go` (`WithRedis`)

- [x] **T3.3** Добавить тесты для Redis fallback — `cache_test.go` или `redis_test.go` содержит тест `TestRedisFallback`: мок Redis возвращает ошибку соединения, cache fallback'ится к base embedder, лог содержит warning — AC-004. Touches: `internal/infrastructure/embedder/cache/redis_test.go`

- [x] **T3.4** Добавить тесты для Redis second-level — тест `TestRedisSecondLevel`: предустанавливает ключ в мок Redis, вызывает `Embed`, проверяет что base embedder не вызывался — AC-005. Touches: `internal/infrastructure/embedder/cache/redis_test.go`

## Покрытие критериев приемки

| AC | Покрытие задачами | Примечание |
|----|-------------------|------------|
| AC-001 Базовое кэширование | T1.1, T1.4, T2.1, T2.6 | Cache structure → lookup flow → tests |
| AC-002 LRU eviction | T1.2, T2.2, T2.6 | LRU structure → eviction logic → tests |
| AC-003 Thread-safety | T2.6 | Race detector в тестах |
| AC-004 Redis fallback | T3.2, T3.3 | Fallback logic + test |
| AC-005 Redis second-level | T3.1, T3.2, T3.4 | Redis client + integration + test |
| AC-006 Хэш консистентности | T2.6 | Unit test с SHA-256 |
| AC-007 Статистика кэша | T1.3, T2.5, T2.6 | Stats structure → integration → tests |

## Покрытие требований

| RQ | Покрытие задачами |
|----|-------------------|
| RQ-001 Реализует `domain.Embedder` | T1.1, T2.1 |
| RQ-002 Принимает базовый `Embedder` | T1.1, T1.4, T2.4 |
| RQ-003 SHA-256 хэш | T1.1, T2.1 |
| RQ-004 LRU-алгоритм | T1.2, T2.2, T2.4 |
| RQ-005 Thread-safe | T1.2 (RWMutex), T2.6 (-race) |
| RQ-006 Msgpack сериализация | T1.5, T3.1 |
| RQ-007 TTL для Redis | T3.1 |
| RQ-008 Методы статистики | T1.3, T2.5 |
| RQ-009 Redis fallback | T3.2 |

## Заметки

- **Порядок задач**: Фаза 1 блокирует Фазу 2 (нужны структуры); Фаза 2 блокирует Фазу 3 (нужен L1 для двухуровневого lookup)
- **Зависимости**: T2.1 зависит от T1.1, T1.2, T1.3; T3.2 зависит от T2.1, T3.1
- **T1.5** (msgpack) может выполняться параллельно с T2.x, но нужен для T3.1
- **T2.6** (тесты) покрывает множество AC — важно запустить с `-race`
