---
status: minor-changes
---

# Data Model: arch-issues

## Status

`minor-changes` — добавляются новые типы для tool calling; существующие типы не меняются.

## Новые типы

### `ToolDefinition` (в `internal/domain/models.go`)

Описание инструмента для LLM tool calling. JSON-совместимый формат для передачи в OpenAI/Anthropic/Mistral API.

```go
type ToolDefinition struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}
```

### `ToolCall` (в `internal/domain/models.go`)

Результат вызова инструмента от LLM.

```go
type ToolCall struct {
    ID       string          `json:"id"`
    Name     string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}
```

### `ToolResult` (в `internal/domain/models.go`)

Результат выполнения инструмента для передачи обратно в LLM.

```go
type ToolResult struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Result string `json:"result"` // text output
}
```

### `ErrPipelineClosed` (в `internal/application/`)

```go
var ErrPipelineClosed = errors.New("pipeline is closed")
```

### `ErrToolsNotSupportedInStream` (в `pkg/draftrag/`)

```go
var ErrToolsNotSupportedInStream = errors.New("tool calling is not supported in streaming mode")
```

## Изменения существующих типов

Нет. Все изменения — добавление новых типов и опционального интерфейса.

## Без изменений (явно)

- `Document`, `Chunk`, `RetrievalResult` — не меняются.
- `HybridConfig`, `MetadataFilter` — не меняются.
- `Config` (YAML) — не меняется (tool calling — runtime параметр, не конфигурационный).
