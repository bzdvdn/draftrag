package domain

import (
	"context"
	"errors"
)

// ErrMiddlewareAbort — sentinel для short-circuit в middleware.
// Middleware может вернуть эту ошибку, чтобы прервать pipeline без
// прохождения последующих middleware и downstream-стадий.
//
// @sk-task middleware-chain#T1.1: ErrMiddlewareAbort sentinel (AC-004)
var ErrMiddlewareAbort = errors.New("middleware aborted pipeline")

// Handler — обработчик стадии pipeline.
// Получает контекст и StageData, возвращает (возможно модифицированные) данные и ошибку.
//
// @sk-task middleware-chain#T1.1: Handler type (AC-001, AC-004)
type Handler func(ctx context.Context, data StageData) (StageData, error)

// Middleware — функциональный тип для обёртки Handler.
// Каждая middleware получает следующий Handler и должна вызвать его
// для продолжения цепочки. Может модифицировать StageData до/после next.
//
// @sk-task middleware-chain#T1.1: Middleware type (AC-001, AC-004)
type Middleware func(next Handler) Handler

// StageData — единая структура данных, передаваемая через middleware-цепочку.
// Поля заполняются в зависимости от стадии (Stage):
//   - chunking: Document
//   - embed:    Query (текст для эмбеддинга)
//   - search:   Query, Embedding, Chunks (результат)
//   - generate: Query, Chunks, Answer (результат)
//
// @sk-task middleware-chain#T1.1: StageData struct (AC-001, AC-003, AC-004)
type StageData struct {
	Stage     HookStage
	Operation string
	Query     string
	Document  Document
	Embedding []float64
	Chunks    []RetrievedChunk
	Answer    string
}
