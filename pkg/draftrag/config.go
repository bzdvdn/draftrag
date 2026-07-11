package draftrag

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// @sk-task config-management#T1.1: Config struct (AC-001, RQ-001, DEC-001)

// Config — единая конфигурация для создания Pipeline.
type Config struct {
	Pipeline     PipelineConfig     `yaml:"pipeline"`
	Store        StoreConfig        `yaml:"store"`
	Embedder     EmbedderConfig     `yaml:"embedder"`
	LLM          LLMConfig          `yaml:"llm"`
	Chunker      ChunkerConfig      `yaml:"chunker,omitempty"`
	Reranker     RerankerConfig     `yaml:"reranker,omitempty"`
	Resilience   ResilienceConfig   `yaml:"resilience,omitempty"`
	CostTracking CostTrackingConfig `yaml:"cost_tracking,omitempty"`
}

// PipelineConfig — общие настройки Pipeline.
type PipelineConfig struct {
	DefaultTopK                  int     `yaml:"default_top_k"`
	SystemPrompt                 string  `yaml:"system_prompt"`
	MaxContextChars              int     `yaml:"max_context_chars"`
	MaxContextChunks             int     `yaml:"max_context_chunks"`
	DedupByParentID              bool    `yaml:"dedup_by_parent_id"`
	MMREnabled                   bool    `yaml:"mmr_enabled"`
	MMRLambda                    float64 `yaml:"mmr_lambda"`
	MMRCandidatePool             int     `yaml:"mmr_candidate_pool"`
	IndexConcurrency             int     `yaml:"index_concurrency"`
	IndexBatchRateLimit          int     `yaml:"index_batch_rate_limit"`
	IndexBatchRateLimitPerWorker bool    `yaml:"index_batch_rate_limit_per_worker"`
	StreamBufferSize             int     `yaml:"stream_buffer_size"`
}

// StoreConfig — настройки векторного хранилища.
type StoreConfig struct {
	Type     string               `yaml:"type"`
	Memory   *MemoryStoreConfig   `yaml:"memory,omitempty"`
	Pgvector *PgvectorStoreConfig `yaml:"pgvector,omitempty"`
	Qdrant   *QdrantStoreConfig   `yaml:"qdrant,omitempty"`
	ChromaDB *ChromaDBStoreConfig `yaml:"chromadb,omitempty"`
	Weaviate *WeaviateStoreConfig `yaml:"weaviate,omitempty"`
	Milvus   *MilvusStoreConfig   `yaml:"milvus,omitempty"`
}

// MemoryStoreConfig — настройки in-memory хранилища.
type MemoryStoreConfig struct{}

// PgvectorStoreConfig — настройки pgvector.
type PgvectorStoreConfig struct {
	TableName          string `yaml:"table_name"`
	EmbeddingDimension int    `yaml:"embedding_dimension"`
	CreateExtension    bool   `yaml:"create_extension"`
	IndexMethod        string `yaml:"index_method"`
	Lists              int    `yaml:"lists"`
}

// QdrantStoreConfig — настройки Qdrant.
type QdrantStoreConfig struct {
	URL        string        `yaml:"url"`
	Collection string        `yaml:"collection"`
	Dimension  int           `yaml:"dimension"`
	Timeout    time.Duration `yaml:"timeout"`
}

// ChromaDBStoreConfig — настройки ChromaDB.
type ChromaDBStoreConfig struct {
	BaseURL    string `yaml:"base_url"`
	Collection string `yaml:"collection"`
	Dimension  int    `yaml:"dimension"`
	AuthToken  string `yaml:"auth_token,omitempty"`
}

// WeaviateStoreConfig — настройки Weaviate.
type WeaviateStoreConfig struct {
	Host       string `yaml:"host"`
	Scheme     string `yaml:"scheme"`
	Collection string `yaml:"collection"`
	APIKey     string `yaml:"api_key"`
}

// MilvusStoreConfig — настройки Milvus.
type MilvusStoreConfig struct {
	Address    string `yaml:"address"`
	Collection string `yaml:"collection"`
	Dimension  int    `yaml:"dimension"`
	User       string `yaml:"user,omitempty"`
	Password   string `yaml:"password,omitempty"`
}

// EmbedderConfig — настройки эмбеддера.
type EmbedderConfig struct {
	Type             string                          `yaml:"type"`
	Ollama           *OllamaEmbedderConfig           `yaml:"ollama,omitempty"`
	OpenAICompatible *OpenAICompatibleEmbedderConfig `yaml:"openai_compatible,omitempty"`
	Mistral          *MistralEmbedderConfig          `yaml:"mistral,omitempty"`
}

// OllamaEmbedderConfig — настройки Ollama embedder.
type OllamaEmbedderConfig struct {
	BaseURL string        `yaml:"base_url"`
	Model   string        `yaml:"model"`
	APIKey  string        `yaml:"api_key"`
	Timeout time.Duration `yaml:"timeout"`
}

// OpenAICompatibleEmbedderConfig — настройки OpenAI-совместимого embedder.
type OpenAICompatibleEmbedderConfig struct {
	BaseURL string        `yaml:"base_url"`
	Model   string        `yaml:"model"`
	APIKey  string        `yaml:"api_key"`
	Timeout time.Duration `yaml:"timeout"`
}

// MistralEmbedderConfig — настройки Mistral embedder.
type MistralEmbedderConfig struct {
	APIKey  string        `yaml:"api_key"`
	Model   string        `yaml:"model"`
	Timeout time.Duration `yaml:"timeout"`
}

// LLMConfig — настройки LLM-провайдера.
type LLMConfig struct {
	Type             string                     `yaml:"type"`
	Ollama           *OllamaLLMConfig           `yaml:"ollama,omitempty"`
	OpenAICompatible *OpenAICompatibleLLMConfig `yaml:"openai_compatible,omitempty"`
	Anthropic        *AnthropicLLMConfig        `yaml:"anthropic,omitempty"`
	DeepSeek         *DeepSeekLLMConfig         `yaml:"deepseek,omitempty"`
	Mistral          *MistralLLMConfig          `yaml:"mistral,omitempty"`
}

// OllamaLLMConfig — настройки Ollama LLM.
type OllamaLLMConfig struct {
	BaseURL string        `yaml:"base_url"`
	Model   string        `yaml:"model"`
	APIKey  string        `yaml:"api_key"`
	Timeout time.Duration `yaml:"timeout"`
}

// OpenAICompatibleLLMConfig — настройки OpenAI-совместимого LLM.
type OpenAICompatibleLLMConfig struct {
	BaseURL string        `yaml:"base_url"`
	Model   string        `yaml:"model"`
	APIKey  string        `yaml:"api_key"`
	Timeout time.Duration `yaml:"timeout"`
}

// AnthropicLLMConfig — настройки Anthropic LLM.
type AnthropicLLMConfig struct {
	BaseURL          string   `yaml:"base_url"`
	APIKey           string   `yaml:"api_key"`
	Model            string   `yaml:"model"`
	AnthropicVersion string   `yaml:"anthropic_version"`
	Temperature      *float64 `yaml:"temperature,omitempty"`
	MaxTokens        *int     `yaml:"max_tokens,omitempty"`
}

// DeepSeekLLMConfig — настройки DeepSeek LLM.
type DeepSeekLLMConfig struct {
	BaseURL string        `yaml:"base_url"`
	APIKey  string        `yaml:"api_key"`
	Model   string        `yaml:"model"`
	Timeout time.Duration `yaml:"timeout"`
}

// MistralLLMConfig — настройки Mistral LLM.
type MistralLLMConfig struct {
	BaseURL string        `yaml:"base_url"`
	APIKey  string        `yaml:"api_key"`
	Model   string        `yaml:"model"`
	Timeout time.Duration `yaml:"timeout"`
}

// ChunkerConfig — настройки чанкера.
type ChunkerConfig struct {
	Type     string                 `yaml:"type"`
	Basic    *BasicChunkerConfig    `yaml:"basic,omitempty"`
	Semantic *SemanticChunkerConfig `yaml:"semantic,omitempty"`
}

// BasicChunkerConfig — настройки BasicChunker.
type BasicChunkerConfig struct {
	ChunkSize    int `yaml:"chunk_size"`
	ChunkOverlap int `yaml:"chunk_overlap"`
}

// @sk-task chunker-semantic#T3.1: YAML semantic chunker config (AC-009)
type SemanticChunkerConfig struct {
	SimilarityThreshold float64 `yaml:"threshold"`
	MinChunkSize        int     `yaml:"min_chunk_size"`
	MaxChunkSize        int     `yaml:"max_chunk_size"`
}

// RerankerConfig — настройки reranker.
type RerankerConfig struct {
	Type string             `yaml:"type"`
	LLM  *LLMRerankerConfig `yaml:"llm,omitempty"`
}

// LLMRerankerConfig — настройки LLM-реранкера.
type LLMRerankerConfig struct {
	BatchSize  int `yaml:"batch_size"`
	MaxRetries int `yaml:"max_retries"`
}

// ResilienceConfig — настройки resilience (retry, circuit breaker).
type ResilienceConfig struct {
	Retry          *RetryConfig          `yaml:"retry,omitempty"`
	CircuitBreaker *CircuitBreakerConfig `yaml:"circuit_breaker,omitempty"`
}

// RetryConfig — настройки retry.
type RetryConfig struct {
	MaxAttempts int           `yaml:"max_attempts"`
	BaseDelay   time.Duration `yaml:"base_delay"`
	MaxDelay    time.Duration `yaml:"max_delay"`
}

// CircuitBreakerConfig — настройки circuit breaker.
type CircuitBreakerConfig struct {
	MaxFailures int           `yaml:"max_failures"`
	ResetAfter  time.Duration `yaml:"reset_after"`
}

// CostTrackingConfig — настройки отслеживания стоимости.
type CostTrackingConfig struct {
	Enabled bool `yaml:"enabled"`
}

// ExternalDeps содержит runtime-зависимости, которые не могут быть сериализованы в YAML.
type ExternalDeps struct {
	DB         *sql.DB
	HTTPClient *http.Client
}

// @sk-task config-management#T2.1: LoadConfig (AC-001, AC-003)

// LoadConfig загружает конфигурацию из YAML-файла и применяет env-оверрайды.
// При пустом path Config заполяется только из переменных окружения.
func LoadConfig(path string) (Config, error) {
	var cfg Config

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config: %w", err)
		}

		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&cfg); err != nil {
			var typeErr *yaml.TypeError
			if errors.As(err, &typeErr) {
				return cfg, fmt.Errorf("%w: %v", ErrUnknownConfigKey, typeErr)
			}
			// io.EOF от пустого файла — не ошибка, остаются zero-значения
			if errors.Is(err, io.EOF) {
				return cfg, nil
			}
			return cfg, fmt.Errorf("parse config: %w", err)
		}
	}

	applyEnvOverrides(&cfg, "DRAFTRAG")
	return cfg, nil
}

// @sk-task config-management#T2.2: LoadConfigFromEnv (AC-007)

// LoadConfigFromEnv загружает конфигурацию только из переменных окружения.
func LoadConfigFromEnv() (Config, error) {
	return LoadConfig("")
}

// applyEnvOverrides рекурсивно обходит struct и переопределяет поля
// из переменных окружения с префиксом DRAFTRAG_.
func applyEnvOverrides(v interface{}, prefix string) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldVal := rv.Field(i)

		if !fieldVal.CanSet() && fieldVal.Kind() != reflect.Struct && fieldVal.Kind() != reflect.Ptr {
			continue
		}

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}
		name := strings.Split(yamlTag, ",")[0]
		envKey := prefix + "_" + strings.ToUpper(strings.ReplaceAll(name, ".", "_"))

		switch fieldVal.Kind() {
		case reflect.Struct:
			if fieldVal.Addr().CanInterface() {
				applyEnvOverrides(fieldVal.Addr().Interface(), envKey)
			}
		case reflect.Ptr:
			if field.Type.Elem().Kind() == reflect.Struct {
				if fieldVal.IsNil() && envVarExistsWithPrefix(envKey) {
					fieldVal.Set(reflect.New(field.Type.Elem()))
				}
				if !fieldVal.IsNil() {
					applyEnvOverrides(fieldVal.Interface(), envKey)
				}
			}
			continue
		default:
			setFieldFromEnv(fieldVal, envKey)
		}
	}
}

func setFieldFromEnv(fieldVal reflect.Value, envKey string) {
	val, ok := os.LookupEnv(envKey)
	if !ok || val == "" {
		return
	}

	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := parseInt(val)
		if err == nil {
			fieldVal.SetInt(v)
		}
	case reflect.Float32, reflect.Float64:
		v, err := parseFloat(val)
		if err == nil {
			fieldVal.SetFloat(v)
		}
	case reflect.Bool:
		fieldVal.SetBool(val == "true" || val == "1" || val == "yes")
	}
}

// envVarExistsWithPrefix проверяет, есть ли хотя бы одна переменная окружения,
// начинающаяся с указанного префикса.
func envVarExistsWithPrefix(prefix string) bool {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix+"_") || strings.HasPrefix(env, prefix+"=") {
			return true
		}
	}
	return false
}

func parseInt(s string) (int64, error) {
	var v int64
	for _, c := range s {
		if c < '0' || c > '9' {
			if c == '-' && v == 0 {
				continue
			}
			return 0, fmt.Errorf("not an integer: %s", s)
		}
		v = v*10 + int64(c-'0')
	}
	return v, nil
}

func parseFloat(s string) (float64, error) {
	var v float64
	var frac bool
	var fracMul float64 = 0.1
	for _, c := range s {
		if c == '.' && !frac {
			frac = true
			continue
		}
		if c < '0' || c > '9' {
			if c == '-' && v == 0 && !frac {
				continue
			}
			return 0, fmt.Errorf("not a float: %s", s)
		}
		if frac {
			v += float64(c-'0') * fracMul
			fracMul *= 0.1
		} else {
			v = v*10 + float64(c-'0')
		}
	}
	return v, nil
}

// @sk-task config-management#T2.3: NewPipelineFromConfig (AC-005)

// NewPipelineFromConfig создаёт Pipeline из Config.
// deps — опциональные runtime-зависимости для store-типов, требующих
// внешнего подключения (pgvector).
func NewPipelineFromConfig(ctx context.Context, cfg Config, deps ...ExternalDeps) (*Pipeline, error) {
	store, err := newStoreFromConfig(cfg.Store, deps)
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}

	embedder, err := newEmbedderFromConfig(cfg.Embedder)
	if err != nil {
		return nil, fmt.Errorf("embedder: %w", err)
	}

	llmProvider, err := newLLMFromConfig(cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	opts := PipelineOptions{
		DefaultTopK:                  cfg.Pipeline.DefaultTopK,
		SystemPrompt:                 cfg.Pipeline.SystemPrompt,
		MaxContextChars:              cfg.Pipeline.MaxContextChars,
		MaxContextChunks:             cfg.Pipeline.MaxContextChunks,
		DedupByParentID:              cfg.Pipeline.DedupByParentID,
		MMREnabled:                   cfg.Pipeline.MMREnabled,
		MMRLambda:                    cfg.Pipeline.MMRLambda,
		MMRCandidatePool:             cfg.Pipeline.MMRCandidatePool,
		IndexConcurrency:             cfg.Pipeline.IndexConcurrency,
		IndexBatchRateLimit:          cfg.Pipeline.IndexBatchRateLimit,
		IndexBatchRateLimitPerWorker: cfg.Pipeline.IndexBatchRateLimitPerWorker,
		StreamBufferSize:             cfg.Pipeline.StreamBufferSize,
	}

	if cfg.Chunker.Type == "basic" && cfg.Chunker.Basic != nil {
		opts.Chunker = NewBasicChunker(BasicChunkerOptions{
			ChunkSize: cfg.Chunker.Basic.ChunkSize,
			Overlap:   cfg.Chunker.Basic.ChunkOverlap,
		})
	}

	if cfg.Chunker.Type == "semantic" && cfg.Chunker.Semantic != nil {
		sc, err := NewSemanticChunker(SemanticChunkerOptions{
			Embedder:            embedder,
			SimilarityThreshold: cfg.Chunker.Semantic.SimilarityThreshold,
			MinChunkSize:        cfg.Chunker.Semantic.MinChunkSize,
			MaxChunkSize:        cfg.Chunker.Semantic.MaxChunkSize,
		})
		if err != nil {
			return nil, fmt.Errorf("semantic chunker: %w", err)
		}
		opts.Chunker = sc
	}

	if cfg.Reranker.Type == "llm" && cfg.Reranker.LLM != nil {
		var rerankerOpts []LLMRerankerOption
		if cfg.Reranker.LLM.BatchSize > 0 {
			rerankerOpts = append(rerankerOpts, WithBatchSize(cfg.Reranker.LLM.BatchSize))
		}
		if cfg.Reranker.LLM.MaxRetries > 0 {
			rerankerOpts = append(rerankerOpts, WithMaxRetries(cfg.Reranker.LLM.MaxRetries))
		}
		rr, err := NewLLMReranker(llmProvider, rerankerOpts...)
		if err != nil {
			return nil, fmt.Errorf("reranker: %w", err)
		}
		opts.Reranker = rr
	}

	return NewPipelineWithOptions(store, llmProvider, embedder, opts)
}

func newStoreFromConfig(cfg StoreConfig, deps []ExternalDeps) (VectorStore, error) {
	switch cfg.Type {
	case "memory":
		return NewInMemoryStore(), nil
	case "pgvector":
		if cfg.Pgvector == nil {
			return nil, fmt.Errorf("%w: pgvector config is nil, add 'pgvector:' section", ErrMissingRequiredField)
		}
		var db *sql.DB
		for _, d := range deps {
			if d.DB != nil {
				db = d.DB
				break
			}
		}
		if db == nil {
			return nil, fmt.Errorf("%w: db (*sql.DB) is required for pgvector store, pass via ExternalDeps", ErrMissingRequiredField)
		}
		return NewPGVectorStoreWithOptions(db, PGVectorStoreOptions{
			PGVectorOptions: PGVectorOptions{
				TableName:          cfg.Pgvector.TableName,
				EmbeddingDimension: cfg.Pgvector.EmbeddingDimension,
				CreateExtension:    cfg.Pgvector.CreateExtension,
				IndexMethod:        cfg.Pgvector.IndexMethod,
				Lists:              cfg.Pgvector.Lists,
			},
		})
	case "qdrant":
		if cfg.Qdrant == nil {
			return nil, fmt.Errorf("%w: qdrant config is nil", ErrMissingRequiredField)
		}
		if cfg.Qdrant.Collection == "" {
			return nil, fmt.Errorf("%w: qdrant collection is required", ErrMissingRequiredField)
		}
		return NewQdrantStore(QdrantOptions{
			URL:        cfg.Qdrant.URL,
			Collection: cfg.Qdrant.Collection,
			Dimension:  cfg.Qdrant.Dimension,
			Timeout:    cfg.Qdrant.Timeout,
		})
	case "chromadb":
		if cfg.ChromaDB == nil {
			return nil, fmt.Errorf("%w: chromadb config is nil", ErrMissingRequiredField)
		}
		if cfg.ChromaDB.Collection == "" {
			return nil, fmt.Errorf("%w: chromadb collection is required", ErrMissingRequiredField)
		}
		return NewChromaDBStore(ChromaDBOptions{
			BaseURL:    cfg.ChromaDB.BaseURL,
			Collection: cfg.ChromaDB.Collection,
			Dimension:  cfg.ChromaDB.Dimension,
		})
	case "weaviate":
		if cfg.Weaviate == nil {
			return nil, fmt.Errorf("%w: weaviate config is nil", ErrMissingRequiredField)
		}
		if cfg.Weaviate.Collection == "" {
			return nil, fmt.Errorf("%w: weaviate collection is required", ErrMissingRequiredField)
		}
		return NewWeaviateStore(WeaviateOptions{
			Host:       cfg.Weaviate.Host,
			Scheme:     cfg.Weaviate.Scheme,
			Collection: cfg.Weaviate.Collection,
			APIKey:     cfg.Weaviate.APIKey,
		})
	case "milvus":
		return nil, fmt.Errorf("%w: milvus store constructor is not yet publicly available", ErrUnknownConfigKey)
	case "":
		return nil, fmt.Errorf("%w: store type is required (memory, pgvector, qdrant, chromadb, weaviate)", ErrMissingRequiredField)
	default:
		return nil, fmt.Errorf("%w: unknown store type %q", ErrUnknownConfigKey, cfg.Type)
	}
}

func newEmbedderFromConfig(cfg EmbedderConfig) (Embedder, error) {
	switch cfg.Type {
	case "ollama":
		if cfg.Ollama == nil {
			return nil, fmt.Errorf("%w: ollama embedder config section is required", ErrMissingRequiredField)
		}
		if cfg.Ollama.Model == "" {
			return nil, fmt.Errorf("%w: embedder model is required for ollama", ErrMissingRequiredField)
		}
		return NewOllamaEmbedder(OllamaEmbedderOptions{
			BaseURL: cfg.Ollama.BaseURL,
			Model:   cfg.Ollama.Model,
			APIKey:  cfg.Ollama.APIKey,
			Timeout: cfg.Ollama.Timeout,
		}), nil
	case "openai_compatible":
		if cfg.OpenAICompatible == nil {
			return nil, fmt.Errorf("%w: openai_compatible embedder config section is required", ErrMissingRequiredField)
		}
		if cfg.OpenAICompatible.Model == "" {
			return nil, fmt.Errorf("%w: embedder model is required for openai_compatible", ErrMissingRequiredField)
		}
		return NewOpenAICompatibleEmbedder(OpenAICompatibleEmbedderOptions{
			BaseURL: cfg.OpenAICompatible.BaseURL,
			Model:   cfg.OpenAICompatible.Model,
			APIKey:  cfg.OpenAICompatible.APIKey,
			Timeout: cfg.OpenAICompatible.Timeout,
		}), nil
	case "mistral":
		if cfg.Mistral == nil {
			return nil, fmt.Errorf("%w: mistral embedder config section is required", ErrMissingRequiredField)
		}
		if cfg.Mistral.Model == "" {
			return nil, fmt.Errorf("%w: embedder model is required for mistral", ErrMissingRequiredField)
		}
		return NewMistralEmbedder(MistralEmbedderOptions{
			APIKey:  cfg.Mistral.APIKey,
			Model:   cfg.Mistral.Model,
			Timeout: cfg.Mistral.Timeout,
		}), nil
	case "":
		return nil, fmt.Errorf("%w: embedder type is required (ollama, openai_compatible, mistral)", ErrMissingRequiredField)
	default:
		return nil, fmt.Errorf("%w: unknown embedder type %q", ErrUnknownConfigKey, cfg.Type)
	}
}

func newLLMFromConfig(cfg LLMConfig) (LLMProvider, error) {
	switch cfg.Type {
	case "ollama":
		if cfg.Ollama == nil {
			return nil, fmt.Errorf("%w: ollama llm config section is required", ErrMissingRequiredField)
		}
		if cfg.Ollama.Model == "" {
			return nil, fmt.Errorf("%w: llm model is required for ollama", ErrMissingRequiredField)
		}
		return NewOllamaLLM(OllamaLLMOptions{
			BaseURL: cfg.Ollama.BaseURL,
			Model:   cfg.Ollama.Model,
			APIKey:  cfg.Ollama.APIKey,
			Timeout: cfg.Ollama.Timeout,
		}), nil
	case "openai_compatible":
		if cfg.OpenAICompatible == nil {
			return nil, fmt.Errorf("%w: openai_compatible llm config section is required", ErrMissingRequiredField)
		}
		if cfg.OpenAICompatible.Model == "" {
			return nil, fmt.Errorf("%w: llm model is required for openai_compatible", ErrMissingRequiredField)
		}
		return NewOpenAICompatibleLLM(OpenAICompatibleLLMOptions{
			BaseURL: cfg.OpenAICompatible.BaseURL,
			Model:   cfg.OpenAICompatible.Model,
			APIKey:  cfg.OpenAICompatible.APIKey,
			Timeout: cfg.OpenAICompatible.Timeout,
		}), nil
	case "anthropic":
		if cfg.Anthropic == nil {
			return nil, fmt.Errorf("%w: anthropic llm config section is required", ErrMissingRequiredField)
		}
		if cfg.Anthropic.Model == "" {
			return nil, fmt.Errorf("%w: llm model is required for anthropic", ErrMissingRequiredField)
		}
		return NewAnthropicLLM(AnthropicLLMOptions{
			BaseURL:          cfg.Anthropic.BaseURL,
			APIKey:           cfg.Anthropic.APIKey,
			Model:            cfg.Anthropic.Model,
			AnthropicVersion: cfg.Anthropic.AnthropicVersion,
			Temperature:      cfg.Anthropic.Temperature,
			MaxTokens:        cfg.Anthropic.MaxTokens,
		}), nil
	case "deepseek":
		if cfg.DeepSeek == nil {
			return nil, fmt.Errorf("%w: deepseek llm config section is required", ErrMissingRequiredField)
		}
		if cfg.DeepSeek.Model == "" {
			return nil, fmt.Errorf("%w: llm model is required for deepseek", ErrMissingRequiredField)
		}
		return NewDeepSeekLLM(DeepSeekLLMOptions{
			BaseURL: cfg.DeepSeek.BaseURL,
			APIKey:  cfg.DeepSeek.APIKey,
			Model:   cfg.DeepSeek.Model,
			Timeout: cfg.DeepSeek.Timeout,
		}), nil
	case "mistral":
		if cfg.Mistral == nil {
			return nil, fmt.Errorf("%w: mistral llm config section is required", ErrMissingRequiredField)
		}
		if cfg.Mistral.Model == "" {
			return nil, fmt.Errorf("%w: llm model is required for mistral", ErrMissingRequiredField)
		}
		return NewMistralLLM(MistralLLMOptions{
			BaseURL: cfg.Mistral.BaseURL,
			APIKey:  cfg.Mistral.APIKey,
			Model:   cfg.Mistral.Model,
			Timeout: cfg.Mistral.Timeout,
		}), nil
	case "":
		return nil, fmt.Errorf("%w: llm type is required (ollama, openai_compatible, anthropic, deepseek, mistral)", ErrMissingRequiredField)
	default:
		return nil, fmt.Errorf("%w: unknown llm type %q", ErrUnknownConfigKey, cfg.Type)
	}
}
