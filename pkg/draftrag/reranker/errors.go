package reranker

import "errors"

// ErrInvalidRerankerConfig возвращается при невалидной конфигурации reranker'а.
//
// @sk-task reranker-cross-encoder#T1.2: sentinel для валидации опций (AC-003)
var ErrInvalidRerankerConfig = errors.New("invalid reranker config")
