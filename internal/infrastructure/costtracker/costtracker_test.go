// @sk-task cost-tracking: T4.1 — unit-тесты CostTracker (AC-001..004, 006, 007)
package costtracker

import (
	"context"
	"sync"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// mockUsageLLM — mock, реализующий UsageAwareLLMProvider.
type mockUsageLLM struct {
	domain.LLMProvider
	usage domain.TokenUsage
	model string
}

func (m *mockUsageLLM) GenerateWithUsage(_ context.Context, _, _ string) (string, domain.TokenUsage, error) {
	return "mock answer", m.usage, nil
}

func (m *mockUsageLLM) ModelName() string { return m.model }

// mockPlainLLM — mock, реализующий только LLMProvider (без UsageAwareLLMProvider).
type mockPlainLLM struct{}

func (m *mockPlainLLM) Generate(_ context.Context, _, userMessage string) (string, error) {
	return "plain:" + userMessage, nil
}

func (m *mockPlainLLM) Health(_ context.Context) error { return nil }

// @sk-task cost-tracking: AC-001 — базовый подсчёт токенов
func TestCostTracker_BasicTokenCount(t *testing.T) {
	usage := domain.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	llm := &mockUsageLLM{usage: usage, model: "gpt-4o"}
	ct := NewCostTracker(llm, nil)

	_, err := ct.Generate(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}

	snap := ct.Snapshot()
	if snap.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", snap.TotalTokens)
	}
	if snap.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", snap.PromptTokens)
	}
	if snap.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", snap.CompletionTokens)
	}
	if snap.CallsCount != 1 {
		t.Errorf("CallsCount = %d, want 1", snap.CallsCount)
	}
}

// @sk-task cost-tracking: AC-002 — расчёт стоимости
func TestCostTracker_CostCalculation(t *testing.T) {
	pricing := map[string]domain.ModelPricing{
		"gpt-4o": {InputCostPer1K: 0.01, OutputCostPer1K: 0.03},
	}
	usage := domain.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	llm := &mockUsageLLM{usage: usage, model: "gpt-4o"}
	ct := NewCostTracker(llm, pricing)

	_, err := ct.Generate(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}

	expectedCost := (100.0/1000.0)*0.01 + (50.0/1000.0)*0.03 // 0.001 + 0.0015 = 0.0025
	snap := ct.Snapshot()
	if snap.TotalCost != expectedCost {
		t.Errorf("TotalCost = %f, want %f", snap.TotalCost, expectedCost)
	}
}

// @sk-task cost-tracking: AC-003 — потокобезопасность (race test)
func TestCostTracker_ConcurrentSafety(t *testing.T) {
	usage := domain.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	llm := &mockUsageLLM{usage: usage, model: "gpt-4o"}
	ct := NewCostTracker(llm, nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ct.Generate(context.Background(), "system", "user")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ct.Snapshot()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ct.Checkpoint()
		}()
	}
	wg.Wait()

	snap := ct.Snapshot()
	if snap.CallsCount != 10 {
		t.Errorf("CallsCount = %d, want 10", snap.CallsCount)
	}
	if snap.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", snap.TotalTokens)
	}
}

// @sk-task cost-tracking: AC-004 — Reset
func TestCostTracker_Reset(t *testing.T) {
	usage := domain.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	llm := &mockUsageLLM{usage: usage, model: "gpt-4o"}
	ct := NewCostTracker(llm, nil)

	_, _ = ct.Generate(context.Background(), "system", "user")
	ct.Reset()

	snap := ct.Snapshot()
	if snap.TotalTokens != 0 {
		t.Errorf("TotalTokens after Reset = %d, want 0", snap.TotalTokens)
	}
	if snap.TotalCost != 0 {
		t.Errorf("TotalCost after Reset = %f, want 0", snap.TotalCost)
	}
	if snap.CallsCount != 0 {
		t.Errorf("CallsCount after Reset = %d, want 0", snap.CallsCount)
	}
}

// @sk-task cost-tracking: AC-006 — graceful degradation (plain LLM без UsageAware)
func TestCostTracker_PlainLLM(t *testing.T) {
	llm := &mockPlainLLM{}
	ct := NewCostTracker(llm, nil)

	_, err := ct.Generate(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}

	snap := ct.Snapshot()
	if snap.CallsCount != 1 {
		t.Errorf("CallsCount = %d, want 1", snap.CallsCount)
	}
	if snap.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0 (no usage)", snap.TotalTokens)
	}
	if snap.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0 (no pricing)", snap.TotalCost)
	}
}

// @sk-task cost-tracking: AC-007 — Checkpoint + Diff
func TestCostTracker_CheckpointDiff(t *testing.T) {
	pricing := map[string]domain.ModelPricing{
		"gpt-4o": {InputCostPer1K: 0.01, OutputCostPer1K: 0.03},
	}
	usage := domain.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	llm := &mockUsageLLM{usage: usage, model: "gpt-4o"}
	ct := NewCostTracker(llm, pricing)

	// Первый вызов
	_, _ = ct.Generate(context.Background(), "system", "user")
	cp1 := ct.Checkpoint()

	// Второй вызов
	_, _ = ct.Generate(context.Background(), "system", "user2")
	cp2 := ct.Checkpoint()

	diff := domain.Diff(cp1, cp2)
	if diff.CallsCount != 1 {
		t.Errorf("Diff.CallsCount = %d, want 1", diff.CallsCount)
	}
	if diff.TotalTokens != 150 {
		t.Errorf("Diff.TotalTokens = %d, want 150", diff.TotalTokens)
	}
	expectedCost := (100.0/1000.0)*0.01 + (50.0/1000.0)*0.03
	if diff.TotalCost != expectedCost {
		t.Errorf("Diff.TotalCost = %f, want %f", diff.TotalCost, expectedCost)
	}
}

// @sk-task cost-tracking: Доп. тест — несколько вызовов с разными моделями
func TestCostTracker_MultipleCallsDifferentModels(t *testing.T) {
	pricing := map[string]domain.ModelPricing{
		"gpt-4o":      {InputCostPer1K: 0.01, OutputCostPer1K: 0.03},
		"gpt-4o-mini": {InputCostPer1K: 0.001, OutputCostPer1K: 0.002},
	}
	llm1 := &mockUsageLLM{
		usage: domain.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		model: "gpt-4o",
	}
	llm2 := &mockUsageLLM{
		usage: domain.TokenUsage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300},
		model: "gpt-4o-mini",
	}

	ct1 := NewCostTracker(llm1, pricing)
	ct2 := NewCostTracker(llm2, pricing)

	_, _ = ct1.Generate(context.Background(), "system", "user")
	_, _ = ct2.Generate(context.Background(), "system", "user")

	cost1 := (100.0/1000.0)*0.01 + (50.0/1000.0)*0.03
	cost2 := (200.0/1000.0)*0.001 + (100.0/1000.0)*0.002

	snap1 := ct1.Snapshot()
	if snap1.TotalCost != cost1 {
		t.Errorf("ct1 TotalCost = %f, want %f", snap1.TotalCost, cost1)
	}
	snap2 := ct2.Snapshot()
	if snap2.TotalCost != cost2 {
		t.Errorf("ct2 TotalCost = %f, want %f", snap2.TotalCost, cost2)
	}
}

// mockStreamingUsageLLM — mock, реализующий StreamingLLMProvider + UsageAwareStreamingLLMProvider.
type mockStreamingUsageLLM struct {
	mockUsageLLM
	streamTokens []string
}

func (m *mockStreamingUsageLLM) GenerateStream(_ context.Context, _, _ string) (<-chan string, error) {
	ch := make(chan string, len(m.streamTokens))
	for _, t := range m.streamTokens {
		ch <- t
	}
	close(ch)
	return ch, nil
}

func (m *mockStreamingUsageLLM) StreamUsage() (domain.TokenUsage, bool) {
	return m.usage, true
}

// @sk-task cost-tracking: T3.4 — streaming usage извлекается после закрытия канала (AC-005, RQ-006)
func TestCostTracker_GenerateStream_WithUsage(t *testing.T) {
	usage := domain.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	llm := &mockStreamingUsageLLM{
		mockUsageLLM: mockUsageLLM{usage: usage, model: "gpt-4o"},
		streamTokens: []string{"token1", "token2"},
	}
	ct := NewCostTracker(llm, nil)

	ch, err := ct.GenerateStream(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for token := range ch {
		got = append(got, token)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(got))
	}

	snap := ct.Snapshot()
	if snap.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", snap.TotalTokens)
	}
	if snap.CallsCount != 1 {
		t.Errorf("CallsCount = %d, want 1", snap.CallsCount)
	}
}

// @sk-task cost-tracking: T3.4 — streaming fallback для LLM без UsageAwareStreaming (AC-006)
func TestCostTracker_GenerateStream_NoUsage(t *testing.T) {
	llm := &mockStreamingLLMOnly{}
	ct := NewCostTracker(llm, nil)

	ch, err := ct.GenerateStream(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}

	for range ch {
	}

	snap := ct.Snapshot()
	if snap.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0 (no usage)", snap.TotalTokens)
	}
	if snap.CallsCount != 1 {
		t.Errorf("CallsCount = %d, want 1", snap.CallsCount)
	}
}

// mockStreamingLLMOnly — реализует только StreamingLLMProvider, без UsageAwareStreamingLLMProvider.
type mockStreamingLLMOnly struct{}

func (m *mockStreamingLLMOnly) Generate(_ context.Context, _, userMessage string) (string, error) {
	return "mock:" + userMessage, nil
}

func (m *mockStreamingLLMOnly) Health(_ context.Context) error { return nil }

func (m *mockStreamingLLMOnly) GenerateStream(_ context.Context, _, _ string) (<-chan string, error) {
	ch := make(chan string)
	close(ch)
	return ch, nil
}

// @sk-task cost-tracking: T3.4 — streaming с расчётом стоимости (AC-005, RQ-006)
func TestCostTracker_GenerateStream_WithCost(t *testing.T) {
	pricing := map[string]domain.ModelPricing{
		"gpt-4o": {InputCostPer1K: 0.01, OutputCostPer1K: 0.03},
	}
	usage := domain.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	llm := &mockStreamingUsageLLM{
		mockUsageLLM: mockUsageLLM{usage: usage, model: "gpt-4o"},
		streamTokens: []string{"hello"},
	}
	ct := NewCostTracker(llm, pricing)

	ch, err := ct.GenerateStream(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}

	expectedCost := (100.0/1000.0)*0.01 + (50.0/1000.0)*0.03
	snap := ct.Snapshot()
	if snap.TotalCost != expectedCost {
		t.Errorf("TotalCost = %f, want %f", snap.TotalCost, expectedCost)
	}
	if snap.CallsCount != 1 {
		t.Errorf("CallsCount = %d, want 1", snap.CallsCount)
	}
}
