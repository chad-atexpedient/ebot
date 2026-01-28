// Package cache provides a caching layer for ebot
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheManager handles all caching operations for ebot
type CacheManager interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Invalidate removes all keys matching a pattern
	Invalidate(ctx context.Context, pattern string) error

	// GetOrSet gets a value from cache, or computes and stores it if missing
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error)

	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)

	// TTL returns the remaining TTL for a key
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Close closes the cache connection
	Close() error

	// Health returns the health status of the cache
	Health(ctx context.Context) error
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Address  string        // Redis server address (host:port)
	Password string        // Optional password
	DB       int           // Database number (0-15)
	PoolSize int           // Connection pool size
	Timeout  time.Duration // Operation timeout
}

// redisCacheManager implements CacheManager using Redis
type redisCacheManager struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisCacheManager creates a new Redis-backed cache manager
func NewRedisCacheManager(config RedisConfig) (CacheManager, error) {
	// Set defaults
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisCacheManager{
		client: client,
		config: config,
	}, nil
}

// Get retrieves a value from cache
func (r *redisCacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("cache get failed: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return result, nil
}

// Set stores a value in cache with TTL
func (r *redisCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cache set failed: %w", err)
	}

	return nil
}

// Delete removes a value from cache
func (r *redisCacheManager) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete failed: %w", err)
	}
	return nil
}

// Invalidate removes all keys matching a pattern
func (r *redisCacheManager) Invalidate(ctx context.Context, pattern string) error {
	var cursor uint64
	var deletedCount int

	for {
		// Scan for keys matching pattern
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("cache scan failed: %w", err)
		}

		// Delete found keys
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("cache delete failed: %w", err)
			}
			deletedCount += len(keys)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// GetOrSet gets a value from cache, or computes and stores it if missing
func (r *redisCacheManager) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	val, err := r.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if err != ErrCacheMiss {
		// Real error, not just a miss
		return nil, err
	}

	// Cache miss, compute the value
	val, err = fn()
	if err != nil {
		return nil, fmt.Errorf("compute function failed: %w", err)
	}

	// Store in cache
	if err := r.Set(ctx, key, val, ttl); err != nil {
		// Log but don't fail - we have the value
		// In production, use structured logging
		fmt.Printf("Warning: failed to cache value: %v\n", err)
	}

	return val, nil
}

// Exists checks if a key exists in cache
func (r *redisCacheManager) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists check failed: %w", err)
	}
	return count > 0, nil
}

// TTL returns the remaining TTL for a key
func (r *redisCacheManager) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache TTL check failed: %w", err)
	}
	return ttl, nil
}

// Close closes the cache connection
func (r *redisCacheManager) Close() error {
	return r.client.Close()
}

// Health returns the health status of the cache
func (r *redisCacheManager) Health(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}
	return nil
}

// MemoryCacheManager is a simple in-memory cache (for development/testing)
type memoryCacheManager struct {
	data map[string]cacheEntry
}

type cacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

// NewMemoryCacheManager creates an in-memory cache (not recommended for production)
func NewMemoryCacheManager() CacheManager {
	return &memoryCacheManager{
		data: make(map[string]cacheEntry),
	}
}

func (m *memoryCacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	entry, ok := m.data[key]
	if !ok {
		return nil, ErrCacheMiss
	}

	if time.Now().After(entry.Expiration) {
		delete(m.data, key)
		return nil, ErrCacheMiss
	}

	return entry.Value, nil
}

func (m *memoryCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = cacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
	return nil
}

func (m *memoryCacheManager) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *memoryCacheManager) Invalidate(ctx context.Context, pattern string) error {
	// Simple pattern matching for in-memory cache
	for key := range m.data {
		// In production, use proper pattern matching
		delete(m.data, key)
	}
	return nil
}

func (m *memoryCacheManager) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	val, err := m.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	val, err = fn()
	if err != nil {
		return nil, err
	}

	_ = m.Set(ctx, key, val, ttl)
	return val, nil
}

func (m *memoryCacheManager) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func (m *memoryCacheManager) TTL(ctx context.Context, key string) (time.Duration, error) {
	entry, ok := m.data[key]
	if !ok {
		return 0, ErrCacheMiss
	}
	return time.Until(entry.Expiration), nil
}

func (m *memoryCacheManager) Close() error {
	return nil
}

func (m *memoryCacheManager) Health(ctx context.Context) error {
	return nil
}

// Common cache key builders
func UserCacheKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func WorkspaceCacheKey(workspaceID string) string {
	return fmt.Sprintf("workspace:%s", workspaceID)
}

func MCPServerCacheKey(serverID string) string {
	return fmt.Sprintf("mcp-server:%s", serverID)
}

func QuotaCacheKey(userID string) string {
	return fmt.Sprintf("quota:%s", userID)
}

func ToolListCacheKey(serverID string) string {
	return fmt.Sprintf("tools:%s", serverID)
}

func ModelListCacheKey() string {
	return "models:list"
}

// Cache TTL constants
const (
	ShortTTL  = 1 * time.Minute  // For frequently changing data
	MediumTTL = 15 * time.Minute // For moderately stable data
	LongTTL   = 1 * time.Hour    // For stable data
	DayTTL    = 24 * time.Hour   // For very stable data
)
