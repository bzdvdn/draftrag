package llm

import (
	"context"
	"errors"
	"time"
)

// MockStreamingLLM — мок-реализация StreamingLLMProvider для тестирования.
// Поддерживает controlled token emission, таймауты и ошибки.
//
// @ds-task T3.2: Создать мок-реализацию StreamingLLMProvider (RQ-007)
type MockStreamingLLM struct {
	// Tokens — токены, которые будут отправлены в канал
	Tokens []string
	// Delay — задержка между токенами
	Delay time.Duration
	// Err — ошибка, которую нужно вернуть при инициализации streaming'а
	Err error
	// GenerateErr — ошибка для синхронного Generate
	GenerateErr error
	// GenerateResult — результат для синхронного Generate
	GenerateResult string
}

// Generate возвращает мок-результат или ошибку.
func (m *MockStreamingLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.GenerateErr != nil {
		return "", m.GenerateErr
	}
	return m.GenerateResult, nil
}

// GenerateStream возвращает канал с токенами и опциональную задержку.
func (m *MockStreamingLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	ch := make(chan string)

	go func() {
		defer close(ch)

		for _, token := range m.Tokens {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if m.Delay > 0 {
				select {
				case <-time.After(m.Delay):
				case <-ctx.Done():
					return
				}
			}

			select {
			case ch <- token:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// MockStreamingLLMWithCancel — мок, который не закрывается до отмены контекста.
// Используется для тестирования обработки отмены контекста.
type MockStreamingLLMWithCancel struct {
	Tokens []string
	Delay  time.Duration
}

// Generate возвращает пустую строку.
func (m *MockStreamingLLMWithCancel) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "", nil
}

// GenerateStream возвращает канал, который не закрывается сам.
func (m *MockStreamingLLMWithCancel) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		for _, token := range m.Tokens {
			select {
			case <-ctx.Done():
				return
			case ch <- token:
			}

			if m.Delay > 0 {
				select {
				case <-time.After(m.Delay):
				case <-ctx.Done():
					return
				}
			}
		}

		// Ждём отмены контекста или закрытия канала извне
		<-ctx.Done()
	}()

	return ch, nil
}

// NonStreamingLLM — мок, который НЕ реализует StreamingLLMProvider.
// Используется для тестирования graceful degradation.
type NonStreamingLLM struct {
	Result string
	Err    error
}

// Generate возвращает мок-результат.
func (m *NonStreamingLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	return m.Result, nil
}

// ErrTestStreaming — тестовая ошибка для streaming.
var ErrTestStreaming = errors.New("test streaming error")
