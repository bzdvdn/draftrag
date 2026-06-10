package application

import (
	"context"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task arch-quality-pass#T1.2: hookStart возвращает context из Hooks.StageStart (AC-001)
// @sk-task arch-quality-pass#T3.1: передаёт возвращённый из StageStart context c span (AC-001, AC-005)
func (p *Pipeline) hookStart(ctx context.Context, op string, stage domain.HookStage) context.Context {
	if p.hooks == nil {
		return ctx
	}
	return p.hooks.StageStart(ctx, domain.StageStartEvent{
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
