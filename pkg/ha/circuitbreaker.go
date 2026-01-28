package ha

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

// CircuitBreaker prevents cascading failures by monitoring and breaking circuit when failures exceed threshold
type CircuitBreaker interface {
	// Execute runs the function with circuit breaker protection
	Execute(ctx context.Context, fn func() error) error
	
	// GetState returns the current circuit state
	GetState() CircuitState
	
	// GetMetrics returns circuit breaker metrics
	GetMetrics() CircuitMetrics
	
	// Reset manually resets the circuit breaker
	Reset()
}

// CircuitMetrics contains circuit breaker statistics
type CircuitMetrics struct {
	State              CircuitState
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	RejectedRequests   int64
	LastStateChange    time.Time
	ConsecutiveFailures int
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	// MaxFailures is the number of consecutive failures before opening
	MaxFailures int
	
	// Timeout is how long to wait in open state before trying half-open
	Timeout time.Duration
	
	// MaxHalfOpenRequests is max requests allowed in half-open state
	MaxHalfOpenRequests int
	
	// SuccessThreshold is successful requests needed in half-open to close
	SuccessThreshold int
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:         5,
		Timeout:             30 * time.Second,
		MaxHalfOpenRequests: 3,
		SuccessThreshold:    2,
	}
}

type circuitBreaker struct {
	config              CircuitBreakerConfig
	state               CircuitState
	consecutiveFailures int
	consecutiveSuccesses int
	halfOpenRequests    int
	metrics             CircuitMetrics
	lastStateChange     time.Time
	mu                  sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) CircuitBreaker {
	return &circuitBreaker{
		config:          config,
		state:           CircuitStateClosed,
		lastStateChange: time.Now(),
		metrics: CircuitMetrics{
			State:           CircuitStateClosed,
			LastStateChange: time.Now(),
		},
	}
}

// Execute runs the function with circuit breaker protection
func (cb *circuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we can proceed
	if err := cb.beforeRequest(); err != nil {
		cb.recordRejection()
		return err
	}
	
	// Execute the function
	err := fn()
	
	// Record the result
	cb.afterRequest(err)
	
	return err
}

// GetState returns the current circuit state
func (cb *circuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns circuit breaker metrics
func (cb *circuitBreaker) GetMetrics() CircuitMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return CircuitMetrics{
		State:               cb.state,
		TotalRequests:       cb.metrics.TotalRequests,
		SuccessfulRequests:  cb.metrics.SuccessfulRequests,
		FailedRequests:      cb.metrics.FailedRequests,
		RejectedRequests:    cb.metrics.RejectedRequests,
		LastStateChange:     cb.lastStateChange,
		ConsecutiveFailures: cb.consecutiveFailures,
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *circuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.state = CircuitStateClosed
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.halfOpenRequests = 0
	cb.lastStateChange = time.Now()
	cb.metrics.State = CircuitStateClosed
	cb.metrics.LastStateChange = time.Now()
}

// beforeRequest checks if request can proceed
func (cb *circuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case CircuitStateClosed:
		// Allow request
		return nil
		
	case CircuitStateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastStateChange) >= cb.config.Timeout {
			// Transition to half-open
			cb.state = CircuitStateHalfOpen
			cb.halfOpenRequests = 0
			cb.consecutiveSuccesses = 0
			cb.lastStateChange = time.Now()
			cb.metrics.State = CircuitStateHalfOpen
			cb.metrics.LastStateChange = time.Now()
			return nil
		}
		return ErrCircuitOpen
		
	case CircuitStateHalfOpen:
		// Check if we can allow more requests
		if cb.halfOpenRequests >= cb.config.MaxHalfOpenRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
		return nil
		
	default:
		return ErrCircuitOpen
	}
}

// afterRequest records the result of a request
func (cb *circuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.metrics.TotalRequests++
	
	if err != nil {
		// Request failed
		cb.metrics.FailedRequests++
		cb.consecutiveFailures++
		cb.consecutiveSuccesses = 0
		
		switch cb.state {
		case CircuitStateClosed:
			// Check if we should open the circuit
			if cb.consecutiveFailures >= cb.config.MaxFailures {
				cb.state = CircuitStateOpen
				cb.lastStateChange = time.Now()
				cb.metrics.State = CircuitStateOpen
				cb.metrics.LastStateChange = time.Now()
			}
			
		case CircuitStateHalfOpen:
			// Single failure in half-open transitions back to open
			cb.state = CircuitStateOpen
			cb.halfOpenRequests = 0
			cb.lastStateChange = time.Now()
			cb.metrics.State = CircuitStateOpen
			cb.metrics.LastStateChange = time.Now()
		}
	} else {
		// Request succeeded
		cb.metrics.SuccessfulRequests++
		cb.consecutiveSuccesses++
		cb.consecutiveFailures = 0
		
		switch cb.state {
		case CircuitStateHalfOpen:
			// Check if we should close the circuit
			if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
				cb.state = CircuitStateClosed
				cb.halfOpenRequests = 0
				cb.lastStateChange = time.Now()
				cb.metrics.State = CircuitStateClosed
				cb.metrics.LastStateChange = time.Now()
			}
		}
	}
}

// recordRejection records a rejected request
func (cb *circuitBreaker) recordRejection() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.metrics.RejectedRequests++
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]CircuitBreaker
	config   CircuitBreakerConfig
	mu       sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(config CircuitBreakerConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]CircuitBreaker),
		config:   config,
	}
}

// GetOrCreate gets or creates a circuit breaker for a specific resource
func (m *CircuitBreakerManager) GetOrCreate(name string) CircuitBreaker {
	m.mu.RLock()
	breaker, exists := m.breakers[name]
	m.mu.RUnlock()
	
	if exists {
		return breaker
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}
	
	breaker = NewCircuitBreaker(m.config)
	m.breakers[name] = breaker
	return breaker
}

// GetAllMetrics returns metrics for all circuit breakers
func (m *CircuitBreakerManager) GetAllMetrics() map[string]CircuitMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics := make(map[string]CircuitMetrics)
	for name, breaker := range m.breakers {
		metrics[name] = breaker.GetMetrics()
	}
	return metrics
}

// ResetAll resets all circuit breakers
func (m *CircuitBreakerManager) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, breaker := range m.breakers {
		breaker.Reset()
	}
}
