# Redis backend для кэша эмбеддингов План

## Phase Contract

Inputs: `.speckeep/specs/redis-cache-backend/spec.md` и минимальный контекст репозитория, затрагивающий `CachedEmbedder` и `internal/infrastructure/embedder/cache/`.
Outputs: `plan.md`, `data-model.md`. Contracts не требуются.
Stop if: возникает необходимость привязаться к конкретной Redis-библиотеке (против RQ-002.1) или требуется менять поведение по умолчанию (против RQ-001).

## Цель

Добавить опциональный Redis L2 к существующему `EmbedderCache` (L1 = in-memory LRU) и вывести включение Redis в публичный API `pkg/draftrag` через адаптер-интерфейс клиента. Реализация должна быть best-effort, безопасной к ошибкам Redis (treat-as-miss + логирование) и без breaking changes для пользователей, которые Redis не включают.

## Scope

- Реализовать Redis L2 кэширование эмбеддингов с TTL и префиксом ключей (с дефолтом).
- Расширить публичную конфигурацию `CachedEmbedder`, не требуя импортов `internal`-пакетов.
- Добавить unit-тесты без реального Redis (fake client + mock embedder).
- Явно не менять поведение по умолчанию: только LRU без Redis.

## Implementation Surfaces

- `internal/infrastructure/embedder/cache/`: основной пакет кэша embedder (L1 и L2), большинство изменений и тестов живут здесь.
- `internal/infrastructure/embedder/cache/cache.go`: L2 lookup и write path, warming L1, обработка ошибок Redis.
- `internal/infrastructure/embedder/cache/options.go`: опции Redis (ttl, prefix) и интерфейс Redis-клиента (перейти на адаптер без go-redis командных типов).
- `internal/infrastructure/embedder/cache/redis.go` (новый): `redisCache` с сериализацией/десериализацией msgpack и построением ключа с префиксом.
- `pkg/draftrag/cached_embedder.go`: публичные `CacheOptions`/конструктор, проброс Redis-настроек в `internal/infrastructure/embedder/cache`.
- `internal/infrastructure/embedder/cache/cache_test.go`: unit-тесты по AC (fake redis client + mock embedder).
- `internal/infrastructure/embedder/cache/redis_test.go`: unit-тесты по AC (fake redis client + mock embedder).
- Документация (опционально): `README.md` / `docs/embedders.md` — только если нужно явно показать новый usage.

## Влияние на архитектуру

- Локально расширяется инфраструктурный слой кэша эмбеддингов; domain и application не затрагиваются.
- Публичный API расширяется добавлением опциональной конфигурации Redis; существующие вызовы не меняются.
- Новая внешняя зависимость не добавляется: Redis подключается пользователем через адаптер-интерфейс.

## Acceptance Approach

- AC-001: оставить текущий путь `NewCachedEmbedder(..., CacheOptions{MaxSize: ...})` без обязательных параметров Redis; тест/сборка подтверждают отсутствие breaking changes (`pkg/draftrag` компилируется, существующие тесты проходят).
- AC-002: при L1 miss и наличии значения в Redis — `Embed` возвращает значение без вызова базового embedder; unit-тест с mock embedder (счётчик вызовов) + fake redis client (отдаёт валидные bytes).
- AC-003: после L2 hit — warming L1; второй вызов `Embed` по тому же тексту обслуживается из L1 и не вызывает `Redis GET`; unit-тест считает `Get` у fake redis client.
- AC-004: при ошибках Redis (Get/Set) — `Embed` не падает, а идёт по fallback на embedder; unit-тест с redis client, который всегда возвращает ошибку.
- AC-005: при записи в Redis — ключ использует префикс (или дефолт), TTL пробрасывается; unit-тест проверяет аргументы `Set` у fake redis client.
- AC-006: при “битых” bytes из Redis — treat-as-miss + логирование, далее вызывается embedder и значение сохраняется; unit-тест с fake redis client, возвращающим некорректные bytes.

## Данные и контракты

- Data model для Redis ключа/значения и TTL зафиксирован в `data-model.md`.
- Внешние API/event contracts не добавляются: это библиотечная конфигурация и внутренняя сериализация кэша.

## Стратегия реализации

- DEC-001 Публичный Redis клиент как адаптер-интерфейс
  Why: требование RQ-002.1 — избежать vendor lock-in и зависимости от конкретной Redis-библиотеки.
  Tradeoff: пользователю может понадобиться написать маленький адаптер под свой клиент.
  Affects: `pkg/draftrag/cached_embedder.go`, `internal/infrastructure/embedder/cache/options.go`.
  Validation: проект компилируется без добавления зависимостей Redis; unit-тесты используют fake client без go-redis типов.

- DEC-002 Сериализация значений в Redis через msgpack
  Why: `msgpack` уже в зависимостях; компактный стабильный формат для `[]float64`.
  Tradeoff: изменение формата в будущем потребует стратегии совместимости/миграции ключей.
  Affects: `internal/infrastructure/embedder/cache/redis.go`, `internal/infrastructure/embedder/cache/cache.go`.
  Validation: unit-тесты на roundtrip encode/decode и обработку “битых” данных (AC-006).

- DEC-003 Формат ключа: `<prefix><sha256(text)>` с дефолтным префиксом
  Why: избегаем коллизий в shared Redis и сохраняем текущую стабильность ключа по SHA-256.
  Tradeoff: смена префикса меняет keyspace; это ожидаемо и контролируемо пользователем.
  Affects: `internal/infrastructure/embedder/cache/redis.go`, публичные опции в `pkg/draftrag`.
  Validation: unit-тесты проверяют, что `Set`/`Get` используют ключ с префиксом (AC-005).

- DEC-004 Ошибки Redis и ошибки decode = treat-as-miss + логирование
  Why: RQ-008 — кэш не должен ломать основной путь `Embed`, Redis не является source of truth.
  Tradeoff: при постоянных ошибках Redis снижается эффективность кэша; это видно по логам/поведению.
  Affects: `internal/infrastructure/embedder/cache/cache.go`, `internal/infrastructure/embedder/cache/redis.go`.
  Validation: unit-тесты AC-004/AC-006; `Embed` возвращает результат embedder при ошибках Redis.

- DEC-005 Warming L1 после L2 hit
  Why: RQ-004/AC-003 — после одного сетевого чтения дальнейшие обращения должны быть локальными.
  Tradeoff: небольшая дополнительная память в L1; контролируется `MaxSize`.
  Affects: `internal/infrastructure/embedder/cache/cache.go`.
  Validation: unit-тест AC-003 (второй вызов без `Redis GET`).

## Incremental Delivery

### MVP (Первая ценность)

- Ввести адаптер-интерфейс Redis клиента и пробросить конфигурацию из `pkg/draftrag` в internal кэш.
- Реализовать Redis `Get/Set` с msgpack и treat-as-miss поведением.
- Покрыть unit-тестами AC-001..AC-006.

Критерий готовности MVP: `go test ./...` проходит, `speckeep check redis-cache-backend .` показывает next `/speckeep.tasks redis-cache-backend`.

### Итеративное расширение

- Добавить/обновить usage-документацию (пример включения Redis и написания адаптера).
- При необходимости — расширить `CacheStats` на раздельные L1/L2 хиты (если это потребуется для эксплуатации; сейчас вне требований).

## Порядок реализации

- Сначала: определить финальный интерфейс адаптера Redis (публичный и internal) и формат ключа (prefix + hash).
- Затем: реализовать `redisCache` (serialize/deserialize, ttl, key builder).
- Затем: интегрировать L2 в `EmbedderCache.Embed` (L2 read -> warming, L2 write best-effort).
- Затем: добавить/обновить unit-тесты по AC.
- В конце: обновить docs (если решим, что это необходимо для usability).

## Риски

- Риск: несовместимость интерфейса адаптера с популярными Redis клиентами.
  Mitigation: держать интерфейс минимальным (GetBytes/SetBytes), показать пример адаптера в docs.
- Риск: “битые” или устаревшие значения в Redis приводят к ошибкам decode.
  Mitigation: treat-as-miss + логирование (RQ-008), unit-тест AC-006.
- Риск: рост нагрузки на embedder при деградации Redis.
  Mitigation: best-effort и отсутствие фатальных ошибок; при необходимости пользователь может мониторить логи/метрики на своей стороне.

## Rollout и compatibility

- Rollout не требует миграций: Redis опционален и включается явной конфигурацией.
- Compatibility: существующий код без Redis работает как прежде (AC-001).

## Проверка

- Unit-тесты в `internal/infrastructure/embedder/cache` покрывают AC-002..AC-006.
- Компиляция/тесты публичного пакета `pkg/draftrag` подтверждают AC-001.
- Проверка `speckeep check redis-cache-backend .` подтверждает readiness к `/speckeep.tasks`.

## Соответствие конституции

- Интерфейсная абстракция: Redis интеграция через Go-интерфейс (адаптер), без жёсткой зависимости от конкретного клиента (соответствует “Интерфейсная абстракция”).
- Контекстная безопасность: все операции Redis используют `context.Context` (соответствует “Контекстная безопасность”).
- Минимальная конфигурация: дефолт — только LRU; Redis включается опционально (соответствует “Минимальная конфигурация”).
- Тестируемость: Redis слой покрывается unit-тестами через fake client, без внешних сервисов (соответствует “Тестируемость”).
