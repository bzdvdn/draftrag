package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// redisCache реализует L2 кэш поверх Redis.
// Данные хранятся как msgpack-encoded []float64 в виде []byte.
type redisCache struct {
	client RedisClient
	ttl    time.Duration
	prefix string
}

type redisDecodeError struct {
	err error
}

func (e *redisDecodeError) Error() string {
	return fmt.Sprintf("msgpack decode: %v", e.err)
}

func (e *redisDecodeError) Unwrap() error {
	return e.err
}

func (r *redisCache) effectivePrefix() string {
	if r.prefix == "" {
		return defaultRedisKeyPrefix
	}
	return r.prefix
}

func (r *redisCache) key(key string) string {
	return r.effectivePrefix() + key
}

func (r *redisCache) Get(ctx context.Context, key string) ([]float64, bool, error) {
	if r.client == nil {
		return nil, false, nil
	}

	data, err := r.client.GetBytes(ctx, r.key(key))
	if err != nil {
		return nil, false, err
	}
	if data == nil {
		return nil, false, nil
	}

	var value []float64
	if err := msgpack.Unmarshal(data, &value); err != nil {
		return nil, false, &redisDecodeError{err: err}
	}

	return value, true, nil
}

func (r *redisCache) Set(ctx context.Context, key string, value []float64) error {
	if r.client == nil {
		return nil
	}

	data, err := msgpack.Marshal(value)
	if err != nil {
		return fmt.Errorf("msgpack encode: %w", err)
	}

	return r.client.SetBytes(ctx, r.key(key), data, r.ttl)
}
