---
report_type: inspect
slug: vectorstore-pgvector
status: pass
docs_language: ru
generated_at: 2026-04-07
---

# Inspect Report: vectorstore-pgvector

## Scope

- snapshot: проверка спецификации vectorstore-pgvector на соответствие конституции и качество acceptance criteria
- artifacts:
  - .draftspec/constitution.summary.md
  - .draftspec/specs/vectorstore-pgvector/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- AC-002/AC-003/AC-004/AC-005 завязаны на интеграционные тесты с реальной БД: в plan/tasks нужно явно зафиксировать механизм запуска (env var DSN) и политику skip по умолчанию, чтобы сохранить требование RQ-005.
- В spec упомянут `SetupPGVector` как создающий расширение: на практике `CREATE EXTENSION` часто требует повышенных прав; в реализации нужно предусмотреть понятную ошибку и возможность отключить auto-extension (или документировать требование прав).
- Формулировка score: указан диапазон [-1, 1] и “cosine distance в БД + преобразование в similarity”; в plan стоит явно зафиксировать точную формулу преобразования и выбранный оператор pgvector (чтобы тесты и реализация не разъехались).

## Questions

- Нужен ли отдельный публичный под-пакет `pkg/draftrag/pgvector` (чтобы не раздувать корневой API), или принимаем функции/типы в `pkg/draftrag`? (см. Открытые вопросы в spec)
- Фиксируем ли только cosine в v1, или делаем опцию метрики (и как это отражается на индексе/DDL)?

## Suggestions

- В `RQ-004` и `AC-002` уточнить, что helper создаёт таблицу и индекс, а создание расширения — best-effort (или отдельный шаг), чтобы ожидания по правам были реалистичными.
- В `Основной сценарий` добавить явное указание на параметр DSN для интеграционных тестов (как “developer workflow”), чтобы downstream plan не терял это требование.

## Traceability

- AC-001..AC-005: критерии приемки определены, Given/When/Then присутствуют, evidence описаны и проверяемы при наличии окружения.

## Next Step

- safe to continue to plan
- Следующая команда: /draftspec.plan vectorstore-pgvector

