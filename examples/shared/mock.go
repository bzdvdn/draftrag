package shared

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// mockLLM — детерминированный mock-LLM, реализующий domain.LLMProvider.
// Echo-ответы с префиксом "[mock] " используются для grep-assertions в CI.
// Compile-time check, что mockLLM реализует domain.LLMProvider.
//
// @sk-task docs-and-examples#T1.2: mockLLM реализует domain.LLMProvider (DEC-005, AC-008).
type mockLLM struct {
	dim int
}

var _ domain.LLMProvider = (*mockLLM)(nil)

// @sk-task health-check-interface#T3.5: Health на mockLLM
func (m *mockLLM) Health(_ context.Context) error { return nil }

// @sk-task docs-and-examples#T1.2: Generate возвращает echo-ответ с префиксом "[mock] " (DEC-007, AC-008).
// Длинные вопросы обрезаются до 200 символов для читаемости в CI-логах.
func (m *mockLLM) Generate(_ context.Context, _, userMessage string) (string, error) {
	truncated := strings.TrimSpace(userMessage)
	if len(truncated) > 200 {
		truncated = truncated[:200] + "..."
	}
	return fmt.Sprintf("[mock] echo answer for: %s", truncated), nil
}

// mockEmbedder — детерминированный mock-эмбеддер, реализующий domain.Embedder.
// Возвращает 1536-мерный (или cfg.Dimension) вектор на основе SHA-256 хэша текста.
// Значения в диапазоне [-1, 1] — косинусное расстояние, не нулевое.
// Compile-time check, что mockEmbedder реализует domain.Embedder.
//
// @sk-task docs-and-examples#T1.2: mockEmbedder реализует domain.Embedder с детерминированным хэшем (DEC-005, DEC-007, AC-008).
type mockEmbedder struct {
	dim int
}

var _ domain.Embedder = (*mockEmbedder)(nil)

// @sk-task health-check-interface#T3.5: Health на mockEmbedder
func (m *mockEmbedder) Health(_ context.Context) error { return nil }

// @sk-task docs-and-examples#T1.2: Embed возвращает детерминированный dim-мерный вектор в [-1, 1] (DEC-007, AC-008).
// Алгоритм: SHA-256 от текста, разбивается на 4-байтные uint32, нормализуется в [-1, 1].
// Детерминизм гарантирует: mockEmbedder.Embed("foo") == mockEmbedder.Embed("foo") при том же dim.
func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	hash := sha256.Sum256([]byte(text))
	vec := make([]float64, m.dim)
	// SHA-256 = 32 байта = 8 uint32 значений. Циклически расширяем до нужной размерности.
	for i := 0; i < m.dim; i++ {
		b := hash[(i*4)%(len(hash)-3) : (i*4)%(len(hash)-3)+4]
		u := binary.BigEndian.Uint32(b)
		// Нормализуем uint32 в [-1, 1] через деление на 2^31.
		vec[i] = float64(int32(u)) / float64(1<<31)
	}
	return vec, nil
}

// @sk-task docs-and-examples#T1.2: NewMockLLM — фабрика для mockLLM (AC-008).
func NewMockLLM() domain.LLMProvider {
	return &mockLLM{}
}

// @sk-task docs-and-examples#T1.2: NewMockEmbedder — фабрика для mockEmbedder с заданной размерностью (AC-008).
func NewMockEmbedder(dim int) domain.Embedder {
	return &mockEmbedder{dim: dim}
}
