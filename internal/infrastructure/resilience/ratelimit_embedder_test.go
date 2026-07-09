package resilience

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmbedderForRL — минимальный mock embedder для rate limit тестов.
type MockEmbedderForRL struct {
	mock.Mock
	mu sync.Mutex
}

func (m *MockEmbedderForRL) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmbedderForRL) Embed(ctx context.Context, text string) ([]float64, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float64), args.Error(1)
}

// @sk-test rate-limiting-llm#T2.2: TestTokenBucketEmbedder_Parallel (AC-004)
func TestTokenBucketEmbedder_Parallel(t *testing.T) {
	mockEmb := new(MockEmbedderForRL)
	mockEmb.On("Embed", mock.Anything, mock.Anything).
		Return([]float64{0.1, 0.2}, nil).Times(10)

	p := NewTokenBucketEmbedder(mockEmb, 5, 5, nil)

	start := time.Now()

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Embed(context.Background(), "text")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	// 5 burst tokens = instant, remaining 5 at 5/sec = ~1s
	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond,
		"10 parallel calls at rate=5 should take >= 900ms")
	mockEmb.AssertExpectations(t)
}
