package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/chad-atexpedient/ebot/pkg/cache"
)

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	// Global limits
	GlobalRequestsPerSecond int
	GlobalBurstSize         int

	// Per-IP limits
	PerIPRequestsPerSecond int
	PerIPBurstSize         int

	// Per-user limits
	PerUserRequestsPerSecond int
	PerUserBurstSize         int

	// Per-endpoint limits
	EndpointLimits map[string]EndpointLimit

	// DDoS protection
	EnableDDoSProtection  bool
	SuspiciousIPThreshold int // Requests per second to be considered suspicious
	BlockDuration         time.Duration

	// Adaptive rate limiting
	EnableAdaptive     bool
	LoadThreshold      float64 // 0.0-1.0, trigger adaptive limiting above this
	AdaptiveMultiplier float64 // Multiply limits by this factor when under load
}

// EndpointLimit configures per-endpoint rate limits
type EndpointLimit struct {
	Path               string
	RequestsPerSecond  int
	BurstSize          int
	RequireAuth        bool
	SkipForAdmins      bool
	CustomHeaderName   string // Optional custom header for rate limit status
}

// RateLimiter provides advanced rate limiting with DDoS protection
type RateLimiter struct {
	config      *RateLimitConfig
	cache       cache.CacheManager
	blockedIPs  sync.Map // map[string]time.Time
	ipCounters  sync.Map // map[string]*ipCounter
	mu          sync.RWMutex
	systemLoad  float64 // Current system load (0.0-1.0)
}

// ipCounter tracks requests from a specific IP
type ipCounter struct {
	count      int64
	windowStart time.Time
	blocked    bool
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig, cacheManager cache.CacheManager) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimiter{
		config: config,
		cache:  cacheManager,
	}

	// Start background cleanup
	go rl.cleanupLoop()

	return rl
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		GlobalRequestsPerSecond:  10000,
		GlobalBurstSize:          20000,
		PerIPRequestsPerSecond:   100,
		PerIPBurstSize:           200,
		PerUserRequestsPerSecond: 500,
		PerUserBurstSize:         1000,
		EndpointLimits: map[string]EndpointLimit{
			"/api/auth/login":          {RequestsPerSecond: 5, BurstSize: 10},
			"/api/auth/register":       {RequestsPerSecond: 2, BurstSize: 5},
			"/api/auth/password-reset": {RequestsPerSecond: 1, BurstSize: 3},
			"/api/costs/export":        {RequestsPerSecond: 10, BurstSize: 20},
		},
		EnableDDoSProtection:  true,
		SuspiciousIPThreshold: 500,
		BlockDuration:         15 * time.Minute,
		EnableAdaptive:        true,
		LoadThreshold:         0.8,
		AdaptiveMultiplier:    0.5,
	}
}

// CheckLimit verifies if a request should be allowed
func (rl *RateLimiter) CheckLimit(r *http.Request, userID string) (*RateLimitResult, error) {
	ctx := r.Context()

	// Extract IP address
	ip := rl.getClientIP(r)

	// Check if IP is blocked
	if blocked, until := rl.isIPBlocked(ip); blocked {
		return &RateLimitResult{
			Allowed:       false,
			Reason:        "IP temporarily blocked due to suspicious activity",
			RetryAfter:    until,
			RemainingHits: 0,
		}, nil
	}

	// Check endpoint-specific limits first
	if limit, ok := rl.config.EndpointLimits[r.URL.Path]; ok {
		result, err := rl.checkEndpointLimit(ctx, r.URL.Path, userID, limit)
		if err != nil || !result.Allowed {
			return result, err
		}
	}

	// Check per-IP limit
	ipResult, err := rl.checkIPLimit(ctx, ip)
	if err != nil || !ipResult.Allowed {
		// Track suspicious activity
		if rl.config.EnableDDoSProtection {
			rl.trackSuspiciousIP(ip)
		}
		return ipResult, err
	}

	// Check per-user limit (if authenticated)
	if userID != "" {
		userResult, err := rl.checkUserLimit(ctx, userID)
		if err != nil || !userResult.Allowed {
			return userResult, err
		}
	}

	// Check global limit
	globalResult, err := rl.checkGlobalLimit(ctx)
	if err != nil || !globalResult.Allowed {
		return globalResult, err
	}

	// All checks passed
	return &RateLimitResult{
		Allowed:       true,
		RemainingHits: ipResult.RemainingHits,
		ResetAt:       ipResult.ResetAt,
	}, nil
}

// checkEndpointLimit checks endpoint-specific rate limits
func (rl *RateLimiter) checkEndpointLimit(ctx context.Context, path, userID string, limit EndpointLimit) (*RateLimitResult, error) {
	key := fmt.Sprintf("ratelimit:endpoint:%s:%s", path, userID)
	return rl.checkLimit(ctx, key, limit.RequestsPerSecond, limit.BurstSize)
}

// checkIPLimit checks per-IP rate limits
func (rl *RateLimiter) checkIPLimit(ctx context.Context, ip string) (*RateLimitResult, error) {
	key := fmt.Sprintf("ratelimit:ip:%s", ip)
	rps := rl.config.PerIPRequestsPerSecond
	burst := rl.config.PerIPBurstSize

	// Apply adaptive rate limiting if enabled
	if rl.config.EnableAdaptive && rl.systemLoad > rl.config.LoadThreshold {
		rps = int(float64(rps) * rl.config.AdaptiveMultiplier)
		burst = int(float64(burst) * rl.config.AdaptiveMultiplier)
	}

	return rl.checkLimit(ctx, key, rps, burst)
}

// checkUserLimit checks per-user rate limits
func (rl *RateLimiter) checkUserLimit(ctx context.Context, userID string) (*RateLimitResult, error) {
	key := fmt.Sprintf("ratelimit:user:%s", userID)
	return rl.checkLimit(ctx, key, rl.config.PerUserRequestsPerSecond, rl.config.PerUserBurstSize)
}

// checkGlobalLimit checks global rate limits
func (rl *RateLimiter) checkGlobalLimit(ctx context.Context) (*RateLimitResult, error) {
	key := "ratelimit:global"
	return rl.checkLimit(ctx, key, rl.config.GlobalRequestsPerSecond, rl.config.GlobalBurstSize)
}

// checkLimit implements token bucket algorithm using cache
func (rl *RateLimiter) checkLimit(ctx context.Context, key string, rps, burst int) (*RateLimitResult, error) {
	now := time.Now()

	// Get current bucket state from cache
	var bucket tokenBucket
	if err := rl.cache.Get(ctx, key, &bucket); err != nil {
		// Initialize new bucket
		bucket = tokenBucket{
			Tokens:     float64(burst),
			LastRefill: now,
		}
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.LastRefill).Seconds()
	tokensToAdd := elapsed * float64(rps)
	bucket.Tokens = min(bucket.Tokens+tokensToAdd, float64(burst))
	bucket.LastRefill = now

	// Check if we have tokens available
	allowed := bucket.Tokens >= 1.0
	if allowed {
		bucket.Tokens -= 1.0
	}

	// Save bucket state
	if err := rl.cache.Set(ctx, key, bucket, time.Second); err != nil {
		return nil, fmt.Errorf("failed to save rate limit state: %w", err)
	}

	result := &RateLimitResult{
		Allowed:       allowed,
		RemainingHits: int(bucket.Tokens),
		ResetAt:       now.Add(time.Second),
	}

	if !allowed {
		result.Reason = "Rate limit exceeded"
		result.RetryAfter = now.Add(time.Second)
	}

	return result, nil
}

// trackSuspiciousIP tracks IPs making excessive requests
func (rl *RateLimiter) trackSuspiciousIP(ip string) {
	val, _ := rl.ipCounters.LoadOrStore(ip, &ipCounter{windowStart: time.Now()})
	counter := val.(*ipCounter)

	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()

	// Reset window if needed (1 second windows)
	if now.Sub(counter.windowStart) > time.Second {
		counter.count = 0
		counter.windowStart = now
	}

	counter.count++

	// Block if threshold exceeded
	if counter.count > int64(rl.config.SuspiciousIPThreshold) && !counter.blocked {
		counter.blocked = true
		rl.blockedIPs.Store(ip, now.Add(rl.config.BlockDuration))
	}
}

// isIPBlocked checks if an IP is currently blocked
func (rl *RateLimiter) isIPBlocked(ip string) (bool, time.Time) {
	val, ok := rl.blockedIPs.Load(ip)
	if !ok {
		return false, time.Time{}
	}

	until := val.(time.Time)
	if time.Now().After(until) {
		// Block expired, remove it
		rl.blockedIPs.Delete(ip)
		
		// Reset counter
		if val, ok := rl.ipCounters.Load(ip); ok {
			counter := val.(*ipCounter)
			counter.mu.Lock()
			counter.blocked = false
			counter.count = 0
			counter.mu.Unlock()
		}
		
		return false, time.Time{}
	}

	return true, until
}

// getClientIP extracts the real client IP from request
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (client's real IP)
		if ip, _, err := net.SplitHostPort(xff); err == nil {
			return ip
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// UpdateSystemLoad updates current system load for adaptive rate limiting
func (rl *RateLimiter) UpdateSystemLoad(load float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.systemLoad = max(0.0, min(1.0, load))
}

// GetSystemLoad returns current system load
func (rl *RateLimiter) GetSystemLoad() float64 {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.systemLoad
}

// cleanupLoop periodically cleans up expired entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		// Clean up blocked IPs
		rl.blockedIPs.Range(func(key, value interface{}) bool {
			until := value.(time.Time)
			if now.After(until) {
				rl.blockedIPs.Delete(key)
			}
			return true
		})

		// Clean up old IP counters
		rl.ipCounters.Range(func(key, value interface{}) bool {
			counter := value.(*ipCounter)
			counter.mu.Lock()
			idle := now.Sub(counter.windowStart) > 5*time.Minute
			counter.mu.Unlock()

			if idle {
				rl.ipCounters.Delete(key)
			}
			return true
		})
	}
}

// GetBlockedIPs returns currently blocked IPs
func (rl *RateLimiter) GetBlockedIPs() map[string]time.Time {
	result := make(map[string]time.Time)
	rl.blockedIPs.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(time.Time)
		return true
	})
	return result
}

// UnblockIP manually unblocks an IP
func (rl *RateLimiter) UnblockIP(ip string) {
	rl.blockedIPs.Delete(ip)
	if val, ok := rl.ipCounters.Load(ip); ok {
		counter := val.(*ipCounter)
		counter.mu.Lock()
		counter.blocked = false
		counter.count = 0
		counter.mu.Unlock()
	}
}

// tokenBucket represents the state of a token bucket
type tokenBucket struct {
	Tokens     float64
	LastRefill time.Time
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed       bool
	Reason        string
	RemainingHits int
	ResetAt       time.Time
	RetryAfter    time.Time
}

// Middleware returns HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(getUserID func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userID string
			if getUserID != nil {
				userID = getUserID(r)
			}

			result, err := rl.CheckLimit(r, userID)
			if err != nil {
				http.Error(w, "Rate limit check failed", http.StatusInternalServerError)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.RemainingHits))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

			if !result.Allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(result.RetryAfter).Seconds())))
				http.Error(w, result.Reason, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
