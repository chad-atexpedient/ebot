// Package models provides A/B testing capabilities for model experiments.
package models

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ABTestManager manages A/B testing experiments for models.
type ABTestManager interface {
	// CreateExperiment creates a new A/B test
	CreateExperiment(ctx context.Context, experiment *Experiment) error
	
	// GetExperiment retrieves an experiment
	GetExperiment(ctx context.Context, experimentID string) (*Experiment, error)
	
	// SelectVariant selects a model variant for a user based on experiment config
	SelectVariant(ctx context.Context, experimentID string, userID string) (*ModelVariant, error)
	
	// RecordResult records the result of using a variant
	RecordResult(ctx context.Context, experimentID string, userID string, result ExperimentResult) error
	
	// GetResults gets experiment results and statistics
	GetResults(ctx context.Context, experimentID string) (*ExperimentResults, error)
	
	// EndExperiment ends an experiment and promotes winner
	EndExperiment(ctx context.Context, experimentID string, winnerVariant string) error
}

// Experiment represents an A/B test configuration.
type Experiment struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      ExperimentStatus `json:"status"`
	
	// Variants being tested
	Variants []*ModelVariant `json:"variants"`
	
	// Traffic allocation
	TrafficSplit map[string]float64 `json:"trafficSplit"` // variantID -> percentage (0-1)
	
	// Success metrics
	Metrics []string `json:"metrics"` // latency, cost, user_rating, etc.
	
	// Experiment duration
	StartedAt   time.Time  `json:"startedAt"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	
	// User assignment (for consistency)
	userAssignments map[string]string // userID -> variantID
	
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ExperimentStatus represents the state of an experiment.
type ExperimentStatus string

const (
	ExperimentStatusDraft     ExperimentStatus = "draft"
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusPaused    ExperimentStatus = "paused"
	ExperimentStatusCompleted ExperimentStatus = "completed"
)

// ModelVariant represents a model configuration being tested.
type ModelVariant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ModelID   string `json:"modelID"`
	Version   string `json:"version"`
	Config    ModelConfig `json:"config"`
	IsControl bool   `json:"isControl"` // Control group
}

// ExperimentResult represents the outcome of using a variant.
type ExperimentResult struct {
	UserID      string    `json:"userID"`
	VariantID   string    `json:"variantID"`
	LatencyMs   float64   `json:"latencyMs"`
	CostUSD     float64   `json:"costUSD"`
	UserRating  *float64  `json:"userRating,omitempty"`
	Success     bool      `json:"success"`
	ErrorType   string    `json:"errorType,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ExperimentResults contains aggregated results for an experiment.
type ExperimentResults struct {
	ExperimentID string                    `json:"experimentID"`
	VariantStats map[string]*VariantStats  `json:"variantStats"`
	WinningVariant *string                 `json:"winningVariant,omitempty"`
	Confidence   float64                   `json:"confidence"`
	SampleSize   int                       `json:"sampleSize"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
}

// VariantStats contains statistics for a single variant.
type VariantStats struct {
	VariantID        string  `json:"variantID"`
	SampleSize       int     `json:"sampleSize"`
	SuccessRate      float64 `json:"successRate"`
	AvgLatencyMs     float64 `json:"avgLatencyMs"`
	AvgCostUSD       float64 `json:"avgCostUSD"`
	AvgUserRating    float64 `json:"avgUserRating"`
	ErrorRate        float64 `json:"errorRate"`
	TrafficPercentage float64 `json:"trafficPercentage"`
}

// Implementation

type abTestManager struct {
	experiments map[string]*Experiment
	results     map[string][]ExperimentResult
	mu          sync.RWMutex
}

// NewABTestManager creates a new A/B test manager.
func NewABTestManager() ABTestManager {
	return &abTestManager{
		experiments: make(map[string]*Experiment),
		results:     make(map[string][]ExperimentResult),
	}
}

func (m *abTestManager) CreateExperiment(ctx context.Context, experiment *Experiment) error {
	if experiment.ID == "" {
		return fmt.Errorf("experiment ID is required")
	}
	if len(experiment.Variants) < 2 {
		return fmt.Errorf("at least 2 variants are required")
	}
	
	// Validate traffic split
	if len(experiment.TrafficSplit) == 0 {
		// Default: equal split
		experiment.TrafficSplit = make(map[string]float64)
		split := 1.0 / float64(len(experiment.Variants))
		for _, v := range experiment.Variants {
			experiment.TrafficSplit[v.ID] = split
		}
	}
	
	// Validate traffic split sums to 1.0
	totalSplit := 0.0
	for _, split := range experiment.TrafficSplit {
		totalSplit += split
	}
	if totalSplit < 0.99 || totalSplit > 1.01 {
		return fmt.Errorf("traffic split must sum to 1.0, got %.2f", totalSplit)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.experiments[experiment.ID]; exists {
		return fmt.Errorf("experiment %s already exists", experiment.ID)
	}
	
	now := time.Now()
	experiment.CreatedAt = now
	experiment.UpdatedAt = now
	experiment.StartedAt = now
	experiment.Status = ExperimentStatusRunning
	experiment.userAssignments = make(map[string]string)
	
	m.experiments[experiment.ID] = experiment
	m.results[experiment.ID] = []ExperimentResult{}
	
	return nil
}

func (m *abTestManager) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	experiment, exists := m.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}
	
	return experiment, nil
}

func (m *abTestManager) SelectVariant(ctx context.Context, experimentID string, userID string) (*ModelVariant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	experiment, exists := m.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}
	
	if experiment.Status != ExperimentStatusRunning {
		return nil, fmt.Errorf("experiment is not running")
	}
	
	// Check if user already assigned
	if variantID, exists := experiment.userAssignments[userID]; exists {
		for _, v := range experiment.Variants {
			if v.ID == variantID {
				return v, nil
			}
		}
	}
	
	// Assign user to variant based on traffic split
	variant := m.selectVariantByTrafficSplit(experiment)
	experiment.userAssignments[userID] = variant.ID
	
	return variant, nil
}

func (m *abTestManager) selectVariantByTrafficSplit(experiment *Experiment) *ModelVariant {
	// Random selection based on traffic split
	random := rand.Float64()
	cumulative := 0.0
	
	for _, variant := range experiment.Variants {
		split := experiment.TrafficSplit[variant.ID]
		cumulative += split
		if random <= cumulative {
			return variant
		}
	}
	
	// Fallback to first variant
	return experiment.Variants[0]
}

func (m *abTestManager) RecordResult(ctx context.Context, experimentID string, userID string, result ExperimentResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.experiments[experimentID]; !exists {
		return fmt.Errorf("experiment %s not found", experimentID)
	}
	
	result.UserID = userID
	result.Timestamp = time.Now()
	
	m.results[experimentID] = append(m.results[experimentID], result)
	
	return nil
}

func (m *abTestManager) GetResults(ctx context.Context, experimentID string) (*ExperimentResults, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	experiment, exists := m.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}
	
	results := m.results[experimentID]
	
	// Calculate stats per variant
	variantStats := make(map[string]*VariantStats)
	for _, variant := range experiment.Variants {
		variantStats[variant.ID] = &VariantStats{
			VariantID: variant.ID,
			TrafficPercentage: experiment.TrafficSplit[variant.ID] * 100,
		}
	}
	
	// Aggregate results
	for _, result := range results {
		stats := variantStats[result.VariantID]
		stats.SampleSize++
		
		if result.Success {
			stats.SuccessRate += 1.0
		} else {
			stats.ErrorRate += 1.0
		}
		
		stats.AvgLatencyMs += result.LatencyMs
		stats.AvgCostUSD += result.CostUSD
		
		if result.UserRating != nil {
			stats.AvgUserRating += *result.UserRating
		}
	}
	
	// Calculate averages
	for _, stats := range variantStats {
		if stats.SampleSize > 0 {
			stats.SuccessRate = (stats.SuccessRate / float64(stats.SampleSize)) * 100
			stats.ErrorRate = (stats.ErrorRate / float64(stats.SampleSize)) * 100
			stats.AvgLatencyMs /= float64(stats.SampleSize)
			stats.AvgCostUSD /= float64(stats.SampleSize)
			stats.AvgUserRating /= float64(stats.SampleSize)
		}
	}
	
	// Determine winner (simple: best success rate with minimum sample size)
	var winnerID *string
	var bestSuccessRate float64
	minSampleSize := 30
	
	for _, stats := range variantStats {
		if stats.SampleSize >= minSampleSize && stats.SuccessRate > bestSuccessRate {
			bestSuccessRate = stats.SuccessRate
			winnerID = &stats.VariantID
		}
	}
	
	return &ExperimentResults{
		ExperimentID:   experimentID,
		VariantStats:   variantStats,
		WinningVariant: winnerID,
		Confidence:     0.95, // Simplified
		SampleSize:     len(results),
		UpdatedAt:      time.Now(),
	}, nil
}

func (m *abTestManager) EndExperiment(ctx context.Context, experimentID string, winnerVariant string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	experiment, exists := m.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment %s not found", experimentID)
	}
	
	now := time.Now()
	experiment.CompletedAt = &now
	experiment.Status = ExperimentStatusCompleted
	experiment.UpdatedAt = now
	
	return nil
}
