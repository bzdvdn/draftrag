# Inspect: hybrid-search-bm25-semantic

## Статус

**APPROVED** — спецификация соответствует конституции и готова к планированию.

## Дата проверки

2026-04-08

## Проверки по конституции

| Принцип | Результат | Примечание |
|---------|-----------|------------|
| Интерфейсная абстракция | ✅ PASS | `HybridSearcher` — capability-интерфейс в domain; реализация будет в infrastructure |
| Чистая архитектура | ✅ PASS | Интерфейс → domain, реализация → pgvector.go, API → pkg/draftrag |
| Контекстная безопасность | ✅ PASS | Все методы принимают `context.Context` первым параметром |
| Минимальная конфигурация | ✅ PASS | Разумные дефолты (`UseRRF: true`, `RRFK: 60`, `SemanticWeight: 0.7`) |
| Языковая политика | ✅ PASS | Спецификация на русском; godoc будет на русском |
| Тестируемость | ✅ PASS | Интерфейс позволяет мокировать; AC-009 требует покрытия ≥80% |

## Проверка требований

| ID | Описание | Статус |
|----|----------|--------|
| RQ-001 | Интерфейс `HybridSearcher` и `HybridConfig` | ✅ Определён |
| RQ-002 | BM25-поиск в PostgreSQL | ✅ Специфицирован |
| RQ-003 | RRF fusion | ✅ Алгоритм описан |
| RQ-004 | Weighted score fusion | ✅ Альтернатива описана |
| RQ-005 | Миграция 0003_add_bm25.sql | ✅ SQL предоставлен |
| RQ-006 | Upsert с поддержкой BM25 | ✅ Backward compatibility указана |
| RQ-007 | Фильтрация в гибридном поиске | ✅ Расширенный интерфейс описан |
| RQ-008 | Публичный API | ✅ Методы и конфигурация описаны |

## Выявленные проблемы

**BLOCKER**: Нет

**WARNING**: Минорные рекомендации (не блокируют планирование):
1. Добавить `DefaultHybridConfig()` helper для консистентности
2. Добавить SQL down-migration в RQ-005
3. Добавить `HybridConfig.Validate()` метод
4. Уточить location `HybridConfig` — `internal/domain/models.go`

## Архитектурные решения

| ID | Решение | Обоснование |
|----|---------|-------------|
| DEC-001 | BM25 в pgvector.go | Capability существующего store |
| DEC-002 | Язык 'english' | PostgreSQL default, расширяемо в будущем |
| DEC-003 | RRF по умолчанию | Устойчивость, не требует tuning |
| DEC-004 | MetadataFilter совместимость | Fusion по пересечению результатов |

## Зависимости

- ✅ Metadata filtering — реализовано в codebase
- ✅ Migration system — уже реализовано

## Критерии приёмки

| ID | Описание | Статус проверки |
|----|----------|-----------------|
| AC-001 | `HybridSearcher` в domain | ✅ Проверено в RQ-001 |
| AC-002 | `SearchBM25` в pgvector | ✅ Проверено в RQ-002 |
| AC-003 | RRF fusion | ✅ Проверено в RQ-003 |
| AC-004 | Weighted fusion | ✅ Проверено в RQ-004 |
| AC-005 | Миграция 0003 | ✅ Проверено в RQ-005 |
| AC-006 | Upsert backward compat | ✅ Проверено в RQ-006 |
| AC-007 | `RetrieveContextHybrid` | ✅ Проверено в RQ-008 |
| AC-008 | Фильтрация в hybrid | ✅ Проверено в RQ-007 |
| AC-009 | Unit-тесты ≥80% | ⚠️ Требуется при implement |
| AC-010 | Benchmark | ⚠️ Требуется при implement |

## Рекомендации для планирования

1. Упомянуть создание `DefaultHybridConfig()` в задачах
2. Добавить down-migration в artifact `0003_add_bm25.sql`
3. Включить `HybridConfig.Validate()` в domain-задачи
4. Проверить соответствие существующего `MetadataFilter` при планировании интеграции

## Решение

Спецификация **APPROVED**. Можно переходить к `/draftspec.plan`.
