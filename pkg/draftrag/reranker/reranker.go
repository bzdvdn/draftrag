// Package reranker provides reranking implementations for improving retrieval quality.
package reranker

import "github.com/bzdvdn/draftrag/internal/domain"

// Reranker — опциональный интерфейс для переранжирования результатов retrieval.
type Reranker = domain.Reranker

// BatchReranker — опциональное расширение Reranker для batch-режима.
//
// @sk-task reranker-cross-encoder#T1.1: re-export BatchReranker interface (AC-008)
type BatchReranker = domain.BatchReranker
