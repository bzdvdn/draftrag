package application

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// runMiddleware выполняет цепочку middleware с указанным final-обработчиком.
// Если middlewares nil или пуст — вызывает final напрямую.
// Любая ошибка из middleware прерывает цепочку (fail-fast).
//
// @sk-task middleware-chain#T1.2: runMiddleware (AC-001, AC-002, AC-005)
// @sk-task middleware-chain#T4.3: ctx first param per revive lint (AC-005)
func runMiddleware(
	ctx context.Context,
	middlewares []domain.Middleware,
	data domain.StageData,
	final domain.Handler,
) (domain.StageData, error) {
	if len(middlewares) == 0 {
		return final(ctx, data)
	}

	handler := final
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler(ctx, data)
}

// execWithStageMiddleware выполняет стадию pipeline, оборачивая её middleware-цепочкой.
// Устанавливает Stage и Operation в StageData перед вызовом.
//
// @sk-task middleware-chain#T2.1: execWithStageMiddleware (AC-001, AC-002, AC-005)
func (p *Pipeline) execWithStageMiddleware(ctx context.Context, stage domain.HookStage, op string, data domain.StageData, stageFn func(context.Context, domain.StageData) (domain.StageData, error)) (domain.StageData, error) {
	data.Stage = stage
	data.Operation = op
	return runMiddleware(ctx, p.middleware, data, stageFn)
}
