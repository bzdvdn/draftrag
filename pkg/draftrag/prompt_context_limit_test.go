package draftrag

// Компиляционная проверка доступности новых полей options.
var _ = PipelineOptions{
	MaxContextChars:  1,
	MaxContextChunks: 1,
}
