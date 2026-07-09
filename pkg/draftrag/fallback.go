package draftrag

import (
	"github.com/bzdvdn/draftrag/internal/infrastructure/resilience"
)

// @sk-task graceful-degradation#T4.1: re-export FallbackStats (RQ-011, AC-008)
type FallbackStats = resilience.FallbackStats

// @sk-task graceful-degradation#T4.1: re-export ErrAllProvidersFailed (RQ-012, AC-003)
var ErrAllProvidersFailed = resilience.ErrAllProvidersFailed

// @sk-task graceful-degradation#T4.1: FallbackLLMProvider public wrapper (RQ-008, AC-001)
type FallbackLLMProvider struct {
	*resilience.FallbackLLMProvider
}

// @sk-task graceful-degradation#T4.1: NewFallbackLLMProvider public constructor (RQ-008, AC-001)
func NewFallbackLLMProvider(providers []LLMProvider, logger Logger, hooks Hooks) (*FallbackLLMProvider, error) {
	internal, err := resilience.NewFallbackLLM(providers, logger, hooks)
	if err != nil {
		return nil, err
	}
	return &FallbackLLMProvider{FallbackLLMProvider: internal}, nil
}

// @sk-task graceful-degradation#T4.1: FallbackStreamingLLMProvider public wrapper (RQ-009, AC-006)
type FallbackStreamingLLMProvider struct {
	*resilience.FallbackStreamingLLMProvider
}

// @sk-task graceful-degradation#T4.1: NewFallbackStreamingLLMProvider public constructor (RQ-009, AC-006)
func NewFallbackStreamingLLMProvider(providers []StreamingLLMProvider, logger Logger, hooks Hooks) (*FallbackStreamingLLMProvider, error) {
	internal, err := resilience.NewFallbackStreamingLLM(providers, logger, hooks)
	if err != nil {
		return nil, err
	}
	return &FallbackStreamingLLMProvider{FallbackStreamingLLMProvider: internal}, nil
}

// @sk-task graceful-degradation#T4.1: FallbackUsageAwareLLMProvider public wrapper (RQ-010, AC-007)
type FallbackUsageAwareLLMProvider struct {
	*resilience.FallbackUsageAwareLLMProvider
}

// @sk-task graceful-degradation#T4.1: NewFallbackUsageAwareLLMProvider public constructor (RQ-010, AC-007)
func NewFallbackUsageAwareLLMProvider(providers []UsageAwareLLMProvider, logger Logger, hooks Hooks) (*FallbackUsageAwareLLMProvider, error) {
	internal, err := resilience.NewFallbackUsageAwareLLM(providers, logger, hooks)
	if err != nil {
		return nil, err
	}
	return &FallbackUsageAwareLLMProvider{FallbackUsageAwareLLMProvider: internal}, nil
}
