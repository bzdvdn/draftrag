package draftrag

import (
	"context"
	"errors"
	"strings"
	"time"
)

type embedCallFunc func(ctx context.Context, text string) ([]float64, error)

func embedWithValidation(ctx context.Context, text string, timeout time.Duration, validate func() error, call embedCallFunc) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}
	if err := validate(); err != nil {
		return nil, err
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return call(ctx, text)
}

type generateCallFunc func(ctx context.Context, systemPrompt, userMessage string) (string, error)

func generateWithValidation(ctx context.Context, systemPrompt, userMessage string, timeout time.Duration, validate func() error, call generateCallFunc) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", errors.New("userMessage is empty")
	}
	if err := validate(); err != nil {
		return "", err
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return call(ctx, systemPrompt, userMessage)
}
