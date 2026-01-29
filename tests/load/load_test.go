// Package load provides load testing utilities for ebot
package load

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// LoadTestConfig holds configuration for load tests
type LoadTestConfig struct {
	// Number of concurrent users/goroutines
	Concurrency int

	// Total number of requests to make
	TotalRequests int

	// Duration to run the test (0 for request-based)
	Duration time.Duration

	// Ramp up period
	RampUpDuration time.Duration

	// Target requests per second (0 for unlimited)
	TargetRPS int

	// Timeout for individual requests
	RequestTimeout time.Duration
}

// LoadTestResult holds the results of a load test
type LoadTestResult struct {
	TotalRequests     int64
	SuccessfulReqs    int64
	FailedReqs        int64
	TotalDuration     time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	AvgLatency        time.Duration
	P50Latency        time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
	RequestsPerSecond float64
	ErrorRate         float64
}

// RequestFunc is a function that makes a single request
type RequestFunc func(ctx context.Context) error

// LoadTester performs load tests
type LoadTester struct {
	config  LoadTestConfig
	results *LoadTestResult

	latencies []time.Duration
	mu        sync.Mutex
}

// NewLoadTester creates a new load tester
func NewLoadTester(config LoadTestConfig) *LoadTester {
	// Set defaults
	if config.Concurrency == 0 {
		config.Concurrency = 10
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	return &LoadTester{
		config: config,
		results: &LoadTestResult{
			MinLatency: time.Hour, // Will be overwritten
		},
		latencies: make([]time.Duration, 0, config.TotalRequests),
	}
}

// Run executes the load test
func (lt *LoadTester) Run(ctx context.Context, fn RequestFunc) (*LoadTestResult, error) {
	startTime := time.Now()

	var wg sync.WaitGroup
	var totalReqs int64
	var successReqs int64
	var failedReqs int64

	// Create rate limiter if needed
	var rateLimiter <-chan time.Time
	if lt.config.TargetRPS > 0 {
		interval := time.Second / time.Duration(lt.config.TargetRPS)
		rateLimiter = time.Tick(interval)
	}

	// Channel to control number of requests
	reqChan := make(chan struct{}, lt.config.TotalRequests)
	if lt.config.Duration > 0 {
		// Duration-based test
		go func() {
			timer := time.NewTimer(lt.config.Duration)
			defer timer.Stop()
			<-timer.C
			close(reqChan)
		}()
	} else {
		// Request-based test
		for i := 0; i < lt.config.TotalRequests; i++ {
			reqChan <- struct{}{}
		}
		close(reqChan)
	}

	// Start workers
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)

		// Ramp up gradually
		if lt.config.RampUpDuration > 0 {
			delay := time.Duration(i) * lt.config.RampUpDuration / time.Duration(lt.config.Concurrency)
			time.Sleep(delay)
		}

		go func() {
			defer wg.Done()

			for range reqChan {
				// Rate limiting
				if rateLimiter != nil {
					<-rateLimiter
				}

				// Make request
				reqCtx, cancel := context.WithTimeout(ctx, lt.config.RequestTimeout)

				reqStart := time.Now()
				err := fn(reqCtx)
				latency := time.Since(reqStart)

				cancel()

				// Record results
				atomic.AddInt64(&totalReqs, 1)
				if err != nil {
					atomic.AddInt64(&failedReqs, 1)
				} else {
					atomic.AddInt64(&successReqs, 1)
				}

				// Record latency
				lt.mu.Lock()
				lt.latencies = append(lt.latencies, latency)
				if latency < lt.results.MinLatency {
					lt.results.MinLatency = latency
				}
				if latency > lt.results.MaxLatency {
					lt.results.MaxLatency = latency
				}
				lt.mu.Unlock()
			}
		}()
	}

	// Wait for completion
	wg.Wait()
	totalDuration := time.Since(startTime)

	// Calculate statistics
	lt.results.TotalRequests = totalReqs
	lt.results.SuccessfulReqs = successReqs
	lt.results.FailedReqs = failedReqs
	lt.results.TotalDuration = totalDuration
	lt.results.RequestsPerSecond = float64(totalReqs) / totalDuration.Seconds()
	if totalReqs > 0 {
		lt.results.ErrorRate = float64(failedReqs) / float64(totalReqs) * 100
	}

	// Calculate latency percentiles
	lt.calculatePercentiles()

	return lt.results, nil
}

// calculatePercentiles calculates latency percentiles
// Uses sort.Slice for O(n log n) performance instead of O(n²) bubble sort
func (lt *LoadTester) calculatePercentiles() {
	if len(lt.latencies) == 0 {
		return
	}

	// Sort latencies - O(n log n) using Go's optimized sort
	sortedLatencies := make([]time.Duration, len(lt.latencies))
	copy(sortedLatencies, lt.latencies)

	sort.Slice(sortedLatencies, func(i, j int) bool {
		return sortedLatencies[i] < sortedLatencies[j]
	})

	// Calculate average
	var total time.Duration
	for _, lat := range sortedLatencies {
		total += lat
	}
	lt.results.AvgLatency = total / time.Duration(len(sortedLatencies))

	// Calculate percentiles
	p50Idx := int(float64(len(sortedLatencies)) * 0.50)
	p95Idx := int(float64(len(sortedLatencies)) * 0.95)
	p99Idx := int(float64(len(sortedLatencies)) * 0.99)

	if p50Idx < len(sortedLatencies) {
		lt.results.P50Latency = sortedLatencies[p50Idx]
	}
	if p95Idx < len(sortedLatencies) {
		lt.results.P95Latency = sortedLatencies[p95Idx]
	}
	if p99Idx < len(sortedLatencies) {
		lt.results.P99Latency = sortedLatencies[p99Idx]
	}
}

// PrintResults prints the test results in a formatted way
func (r *LoadTestResult) PrintResults() {
	fmt.Println("╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║           Load Test Results                           ║")
	fmt.Println("╠═══════════════════════════════════════════════════════╣")
	fmt.Printf("║ Total Requests:        %10d                     ║\n", r.TotalRequests)
	if r.TotalRequests > 0 {
		fmt.Printf("║ Successful:            %10d (%.1f%%)              ║\n", r.SuccessfulReqs, float64(r.SuccessfulReqs)/float64(r.TotalRequests)*100)
	} else {
		fmt.Printf("║ Successful:            %10d (0.0%%)              ║\n", r.SuccessfulReqs)
	}
	fmt.Printf("║ Failed:                %10d (%.1f%%)              ║\n", r.FailedReqs, r.ErrorRate)
	fmt.Printf("║ Duration:              %10s                     ║\n", r.TotalDuration.Round(time.Millisecond))
	fmt.Printf("║ Requests/sec:          %10.2f                     ║\n", r.RequestsPerSecond)
	fmt.Println("╠═══════════════════════════════════════════════════════╣")
	fmt.Println("║           Latency Statistics                          ║")
	fmt.Println("╠═══════════════════════════════════════════════════════╣")
	fmt.Printf("║ Min:                   %10s                     ║\n", r.MinLatency.Round(time.Millisecond))
	fmt.Printf("║ Average:               %10s                     ║\n", r.AvgLatency.Round(time.Millisecond))
	fmt.Printf("║ Max:                   %10s                     ║\n", r.MaxLatency.Round(time.Millisecond))
	fmt.Printf("║ P50:                   %10s                     ║\n", r.P50Latency.Round(time.Millisecond))
	fmt.Printf("║ P95:                   %10s                     ║\n", r.P95Latency.Round(time.Millisecond))
	fmt.Printf("║ P99:                   %10s                     ║\n", r.P99Latency.Round(time.Millisecond))
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
}

// Example load test scenarios

// BasicLoadTest runs a simple load test
func BasicLoadTest(ctx context.Context, fn RequestFunc) (*LoadTestResult, error) {
	config := LoadTestConfig{
		Concurrency:    10,
		TotalRequests:  1000,
		RequestTimeout: 10 * time.Second,
	}

	tester := NewLoadTester(config)
	return tester.Run(ctx, fn)
}

// StressTest runs a high-concurrency stress test
func StressTest(ctx context.Context, fn RequestFunc) (*LoadTestResult, error) {
	config := LoadTestConfig{
		Concurrency:    100,
		TotalRequests:  10000,
		RequestTimeout: 30 * time.Second,
		RampUpDuration: 10 * time.Second,
	}

	tester := NewLoadTester(config)
	return tester.Run(ctx, fn)
}

// SpikeTest runs a test with sudden traffic spike
func SpikeTest(ctx context.Context, fn RequestFunc) (*LoadTestResult, error) {
	config := LoadTestConfig{
		Concurrency:    200,
		TotalRequests:  5000,
		RequestTimeout: 15 * time.Second,
		// No ramp up - instant spike
	}

	tester := NewLoadTester(config)
	return tester.Run(ctx, fn)
}

// EnduranceTest runs a long-duration test
func EnduranceTest(ctx context.Context, fn RequestFunc) (*LoadTestResult, error) {
	config := LoadTestConfig{
		Concurrency:    25,
		Duration:       30 * time.Minute,
		RequestTimeout: 20 * time.Second,
		TargetRPS:      50, // Sustained 50 RPS
	}

	tester := NewLoadTester(config)
	return tester.Run(ctx, fn)
}
