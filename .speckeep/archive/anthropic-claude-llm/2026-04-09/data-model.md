# Data Model: Anthropic Claude LLM

## Сущности

Нет persisted entities — клиент работает с эфемерными HTTP запросами/ответами.

## Request/Response структуры

### messagesRequest (non-streaming)
```go
type messagesRequest struct {
    Model       string           `json:"model"`
    MaxTokens   int              `json:"max_tokens"`
    System      string           `json:"system,omitempty"`
    Messages    []messageContent `json:"messages"`
    Temperature *float64         `json:"temperature,omitempty"`
}
```

### messagesResponse
```go
type messagesResponse struct {
    Content []contentBlock `json:"content"`
    Role    string         `json:"role"`
}

type contentBlock struct {
    Type string `json:"type"`
    Text string `json:"text"`
}
```

### messagesStreamRequest
```go
type messagesStreamRequest struct {
    Model       string           `json:"model"`
    MaxTokens   int              `json:"max_tokens"`
    System      string           `json:"system,omitempty"`
    Messages    []messageContent `json:"messages"`
    Temperature *float64         `json:"temperature,omitempty"`
    Stream      bool             `json:"stream"`
}
```

### streamEvent (SSE)
```go
type streamEvent struct {
    Type  string `json:"type"`  // "content_block_delta", "message_stop"
    Delta struct {
        Type string `json:"type"`
        Text string `json:"text"`
    } `json:"delta"`
}
```

## Invariants

- `model` — непустая строка, по умолчанию `"claude-3-haiku-20240307"`
- `max_tokens` — положительное число, по умолчанию `1024`
- `anthropic-version` заголовок — обязателен, по умолчанию `"2023-06-01"`
- `messages` — всегда содержит хотя бы одно сообщение (user)
- `system` — опционально, если пустое — не включается в запрос

## Lifecycle

Нет persisted lifecycle — каждый вызов `Generate()` / `GenerateStream()` создаёт новый HTTP запрос.

## Отображение на AC

| AC | Data Model Impact |
|----|-------------------|
| AC-002 | Структуры `messagesRequest`, `messagesResponse` |
| AC-003 | Константа `defaultAnthropicVersion` |
| AC-004 | `messagesStreamRequest`, `streamEvent` |
