package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type capturedLog struct {
	level  domain.LogLevel
	msg    string
	fields map[string]any
}

type fakeLogger struct {
	entries []capturedLog
	panic   bool
}

func (l *fakeLogger) Log(ctx context.Context, level domain.LogLevel, msg string, fields ...domain.LogField) {
	_ = ctx
	if l.panic {
		panic("logger panic")
	}
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	l.entries = append(l.entries, capturedLog{level: level, msg: msg, fields: m})
}

func TestRetryEmbedder_LogsRetryAttempt(t *testing.T) {
	mockEmb := new(MockEmbedder)
	logger := &fakeLogger{}

	// 1-я попытка — ошибка, 2-я — успех
	mockEmb.On("Embed", mock.Anything, "t").
		Return(nil, errors.New("transient error")).Once()
	mockEmb.On("Embed", mock.Anything, "t").
		Return([]float64{0.1}, nil).Once()

	config := &RetryConfig{
		MaxRetries: 1,
		Backoff: &Backoff{
			BaseDelay:    1 * time.Millisecond,
			MaxDelay:     1 * time.Millisecond,
			Multiplier:   1,
			JitterFactor: 0,
		},
	}

	retryEmb := NewRetryEmbedder(mockEmb, config, nil, nil, logger)
	_, err := retryEmb.Embed(context.Background(), "t")
	require.NoError(t, err)

	require.NotEmpty(t, logger.entries)
	found := false
	for _, e := range logger.entries {
		if e.fields["operation"] == "embed" && e.fields["attempt"] == 0 {
			found = true
			assert.Equal(t, "resilience_retry", e.fields["component"])
			assert.Equal(t, false, e.fields["rejected"])
			break
		}
	}
	assert.True(t, found, "должен быть лог retry attempt для embed")
}

func TestRetryEmbedder_LogsCircuitBreakerRejection(t *testing.T) {
	mockEmb := new(MockEmbedder)
	logger := &fakeLogger{}

	mockEmb.On("Embed", mock.Anything, "t").
		Return(nil, errors.New("persistent error"))

	config := &RetryConfig{
		MaxRetries: 1,
		Backoff: &Backoff{
			BaseDelay:    1 * time.Millisecond,
			MaxDelay:     1 * time.Millisecond,
			Multiplier:   1,
			JitterFactor: 0,
		},
	}
	cbConfig := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   10 * time.Second,
	}

	retryEmb := NewRetryEmbedder(mockEmb, config, cbConfig, nil, logger)

	// 1-й вызов запишет failure и откроет CB
	_, _ = retryEmb.Embed(context.Background(), "t")
	// 2-й вызов должен быть отклонён
	_, err := retryEmb.Embed(context.Background(), "t")
	require.Error(t, err)

	found := false
	for _, e := range logger.entries {
		if e.fields["operation"] == "embed" && e.fields["rejected"] == true {
			found = true
			break
		}
	}
	assert.True(t, found, "должен быть лог CB rejection для embed")
}

func TestRetryEmbedder_LoggerPanicDoesNotBreak(t *testing.T) {
	mockEmb := new(MockEmbedder)
	logger := &fakeLogger{panic: true}

	mockEmb.On("Embed", mock.Anything, "t").
		Return(nil, errors.New("transient error")).Times(2)

	config := &RetryConfig{
		MaxRetries: 1,
		Backoff: &Backoff{
			BaseDelay:    1 * time.Millisecond,
			MaxDelay:     1 * time.Millisecond,
			Multiplier:   1,
			JitterFactor: 0,
		},
	}

	retryEmb := NewRetryEmbedder(mockEmb, config, nil, nil, logger)
	assert.NotPanics(t, func() {
		_, _ = retryEmb.Embed(context.Background(), "t")
	})
}

func TestRetryLLMProvider_LogsCircuitBreakerRejection(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	logger := &fakeLogger{}

	mockLLM.On("Generate", mock.Anything, "s", "u").
		Return("", errors.New("persistent error"))

	config := &RetryConfig{
		MaxRetries: 1,
		Backoff: &Backoff{
			BaseDelay:    1 * time.Millisecond,
			MaxDelay:     1 * time.Millisecond,
			Multiplier:   1,
			JitterFactor: 0,
		},
	}
	cbConfig := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   10 * time.Second,
	}

	retryLLM := NewRetryLLMProvider(mockLLM, config, cbConfig, nil, logger)

	_, _ = retryLLM.Generate(context.Background(), "s", "u")
	_, err := retryLLM.Generate(context.Background(), "s", "u")
	require.Error(t, err)

	found := false
	for _, e := range logger.entries {
		if e.fields["operation"] == "generate" && e.fields["rejected"] == true {
			found = true
			break
		}
	}
	assert.True(t, found, "должен быть лог CB rejection для generate")
}
