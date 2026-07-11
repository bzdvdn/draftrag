package decomposer

import (
	"context"
	"testing"
)

// @sk-test sub-query-decomposition#T4.1: TestRuleQueryDecomposer_SplitByИ (AC-003)
func TestRuleQueryDecomposer_SplitByИ(t *testing.T) {
	d := NewRuleQueryDecomposer()
	subs, err := d.Decompose(context.Background(), "что такое RAG и как работает retrieval")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 sub-queries, got %d: %v", len(subs), subs)
	}
	if subs[0] != "что такое RAG" {
		t.Fatalf("expected 'что такое RAG', got %q", subs[0])
	}
	if subs[1] != "как работает retrieval" {
		t.Fatalf("expected 'как работает retrieval', got %q", subs[1])
	}
}

// @sk-test sub-query-decomposition#T4.1: TestRuleQueryDecomposer_SplitByИли (AC-003)
func TestRuleQueryDecomposer_SplitByИли(t *testing.T) {
	d := NewRuleQueryDecomposer()
	subs, err := d.Decompose(context.Background(), "найди про кошек или про собак")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 sub-queries, got %d: %v", len(subs), subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestRuleQueryDecomposer_SplitByComma (AC-003)
func TestRuleQueryDecomposer_SplitByComma(t *testing.T) {
	d := NewRuleQueryDecomposer()
	subs, err := d.Decompose(context.Background(), "query1, query2, query3")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 3 {
		t.Fatalf("expected 3 sub-queries, got %d: %v", len(subs), subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestRuleQueryDecomposer_NoSplit (AC-003)
func TestRuleQueryDecomposer_NoSplit(t *testing.T) {
	d := NewRuleQueryDecomposer()
	subs, err := d.Decompose(context.Background(), "простой запрос без разделителей")
	if err != nil {
		t.Fatal(err)
	}
	if subs != nil {
		t.Fatalf("expected nil for single-query fallback, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestRuleQueryDecomposer_EmptyQuery (AC-003)
func TestRuleQueryDecomposer_EmptyQuery(t *testing.T) {
	d := NewRuleQueryDecomposer()
	subs, err := d.Decompose(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if subs != nil {
		t.Fatalf("expected nil for empty query, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestRuleQueryDecomposer_PicksBestSeparator (AC-003)
func TestRuleQueryDecomposer_PicksBestSeparator(t *testing.T) {
	d := NewRuleQueryDecomposer()
	// Должен выбрать " и " (даёт 2 части), а не ", " (даёт 1 часть)
	subs, err := d.Decompose(context.Background(), "кошки и собаки, птицы")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 sub-queries (split by ' и '), got %d: %v", len(subs), subs)
	}
}
