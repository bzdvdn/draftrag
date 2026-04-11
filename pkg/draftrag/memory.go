package draftrag

import "github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"

// NewInMemoryStore создаёт in-memory реализацию VectorStore.
//
// Подходит для прототипирования и тестирования — данные хранятся только в памяти
// и не сохраняются между перезапусками процесса.
func NewInMemoryStore() VectorStore {
	return vectorstore.NewInMemoryStore()
}
