package vectorstore

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test contract-tests-stores#T2.2: MemoryStore registration for VectorStore contract
func TestContract_VectorStore(t *testing.T) {
	t.Run("memory", func(t *testing.T) {
		runVectorStoreContract(t, func() domain.VectorStore {
			return NewInMemoryStore()
		})
	})
}

// @sk-test contract-tests-stores#T3.2: MemoryStore full run 15 scenarios
func TestContract_VectorStoreWithFilters(t *testing.T) {
	t.Run("memory", func(t *testing.T) {
		runFilterContract(t, func() domain.VectorStore {
			return NewInMemoryStore()
		})
	})
}

func TestContract(t *testing.T) {
	t.Run("VectorStore/memory", func(t *testing.T) {
		runVectorStoreContract(t, func() domain.VectorStore {
			return NewInMemoryStore()
		})
	})
	t.Run("VectorStoreWithFilters/memory", func(t *testing.T) {
		runFilterContract(t, func() domain.VectorStore {
			return NewInMemoryStore()
		})
	})
}
