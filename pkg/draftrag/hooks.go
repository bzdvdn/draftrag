package draftrag

import "github.com/bzdvdn/draftrag/internal/domain"

// HookStage описывает стадию выполнения pipeline, которую можно наблюдать через Hooks.
type HookStage = domain.HookStage

const (
	// HookStageChunking — разбиение документа на чанки (только при наличии Chunker).
	HookStageChunking = domain.HookStageChunking
	// HookStageEmbed — генерация embedding для текста.
	HookStageEmbed = domain.HookStageEmbed
	// HookStageSearch — поиск в VectorStore.
	HookStageSearch = domain.HookStageSearch
	// HookStageGenerate — генерация ответа LLM.
	HookStageGenerate = domain.HookStageGenerate
)

// StageStartEvent — событие начала стадии pipeline.
type StageStartEvent = domain.StageStartEvent

// StageEndEvent — событие завершения стадии pipeline.
type StageEndEvent = domain.StageEndEvent
