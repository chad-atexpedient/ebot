// Package testutil provides shared testing utilities
package testutil

import (
	"context"
	"sync"
	"time"
)

// MockLogger implements a logger for testing
type MockLogger struct {
	mu       sync.Mutex
	Messages []LogMessage
}

type LogMessage struct {
	Level   string
	Message string
	Args    []interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		Messages: []LogMessage{},
	}
}

func (l *MockLogger) Info(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Messages = append(l.Messages, LogMessage{Level: "info", Message: msg, Args: args})
}

func (l *MockLogger) Error(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Messages = append(l.Messages, LogMessage{Level: "error", Message: msg, Args: args})
}

func (l *MockLogger) Warn(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Messages = append(l.Messages, LogMessage{Level: "warn", Message: msg, Args: args})
}

func (l *MockLogger) Debug(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Messages = append(l.Messages, LogMessage{Level: "debug", Message: msg, Args: args})
}

func (l *MockLogger) GetMessages(level string) []LogMessage {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	var filtered []LogMessage
	for _, msg := range l.Messages {
		if level == "" || msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

func (l *MockLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Messages = []LogMessage{}
}

// MockCache implements a simple in-memory cache for testing
type MockCache struct {
	mu    sync.RWMutex
	data  map[string]cacheEntry
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]cacheEntry),
	}
}

func (c *MockCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.data[key]
	if !exists {
		return ErrCacheMiss
	}
	
	if time.Now().After(entry.expiresAt) {
		return ErrCacheExpired
	}
	
	// Simple assignment for testing (real impl would use reflection)
	return nil
}

func (c *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (c *MockCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

// Cache errors
type CacheError string

func (e CacheError) Error() string { return string(e) }

const (
	ErrCacheMiss    CacheError = "cache miss"
	ErrCacheExpired CacheError = "cache expired"
)

// MockHTTPClient for testing HTTP requests
type MockHTTPClient struct {
	mu           sync.Mutex
	Requests     []MockRequest
	ResponseFunc func(req MockRequest) (int, []byte, error)
}

type MockRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		Requests: []MockRequest{},
	}
}

func (c *MockHTTPClient) RecordRequest(method, url string, headers map[string]string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Requests = append(c.Requests, MockRequest{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
	})
}

func (c *MockHTTPClient) GetRequests() []MockRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Requests
}

func (c *MockHTTPClient) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Requests = []MockRequest{}
}

// MockDatabase for testing database operations
type MockDatabase struct {
	mu     sync.RWMutex
	Tables map[string][]map[string]interface{}
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		Tables: make(map[string][]map[string]interface{}),
	}
}

func (db *MockDatabase) Insert(table string, record map[string]interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	
	if db.Tables[table] == nil {
		db.Tables[table] = []map[string]interface{}{}
	}
	db.Tables[table] = append(db.Tables[table], record)
	return nil
}

func (db *MockDatabase) Find(table string, query map[string]interface{}) []map[string]interface{} {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	var results []map[string]interface{}
	for _, record := range db.Tables[table] {
		match := true
		for k, v := range query {
			if record[k] != v {
				match = false
				break
			}
		}
		if match {
			results = append(results, record)
		}
	}
	return results
}

func (db *MockDatabase) Clear() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.Tables = make(map[string][]map[string]interface{})
}

// MockTimeProvider for testing time-dependent code
type MockTimeProvider struct {
	mu          sync.Mutex
	CurrentTime time.Time
}

func NewMockTimeProvider(t time.Time) *MockTimeProvider {
	return &MockTimeProvider{CurrentTime: t}
}

func (p *MockTimeProvider) Now() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.CurrentTime
}

func (p *MockTimeProvider) SetTime(t time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.CurrentTime = t
}

func (p *MockTimeProvider) Advance(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.CurrentTime = p.CurrentTime.Add(d)
}
