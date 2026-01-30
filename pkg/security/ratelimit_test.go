package security

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockCacheManager implements cache.CacheManager for testing
type mockCacheManager struct {
	data map[string]interface{}
}

func newMockCacheManager() *mockCacheManager {
	return &mockCacheManager{
		data: make(map[string]interface{}),
	}
}

func (m *mockCacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	if val, ok := m.data[key]; ok {
		// For testing, we just check existence
		_ = val
		return nil
	}
	return fmt.Errorf("key not found")
}

func (m *mockCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCacheManager) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func TestNewRateLimiter(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	
	rl := NewRateLimiter(config, cache)
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()
	
	if config.GlobalRequestsPerSecond != 10000 {
		t.Errorf("Expected GlobalRequestsPerSecond 10000, got %d", config.GlobalRequestsPerSecond)
	}
	if config.PerIPRequestsPerSecond != 100 {
		t.Errorf("Expected PerIPRequestsPerSecond 100, got %d", config.PerIPRequestsPerSecond)
	}
	if config.PerUserRequestsPerSecond != 500 {
		t.Errorf("Expected PerUserRequestsPerSecond 500, got %d", config.PerUserRequestsPerSecond)
	}
	if !config.EnableDDoSProtection {
		t.Error("Expected EnableDDoSProtection to be true")
	}
	if !config.EnableAdaptive {
		t.Error("Expected EnableAdaptive to be true")
	}
}

func TestCheckLimit_AllowsRequest(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	
	result, err := rl.CheckLimit(req, "user123")
	if err != nil {
		t.Fatalf("CheckLimit failed: %v", err)
	}
	
	if !result.Allowed {
		t.Error("Expected request to be allowed")
	}
}

func TestCheckLimit_BlockedIP(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	// Manually block an IP
	testIP := "10.0.0.1"
	rl.blockedIPs.Store(testIP, time.Now().Add(time.Hour))
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = testIP + ":12345"
	
	result, err := rl.CheckLimit(req, "user123")
	if err != nil {
		t.Fatalf("CheckLimit failed: %v", err)
	}
	
	if result.Allowed {
		t.Error("Expected blocked IP to be denied")
	}
	if result.Reason != "IP temporarily blocked due to suspicious activity" {
		t.Errorf("Unexpected reason: %s", result.Reason)
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	
	ip := rl.getClientIP(req)
	if ip != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", ip)
	}
}

func TestGetClientIP_TrustedProxy(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	config.TrustedProxies = []string{"10.0.0.0/8"}
	rl := NewRateLimiter(config, cache)
	
	// Explicitly enable trusting X-Forwarded-For from trusted proxies
	rl.EnableXForwardedForTrust(true)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")
	
	ip := rl.getClientIP(req)
	// Should trust X-Forwarded-For from trusted proxy
	if ip != "203.0.113.50" {
		t.Errorf("Expected IP 203.0.113.50 from trusted proxy, got %s", ip)
	}
}

func TestGetClientIP_UntrustedProxy(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	// No trusted proxies configured
	rl := NewRateLimiter(config, cache)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	req.Header.Set("X-Forwarded-For", "spoofed-ip")
	
	ip := rl.getClientIP(req)
	// Should NOT trust X-Forwarded-For from untrusted source
	if ip != "192.168.1.1" {
		t.Errorf("Expected RemoteAddr IP, got %s", ip)
	}
}

func TestUnblockIP(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	testIP := "10.0.0.5"
	
	// Block the IP
	rl.blockedIPs.Store(testIP, time.Now().Add(time.Hour))
	
	// Verify blocked
	blocked, _ := rl.isIPBlocked(testIP)
	if !blocked {
		t.Error("Expected IP to be blocked")
	}
	
	// Unblock
	rl.UnblockIP(testIP)
	
	// Verify unblocked
	blocked, _ = rl.isIPBlocked(testIP)
	if blocked {
		t.Error("Expected IP to be unblocked")
	}
}

func TestUpdateSystemLoad(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	// Test valid load
	rl.UpdateSystemLoad(0.75)
	if rl.GetSystemLoad() != 0.75 {
		t.Errorf("Expected load 0.75, got %f", rl.GetSystemLoad())
	}
	
	// Test load clamping (above 1.0)
	rl.UpdateSystemLoad(1.5)
	if rl.GetSystemLoad() != 1.0 {
		t.Errorf("Expected load clamped to 1.0, got %f", rl.GetSystemLoad())
	}
	
	// Test load clamping (below 0.0)
	rl.UpdateSystemLoad(-0.5)
	if rl.GetSystemLoad() != 0.0 {
		t.Errorf("Expected load clamped to 0.0, got %f", rl.GetSystemLoad())
	}
}

func TestGetBlockedIPs(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	// Block some IPs
	rl.blockedIPs.Store("1.1.1.1", time.Now().Add(time.Hour))
	rl.blockedIPs.Store("2.2.2.2", time.Now().Add(time.Hour))
	
	blocked := rl.GetBlockedIPs()
	if len(blocked) != 2 {
		t.Errorf("Expected 2 blocked IPs, got %d", len(blocked))
	}
	
	if _, ok := blocked["1.1.1.1"]; !ok {
		t.Error("Expected 1.1.1.1 to be in blocked list")
	}
	if _, ok := blocked["2.2.2.2"]; !ok {
		t.Error("Expected 2.2.2.2 to be in blocked list")
	}
}

func TestMiddleware(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	// Create a handler
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	
	// Apply middleware
	middleware := rl.Middleware(nil)
	handler := middleware(next)
	
	// Create request
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	
	// Create response recorder
	rr := httptest.NewRecorder()
	
	// Execute
	handler.ServeHTTP(rr, req)
	
	// Verify
	if !nextCalled {
		t.Error("Expected next handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	// Check rate limit headers
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}
}

func TestEndpointSpecificLimits(t *testing.T) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	// Login endpoint has stricter limits
	config.EndpointLimits["/api/auth/login"] = EndpointLimit{
		RequestsPerSecond: 5,
		BurstSize:         10,
	}
	rl := NewRateLimiter(config, cache)
	
	// First few requests should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		
		result, err := rl.CheckLimit(req, "user123")
		if err != nil {
			t.Fatalf("CheckLimit failed: %v", err)
		}
		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}
}

func BenchmarkCheckLimit(b *testing.B) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.CheckLimit(req, "user123")
	}
}

func BenchmarkGetClientIP(b *testing.B) {
	cache := newMockCacheManager()
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config, cache)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.getClientIP(req)
	}
}
