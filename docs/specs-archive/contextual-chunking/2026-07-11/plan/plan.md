# Contextual Chunking — План

## Phase Contract

Inputs: spec, inspect (pass), конституция, minimal repo-контекст.
Outputs: plan, data-model.md.
Stop if: нет.

## Цель

Добавить `ContextualChunker` — декоратор над `domain.Chunker`, обогащающий каждый чанк документным контекстом из `Document.Metadata`. Новый файл `internal/infrastructure/chunker/contextual.go` + публичный wrapper в `pkg/draftrag/contextual_chunker.go`. Никакие модели, интерфейсы или другие части пайплайна не меняются.

## MVP Slice

`ContextualChunker` с настраиваемым ключом метаданных, шаблоном и дефолтным поведением. Закрывает AC-001, AC-002, AC-003, AC-004, AC-006.

## First Validation Path

Юнит-тест: создать `ContextualChunker` с `BasicChunker`, вызвать `Chunk` на документе с `Metadata["title"]="Research"`, проверить что каждый чанк имеет префикс `"[CONTEXT] Research\n"`.

## Scope

- Внутренняя реализация в `internal/infrastructure/chunker/contextual.go`
- Публичный wrapper + валидация в `pkg/draftrag/contextual_chunker.go`
- Unit-тесты для contextual chunker + тесты валидации
- Pipeline options не расширяются — пользователь передаёт Chunker напрямую в `PipelineOptions.Chunker`

## Performance Budget

- `none` — фича добавляет O(n) конкатенацию строк на чанк, что несущественно на фоне эмбеддинга.

## Implementation Surfaces

- `internal/infrastructure/chunker/contextual.go` — новая; внутренняя реализация `ContextualChunker`
- `pkg/draftrag/contextual_chunker.go` — новая; публичный тип `ContextualChunkerOptions` + `NewContextualChunker`
- `pkg/draftrag/errors.go` — существующая; `ErrInvalidChunkerConfig` уже есть, повторный sentinel не нужен
- `internal/infrastructure/chunker/contextual_test.go` — новая; юнит-тесты
- `pkg/draftrag/contextual_chunker_test.go` — новая; тесты публичного wrapper + валидация

## Bootstrapping Surfaces

- `none` — все нужные директории и файлы в репозитории уже существуют.

## Влияние на архитектуру

- Локальное: новый файл в `internal/infrastructure/chunker/`, новый файл в `pkg/draftrag/`.
- Нет влияния на интерфейсы, модели, интеграции, совместимость или rollout.

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|---|---|---|---|
| AC-001 | Юнит-тест: дефолтный шаблон + контекст | `contextual.go`, `contextual_chunker.go` | `HasPrefix` на каждом чанке |
| AC-002 | Юнит-тест: кастомный шаблон | `contextual.go` | `HasPrefix` + `Contains` separator |
| AC-003 | Юнит-тест: nil/пустой Metadata | `contextual.go` | Чанки идентичны базовому чанкеру |
| AC-004 | Юнит-тест: отменённый ctx | `contextual.go` | Возвращается `ctx.Err()` |
| AC-005 | Интеграционный тест через Pipeline | `contextual_chunker.go` + Pipeline | Поиск по контекстному слову |
| AC-006 | Юнит-тест: `ContextKey="description"` | `contextual.go`, `contextual_chunker.go` | Чанки с префиксом "Annual Report 2025" |

## Данные и контракты

- Data model не меняется. См. `data-model.md`.
- API контракты не меняются. `ContextualChunker` реализует существующий `domain.Chunker`.

## Стратегия реализации

### DEC-001 Декоратор, а не отдельный чанкер с нуля

Why: контекстуализация ортогональна стратегии чанкинга — пользователь хочет добавить контекст к любому существующему чанкеру (Basic, Semantic, будущие). Наследование или форк BasicChunker/СеmanticChunker привело бы к дублированию кода.
Tradeoff: небольшой overhead на вызов внутреннего чанкера + проход по чанкам.
Affects: `internal/infrastructure/chunker/contextual.go`, `pkg/draftrag/contextual_chunker.go`
Validation: `ContextualChunker` принимает любой `domain.Chunker` и корректно оборачивает его.

### DEC-002 Контекст модифицирует Chunk.Content, а не добавляет отдельное поле

Why: контекст должен влиять на эмбеддинг и поиск. Отдельное поле Chunk.Context потребовало бы изменений в эмбеддере и VectorStore, что расширяет scope и ломает обратную совместимость.
Tradeoff: контекст не хранится отдельно — если retrieval вернёт чанк, контекст будет частью контента. Для отображения "чистого" контента потребуется пост-обработка.
Affects: `internal/infrastructure/chunker/contextual.go`
Validation: после `Chunk()` каждый `Chunk.Content` содержит контекстный префикс.

### DEC-003 Одно поле метаданных, без поддержки множественных источников

Why: spec явно исключает множественные источники. Одно поле покрывает основной юзкейс (заголовок документа) без усложнения API.
Tradeoff: для комбинированного контекста (title + description) пользователь заранее объединяет значения в одном поле Metadata.
Affects: `ContextualChunkerOptions.ContextKey`
Validation: AC-006.

## Incremental Delivery

### MVP (Первая ценность)

- `internal/infrastructure/chunker/contextual.go` — реализация декоратора
- `pkg/draftrag/contextual_chunker.go` — публичный конструктор + валидация
- Тесты: unit (AC-001–004, AC-006)
- Критерий: `go test ./internal/infrastructure/chunker/ -run TestContextual -v && go test ./pkg/draftrag/ -run TestContextual -v`

### Итеративное расширение

- Интеграционный тест через Pipeline (AC-005) — можно отложить, если не критично для первого внедрения.

## Порядок реализации

1. `internal/infrastructure/chunker/contextual.go` — внутренний декоратор
2. `internal/infrastructure/chunker/contextual_test.go` — unit-тесты внутреннего
3. `pkg/draftrag/contextual_chunker.go` — публичный wrapper + валидация
4. `pkg/draftrag/contextual_chunker_test.go` — тесты публичного API + валидации
5. Интеграционный тест AC-005 (опционально)

Шаги 1–4 можно выполнять последовательно. Шаг 5 независим.

## Риски

- Риск: контекст может исказить семантическое пространство эмбеддингов (контекстный префикс во всех чанках документа может сделать их более похожими между собой).
  Mitigation: AC-005 проверяет что поиск по контекстному слову работает; решение о влиянии на качество — за пользователем (выбор шаблона и источника контекста).
- Риск: очень длинный контекст (10k+ символов) может доминировать над содержимым чанка.
  Mitigation: spec явно разрешает длинный контекст без обрезки; если это станет проблемой — будущее расширение с `MaxContextSize`.

## Rollout и compatibility

- Специальных rollout-действий не требуется — новая опциональная фича, не ломает существующие индексы.

## Проверка

| Проверка | Тип | Покрывает |
|---|---|---|
| Unit: дефолтный шаблон | automated | AC-001 |
| Unit: кастомный шаблон | automated | AC-002 |
| Unit: пустой/отсутствующий Metadata | automated | AC-003 |
| Unit: отменённый контекст | automated | AC-004 |
| Integration: поиск по контексту | automated | AC-005 |
| Unit: кастомный ключ метаданных | automated | AC-006 |
| Unit: валидация опций (пустой шаблон, без {content}) | automated | RQ-005, краевые случаи |
| `go vet ./...` | automated | — |
| `golangci-lint` | automated | — |

## Соответствие конституции

- нет конфликтов: фича следует Clean Architecture (domain → application → infrastructure), не вводит HTTP-сервер/CLI, использует `context.Context`, все внешние зависимости через Go-интерфейсы.
