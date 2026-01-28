package quota

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// manager implements the Manager interface
type manager struct {
	// In-memory storage for quotas and usage
	// In production, this should be backed by database
	quotas map[string]*ResourceQuota
	usage  map[string]*ResourceUsage
	mu     sync.RWMutex
	
	// For time-based quota tracking (daily/monthly)
	dailyResetTime  time.Time
	monthlyResetTime time.Time
}

// NewManager creates a new quota manager instance
func NewManager() Manager {
	now := time.Now()
	return &manager{
		quotas: make(map[string]*ResourceQuota),
		usage:  make(map[string]*ResourceUsage),
		dailyResetTime: getNextDayReset(now),
		monthlyResetTime: getNextMonthReset(now),
	}
}

// CheckQuota verifies if a resource allocation request can be satisfied
func (m *manager) CheckQuota(ctx context.Context, userID, resourceType string, requested int64) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	quota, err := m.getQuotaLocked(userID)
	if err != nil {
		return err
	}
	
	usage, err := m.getUsageLocked(userID)
	if err != nil {
		return err
	}
	
	// Check time-based resets
	m.checkAndResetTimeLocked(usage)
	
	return m.checkResourceQuota(quota, usage, ResourceType(resourceType), requested)
}

// ReserveQuota reserves resources for a user
func (m *manager) ReserveQuota(ctx context.Context, userID, resourceType string, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	usage, err := m.getUsageLocked(userID)
	if err != nil {
		return err
	}
	
	// Check time-based resets
	m.checkAndResetTimeLocked(usage)
	
	// Update usage based on resource type
	m.updateUsageLocked(usage, ResourceType(resourceType), amount)
	
	return nil
}

// ReleaseQuota releases reserved resources
func (m *manager) ReleaseQuota(ctx context.Context, userID, resourceType string, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	usage, err := m.getUsageLocked(userID)
	if err != nil {
		return err
	}
	
	// Decrease usage based on resource type
	m.updateUsageLocked(usage, ResourceType(resourceType), -amount)
	
	return nil
}

// GetUsage returns current usage for a user
func (m *manager) GetUsage(ctx context.Context, userID string) (*ResourceUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usage, err := m.getUsageLocked(userID)
	if err != nil {
		return nil, err
	}
	
	// Create a copy to avoid external modifications
	usageCopy := *usage
	return &usageCopy, nil
}

// GetQuota returns configured quotas for a user
func (m *manager) GetQuota(ctx context.Context, userID string) (*ResourceQuota, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	quota, err := m.getQuotaLocked(userID)
	if err != nil {
		return nil, err
	}
	
	// Create a copy to avoid external modifications
	quotaCopy := *quota
	return &quotaCopy, nil
}

// UpdateQuota updates quota configuration for a user
func (m *manager) UpdateQuota(ctx context.Context, userID string, quota *ResourceQuota) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	quota.UserID = userID
	quota.UpdatedAt = time.Now()
	if quota.CreatedAt.IsZero() {
		quota.CreatedAt = quota.UpdatedAt
	}
	
	m.quotas[userID] = quota
	return nil
}

// GetWorkspaceUsage returns usage for a workspace
func (m *manager) GetWorkspaceUsage(ctx context.Context, workspaceID string) (*ResourceUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Aggregate usage across all users in workspace
	// For now, using workspace ID as key
	usage, exists := m.usage[workspaceID]
	if !exists {
		usage = &ResourceUsage{
			WorkspaceID: workspaceID,
			Timestamp:   time.Now(),
		}
		m.usage[workspaceID] = usage
	}
	
	usageCopy := *usage
	return &usageCopy, nil
}

// getQuotaLocked retrieves quota for a user (must be called with lock held)
func (m *manager) getQuotaLocked(userID string) (*ResourceQuota, error) {
	quota, exists := m.quotas[userID]
	if !exists {
		// Return default quota if none configured
		quota = DefaultQuota()
		quota.UserID = userID
		quota.CreatedAt = time.Now()
		quota.UpdatedAt = quota.CreatedAt
		m.quotas[userID] = quota
	}
	return quota, nil
}

// getUsageLocked retrieves usage for a user (must be called with lock held)
func (m *manager) getUsageLocked(userID string) (*ResourceUsage, error) {
	usage, exists := m.usage[userID]
	if !exists {
		usage = &ResourceUsage{
			UserID:    userID,
			Timestamp: time.Now(),
		}
		m.usage[userID] = usage
	}
	return usage, nil
}

// checkResourceQuota verifies if quota is available for the requested resource
func (m *manager) checkResourceQuota(quota *ResourceQuota, usage *ResourceUsage, resourceType ResourceType, requested int64) error {
	var limit, current int64
	
	switch resourceType {
	case ResourceTypeMCPServer:
		limit = int64(quota.MaxMCPServers)
		current = int64(usage.MCPServerCount)
		
	case ResourceTypeThread:
		limit = int64(quota.MaxThreads)
		current = int64(usage.ThreadCount)
		
	case ResourceTypeKnowledgeSet:
		limit = int64(quota.MaxKnowledgeSets)
		current = int64(usage.KnowledgeSetCount)
		
	case ResourceTypeKnowledgeFile:
		limit = int64(quota.MaxKnowledgeFiles)
		current = int64(usage.KnowledgeFileCount)
		
	case ResourceTypeStorage:
		limit = quota.MaxStorageBytes
		current = usage.TotalStorageBytes
		
	case ResourceTypeCPU:
		limit = int64(quota.MaxCPUCores * 1000) // Convert to millicores
		current = int64(usage.CurrentCPUCores * 1000)
		
	case ResourceTypeMemory:
		limit = quota.MaxMemoryBytes
		current = usage.CurrentMemoryBytes
		
	case ResourceTypeLLMTokens:
		limit = quota.MaxLLMTokensPerDay
		current = usage.LLMTokensToday
		
	case ResourceTypeLLMCost:
		limit = int64(quota.MaxLLMCostPerDay * 100) // Convert to cents
		current = int64(usage.LLMCostToday * 100)
		
	case ResourceTypeAPIRequest:
		limit = int64(quota.MaxRequestsPerMinute)
		current = int64(usage.RequestsThisMinute)
		
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	
	// -1 means unlimited
	if limit == -1 {
		return nil
	}
	
	available := limit - current
	if available < requested {
		return &QuotaExceededError{
			UserID:       usage.UserID,
			ResourceType: resourceType,
			Requested:    requested,
			Available:    available,
			Limit:        limit,
			Message:      fmt.Sprintf("quota exceeded: requested %d %s, only %d available (limit: %d, current: %d)", 
				requested, resourceType, available, limit, current),
		}
	}
	
	return nil
}

// updateUsageLocked updates usage counters (must be called with lock held)
func (m *manager) updateUsageLocked(usage *ResourceUsage, resourceType ResourceType, amount int64) {
	usage.Timestamp = time.Now()
	
	switch resourceType {
	case ResourceTypeMCPServer:
		usage.MCPServerCount += int(amount)
		if usage.MCPServerCount < 0 {
			usage.MCPServerCount = 0
		}
		
	case ResourceTypeThread:
		usage.ThreadCount += int(amount)
		if usage.ThreadCount < 0 {
			usage.ThreadCount = 0
		}
		
	case ResourceTypeKnowledgeSet:
		usage.KnowledgeSetCount += int(amount)
		if usage.KnowledgeSetCount < 0 {
			usage.KnowledgeSetCount = 0
		}
		
	case ResourceTypeKnowledgeFile:
		usage.KnowledgeFileCount += int(amount)
		if usage.KnowledgeFileCount < 0 {
			usage.KnowledgeFileCount = 0
		}
		
	case ResourceTypeStorage:
		usage.TotalStorageBytes += amount
		if usage.TotalStorageBytes < 0 {
			usage.TotalStorageBytes = 0
		}
		
	case ResourceTypeCPU:
		usage.CurrentCPUCores = float64(amount) / 1000.0 // Convert from millicores
		if usage.CurrentCPUCores < 0 {
			usage.CurrentCPUCores = 0
		}
		
	case ResourceTypeMemory:
		usage.CurrentMemoryBytes = amount
		if usage.CurrentMemoryBytes < 0 {
			usage.CurrentMemoryBytes = 0
		}
		
	case ResourceTypeLLMTokens:
		usage.LLMTokensToday += amount
		usage.LLMTokensThisMonth += amount
		if usage.LLMTokensToday < 0 {
			usage.LLMTokensToday = 0
		}
		if usage.LLMTokensThisMonth < 0 {
			usage.LLMTokensThisMonth = 0
		}
		
	case ResourceTypeLLMCost:
		costDelta := float64(amount) / 100.0 // Convert from cents
		usage.LLMCostToday += costDelta
		usage.LLMCostThisMonth += costDelta
		if usage.LLMCostToday < 0 {
			usage.LLMCostToday = 0
		}
		if usage.LLMCostThisMonth < 0 {
			usage.LLMCostThisMonth = 0
		}
		
	case ResourceTypeAPIRequest:
		usage.RequestsThisMinute += int(amount)
		if usage.RequestsThisMinute < 0 {
			usage.RequestsThisMinute = 0
		}
	}
}

// checkAndResetTimeLocked checks if time-based quotas need resetting
func (m *manager) checkAndResetTimeLocked(usage *ResourceUsage) {
	now := time.Now()
	
	// Reset daily counters
	if now.After(m.dailyResetTime) {
		usage.LLMTokensToday = 0
		usage.LLMCostToday = 0
		m.dailyResetTime = getNextDayReset(now)
	}
	
	// Reset monthly counters
	if now.After(m.monthlyResetTime) {
		usage.LLMTokensThisMonth = 0
		usage.LLMCostThisMonth = 0
		m.monthlyResetTime = getNextMonthReset(now)
	}
	
	// Reset per-minute counters (simple approach - reset every minute)
	if now.Sub(usage.Timestamp) > time.Minute {
		usage.RequestsThisMinute = 0
	}
}

// getNextDayReset returns the timestamp for the next daily reset (midnight UTC)
func getNextDayReset(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

// getNextMonthReset returns the timestamp for the next monthly reset (first day of next month, midnight UTC)
func getNextMonthReset(now time.Time) time.Time {
	year, month, _ := now.Date()
	if month == 12 {
		return time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
}
