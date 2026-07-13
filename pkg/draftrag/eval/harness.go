// Package eval provides a small evaluation harness for measuring retrieval quality (Hit@K, MRR, etc).
package eval

import (
	"context"
	"errors"
	"strings"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// RetrievalRunner — минимальный интерфейс для eval harness.
// Pipeline удовлетворяет этому интерфейсу напрямую.
type RetrievalRunner interface {
	Retrieve(ctx context.Context, question string, topK int) (draftrag.RetrievalResult, error)
}

// Options задаёт параметры запуска harness.
type Options struct {
	// DefaultTopK — topK по умолчанию, если в кейсе TopK=0.
	// Если 0, используется 5.
	DefaultTopK int
	// @sk-task T2.3: Добавить флаг EnableNDCG для включения вычисления NDCG (AC-003)
	EnableNDCG bool
	// @sk-task T2.3: Добавить флаг EnablePrecision для включения вычисления Precision (AC-003)
	EnablePrecision bool
	// @sk-task T2.3: Добавить флаг EnableRecall для включения вычисления Recall (AC-003)
	EnableRecall bool
	// @sk-task eval-ragas-metrics#T1.1: Добавить флаг EnableFaithfulness для RAGAS-метрик (DM)
	EnableFaithfulness bool
	// @sk-task eval-ragas-metrics#T1.1: Добавить флаг EnableAnswerRelevance для RAGAS-метрик (DM)
	EnableAnswerRelevance bool
	// @sk-task eval-ragas-metrics#T1.1: Добавить флаг EnableContextRelevance для RAGAS-метрик (DM)
	EnableContextRelevance bool
}

// Run прогоняет датасет кейсов и возвращает отчёт с базовыми retrieval-метриками.
// @sk-task arch-generics#T4.1: nil context guard вместо panic (AC-002)
func Run(ctx context.Context, runner RetrievalRunner, cases []Case, opts Options) (Report, error) {
	if ctx == nil {
		return Report{}, draftrag.ErrNilContext
	}
	if runner == nil {
		return Report{}, errors.New("nil runner")
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}

	defaultTopK := 5
	if opts.DefaultTopK < 0 {
		return Report{}, errors.New("DefaultTopK must be >= 0")
	}
	if opts.DefaultTopK > 0 {
		defaultTopK = opts.DefaultTopK
	}

	results := make([]CaseResult, 0, len(cases))

	for i, c := range cases {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		if c.Question == "" {
			return Report{}, errors.New("case question is empty")
		}

		// @sk-task T2.5: Валидация ExpectedParentIDs на пустые строки после нормализации (AC-005)
		for _, id := range c.ExpectedParentIDs {
			if normalizeID(id) == "" {
				return Report{}, errors.New("case ExpectedParentIDs contains empty string after normalization")
			}
		}

		topK := defaultTopK
		if c.TopK < 0 {
			return Report{}, errors.New("case TopK must be >= 0")
		}
		if c.TopK > 0 {
			topK = c.TopK
		}

		retrieval, err := runner.Retrieve(ctx, c.Question, topK)
		if err != nil {
			return Report{}, err
		}

		retrievedParentIDs := make([]string, 0, len(retrieval.Chunks))
		for _, rc := range retrieval.Chunks {
			retrievedParentIDs = append(retrievedParentIDs, rc.Chunk.ParentID)
		}

		rank := rankByParentID(c.ExpectedParentIDs, retrievedParentIDs)
		found := rank > 0

		caseID := c.ID
		if caseID == "" {
			caseID = "case-" + itoa(i+1)
		}

		results = append(results, CaseResult{
			CaseID:             caseID,
			Found:              found,
			Rank:               rank,
			RetrievedParentIDs: retrievedParentIDs,
		})
	}

	// @sk-task T2.5: Передача opts в computeMetrics для условного вычисления метрик (AC-003)
	metrics := computeMetrics(cases, results, opts)
	return Report{Metrics: metrics, Cases: results}, nil
}

// RunWithAnswer прогоняет датасет кейсов с генерацией ответа LLM и вычислением RAGAS-метрик.
// @sk-task eval-ragas-metrics#T3.1: RunWithAnswer (AC-004)
//
//nolint:gocyclo // complexity is inherent to multi-metric evaluation
func RunWithAnswer(ctx context.Context, runner RetrievalRunner, llm draftrag.LLMProvider, embedder draftrag.Embedder, cases []Case, opts Options) (Report, error) {
	if ctx == nil {
		return Report{}, draftrag.ErrNilContext
	}
	if runner == nil {
		return Report{}, errors.New("nil runner")
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}

	defaultTopK := 5
	if opts.DefaultTopK < 0 {
		return Report{}, errors.New("DefaultTopK must be >= 0")
	}
	if opts.DefaultTopK > 0 {
		defaultTopK = opts.DefaultTopK
	}

	results := make([]CaseResult, 0, len(cases))
	faithfulnessSum := 0.0
	answerRelSum := 0.0
	contextRelSum := 0.0
	ragasCount := 0

	for i, c := range cases {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		if c.Question == "" {
			return Report{}, errors.New("case question is empty")
		}

		for _, id := range c.ExpectedParentIDs {
			if normalizeID(id) == "" {
				return Report{}, errors.New("case ExpectedParentIDs contains empty string after normalization")
			}
		}

		topK := defaultTopK
		if c.TopK < 0 {
			return Report{}, errors.New("case TopK must be >= 0")
		}
		if c.TopK > 0 {
			topK = c.TopK
		}

		retrieval, err := runner.Retrieve(ctx, c.Question, topK)
		if err != nil {
			return Report{}, err
		}

		retrievedParentIDs := make([]string, 0, len(retrieval.Chunks))
		contextChunks := make([]string, 0, len(retrieval.Chunks))
		for _, rc := range retrieval.Chunks {
			retrievedParentIDs = append(retrievedParentIDs, rc.Chunk.ParentID)
			contextChunks = append(contextChunks, rc.Chunk.Content)
		}

		rank := rankByParentID(c.ExpectedParentIDs, retrievedParentIDs)
		found := rank > 0

		caseID := c.ID
		if caseID == "" {
			caseID = "case-" + itoa(i+1)
		}

		results = append(results, CaseResult{
			CaseID:             caseID,
			Found:              found,
			Rank:               rank,
			RetrievedParentIDs: retrievedParentIDs,
		})

		answer := c.ExpectedAnswer
		if answer == "" && llm != nil && len(contextChunks) > 0 {
			contextStr := strings.Join(contextChunks, "\n\n")
			answer, err = llm.Generate(ctx, "You are a helpful assistant. Answer based on the provided context.",
				"Context:\n"+contextStr+"\n\nQuestion: "+c.Question)
			if err != nil {
				return Report{}, err
			}
		}

		if answer == "" {
			continue
		}
		ragasCount++

		if opts.EnableFaithfulness && llm != nil {
			contextStr := strings.Join(contextChunks, "\n\n")
			score, err := ComputeFaithfulness(ctx, answer, contextStr, llm)
			if err != nil {
				return Report{}, err
			}
			faithfulnessSum += score
		}

		if opts.EnableAnswerRelevance && embedder != nil {
			score, err := ComputeAnswerRelevance(ctx, c.Question, answer, embedder)
			if err != nil {
				return Report{}, err
			}
			answerRelSum += score
		}

		if opts.EnableContextRelevance && embedder != nil && len(contextChunks) > 0 {
			score, err := ComputeContextRelevance(ctx, c.Question, contextChunks, embedder)
			if err != nil {
				return Report{}, err
			}
			contextRelSum += score
		}
	}

	metrics := computeMetrics(cases, results, opts)
	if opts.EnableFaithfulness && ragasCount > 0 {
		metrics.Faithfulness = faithfulnessSum / float64(ragasCount)
	}
	if opts.EnableAnswerRelevance && ragasCount > 0 {
		metrics.AnswerRelevance = answerRelSum / float64(ragasCount)
	}
	if opts.EnableContextRelevance && ragasCount > 0 {
		metrics.ContextRelevance = contextRelSum / float64(ragasCount)
	}

	return Report{Metrics: metrics, Cases: results}, nil
}

func itoa(n int) string {
	// В этой фиче нам не критичен perf; держим зависимость минимальной.
	// Используем стандартный подход через преобразование rune.
	if n == 0 {
		return "0"
	}

	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
