package quota

import (
	"context"
	"time"
)

// Manager defines the interface for quota management operations
type Manager interface {
	// CheckQuota verifies if a resource allocation request can be satisfied
	CheckQuota(ctx context.Context, userID, resourceType string, requested int64) error
	
	// ReserveQuota reserves resources for a user
	ReserveQuota(ctx context.Context, userID, resourceType string, amount int64) error
	
	// ReleaseQuota releases reserved resources
	ReleaseQuota(ctx context.Context, userID, resourceType string, amount int64) error
	
	// GetUsage returns current usage for a user
	GetUsage(ctx context.Context, userID string) (*ResourceUsage, error)
	
	// GetQuota returns configured quotas for a user
	GetQuota(ctx context.Context, userID string) (*ResourceQuota, error)
	
	// UpdateQuota updates quota configuration for a user
	UpdateQuota(ctx context.Context, userID string, quota *ResourceQuota) error
	
	// GetWorkspaceUsage returns usage for a workspace
	GetWorkspaceUsage(ctx context.Context, workspaceID string) (*ResourceUsage, error)
}

// ResourceQuota defines resource limits for a user or workspace
type ResourceQuota struct {
	UserID      string    `json:"userID,omitempty"`
	WorkspaceID string    `json:"workspaceID,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	
	// MCP Server limits
	MaxMCPServers int `json:"maxMCPServers,omitempty"`
	
	// Thread and conversation limits
	MaxThreads int `json:"maxThreads,omitempty"`
	MaxMessagesPerThread int `json:"maxMessagesPerThread,omitempty"`
	
	// Knowledge limits
	MaxKnowledgeSets int `json:"maxKnowledgeSets,omitempty"`
	MaxKnowledgeFiles int `json:"maxKnowledgeFiles,omitempty"`
	MaxKnowledgeFileSizeBytes int64 `json:"maxKnowledgeFileSizeBytes,omitempty"`
	MaxTotalKnowledgeSizeBytes int64 `json:"maxTotalKnowledgeSizeBytes,omitempty"`
	
	// Compute resource limits
	MaxCPUCores float64 `json:"maxCPUCores,omitempty"`
	MaxMemoryBytes int64 `json:"maxMemoryBytes,omitempty"`
	
	// Storage limits
	MaxStorageBytes int64 `json:"maxStorageBytes,omitempty"`
	
	// Rate limits
	MaxRequestsPerMinute int `json:"maxRequestsPerMinute,omitempty"`
	MaxConcurrentRequests int `json:"maxConcurrentRequests,omitempty"`
	
	// LLM-specific limits
	MaxLLMTokensPerDay int64 `json:"maxLLMTokensPerDay,omitempty"`
	MaxLLMTokensPerMonth int64 `json:"maxLLMTokensPerMonth,omitempty"`
	MaxLLMCostPerDay float64 `json:"maxLLMCostPerDay,omitempty"`
	MaxLLMCostPerMonth float64 `json:"maxLLMCostPerMonth,omitempty"`
}

// ResourceUsage tracks current resource consumption
type ResourceUsage struct {
	UserID      string    `json:"userID,omitempty"`
	WorkspaceID string    `json:"workspaceID,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	
	// MCP Server usage
	MCPServerCount int `json:"mcpServerCount"`
	
	// Thread usage
	ThreadCount int `json:"threadCount"`
	TotalMessages int `json:"totalMessages"`
	
	// Knowledge usage
	KnowledgeSetCount int `json:"knowledgeSetCount"`
	KnowledgeFileCount int `json:"knowledgeFileCount"`
	TotalKnowledgeSizeBytes int64 `json:"totalKnowledgeSizeBytes"`
	
	// Compute usage (current)
	CurrentCPUCores float64 `json:"currentCPUCores"`
	CurrentMemoryBytes int64 `json:"currentMemoryBytes"`
	
	// Storage usage
	TotalStorageBytes int64 `json:"totalStorageBytes"`
	
	// Rate tracking (current window)
	RequestsThisMinute int `json:"requestsThisMinute"`
	ConcurrentRequests int `json:"concurrentRequests"`
	
	// LLM usage tracking
	LLMTokensToday int64 `json:"llmTokensToday"`
	LLMTokensThisMonth int64 `json:"llmTokensThisMonth"`
	LLMCostToday float64 `json:"llmCostToday"`
	LLMCostThisMonth float64 `json:"llmCostThisMonth"`
}

// ResourceType represents different types of resources
type ResourceType string

const (
	ResourceTypeMCPServer      ResourceType = "mcp-server"
	ResourceTypeThread         ResourceType = "thread"
	ResourceTypeKnowledgeSet   ResourceType = "knowledge-set"
	ResourceTypeKnowledgeFile  ResourceType = "knowledge-file"
	ResourceTypeCPU            ResourceType = "cpu"
	ResourceTypeMemory         ResourceType = "memory"
	ResourceTypeStorage        ResourceType = "storage"
	ResourceTypeLLMTokens      ResourceType = "llm-tokens"
	ResourceTypeLLMCost        ResourceType = "llm-cost"
	ResourceTypeAPIRequest     ResourceType = "api-request"
)

// QuotaExceededError indicates a quota has been exceeded
type QuotaExceededError struct {
	UserID       string
	ResourceType ResourceType
	Requested    int64
	Available    int64
	Limit        int64
	Message      string
}

func (e *QuotaExceededError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "quota exceeded for resource: " + string(e.ResourceType)
}

// DefaultQuota returns default quota values for standard users
func DefaultQuota() *ResourceQuota {
	return &ResourceQuota{
		MaxMCPServers:              10,
		MaxThreads:                 100,
		MaxMessagesPerThread:       1000,
		MaxKnowledgeSets:           5,
		MaxKnowledgeFiles:          1000,
		MaxKnowledgeFileSizeBytes:  100 * 1024 * 1024, // 100 MB per file
		MaxTotalKnowledgeSizeBytes: 10 * 1024 * 1024 * 1024, // 10 GB total
		MaxCPUCores:                4.0,
		MaxMemoryBytes:             8 * 1024 * 1024 * 1024, // 8 GB
		MaxStorageBytes:            50 * 1024 * 1024 * 1024, // 50 GB
		MaxRequestsPerMinute:       100,
		MaxConcurrentRequests:      10,
		MaxLLMTokensPerDay:         1000000, // 1M tokens per day
		MaxLLMTokensPerMonth:       10000000, // 10M tokens per month
		MaxLLMCostPerDay:           50.0,  // $50 per day
		MaxLLMCostPerMonth:         1000.0, // $1000 per month
	}
}

// PowerUserQuota returns elevated quota values for power users
func PowerUserQuota() *ResourceQuota {
	return &ResourceQuota{
		MaxMCPServers:              50,
		MaxThreads:                 500,
		MaxMessagesPerThread:       5000,
		MaxKnowledgeSets:           25,
		MaxKnowledgeFiles:          10000,
		MaxKnowledgeFileSizeBytes:  500 * 1024 * 1024, // 500 MB per file
		MaxTotalKnowledgeSizeBytes: 100 * 1024 * 1024 * 1024, // 100 GB total
		MaxCPUCores:                16.0,
		MaxMemoryBytes:             32 * 1024 * 1024 * 1024, // 32 GB
		MaxStorageBytes:            500 * 1024 * 1024 * 1024, // 500 GB
		MaxRequestsPerMinute:       1000,
		MaxConcurrentRequests:      50,
		MaxLLMTokensPerDay:         10000000, // 10M tokens per day
		MaxLLMTokensPerMonth:       100000000, // 100M tokens per month
		MaxLLMCostPerDay:           500.0,  // $500 per day
		MaxLLMCostPerMonth:         10000.0, // $10000 per month
	}
}

// EnterpriseQuota returns unlimited quota values for enterprise users
func EnterpriseQuota() *ResourceQuota {
	return &ResourceQuota{
		MaxMCPServers:              -1, // unlimited
		MaxThreads:                 -1,
		MaxMessagesPerThread:       -1,
		MaxKnowledgeSets:           -1,
		MaxKnowledgeFiles:          -1,
		MaxKnowledgeFileSizeBytes:  -1,
		MaxTotalKnowledgeSizeBytes: -1,
		MaxCPUCores:                -1,
		MaxMemoryBytes:             -1,
		MaxStorageBytes:            -1,
		MaxRequestsPerMinute:       -1,
		MaxConcurrentRequests:      -1,
		MaxLLMTokensPerDay:         -1,
		MaxLLMTokensPerMonth:       -1,
		MaxLLMCostPerDay:           -1,
		MaxLLMCostPerMonth:         -1,
	}
}
