package metering

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
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
	mu            sync.RWMutex
}

// NewUsageCollector creates a new usage collector
func NewUsageCollector() UsageCollector {
	return &usageCollector{
		llmEvents:     []LLMUsageEvent{},
		computeEvents: []ComputeUsageEvent{},
		storageEvents: []StorageUsageEvent{},
		calculator:    NewCostCalculator(),
	}
}

// RecordLLMUsage records an LLM usage event
func (c *usageCollector) RecordLLMUsage(ctx context.Context, event LLMUsageEvent) error {
	// Calculate cost if not provided
	if event.CostUSD == 0 {
		cost, err := c.calculator.CalculateLLMCost(event.ModelProvider, event.ModelName, event.PromptTokens, event.CompletionTokens)
		if err != nil {
			return fmt.Errorf("failed to calculate cost: %w", err)
		}
		event.CostUSD = cost
	}

	c.mu.Lock()
	c.llmEvents = append(c.llmEvents, event)
	c.mu.Unlock()

	return nil
}

// RecordComputeUsage records a compute usage event
func (c *usageCollector) RecordComputeUsage(ctx context.Context, event ComputeUsageEvent) error {
	// Calculate cost if not provided
	if event.CostUSD == 0 {
		cost := c.calculator.CalculateComputeCost(event.CPUSeconds, event.MemoryGBSeconds)
		event.CostUSD = cost
	}

	c.mu.Lock()
	c.computeEvents = append(c.computeEvents, event)
	c.mu.Unlock()

	return nil
}

// RecordStorageUsage records a storage usage event
func (c *usageCollector) RecordStorageUsage(ctx context.Context, event StorageUsageEvent) error {
	// Calculate cost if not provided
	if event.CostUSD == 0 {
		cost := c.calculator.CalculateStorageCost(event.Bytes)
		event.CostUSD = cost
	}

	c.mu.Lock()
	c.storageEvents = append(c.storageEvents, event)
	c.mu.Unlock()

	return nil
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

	// Calculate costs by category
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
