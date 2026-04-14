# Metadata filtering — Модель данных

## Scope

- Связанные `AC-*`: AC-001, AC-002, AC-003, AC-004, AC-005
- Связанные `DEC-*`: DEC-001, DEC-004
- Изменения касаются только domain-типов; персистентная схема (SQL-таблица, migration) не меняется.

## Сущности

### DM-001 MetadataFilter

- Назначение: выражает условие точного совпадения по полям метаданных документа при поиске; передаётся в `SearchWithMetadataFilter`.
- Источник истины: `internal/domain/models.go`
- Инварианты:
  - Пустой `Fields` (nil или len==0) семантически означает «без фильтра»; реализации должны делегировать в `Search`.
  - Все условия в `Fields` применяются как AND (все ключи должны совпасть).
  - Значение `""` в `Fields` трактуется как точное совпадение с пустой строкой, не как wildcard.
- Связанные `AC-*`: AC-001, AC-002, AC-004
- Связанные `DEC-*`: DEC-001
- Поля:
  - `Fields map[string]string` — required (может быть nil); ключи — имена полей метаданных, значения — ожидаемые строки точного совпадения
- Жизненный цикл:
  - Создаётся вызывающим кодом per-request; не персистируется.
  - Не обновляется и не удаляется — immutable value object.
- Замечания по консистентности:
  - Тип — value object без identity; никакого персистентного состояния нет, проблем консистентности нет.

### DM-002 VectorStoreWithFilters (расширенный интерфейс)

- Назначение: опциональная capability VectorStore, объединяющая фильтрацию по ParentID и по метаданным; реализуется бэкендами, которые поддерживают оба метода.
- Источник истины: `internal/domain/interfaces.go`
- Инварианты:
  - Реализатор ДОЛЖЕН реализовать оба метода: `SearchWithFilter` и `SearchWithMetadataFilter`.
  - Compile-time assert `var _ domain.VectorStoreWithFilters = (*Impl)(nil)` обязателен для каждого реализатора.
- Связанные `AC-*`: AC-001, AC-002, AC-003, AC-005
- Связанные `DEC-*`: DEC-001
- Поля (методы):
  - `SearchWithFilter(ctx, embedding, topK, ParentIDFilter) (RetrievalResult, error)` — существующий, не меняется
  - `SearchWithMetadataFilter(ctx, embedding, topK, MetadataFilter) (RetrievalResult, error)` — новый
- Жизненный цикл: интерфейс статичен; реализации инициализируются при создании store.
- Замечания по консистентности: нет состояния; интерфейс является контрактом, а не сущностью с жизненным циклом.

## Связи

- `MetadataFilter` используется как параметр `VectorStoreWithFilters.SearchWithMetadataFilter` — coupling только через сигнатуру метода, не через поле сущности.
- `Document.Metadata map[string]string` → pgvector JSONB `metadata`-колонка → `MetadataFilter.Fields`: данные, по которым фильтруем, уже хранятся при индексации документа; новый тип лишь задаёт условие выборки.

## Производные правила

- При `len(MetadataFilter.Fields) == 0`: реализация делегирует в `Search(ctx, embedding, topK)` — эквивалентность без фильтра (AC-002).
- В pgvector: `Fields` сериализуется в JSON и используется как аргумент JSONB-оператора `@>` — строгое подмножество хранимых метаданных.
- В in-memory: для каждого чанка проверяется, что все пары `(k, v)` из `Fields` присутствуют в `chunk` (в `Chunk` нет поля `Metadata` — хранится при индексации; in-memory store не хранит оригинальные метаданные документа в `Chunk`).

  > **Замечание**: `Chunk` в domain не имеет поля `Metadata` — только `ID`, `Content`, `ParentID`, `Embedding`, `Position`. Метаданные документа хранятся в pgvector JSONB-колонке при upsert. Для in-memory `SearchWithMetadataFilter` нужно решить, что хранить: либо расширить `Chunk` полем `Metadata map[string]string`, либо хранить отдельный map в `InMemoryStore`. **Это единственное открытое решение, влияющее на data model.**

  Рекомендуемое решение (DEC-001 scope): добавить `Metadata map[string]string` в `domain.Chunk` — это naturalное место хранения, данные уже есть в pgvector-колонке, и in-memory store получит консистентное поведение. pgvector-Upsert заполняет колонку `metadata` из chunk; при scan — читает обратно в `Chunk.Metadata`. Это согласовано с тем, что `Document.Metadata` — источник данных.

  Если `Chunk.Metadata` не добавлять: in-memory `SearchWithMetadataFilter` не может фильтровать (нет данных), что делает AC-005 неверифицируемым без дополнительного store-level map. Добавление поля в `Chunk` — breaking для тестов, которые создают `Chunk` без `Metadata`; это приемлемо, поле optional (nil = нет метаданных).

## Переходы состояний

Жизненный цикл достаточно прост — отдельный список переходов не нужен. `MetadataFilter` — immutable value object per request.

## Вне scope

- Изменения SQL-схемы (таблица, migration): не нужны.
- Числовые или range-условия в `MetadataFilter`: не моделируются в этой фиче.
- Комбинация `ParentIDFilter + MetadataFilter` в одном вызове: не в scope.
- Персистирование или логирование применённых фильтров.
