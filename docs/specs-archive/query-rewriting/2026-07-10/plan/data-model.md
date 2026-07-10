# Query Rewriting — Модель данных

## Scope

- Связанные `AC-*`: `AC-001`, `AC-003`, `AC-004`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`
- Статус: `changed`

## Сущности

### DM-001 QueryRewriter (интерфейс)

- Назначение: контракт для стратегий переписывания запросов
- Источник истины: `internal/domain/interfaces.go`
- Инварианты:
  - `Rewrite` возвращает не-nil слайс (может быть пустым)
  - `Rewrite` не модифицирует входные параметры
- Связанные `AC-*`: `AC-001`
- Связанные `DEC-*`: `DEC-001`
- Поля: (интерфейс, без полей)
  - `Rewrite(ctx context.Context, query string, history QueryHistory) ([]RewrittenQuery, error)`
- Жизненный цикл: не управляется — интерфейс, реализуется вызывающей стороной или встроенным LLMRewriter
- Замечания по консистентности: не применимо

### DM-002 RewrittenQuery

- Назначение: результат одной переформулировки запроса
- Источник истины: `internal/domain/models.go`
- Инварианты:
  - `Query` не пуста (post-validation)
  - `Weight >= 0`; 0 интерпретируется как 1.0
- Связанные `AC-*`: `AC-003`
- Связанные `DEC-*`: `DEC-002`
- Поля:
  - `Query string` — required, переписанный текст запроса
  - `Weight float64` — optional, вес при fusion (default 1.0)
- Жизненный цикл:
  - создаётся: внутри `QueryRewriter.Rewrite`
  - потребляется: pipeline для embedding + retrieval (1:1 или 1:N)
  - удаляется: после завершения retrieval/fusion
- Замечания по консистентности: слайс переформулировок не должен содержать дубликатов по `Query` (проверка в тесте)

### DM-003 QueryHistory

- Назначение: контекст предыдущих сообщений диалога для multi-turn rewriting
- Источник истины: `internal/domain/models.go`
- Инварианты:
  - `Entries` упорядочены от старого к новому
  - Каждая запись имеет валидный Role
- Связанные `AC-*`: `AC-004`
- Поля:
  - `Entries []Message` — required, список сообщений
    - `Message.Role string` — "user" или "assistant"
    - `Message.Content string` — текст сообщения
- Жизненный цикл:
  - создаётся: caller перед вызовом Search
  - передаётся: в `QueryRewriter.Rewrite`
  - не хранится: pipeline не сохраняет историю
- Замечания по консистентности: caller отвечает за актуальность и размер истории

## Связи

- `DM-001` (QueryRewriter) → создаёт → `[]DM-002` (RewrittenQuery): 1:N
- `DM-003` (QueryHistory) → передаётся в → `DM-001.Rewrite`: 1:1 per call

## Производные правила

- При пустом `[]RewrittenQuery` (nil или len==0) pipeline использует исходный запрос как единственную переформулировку.
- При 1:N fusion: все `RewrittenQuery.Weight` считаются равными 1.0 (weighted fusion отложен).

## Переходы состояний

none — все сущности value objects без жизненного цикла.

## Вне scope

- Персистентное хранение QueryHistory
- Кэширование RewrittenQuery
- Weighted fusion (использование RewrittenQuery.Weight)
