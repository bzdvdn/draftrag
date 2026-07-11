package draftrag

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// @sk-test config-management#T2.4: TestLoadConfigMemoryOllama (AC-001)

func TestLoadConfigMemoryOllama(t *testing.T) {
	// @sk-test config-management#T2.4: YAML → Config for memory+ollama (AC-001)
	yamlContent := `
pipeline:
  default_top_k: 10
  mmr_enabled: true
store:
  type: memory
embedder:
  type: ollama
  ollama:
    model: nomic-embed-text
    base_url: http://localhost:11434
llm:
  type: ollama
  ollama:
    model: llama3
    base_url: http://localhost:11434
`
	path := writeTempYAML(t, yamlContent)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, 10, cfg.Pipeline.DefaultTopK)
	assert.True(t, cfg.Pipeline.MMREnabled)
	assert.Equal(t, "memory", cfg.Store.Type)
	assert.Equal(t, "ollama", cfg.Embedder.Type)
	assert.NotNil(t, cfg.Embedder.Ollama)
	assert.Equal(t, "nomic-embed-text", cfg.Embedder.Ollama.Model)
	assert.Equal(t, "http://localhost:11434", cfg.Embedder.Ollama.BaseURL)
	assert.Equal(t, "ollama", cfg.LLM.Type)
	assert.NotNil(t, cfg.LLM.Ollama)
	assert.Equal(t, "llama3", cfg.LLM.Ollama.Model)
}

// @sk-test config-management#T2.4: TestLoadConfigEnvOverride (AC-002)

func TestLoadConfigEnvOverride(t *testing.T) {
	// @sk-test config-management#T2.4: env переопределяет YAML-поле (AC-002)
	yamlContent := `
store:
  type: memory
embedder:
  type: ollama
  ollama:
    model: nomic-embed-text
llm:
  type: ollama
  ollama:
    model: llama3
    api_key: placeholder
`
	path := writeTempYAML(t, yamlContent)
	defer os.Remove(path)

	t.Setenv("DRAFTRAG_LLM_OLLAMA_API_KEY", "real-key")

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg.LLM.Ollama)
	assert.Equal(t, "real-key", cfg.LLM.Ollama.APIKey)
}

// @sk-test config-management#T2.4: TestLoadConfigEnvOnly (AC-007)

func TestLoadConfigEnvOnly(t *testing.T) {
	// @sk-test config-management#T2.4: только env, без YAML (AC-007)
	t.Setenv("DRAFTRAG_STORE_TYPE", "memory")
	t.Setenv("DRAFTRAG_EMBEDDER_TYPE", "ollama")
	t.Setenv("DRAFTRAG_EMBEDDER_OLLAMA_MODEL", "nomic-embed-text")
	t.Setenv("DRAFTRAG_LLM_TYPE", "ollama")
	t.Setenv("DRAFTRAG_LLM_OLLAMA_MODEL", "llama3")

	cfg, err := LoadConfig("")
	require.NoError(t, err)
	assert.Equal(t, "memory", cfg.Store.Type)
	assert.Equal(t, "ollama", cfg.Embedder.Type)
	require.NotNil(t, cfg.Embedder.Ollama)
	assert.Equal(t, "nomic-embed-text", cfg.Embedder.Ollama.Model)
	assert.Equal(t, "ollama", cfg.LLM.Type)
	require.NotNil(t, cfg.LLM.Ollama)
	assert.Equal(t, "llama3", cfg.LLM.Ollama.Model)
}

// @sk-test config-management#T2.4: TestLoadConfigUnknownKey (AC-003)

func TestLoadConfigUnknownKey(t *testing.T) {
	// @sk-test config-management#T2.4: неизвестный ключ → ErrUnknownConfigKey (AC-003)
	yamlContent := `
store:
  type: memory
  ttl: 3600
embedder:
  type: ollama
  ollama:
    model: test
llm:
  type: ollama
  ollama:
    model: test
`
	path := writeTempYAML(t, yamlContent)
	defer os.Remove(path)

	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownConfigKey)
}

// @sk-test config-management#T2.4: TestNewPipelineFromConfigMemoryOllama (AC-005)

func TestNewPipelineFromConfigMemoryOllama(t *testing.T) {
	// @sk-test config-management#T2.4: NewPipelineFromConfig не паникует (AC-005)
	cfg := Config{
		Store: StoreConfig{
			Type: "memory",
		},
		Embedder: EmbedderConfig{
			Type: "ollama",
			Ollama: &OllamaEmbedderConfig{
				Model:   "nomic-embed-text",
				BaseURL: "http://localhost:11434",
			},
		},
		LLM: LLMConfig{
			Type: "ollama",
			Ollama: &OllamaLLMConfig{
				Model:   "llama3",
				BaseURL: "http://localhost:11434",
			},
		},
	}

	p, err := NewPipelineFromConfig(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Query должен вернуть транспортную ошибку, не панику
	_, err = p.Query(context.Background(), "test")
	// Ошибка ожидаема (ollama недоступен), но не паника
	assert.Error(t, err)
}

// @sk-test config-management#T2.4: TestLoadConfigEnvOverridePrecedence (AC-002)

func TestLoadConfigEmptyPath(t *testing.T) {
	// @sk-test config-management#T2.4: пустой path = только env (AC-007)
	t.Setenv("DRAFTRAG_STORE_TYPE", "memory")
	t.Setenv("DRAFTRAG_EMBEDDER_TYPE", "ollama")
	t.Setenv("DRAFTRAG_EMBEDDER_OLLAMA_MODEL", "test-model")
	t.Setenv("DRAFTRAG_LLM_TYPE", "ollama")
	t.Setenv("DRAFTRAG_LLM_OLLAMA_MODEL", "test-llm")

	cfg, err := LoadConfigFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "memory", cfg.Store.Type)
	require.NotNil(t, cfg.Embedder.Ollama)
	assert.Equal(t, "test-model", cfg.Embedder.Ollama.Model)
	require.NotNil(t, cfg.LLM.Ollama)
	assert.Equal(t, "test-llm", cfg.LLM.Ollama.Model)
}

// @sk-test config-management#T3.4: TestNewPipelineFromConfigDispatch (AC-006)

func TestNewPipelineFromConfigDispatch(t *testing.T) {
	// @sk-test config-management#T3.4: memory — успех (AC-006)
	cfg := Config{
		Store: StoreConfig{Type: "memory"},
		Embedder: EmbedderConfig{
			Type:   "ollama",
			Ollama: &OllamaEmbedderConfig{Model: "test", BaseURL: "http://localhost:11434"},
		},
		LLM: LLMConfig{
			Type:   "ollama",
			Ollama: &OllamaLLMConfig{Model: "test", BaseURL: "http://localhost:11434"},
		},
	}
	p, err := NewPipelineFromConfig(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
}

// @sk-test config-management#T3.4: TestNewPipelineFromConfigStoreDispatch (AC-006)

func TestNewPipelineFromConfigMissingStoreType(t *testing.T) {
	// @sk-test config-management#T3.4: пустой store type → ErrMissingRequiredField (AC-004)
	cfg := Config{
		Store: StoreConfig{Type: ""},
	}
	_, err := NewPipelineFromConfig(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingRequiredField)
}

// @sk-test config-management#T3.4: TestNewPipelineFromConfigInvalidStoreType (AC-006)

func TestNewPipelineFromConfigInvalidStoreType(t *testing.T) {
	// @sk-test config-management#T3.4: неизвестный store type → ErrUnknownConfigKey (AC-006)
	cfg := Config{
		Store: StoreConfig{Type: "nonexistent"},
	}
	_, err := NewPipelineFromConfig(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownConfigKey)
}

// @sk-test config-management#T3.4: TestNewPipelineFromConfigMissingEmbedderModel (AC-004)

func TestNewPipelineFromConfigMissingEmbedderModel(t *testing.T) {
	// @sk-test config-management#T3.4: embedder без model → ErrMissingRequiredField (AC-004)
	cfg := Config{
		Store: StoreConfig{Type: "memory"},
		Embedder: EmbedderConfig{
			Type:   "ollama",
			Ollama: &OllamaEmbedderConfig{BaseURL: "http://localhost:11434"},
		},
		LLM: LLMConfig{
			Type:   "ollama",
			Ollama: &OllamaLLMConfig{Model: "test", BaseURL: "http://localhost:11434"},
		},
	}
	_, err := NewPipelineFromConfig(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingRequiredField)
}

// @sk-test config-management#T3.4: TestNewPipelineFromConfigPgvectorRequiresDB (AC-006)

func TestNewPipelineFromConfigPgvectorRequiresDB(t *testing.T) {
	// @sk-test config-management#T3.4: pgvector без *sql.DB → ErrMissingRequiredField (AC-006)
	cfg := Config{
		Store: StoreConfig{
			Type: "pgvector",
			Pgvector: &PgvectorStoreConfig{
				TableName:          "test",
				EmbeddingDimension: 384,
			},
		},
		Embedder: EmbedderConfig{
			Type:   "ollama",
			Ollama: &OllamaEmbedderConfig{Model: "test", BaseURL: "http://localhost:11434"},
		},
		LLM: LLMConfig{
			Type:   "ollama",
			Ollama: &OllamaLLMConfig{Model: "test", BaseURL: "http://localhost:11434"},
		},
	}
	_, err := NewPipelineFromConfig(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingRequiredField)
}

// @sk-test config-management#T4.1: TestLoadConfigEmptyYAML (AC-004, AC-003)

func TestLoadConfigEmptyYAML(t *testing.T) {
	// @sk-test config-management#T4.1: пустой YAML → zero values (не ошибка парсинга)
	yamlContent := ``
	path := writeTempYAML(t, yamlContent)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Empty(t, cfg.Store.Type)
	assert.Empty(t, cfg.Embedder.Type)
}

// @sk-test config-management#T4.1: TestLoadConfigInvalidYAML

func TestLoadConfigInvalidYAML(t *testing.T) {
	// @sk-test config-management#T4.1: некорректный YAML → ошибка парсинга
	yamlContent := `store: [invalid yaml`
	path := writeTempYAML(t, yamlContent)
	defer os.Remove(path)

	_, err := LoadConfig(path)
	require.Error(t, err)
	// Ошибка парсинга от yaml.v3, не ErrUnknownConfigKey
	assert.False(t, errors.Is(err, ErrUnknownConfigKey))
}

// @sk-test config-management#T4.1: TestEnvEmptyValueDoesNotOverride

func TestEnvEmptyValueDoesNotOverride(t *testing.T) {
	// @sk-test config-management#T4.1: пустая env-переменная не переопределяет YAML-значение
	yamlContent := `
store:
  type: memory
embedder:
  type: ollama
  ollama:
    model: original-model
llm:
  type: ollama
  ollama:
    model: original-llm
`
	path := writeTempYAML(t, yamlContent)
	defer os.Remove(path)

	t.Setenv("DRAFTRAG_EMBEDDER_OLLAMA_MODEL", "")

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg.Embedder.Ollama)
	assert.Equal(t, "original-model", cfg.Embedder.Ollama.Model)
}

// @sk-test config-management#T4.1: TestNewPipelineFromConfigResilienceNoop

func TestNewPipelineFromConfigResilienceNoop(t *testing.T) {
	// @sk-test config-management#T4.1: resilience-секция без стратегии не вызывает ошибку
	cfg := Config{
		Store:      StoreConfig{Type: "memory"},
		Embedder:   EmbedderConfig{Type: "ollama", Ollama: &OllamaEmbedderConfig{Model: "t", BaseURL: "http://localhost:11434"}},
		LLM:        LLMConfig{Type: "ollama", Ollama: &OllamaLLMConfig{Model: "t", BaseURL: "http://localhost:11434"}},
		Resilience: ResilienceConfig{
			// nil Retry и nil CircuitBreaker — no-op
		},
	}
	p, err := NewPipelineFromConfig(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}
