package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

type fakeRedisClient struct {
	store map[string][]byte

	getCalls int
	setCalls int

	lastGetKey string

	lastSetKey   string
	lastSetValue []byte
	lastSetTTL   time.Duration

	getErr error
	setErr error
}

type logEntry struct {
	level  domain.LogLevel
	msg    string
	fields map[string]any
}

type fakeLogger struct {
	entries []logEntry
}

func (l *fakeLogger) Log(ctx context.Context, level domain.LogLevel, msg string, fields ...domain.LogField) {
	_ = ctx
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	l.entries = append(l.entries, logEntry{level: level, msg: msg, fields: m})
}

func (f *fakeRedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	_ = ctx
	f.getCalls++
	f.lastGetKey = key
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.store == nil {
		return nil, nil
	}
	v, ok := f.store[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (f *fakeRedisClient) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	_ = ctx
	f.setCalls++
	f.lastSetKey = key
	f.lastSetValue = value
	f.lastSetTTL = ttl
	if f.setErr != nil {
		return f.setErr
	}
	if f.store == nil {
		f.store = make(map[string][]byte)
	}
	f.store[key] = value
	return nil
}

// TestRedisL2HitNoEmbedder проверяет, что при L2 hit базовый embedder не вызывается (AC-002),
// а значение прогревает L1 и второй вызов не делает Redis GET (AC-003).
func TestRedisL2HitNoEmbedder(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}
	redis := &fakeRedisClient{store: make(map[string][]byte)}

	c, err := NewEmbedderCache(mock, WithRedis(redis, 0, ""))
	require.NoError(t, err)

	text := "hello"
	key := c.hashKey(text)
	redisKey := defaultRedisKeyPrefix + key

	expected := []float64{1, 2, 3}
	b, err := msgpack.Marshal(expected)
	require.NoError(t, err)
	redis.store[redisKey] = b

	// L2 hit: embedder не должен вызываться
	got, err := c.Embed(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, 0, mock.callCount)
	assert.Equal(t, expected, got)
	assert.Equal(t, 1, redis.getCalls)

	// Второй вызов: должен быть L1 hit, без Redis GET
	got2, err := c.Embed(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, expected, got2)
	assert.Equal(t, 1, redis.getCalls, "второй вызов не должен обращаться к Redis")
}

// TestRedisErrorsTreatAsMiss проверяет treat-as-miss при ошибках Redis (AC-004).
func TestRedisErrorsTreatAsMiss(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}
	logger := &fakeLogger{}
	redis := &fakeRedisClient{
		getErr: errors.New("redis down"),
		setErr: errors.New("redis down"),
	}

	c, err := NewEmbedderCache(mock, WithLogger(logger), WithRedis(redis, 0, ""))
	require.NoError(t, err)

	got, err := c.Embed(ctx, "x")
	require.NoError(t, err)
	assert.Equal(t, 1, mock.callCount, "должен вызвать embedder при Redis ошибке")
	assert.NotNil(t, got)

	require.GreaterOrEqual(t, len(logger.entries), 1)
	assert.Equal(t, domain.LogLevelWarn, logger.entries[0].level)
	assert.Equal(t, "embedder_cache", logger.entries[0].fields["component"])
	assert.Equal(t, "redis_get", logger.entries[0].fields["operation"])
}

// TestRedisBadDataTreatAsMiss проверяет treat-as-miss при битых данных Redis (AC-006).
func TestRedisBadDataTreatAsMiss(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}
	logger := &fakeLogger{}
	redis := &fakeRedisClient{store: make(map[string][]byte)}

	c, err := NewEmbedderCache(mock, WithLogger(logger), WithRedis(redis, 0, ""))
	require.NoError(t, err)

	text := "bad"
	key := c.hashKey(text)
	redisKey := defaultRedisKeyPrefix + key
	redis.store[redisKey] = []byte{0xc1} // invalid msgpack code -> decode error

	got, err := c.Embed(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, 1, mock.callCount, "при битых данных должен вызвать embedder")
	assert.NotNil(t, got)

	require.GreaterOrEqual(t, len(logger.entries), 1)
	assert.Equal(t, "redis_decode", logger.entries[0].fields["operation"])
}

// TestRedisTTLAndPrefix проверяет, что TTL и prefix учитываются при записи (AC-005).
func TestRedisTTLAndPrefix(t *testing.T) {
	ctx := context.Background()
	mock := &mockEmbedder{vectors: make(map[string][]float64)}
	redis := &fakeRedisClient{store: make(map[string][]byte)}

	ttl := 5 * time.Minute
	prefix := "myapp:embedder:"

	c, err := NewEmbedderCache(mock, WithRedis(redis, ttl, prefix))
	require.NoError(t, err)

	text := "ttl"
	_, err = c.Embed(ctx, text) // miss -> embedder -> set
	require.NoError(t, err)

	key := c.hashKey(text)
	assert.Equal(t, prefix+key, redis.lastSetKey)
	assert.Equal(t, ttl, redis.lastSetTTL)
	assert.GreaterOrEqual(t, redis.setCalls, 1)
}
