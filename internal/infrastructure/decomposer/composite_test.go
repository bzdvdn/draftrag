package decomposer

import (
	"context"
	"errors"
	"testing"
)

type mockDecomposer struct {
	subs []string
	err  error
}

func (m *mockDecomposer) Decompose(_ context.Context, _ string) ([]string, error) {
	return m.subs, m.err
}

// @sk-test sub-query-decomposition#T4.1: TestCompositeDecomposer_PrimarySuccess (AC-005)
func TestCompositeDecomposer_PrimarySuccess(t *testing.T) {
	primary := &mockDecomposer{subs: []string{"q1", "q2"}}
	secondary := &mockDecomposer{subs: []string{"fallback"}}

	d, err := NewCompositeDecomposer(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 || subs[0] != "q1" {
		t.Fatalf("expected primary results, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestCompositeDecomposer_PrimaryError_FallbackToSecondary (AC-005)
func TestCompositeDecomposer_PrimaryError_FallbackToSecondary(t *testing.T) {
	primary := &mockDecomposer{err: errors.New("primary failed")}
	secondary := &mockDecomposer{subs: []string{"fallback1", "fallback2"}}

	d, err := NewCompositeDecomposer(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 || subs[0] != "fallback1" {
		t.Fatalf("expected secondary results, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestCompositeDecomposer_PrimaryEmpty_FallbackToSecondary (AC-005)
func TestCompositeDecomposer_PrimaryEmpty_FallbackToSecondary(t *testing.T) {
	primary := &mockDecomposer{subs: nil}
	secondary := &mockDecomposer{subs: []string{"fallback"}}

	d, err := NewCompositeDecomposer(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0] != "fallback" {
		t.Fatalf("expected secondary results, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestCompositeDecomposer_BothFail_ReturnsNil (AC-005)
func TestCompositeDecomposer_BothFail_ReturnsNil(t *testing.T) {
	primary := &mockDecomposer{err: errors.New("primary failed")}
	secondary := &mockDecomposer{err: errors.New("secondary failed")}

	d, err := NewCompositeDecomposer(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if subs != nil {
		t.Fatalf("expected nil for full fallback, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestNewCompositeDecomposer_NilPrimary (AC-005)
func TestNewCompositeDecomposer_NilPrimary(t *testing.T) {
	_, err := NewCompositeDecomposer(nil, &mockDecomposer{})
	if err == nil {
		t.Fatal("expected error for nil primary")
	}
}

// @sk-test sub-query-decomposition#T4.1: TestNewCompositeDecomposer_NilSecondary (AC-005)
func TestNewCompositeDecomposer_NilSecondary(t *testing.T) {
	_, err := NewCompositeDecomposer(&mockDecomposer{}, nil)
	if err == nil {
		t.Fatal("expected error for nil secondary")
	}
}

// @sk-test sub-query-decomposition#T4.1: TestCompositeDecomposer_PrimaryEmptyStrings_Fallback (AC-005)
func TestCompositeDecomposer_PrimaryEmptyStrings_Fallback(t *testing.T) {
	primary := &mockDecomposer{subs: []string{}}
	secondary := &mockDecomposer{subs: []string{"fallback"}}

	d, err := NewCompositeDecomposer(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0] != "fallback" {
		t.Fatalf("expected secondary results for empty string slice, got %v", subs)
	}
}
