package metering

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Default configuration values
const (
	DefaultMaxEvents       = 100000           // Maximum events per slice before forced cleanup
	DefaultRetentionPeriod = 30 * 24 * time.Hour // 30 days retention
	DefaultCleanupInterval = 1 * time.Hour    // Cleanup check interval
)

// UsageCollector collects and tracks resource usage for cost management
type UsageCollector interface {
	// RecordLLMUsage records LLM API usage
	RecordLLMUsage(ctx context.Context, event LLMUsageEvent) error

	// RecordComputeUsage records compute resource usage
	RecordComputeUsage(ctx context.Context, event ComputeUsageEvent) error

	// RecordStorageUsage records storage usage
	RecordStorageUsage(ctx context.Context, event StorageUsageEvent) error

	// GetUsageReport generates a usage report
	GetUsageReport(ctx context.Context, filters UsageFilters) (*UsageReport, error)

	// GetCostReport generates a cost report
	GetCostReport(ctx context.Context, filters UsageFilters) (*CostReport, error)

	// Cleanup removes old events based on retention policy
	Cleanup(ctx context.Context) error

	// GetStats returns collector statistics
	GetStats() CollectorStats

	// Close stops background cleanup and releases resources
	Close() error
}

// CollectorConfig configures the usage collector
type CollectorConfig struct {
	MaxEvents       int           // Maximum events per type before forced cleanup
	RetentionPeriod time.Duration // How long to keep events
	CleanupInterval time.Duration // How often to run cleanup
	EnableAutoCleanup bool        // Whether to run background cleanup
}

// DefaultCollectorConfig returns the default configuration
func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		MaxEvents:       DefaultMaxEvents,
		RetentionPeriod: DefaultRetentionPeriod,
		CleanupInterval: DefaultCleanupInterval,
		EnableAutoCleanup: true,
	}
}

// CollectorStats provides statistics about the collector
type CollectorStats struct {
	LLMEventCount     int
	ComputeEventCount int
	StorageEventCount int
	OldestEvent       time.Time
	NewestEvent       time.Time
	LastCleanup       time.Time
	TotalCleanups     int64
	EventsRemoved     int64
}

// LLMUsageEvent represents an LLM API call
type LLMUsageEvent struct {
	Timestamp        time.Time
	UserID           string
	WorkspaceID      string
	ThreadID         string
	ModelProvider    string // "openai", "anthropic", etc.
	ModelName        string // "gpt-4", "claude-3-opus", etc.
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CostUSD          float64
	Latency          time.Duration
	Success          bool
	Error            string
}

// ComputeUsageEvent represents compute resource usage
type ComputeUsageEvent struct {
	Timestamp       time.Time
	UserID          string
	WorkspaceID     string
	ResourceType    string // "mcp-server", "knowledge-ingestion", etc.
	ResourceID      string
	CPUSeconds      float64
	MemoryGBSeconds float64
	CostUSD         float64
	Success         bool
}

// StorageUsageEvent represents storage usage
type StorageUsageEvent struct {
	Timestamp   time.Time
	UserID      string
	WorkspaceID string
	StorageType string // "knowledge-files", "audit-logs", etc.
	Bytes       int64
	CostUSD     float64
}

// UsageFilters filters usage queries
type UsageFilters struct {
	StartTime    time.Time
	EndTime      time.Time
	UserID       string
	WorkspaceID  string
	ResourceType string
}

// UsageReport summarizes usage statistics
type UsageReport struct {
	Period         Period
	TotalRequests  int64
	TotalTokens    int64
	TotalCostUSD   float64
	ByUser         map[string]UserUsage
	ByWorkspace    map[string]WorkspaceUsage
	ByModel        map[string]ModelUsage
	ByResourceType map[string]ResourceUsage
}

// Period represents a time period
type Period struct {
	Start time.Time
	End   time.Time
}

// UserUsage represents usage by a specific user
type UserUsage struct {
	UserID        string
	TotalRequests int64
	TotalTokens   int64
	TotalCostUSD  float64
	ByModel       map[string]ModelUsage
}

// WorkspaceUsage represents usage by a workspace
type WorkspaceUsage struct {
	WorkspaceID   string
	TotalRequests int64
	TotalTokens   int64
	TotalCostUSD  float64
	ByUser        map[string]UserUsage
}

// ModelUsage represents usage of a specific model
type ModelUsage struct {
	ModelProvider string
	ModelName     string
	RequestCount  int64
	TotalTokens   int64
	TotalCostUSD  float64
	AvgLatency    time.Duration
}

// ResourceUsage represents usage by resource type
type ResourceUsage struct {
	ResourceType    string
	Count           int64
	CPUSeconds      float64
	MemoryGBSeconds float64
	StorageBytes    int64
	TotalCostUSD    float64
}

// CostReport provides cost breakdown and analysis
type CostReport struct {
	Period          Period
	TotalCostUSD    float64
	CostByCategory  map[string]float64
	CostByUser      map[string]float64
	CostByWorkspace map[string]float64
	TopCostDrivers  []CostDriver
	BudgetAlerts    []BudgetAlert
	CostTrend       []TrendPoint
	Recommendations []CostRecommendation
}

// CostDriver identifies major cost contributors
type CostDriver struct {
	Category    string
	Description string
	CostUSD     float64
	Percentage  float64
}

// BudgetAlert represents a budget threshold alert
type BudgetAlert struct {
	Severity     string // "warning", "critical"
	Message      string
	CurrentSpend float64
	BudgetLimit  float64
	Percentage   float64
}

// TrendPoint represents a point in the cost trend
type TrendPoint struct {
	Timestamp time.Time
	CostUSD   float64
}

// CostRecommendation suggests ways to reduce costs
type CostRecommendation struct {
	Priority    string // "high", "medium", "low"
	Title       string
	Description string
	Savings     float64
}

// usageCollector implements UsageCollector
type usageCollector struct {
	llmEvents     []LLMUsageEvent
	computeEvents []ComputeUsageEvent
	storageEvents []StorageUsageEvent
	calculator    *CostCalculator
	config        CollectorConfig
	mu            sync.RWMutex

	// Statistics
	lastCleanup   time.Time
	totalCleanups int64
	eventsRemoved int64

	// Background cleanup
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewUsageCollector creates a new usage collector with default configuration
func NewUsageCollector() UsageCollector {
	return NewUsageCollectorWithConfig(DefaultCollectorConfig())
}

// NewUsageCollectorWithConfig creates a new usage collector with custom configuration
func NewUsageCollectorWithConfig(config CollectorConfig) UsageCollector {
	c := &usageCollector{
		llmEvents:     make([]LLMUsageEvent, 0, 1000),      // Pre-allocate with reasonable capacity
		computeEvents: make([]ComputeUsageEvent, 0, 1000),
		storageEvents: make([]StorageUsageEvent, 0, 1000),
		calculator:    NewCostCalculator(),
		config:        config,
		stopCh:        make(chan struct{}),
	}

	// Start background cleanup if enabled
	if config.EnableAutoCleanup && config.CleanupInterval > 0 {
		c.wg.Add(1)
		go c.backgroundCleanup()
	}

	return c
}

// backgroundCleanup runs periodic cleanup in the background
func (c *usageCollector) backgroundCleanup() {
	defer c.wg.Done()
	
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = c.Cleanup(context.Background())
		case <-c.stopCh:
			return
		}
	}
}

// Close stops background cleanup and releases resources
func (c *usageCollector) Close() error {
	close(c.stopCh)
	c.wg.Wait()
	return nil
}

// RecordLLMUsage records an LLM usage event
func (c *usageCollector) RecordLLMUsage(ctx context.Context, event LLMUsageEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Calculate cost if not provided
	if event.CostUSD == 0 {
		cost, err := c.calculator.CalculateLLMCost(event.ModelProvider, event.ModelName, event.PromptTokens, event.CompletionTokens)
		if err != nil {
			return fmt.Errorf("failed to calculate cost: %w", err)
		}
		event.CostUSD = cost
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to force cleanup due to max events
	if len(c.llmEvents) >= c.config.MaxEvents {
		c.cleanupLLMEventsLocked()
	}

	c.llmEvents = append(c.llmEvents, event)
	return nil
}

// RecordComputeUsage records a compute usage event
func (c *usageCollector) RecordComputeUsage(ctx context.Context, event ComputeUsageEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Calculate cost if not provided
	if event.CostUSD == 0 {
		cost := c.calculator.CalculateComputeCost(event.CPUSeconds, event.MemoryGBSeconds)
		event.CostUSD = cost
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to force cleanup due to max events
	if len(c.computeEvents) >= c.config.MaxEvents {
		c.cleanupComputeEventsLocked()
	}

	c.computeEvents = append(c.computeEvents, event)
	return nil
}

// RecordStorageUsage records a storage usage event
func (c *usageCollector) RecordStorageUsage(ctx context.Context, event StorageUsageEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Calculate cost if not provided
	if event.CostUSD == 0 {
		cost := c.calculator.CalculateStorageCost(event.Bytes)
		event.CostUSD = cost
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to force cleanup due to max events
	if len(c.storageEvents) >= c.config.MaxEvents {
		c.cleanupStorageEventsLocked()
	}

	c.storageEvents = append(c.storageEvents, event)
	return nil
}

// Cleanup removes old events based on retention policy
func (c *usageCollector) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cleanupLLMEventsLocked()
	c.cleanupComputeEventsLocked()
	c.cleanupStorageEventsLocked()

	c.lastCleanup = time.Now()
	c.totalCleanups++

	return nil
}

// cleanupLLMEventsLocked removes old LLM events (must be called with lock held)
func (c *usageCollector) cleanupLLMEventsLocked() {
	if len(c.llmEvents) == 0 {
		return
	}

	cutoff := time.Now().Add(-c.config.RetentionPeriod)
	originalLen := len(c.llmEvents)

	// Filter events to keep only those within retention period
	filtered := make([]LLMUsageEvent, 0, len(c.llmEvents)/2)
	for _, event := range c.llmEvents {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}

	c.llmEvents = filtered
	removed := originalLen - len(filtered)
	c.eventsRemoved += int64(removed)
}

// cleanupComputeEventsLocked removes old compute events (must be called with lock held)
func (c *usageCollector) cleanupComputeEventsLocked() {
	if len(c.computeEvents) == 0 {
		return
	}

	cutoff := time.Now().Add(-c.config.RetentionPeriod)
	originalLen := len(c.computeEvents)

	// Filter events to keep only those within retention period
	filtered := make([]ComputeUsageEvent, 0, len(c.computeEvents)/2)
	for _, event := range c.computeEvents {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}

	c.computeEvents = filtered
	removed := originalLen - len(filtered)
	c.eventsRemoved += int64(removed)
}

// cleanupStorageEventsLocked removes old storage events (must be called with lock held)
func (c *usageCollector) cleanupStorageEventsLocked() {
	if len(c.storageEvents) == 0 {
		return
	}

	cutoff := time.Now().Add(-c.config.RetentionPeriod)
	originalLen := len(c.storageEvents)

	// Filter events to keep only those within retention period
	filtered := make([]StorageUsageEvent, 0, len(c.storageEvents)/2)
	for _, event := range c.storageEvents {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}

	c.storageEvents = filtered
	removed := originalLen - len(filtered)
	c.eventsRemoved += int64(removed)
}

// GetStats returns collector statistics
func (c *usageCollector) GetStats() CollectorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CollectorStats{
		LLMEventCount:     len(c.llmEvents),
		ComputeEventCount: len(c.computeEvents),
		StorageEventCount: len(c.storageEvents),
		LastCleanup:       c.lastCleanup,
		TotalCleanups:     c.totalCleanups,
		EventsRemoved:     c.eventsRemoved,
	}

	// Find oldest and newest events
	var oldest, newest time.Time

	for _, e := range c.llmEvents {
		if oldest.IsZero() || e.Timestamp.Before(oldest) {
			oldest = e.Timestamp
		}
		if newest.IsZero() || e.Timestamp.After(newest) {
			newest = e.Timestamp
		}
	}

	for _, e := range c.computeEvents {
		if oldest.IsZero() || e.Timestamp.Before(oldest) {
			oldest = e.Timestamp
		}
		if newest.IsZero() || e.Timestamp.After(newest) {
			newest = e.Timestamp
		}
	}

	for _, e := range c.storageEvents {
		if oldest.IsZero() || e.Timestamp.Before(oldest) {
			oldest = e.Timestamp
		}
		if newest.IsZero() || e.Timestamp.After(newest) {
			newest = e.Timestamp
		}
	}

	stats.OldestEvent = oldest
	stats.NewestEvent = newest

	return stats
}

// GetUsageReport generates a usage report
func (c *usageCollector) GetUsageReport(ctx context.Context, filters UsageFilters) (*UsageReport, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	report := &UsageReport{
		Period: Period{
			Start: filters.StartTime,
			End:   filters.EndTime,
		},
		ByUser:         make(map[string]UserUsage),
		ByWorkspace:    make(map[string]WorkspaceUsage),
		ByModel:        make(map[string]ModelUsage),
		ByResourceType: make(map[string]ResourceUsage),
	}

	// Process LLM events
	for _, event := range c.llmEvents {
		if !c.matchesFilters(event, filters) {
			continue
		}

		report.TotalRequests++
		report.TotalTokens += event.TotalTokens
		report.TotalCostUSD += event.CostUSD

		// Aggregate by user
		if event.UserID != "" {
			usage := report.ByUser[event.UserID]
			usage.UserID = event.UserID
			usage.TotalRequests++
			usage.TotalTokens += event.TotalTokens
			usage.TotalCostUSD += event.CostUSD
			report.ByUser[event.UserID] = usage
		}

		// Aggregate by workspace
		if event.WorkspaceID != "" {
			usage := report.ByWorkspace[event.WorkspaceID]
			usage.WorkspaceID = event.WorkspaceID
			usage.TotalRequests++
			usage.TotalTokens += event.TotalTokens
			usage.TotalCostUSD += event.CostUSD
			report.ByWorkspace[event.WorkspaceID] = usage
		}

		// Aggregate by model
		modelKey := fmt.Sprintf("%s/%s", event.ModelProvider, event.ModelName)
		usage := report.ByModel[modelKey]
		usage.ModelProvider = event.ModelProvider
		usage.ModelName = event.ModelName
		usage.RequestCount++
		usage.TotalTokens += event.TotalTokens
		usage.TotalCostUSD += event.CostUSD
		report.ByModel[modelKey] = usage
	}

	// Process compute events
	for _, event := range c.computeEvents {
		if !filters.StartTime.IsZero() && event.Timestamp.Before(filters.StartTime) {
			continue
		}
		if !filters.EndTime.IsZero() && event.Timestamp.After(filters.EndTime) {
			continue
		}

		usage := report.ByResourceType[event.ResourceType]
		usage.ResourceType = event.ResourceType
		usage.Count++
		usage.CPUSeconds += event.CPUSeconds
		usage.MemoryGBSeconds += event.MemoryGBSeconds
		usage.TotalCostUSD += event.CostUSD
		report.ByResourceType[event.ResourceType] = usage

		report.TotalCostUSD += event.CostUSD
	}

	return report, nil
}

// GetCostReport generates a cost report with analysis
func (c *usageCollector) GetCostReport(ctx context.Context, filters UsageFilters) (*CostReport, error) {
	usageReport, err := c.GetUsageReport(ctx, filters)
	if err != nil {
		return nil, err
	}

	report := &CostReport{
		Period:          usageReport.Period,
		TotalCostUSD:    usageReport.TotalCostUSD,
		CostByCategory:  make(map[string]float64),
		CostByUser:      make(map[string]float64),
		CostByWorkspace: make(map[string]float64),
	}

	// Calculate costs by category (use read lock for iteration)
	c.mu.RLock()
	report.CostByCategory["LLM"] = 0
	report.CostByCategory["Compute"] = 0
	report.CostByCategory["Storage"] = 0

	for _, event := range c.llmEvents {
		if c.matchesFilters(event, filters) {
			report.CostByCategory["LLM"] += event.CostUSD
		}
	}

	for _, event := range c.computeEvents {
		if c.matchesComputeFilters(event, filters) {
			report.CostByCategory["Compute"] += event.CostUSD
		}
	}

	for _, event := range c.storageEvents {
		if c.matchesStorageFilters(event, filters) {
			report.CostByCategory["Storage"] += event.CostUSD
		}
	}
	c.mu.RUnlock()

	// Aggregate by user and workspace
	for userID, usage := range usageReport.ByUser {
		report.CostByUser[userID] = usage.TotalCostUSD
	}

	for wsID, usage := range usageReport.ByWorkspace {
		report.CostByWorkspace[wsID] = usage.TotalCostUSD
	}

	// Identify top cost drivers
	report.TopCostDrivers = c.identifyTopCostDrivers(report)

	// Generate recommendations
	report.Recommendations = c.generateRecommendations(usageReport)

	return report, nil
}

// matchesFilters checks if an LLM event matches the filters
func (c *usageCollector) matchesFilters(event LLMUsageEvent, filters UsageFilters) bool {
	if !filters.StartTime.IsZero() && event.Timestamp.Before(filters.StartTime) {
		return false
	}
	if !filters.EndTime.IsZero() && event.Timestamp.After(filters.EndTime) {
		return false
	}
	if filters.UserID != "" && event.UserID != filters.UserID {
		return false
	}
	if filters.WorkspaceID != "" && event.WorkspaceID != filters.WorkspaceID {
		return false
	}
	return true
}

func (c *usageCollector) matchesComputeFilters(event ComputeUsageEvent, filters UsageFilters) bool {
	if !filters.StartTime.IsZero() && event.Timestamp.Before(filters.StartTime) {
		return false
	}
	if !filters.EndTime.IsZero() && event.Timestamp.After(filters.EndTime) {
		return false
	}
	return true
}

func (c *usageCollector) matchesStorageFilters(event StorageUsageEvent, filters UsageFilters) bool {
	if !filters.StartTime.IsZero() && event.Timestamp.Before(filters.StartTime) {
		return false
	}
	if !filters.EndTime.IsZero() && event.Timestamp.After(filters.EndTime) {
		return false
	}
	return true
}

// identifyTopCostDrivers identifies the top cost drivers
// Uses sort.Slice for O(n log n) performance instead of O(n²) bubble sort
func (c *usageCollector) identifyTopCostDrivers(report *CostReport) []CostDriver {
	drivers := []CostDriver{}

	for category, cost := range report.CostByCategory {
		if cost > 0 {
			percentage := 0.0
			if report.TotalCostUSD > 0 {
				percentage = (cost / report.TotalCostUSD) * 100
			}
			drivers = append(drivers, CostDriver{
				Category:    category,
				Description: fmt.Sprintf("%s costs", category),
				CostUSD:     cost,
				Percentage:  percentage,
			})
		}
	}

	// Sort by cost descending using sort.Slice - O(n log n)
	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].CostUSD > drivers[j].CostUSD
	})

	// Return top 5
	if len(drivers) > 5 {
		drivers = drivers[:5]
	}

	return drivers
}

// generateRecommendations generates cost optimization recommendations
func (c *usageCollector) generateRecommendations(report *UsageReport) []CostRecommendation {
	recommendations := []CostRecommendation{}

	// Check for expensive models
	for _, modelUsage := range report.ByModel {
		if modelUsage.TotalCostUSD > 1000 && modelUsage.RequestCount > 0 {
			avgCost := modelUsage.TotalCostUSD / float64(modelUsage.RequestCount)
			if avgCost > 0.50 {
				recommendations = append(recommendations, CostRecommendation{
					Priority:    "high",
					Title:       fmt.Sprintf("Consider using a less expensive model than %s", modelUsage.ModelName),
					Description: fmt.Sprintf("Average cost per request: $%.4f", avgCost),
					Savings:     modelUsage.TotalCostUSD * 0.3, // Estimate 30% savings
				})
			}
		}
	}

	// Check for high-usage users
	for userID, usage := range report.ByUser {
		if usage.TotalCostUSD > 500 {
			recommendations = append(recommendations, CostRecommendation{
				Priority:    "medium",
				Title:       fmt.Sprintf("Review usage patterns for user %s", userID),
				Description: fmt.Sprintf("High cost: $%.2f", usage.TotalCostUSD),
				Savings:     usage.TotalCostUSD * 0.2, // Estimate 20% savings
			})
		}
	}

	return recommendations
}
