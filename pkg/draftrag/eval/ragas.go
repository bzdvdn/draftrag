package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

const faithfulnessPrompt = `You are evaluating the faithfulness of an answer given a context.
Analyze the answer, decompose it into individual factual claims, and check each claim against the provided context.
Return a JSON object with exactly this structure:
{
  "faithfulness_score": <float between 0 and 1>,
  "claims": ["<claim 1>", ...],
  "supported_claims": ["<supported claim 1>", ...],
  "unsupported_claims": ["<unsupported claim 1>", ...]
}
faithfulness_score = number_of_supported_claims / total_number_of_claims (0 if no claims)

Context:
---
%s
---

Answer:
---
%s
---

Return ONLY valid JSON.`

// ComputeFaithfulness вычисляет faithfulness score ответа относительно контекста через LLM-декомпозицию.
// @sk-task eval-ragas-metrics#T2.1: ComputeFaithfulness (AC-001)
func ComputeFaithfulness(ctx context.Context, answer, contextStr string, llmProvider draftrag.LLMProvider) (float64, error) {
	if llmProvider == nil {
		return 0, nil
	}
	if strings.TrimSpace(answer) == "" || strings.TrimSpace(contextStr) == "" {
		return 0, nil
	}

	prompt := fmt.Sprintf(faithfulnessPrompt, contextStr, answer)
	resp, err := llmProvider.Generate(ctx, "You are a faithful evaluator.", prompt)
	if err != nil {
		return 0, err
	}

	var result struct {
		FaithfulnessScore float64 `json:"faithfulness_score"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return 0, nil
	}

	if result.FaithfulnessScore < 0 {
		return 0, nil
	}
	if result.FaithfulnessScore > 1 {
		return 1, nil
	}
	return result.FaithfulnessScore, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	dot, normA, normB := 0.0, 0.0, 0.0
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ComputeContextRelevance вычисляет context relevance score вопроса относительно набора чанков контекста.
// @sk-task eval-ragas-metrics#T2.2: ComputeContextRelevance (AC-003)
func ComputeContextRelevance(ctx context.Context, question string, contextChunks []string, embedder draftrag.Embedder) (float64, error) {
	if embedder == nil {
		return 0, nil
	}
	if len(contextChunks) == 0 || strings.TrimSpace(question) == "" {
		return 0, nil
	}

	qEmb, err := embedder.Embed(ctx, question)
	if err != nil {
		return 0, err
	}

	sum := 0.0
	for _, chunk := range contextChunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		cEmb, err := embedder.Embed(ctx, chunk)
		if err != nil {
			return 0, err
		}
		sum += cosineSimilarity(qEmb, cEmb)
	}

	if len(contextChunks) == 0 {
		return 0, nil
	}
	return sum / float64(len(contextChunks)), nil
}

// ComputeAnswerRelevance вычисляет answer relevance score — семантическую близость между ответом и вопросом.
// @sk-task eval-ragas-metrics#T2.3: ComputeAnswerRelevance (AC-002)
func ComputeAnswerRelevance(ctx context.Context, question, answer string, embedder draftrag.Embedder) (float64, error) {
	if embedder == nil {
		return 0, nil
	}
	if strings.TrimSpace(answer) == "" || strings.TrimSpace(question) == "" {
		return 0, nil
	}

	qEmb, err := embedder.Embed(ctx, question)
	if err != nil {
		return 0, err
	}

	aEmb, err := embedder.Embed(ctx, answer)
	if err != nil {
		return 0, err
	}

	return cosineSimilarity(qEmb, aEmb), nil
}
