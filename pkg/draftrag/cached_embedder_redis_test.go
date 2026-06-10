package draftrag

import (
	"context"
	"sync"
	"testing"
	"time"
)

type mockRedisClient struct {
	mu       sync.Mutex
	store    map[string][]byte
	getCalls int
	setCalls int
}

func (m *mockRedisClient) GetBytes(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	if m.store == nil {
		return nil, nil
	}
	v, ok := m.store[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (m *mockRedisClient) SetBytes(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	if m.store == nil {
		m.store = make(map[string][]byte)
	}
	m.store[key] = value
	return nil
}

// @sk-test hardening-2026q2#T2.2: TestNewRedisCache_Constructs (AC-006)
func TestNewRedisCache_Constructs(t *testing.T) {
	ctx := context.Background()
	base := &countEmbedder{vec: []float64{1, 2, 3}}
	client := &mockRedisClient{}

	cached, err := NewRedisCache(ctx, base, client, time.Minute)
	if err != nil {
		t.Fatalf("NewRedisCache failed: %v", err)
	}
	if cached == nil {
		t.Fatal("expected non-nil CachedEmbedder")
	}
}

// @sk-test hardening-2026q2#T2.2: TestNewRedisCache_UsesRedis (AC-006)
func TestNewRedisCache_UsesRedis(t *testing.T) {
	ctx := context.Background()
	base := &countEmbedder{vec: []float64{0.5, 0.25}}
	client := &mockRedisClient{}

	cached, err := NewRedisCache(ctx, base, client, time.Minute)
	if err != nil {
		t.Fatalf("NewRedisCache failed: %v", err)
	}

	v, err := cached.Embed(ctx, "hello")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(v) != 2 || v[0] != 0.5 || v[1] != 0.25 {
		t.Fatalf("unexpected vector: %v", v)
	}

	if client.getCalls < 1 {
		t.Fatal("expected at least one Redis GetBytes call after Embed")
	}
	if client.setCalls < 1 {
		t.Fatal("expected at least one Redis SetBytes call after Embed")
	}
}
