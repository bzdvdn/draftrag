package resilience

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func recordHookEvent(ctx context.Context, hooks domain.Hooks, stage domain.HookStage, operation string, attempt int, err error, rejected bool) {
	if hooks == nil {
		return
	}

	ev := domain.StageStartEvent{
		Operation: fmt.Sprintf("%s:attempt=%d", operation, attempt),
		Stage:     stage,
	}
	if rejected {
		ev.Operation = fmt.Sprintf("%s:rejected", operation)
	}

	hooks.StageStart(ctx, ev)

	// Для завершения используем нулевую длительность (event-based hooks).
	hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: ev.Operation,
		Stage:     stage,
		Duration:  0,
		Err:       err,
	})
}
