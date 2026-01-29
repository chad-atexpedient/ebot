// Package models provides model management capabilities including versioning,
// A/B testing, performance tracking, and custom model registration.
package models

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ModelRegistry manages model versions, custom models, and deployment strategies.
type ModelRegistry interface {
	// RegisterModel registers a new model version
	RegisterModel(ctx context.Context, model *ModelVersion) error
	
	// GetModel retrieves a specific model version
	GetModel(ctx context.Context, modelID string, version string) (*ModelVersion, error)
	
	// ListModels lists all available models with optional filtering
	ListModels(ctx context.Context, filters ModelFilters) ([]*ModelVersion, error)
	
	// UpdateModel updates model configuration or metadata
	UpdateModel(ctx context.Context, modelID string, version string, updates ModelUpdates) error
	
	// DeprecateModel marks a model version as deprecated
	DeprecateModel(ctx context.Context, modelID string, version string) error
	
	// SetDefaultVersion sets the default version for a model
	SetDefaultVersion(ctx context.Context, modelID string, version string) error
	
	// GetDefaultVersion gets the default version for a model
	GetDefaultVersion(ctx context.Context, modelID string) (*ModelVersion, error)
}

// ModelVersion represents a specific version of a model.
type ModelVersion struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Provider    string            `json:"provider"` // openai, anthropic, custom, etc.
	Endpoint    string            `json:"endpoint,omitempty"`
	IsDefault   bool              `json:"isDefault"`
	Status      ModelStatus       `json:"status"`
	Capabilities []ModelCapability `json:"capabilities"`
	
	// Configuration
	Config ModelConfig `json:"config"`
	
	// Metadata
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	
	// Lifecycle
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeprecatedAt *time.Time `json:"deprecatedAt,omitempty"`
	
	// Performance tracking
	Performance *ModelPerformance `json:"performance,omitempty"`
}

// ModelStatus represents the deployment status of a model.
type ModelStatus string

const (
	ModelStatusDraft      ModelStatus = "draft"
	ModelStatusTesting    ModelStatus = "testing"
	ModelStatusActive     ModelStatus = "active"
	ModelStatusDeprecated ModelStatus = "deprecated"
	ModelStatusRetired    ModelStatus = "retired"
)

// ModelCapability represents what a model can do.
type ModelCapability string

const (
	CapabilityTextGeneration  ModelCapability = "text_generation"
	CapabilityCodeGeneration  ModelCapability = "code_generation"
	CapabilityImageGeneration ModelCapability = "image_generation"
	CapabilityVision          ModelCapability = "vision"
	CapabilityFunctionCalling ModelCapability = "function_calling"
	CapabilityStreaming       ModelCapability = "streaming"
)

// ModelConfig holds model-specific configuration.
type ModelConfig struct {
	MaxTokens       int     `json:"maxTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP,omitempty"`
	FrequencyPenalty float64 `json:"frequencyPenalty,omitempty"`
	PresencePenalty float64 `json:"presencePenalty,omitempty"`
	Timeout         int     `json:"timeout"` // seconds
	RetryAttempts   int     `json:"retryAttempts"`
}

// ModelPerformance tracks model performance metrics.
type ModelPerformance struct {
	// Usage metrics
	TotalRequests      int64   `json:"totalRequests"`
	SuccessfulRequests int64   `json:"successfulRequests"`
	FailedRequests     int64   `json:"failedRequests"`
	SuccessRate        float64 `json:"successRate"`
	
	// Latency metrics (milliseconds)
	AvgLatency float64 `json:"avgLatency"`
	P50Latency float64 `json:"p50Latency"`
	P95Latency float64 `json:"p95Latency"`
	P99Latency float64 `json:"p99Latency"`
	
	// Cost metrics
	TotalCost       float64 `json:"totalCost"`
	AvgCostPerRequest float64 `json:"avgCostPerRequest"`
	
	// Quality metrics
	AvgUserRating    float64 `json:"avgUserRating,omitempty"`
	ThumbsUpRate     float64 `json:"thumbsUpRate,omitempty"`
	
	// Token usage
	TotalTokens       int64   `json:"totalTokens"`
	TotalPromptTokens int64   `json:"totalPromptTokens"`
	TotalCompletionTokens int64 `json:"totalCompletionTokens"`
	
	LastUpdated time.Time `json:"lastUpdated"`
}

// ModelFilters for querying models.
type ModelFilters struct {
	Provider     string          `json:"provider,omitempty"`
	Status       ModelStatus     `json:"status,omitempty"`
	Capability   ModelCapability `json:"capability,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	IncludeDeprecated bool       `json:"includeDeprecated"`
}

// ModelUpdates represents updates to a model.
type ModelUpdates struct {
	Name        *string       `json:"name,omitempty"`
	Description *string       `json:"description,omitempty"`
	Status      *ModelStatus  `json:"status,omitempty"`
	Config      *ModelConfig  `json:"config,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Implementation

type modelRegistry struct {
	models map[string]map[string]*ModelVersion // modelID -> version -> ModelVersion
	defaults map[string]string // modelID -> default version
	mu sync.RWMutex
}

// NewModelRegistry creates a new model registry.
func NewModelRegistry() ModelRegistry {
	return &modelRegistry{
		models:   make(map[string]map[string]*ModelVersion),
		defaults: make(map[string]string),
	}
}

func (r *modelRegistry) RegisterModel(ctx context.Context, model *ModelVersion) error {
	if model.ID == "" {
		return fmt.Errorf("model ID is required")
	}
	if model.Version == "" {
		return fmt.Errorf("model version is required")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Initialize model versions map if needed
	if r.models[model.ID] == nil {
		r.models[model.ID] = make(map[string]*ModelVersion)
	}
	
	// Check if version already exists
	if _, exists := r.models[model.ID][model.Version]; exists {
		return fmt.Errorf("model %s version %s already exists", model.ID, model.Version)
	}
	
	// Set timestamps
	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now
	
	// Store model
	r.models[model.ID][model.Version] = model
	
	// Set as default if it's the first version or explicitly marked
	if len(r.models[model.ID]) == 1 || model.IsDefault {
		r.defaults[model.ID] = model.Version
		model.IsDefault = true
	}
	
	return nil
}

func (r *modelRegistry) GetModel(ctx context.Context, modelID string, version string) (*ModelVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	versions, exists := r.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}
	
	model, exists := versions[version]
	if !exists {
		return nil, fmt.Errorf("model %s version %s not found", modelID, version)
	}
	
	return model, nil
}

func (r *modelRegistry) ListModels(ctx context.Context, filters ModelFilters) ([]*ModelVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []*ModelVersion
	
	for _, versions := range r.models {
		for _, model := range versions {
			// Apply filters
			if filters.Provider != "" && model.Provider != filters.Provider {
				continue
			}
			if filters.Status != "" && model.Status != filters.Status {
				continue
			}
			if !filters.IncludeDeprecated && model.DeprecatedAt != nil {
				continue
			}
			if filters.Capability != "" {
				hasCapability := false
				for _, cap := range model.Capabilities {
					if cap == filters.Capability {
						hasCapability = true
						break
					}
				}
				if !hasCapability {
					continue
				}
			}
			if len(filters.Tags) > 0 {
				hasTag := false
				for _, filterTag := range filters.Tags {
					for _, modelTag := range model.Tags {
						if modelTag == filterTag {
							hasTag = true
							break
						}
					}
					if hasTag {
						break
					}
				}
				if !hasTag {
					continue
				}
			}
			
			result = append(result, model)
		}
	}
	
	return result, nil
}

func (r *modelRegistry) UpdateModel(ctx context.Context, modelID string, version string, updates ModelUpdates) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	versions, exists := r.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}
	
	model, exists := versions[version]
	if !exists {
		return fmt.Errorf("model %s version %s not found", modelID, version)
	}
	
	// Apply updates
	if updates.Name != nil {
		model.Name = *updates.Name
	}
	if updates.Description != nil {
		model.Description = *updates.Description
	}
	if updates.Status != nil {
		model.Status = *updates.Status
	}
	if updates.Config != nil {
		model.Config = *updates.Config
	}
	if updates.Tags != nil {
		model.Tags = updates.Tags
	}
	if updates.Metadata != nil {
		if model.Metadata == nil {
			model.Metadata = make(map[string]string)
		}
		for k, v := range updates.Metadata {
			model.Metadata[k] = v
		}
	}
	
	model.UpdatedAt = time.Now()
	
	return nil
}

func (r *modelRegistry) DeprecateModel(ctx context.Context, modelID string, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	versions, exists := r.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}
	
	model, exists := versions[version]
	if !exists {
		return fmt.Errorf("model %s version %s not found", modelID, version)
	}
	
	now := time.Now()
	model.DeprecatedAt = &now
	model.Status = ModelStatusDeprecated
	model.UpdatedAt = now
	
	// If this was the default, clear it
	if r.defaults[modelID] == version {
		delete(r.defaults, modelID)
	}
	
	return nil
}

func (r *modelRegistry) SetDefaultVersion(ctx context.Context, modelID string, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	versions, exists := r.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}
	
	model, exists := versions[version]
	if !exists {
		return fmt.Errorf("model %s version %s not found", modelID, version)
	}
	
	// Clear old default
	if oldDefault, exists := r.defaults[modelID]; exists {
		if oldModel, exists := versions[oldDefault]; exists {
			oldModel.IsDefault = false
		}
	}
	
	// Set new default
	r.defaults[modelID] = version
	model.IsDefault = true
	model.UpdatedAt = time.Now()
	
	return nil
}

func (r *modelRegistry) GetDefaultVersion(ctx context.Context, modelID string) (*ModelVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	defaultVersion, exists := r.defaults[modelID]
	if !exists {
		return nil, fmt.Errorf("no default version set for model %s", modelID)
	}
	
	return r.GetModel(ctx, modelID, defaultVersion)
}
