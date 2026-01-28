package cache

import "errors"

// Common cache errors
var (
	// ErrCacheMiss indicates the key was not found in cache
	ErrCacheMiss = errors.New("cache miss")

	// ErrCacheUnavailable indicates the cache is not available
	ErrCacheUnavailable = errors.New("cache unavailable")

	// ErrInvalidKey indicates the cache key is invalid
	ErrInvalidKey = errors.New("invalid cache key")

	// ErrSerializationFailed indicates value serialization failed
	ErrSerializationFailed = errors.New("serialization failed")

	// ErrDeserializationFailed indicates value deserialization failed
	ErrDeserializationFailed = errors.New("deserialization failed")
)
