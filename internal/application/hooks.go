package application

import (
	"context"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) hookStart(ctx context.Context, op string, stage domain.HookStage) {
	if p.hooks == nil {
		return
	}
	p.hooks.StageStart(ctx, domain.StageStartEvent{
		Operation: op,
		Stage:     stage,
	})
}

func (p *Pipeline) hookEnd(ctx context.Context, op string, stage domain.HookStage, started time.Time, err error) {
	if p.hooks == nil {
		return
	}
	p.hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: op,
		Stage:     stage,
		Duration:  time.Since(started),
		Err:       err,
	})
}
