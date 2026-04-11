# LLM провайдеры

## OpenAI-compatible (Responses API)

Работает с OpenAI Responses API (`POST /v1/responses`) и совместимыми провайдерами.

```go
llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "gpt-4o-mini",
})
```

Реализует `LLMProvider` и `StreamingLLMProvider`.

### Опции

| Поле | Описание |
|---|---|
| `BaseURL` | **Обязательно.** Базовый URL (без `/v1/responses`) |
| `APIKey` | **Обязательно.** Ключ API |
| `Model` | **Обязательно.** Имя модели |
| `Temperature` | `*float64`. `nil` → не передаётся в запросе |
| `MaxOutputTokens` | `*int`. `nil` → не передаётся |
| `HTTPClient` | `nil` → `http.DefaultClient` |
| `Timeout` | Таймаут на `Generate`. Не применяется к `GenerateStream` |

### Streaming

```go
// type assertion для получения streaming capability
streamingLLM, ok := llm.(draftrag.StreamingLLMProvider)
if !ok {
    // не поддерживается
}
ch, err := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
```

Через Pipeline:

```go
tokenChan, err := pipeline.AnswerStream(ctx, "вопрос", 5)
if errors.Is(err, draftrag.ErrStreamingNotSupported) {
    // LLM не поддерживает streaming
}
for token := range tokenChan {
    fmt.Print(token)
}
```

---

## Anthropic Claude

Нативная поддержка [Anthropic Messages API](https://docs.anthropic.com/en/api/messages).

```go
llm := draftrag.NewAnthropicLLM(draftrag.AnthropicLLMOptions{
    BaseURL: "https://api.anthropic.com",
    APIKey:  "sk-ant-...",
    Model:   "claude-3-haiku-20240307",
})
```

Реализует `LLMProvider` и `StreamingLLMProvider`.

### Опции

| Поле | По умолчанию | Описание |
|---|---|---|
| `BaseURL` | — | **Обязательно.** `https://api.anthropic.com` |
| `APIKey` | — | **Обязательно.** Ключ API (`X-API-Key` заголовок) |
| `Model` | `claude-3-haiku-20240307` | Модель (пустая строка → дефолт) |
| `AnthropicVersion` | `2023-06-01` | Версия API (`anthropic-version` заголовок) |
| `Temperature` | `nil` | `*float64` |
| `MaxTokens` | `nil` (→ 1024) | `*int` |
| `HTTPClient` | `http.DefaultClient` | |
| `Timeout` | `0` | Таймаут на `Generate` |

### Популярные модели

| Модель | Скорость | Качество |
|---|---|---|
| `claude-3-haiku-20240307` | Быстрый | Хорошее |
| `claude-3-5-sonnet-20241022` | Средний | Отличное |
| `claude-opus-4-6` | Медленный | Максимальное |

---

## Ollama

Локальные LLM-модели через [Ollama](https://ollama.com/).

```go
llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
    Model: "llama3.2",
})
```

### Опции

| Поле | По умолчанию | Описание |
|---|---|---|
| `Model` | — | **Обязательно.** |
| `BaseURL` | `http://localhost:11434` | URL Ollama сервера |
| `APIKey` | `""` | Опционально (для авторизации) |
| `Temperature` | `nil` | `*float64` |
| `MaxTokens` | `nil` | `*int` |
| `HTTPClient` | `http.DefaultClient` | |
| `Timeout` | `0` | Таймаут на `Generate` |

### Установка моделей

```bash
ollama pull llama3.2        # 2B, быстрый
ollama pull llama3.1:8b     # 8B, хорошее качество
ollama pull mistral         # 7B
ollama pull qwen2.5:7b      # хорошо для русского языка
```

**Ollama не поддерживает streaming** через draftRAG. `ErrStreamingNotSupported` при попытке использовать `AnswerStream`.

---

## Кастомный LLM

Реализуйте интерфейс для любого провайдера:

```go
type MyLLM struct{}

func (m *MyLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
    // ваша реализация
}

pipeline := draftrag.NewPipeline(store, &MyLLM{}, embedder)
```

Для streaming дополнительно реализуйте `GenerateStream(ctx, systemPrompt, userMessage) (<-chan string, error)`.
