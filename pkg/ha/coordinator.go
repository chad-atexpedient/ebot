package ha

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HACoordinator manages high availability operations for ebot cluster
type HACoordinator interface {
	// IsLeader returns true if this instance is the leader
	IsLeader(ctx context.Context) (bool, error)
	
	// GetCurrentLeader returns the ID of the current leader
	GetCurrentLeader(ctx context.Context) (string, error)
	
	// RegisterHealthCheck registers a health checker for monitoring
	RegisterHealthCheck(name string, checker HealthChecker) error
	
	// GetClusterStatus returns the overall cluster status
	GetClusterStatus(ctx context.Context) (*ClusterStatus, error)
	
	// Start begins the HA coordinator
	Start(ctx context.Context) error
	
	// Stop gracefully stops the HA coordinator
	Stop(ctx context.Context) error
}

// HealthChecker defines an interface for health checks
type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
}

// HealthStatus represents the health state
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ClusterStatus represents the overall cluster state
type ClusterStatus struct {
	LeaderID       string
	Members        []MemberStatus
	OverallHealth  HealthStatus
	LastUpdated    time.Time
	HealthChecks   map[string]HealthCheckResult
}

// MemberStatus represents the status of a cluster member
type MemberStatus struct {
	ID          string
	IsLeader    bool
	LastSeen    time.Time
	Health      HealthStatus
	Version     string
}

// HealthCheckResult contains the result of a health check
type HealthCheckResult struct {
	Name      string
	Status    HealthStatus
	Error     error
	CheckedAt time.Time
	Duration  time.Duration
}

// haCoordinator implements HACoordinator
type haCoordinator struct {
	instanceID    string
	isLeader      bool
	healthChecks  map[string]HealthChecker
	checkResults  map[string]HealthCheckResult
	mu            sync.RWMutex
	stopCh        chan struct{}
	healthTicker  *time.Ticker
}

// NewHACoordinator creates a new HA coordinator
func NewHACoordinator(instanceID string) HACoordinator {
	return &haCoordinator{
		instanceID:   instanceID,
		healthChecks: make(map[string]HealthChecker),
		checkResults: make(map[string]HealthCheckResult),
		stopCh:       make(chan struct{}),
	}
}

// IsLeader returns true if this instance is the leader
func (h *haCoordinator) IsLeader(ctx context.Context) (bool, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isLeader, nil
}

// GetCurrentLeader returns the ID of the current leader
func (h *haCoordinator) GetCurrentLeader(ctx context.Context) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.isLeader {
		return h.instanceID, nil
	}
	
	// In a real implementation, this would query the leader election mechanism
	// For now, return this instance if it's the leader
	return "", fmt.Errorf("leader election not fully implemented")
}

// RegisterHealthCheck registers a health checker
func (h *haCoordinator) RegisterHealthCheck(name string, checker HealthChecker) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if _, exists := h.healthChecks[name]; exists {
		return fmt.Errorf("health check %s already registered", name)
	}
	
	h.healthChecks[name] = checker
	return nil
}

// GetClusterStatus returns the overall cluster status
func (h *haCoordinator) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// Calculate overall health based on check results
	overallHealth := h.calculateOverallHealth()
	
	status := &ClusterStatus{
		LeaderID:      h.instanceID,
		Members:       h.getMembers(),
		OverallHealth: overallHealth,
		LastUpdated:   time.Now(),
		HealthChecks:  make(map[string]HealthCheckResult),
	}
	
	// Copy health check results
	for name, result := range h.checkResults {
		status.HealthChecks[name] = result
	}
	
	return status, nil
}

// Start begins the HA coordinator
func (h *haCoordinator) Start(ctx context.Context) error {
	h.mu.Lock()
	h.healthTicker = time.NewTicker(30 * time.Second)
	h.mu.Unlock()
	
	// Start health check goroutine
	go h.runHealthChecks(ctx)
	
	// Start leader election (simplified for now)
	h.mu.Lock()
	h.isLeader = true // In production, this would use proper leader election
	h.mu.Unlock()
	
	return nil
}

// Stop gracefully stops the HA coordinator
func (h *haCoordinator) Stop(ctx context.Context) error {
	close(h.stopCh)
	
	h.mu.Lock()
	if h.healthTicker != nil {
		h.healthTicker.Stop()
	}
	h.mu.Unlock()
	
	return nil
}

// runHealthChecks runs health checks periodically
func (h *haCoordinator) runHealthChecks(ctx context.Context) {
	// Run initial checks
	h.performHealthChecks(ctx)
	
	for {
		select {
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		case <-h.healthTicker.C:
			h.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks executes all registered health checks
func (h *haCoordinator) performHealthChecks(ctx context.Context) {
	h.mu.RLock()
	checks := make(map[string]HealthChecker)
	for name, checker := range h.healthChecks {
		checks[name] = checker
	}
	h.mu.RUnlock()
	
	// Run checks in parallel
	var wg sync.WaitGroup
	resultsCh := make(chan HealthCheckResult, len(checks))
	
	for name, checker := range checks {
		wg.Add(1)
		go func(n string, c HealthChecker) {
			defer wg.Done()
			
			start := time.Now()
			err := c.Check(ctx)
			duration := time.Since(start)
			
			status := HealthStatusHealthy
			if err != nil {
				status = HealthStatusUnhealthy
			}
			
			resultsCh <- HealthCheckResult{
				Name:      n,
				Status:    status,
				Error:     err,
				CheckedAt: time.Now(),
				Duration:  duration,
			}
		}(name, checker)
	}
	
	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsCh)
	}()
	
	// Collect results
	results := make(map[string]HealthCheckResult)
	for result := range resultsCh {
		results[result.Name] = result
	}
	
	// Update stored results
	h.mu.Lock()
	h.checkResults = results
	h.mu.Unlock()
}

// calculateOverallHealth determines overall cluster health
func (h *haCoordinator) calculateOverallHealth() HealthStatus {
	if len(h.checkResults) == 0 {
		return HealthStatusHealthy
	}
	
	unhealthyCount := 0
	degradedCount := 0
	
	for _, result := range h.checkResults {
		switch result.Status {
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusDegraded:
			degradedCount++
		}
	}
	
	totalChecks := len(h.checkResults)
	
	// If more than 50% are unhealthy, cluster is unhealthy
	if float64(unhealthyCount)/float64(totalChecks) > 0.5 {
		return HealthStatusUnhealthy
	}
	
	// If any are unhealthy or degraded, cluster is degraded
	if unhealthyCount > 0 || degradedCount > 0 {
		return HealthStatusDegraded
	}
	
	return HealthStatusHealthy
}

// getMembers returns current cluster members
func (h *haCoordinator) getMembers() []MemberStatus {
	// In a real implementation, this would query the actual cluster state
	return []MemberStatus{
		{
			ID:       h.instanceID,
			IsLeader: h.isLeader,
			LastSeen: time.Now(),
			Health:   h.calculateOverallHealth(),
			Version:  "0.1.0", // TODO: Get actual version
		},
	}
}

// Common health checkers

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	name string
	ping func(context.Context) error
}

func NewDatabaseHealthChecker(ping func(context.Context) error) HealthChecker {
	return &DatabaseHealthChecker{
		name: "database",
		ping: ping,
	}
}

func (d *DatabaseHealthChecker) Check(ctx context.Context) error {
	return d.ping(ctx)
}

func (d *DatabaseHealthChecker) Name() string {
	return d.name
}

// StorageHealthChecker checks storage system health
type StorageHealthChecker struct {
	name  string
	check func(context.Context) error
}

func NewStorageHealthChecker(check func(context.Context) error) HealthChecker {
	return &StorageHealthChecker{
		name:  "storage",
		check: check,
	}
}

func (s *StorageHealthChecker) Check(ctx context.Context) error {
	return s.check(ctx)
}

func (s *StorageHealthChecker) Name() string {
	return s.name
}

// MCPHealthChecker checks MCP server health
type MCPHealthChecker struct {
	name  string
	check func(context.Context) error
}

func NewMCPHealthChecker(check func(context.Context) error) HealthChecker {
	return &MCPHealthChecker{
		name:  "mcp-servers",
		check: check,
	}
}

func (m *MCPHealthChecker) Check(ctx context.Context) error {
	return m.check(ctx)
}

func (m *MCPHealthChecker) Name() string {
	return m.name
}
