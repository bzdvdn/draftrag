package domain

import (
	"context"
	"time"
)

// HookStage описывает стадию выполнения pipeline, которую можно наблюдать через Hooks.
type HookStage string

const (
	// HookStageChunking — разбиение документа на чанки (только при наличии Chunker).
	HookStageChunking HookStage = "chunking"
	// HookStageEmbed — генерация embedding для текста.
	HookStageEmbed HookStage = "embed"
	// HookStageSearch — поиск в VectorStore.
	HookStageSearch HookStage = "search"
	// HookStageGenerate — генерация ответа LLM.
	HookStageGenerate HookStage = "generate"
)

// StageStartEvent — событие начала стадии pipeline.
type StageStartEvent struct {
	Operation string
	Stage     HookStage
}

// StageEndEvent — событие завершения стадии pipeline.
type StageEndEvent struct {
	Operation string
	Stage     HookStage
	Duration  time.Duration
	Err       error
}

// Hooks — опциональный интерфейс наблюдаемости для pipeline стадий.
//
// Hooks вызываются синхронно: обработчики ДОЛЖНЫ быть лёгкими и быстрыми.
// При nil hooks pipeline работает как обычно (no-op).
//
// StageStart возвращает context.Context, который может содержать span или другие
// инструментационные данные, пробрасываемые в StageEnd.
//
// @sk-task arch-quality-pass#T1.2: StageStart returns context.Context (AC-001)
type Hooks interface {
	StageStart(ctx context.Context, ev StageStartEvent) context.Context
	StageEnd(ctx context.Context, ev StageEndEvent)
}
