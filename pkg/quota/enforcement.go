package quota

import (
	"context"
	"fmt"
	"log/slog"
)

// Enforcer provides middleware and helpers for quota enforcement
type Enforcer struct {
	manager Manager
	logger  *slog.Logger
}

// NewEnforcer creates a new quota enforcer
func NewEnforcer(manager Manager, logger *slog.Logger) *Enforcer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Enforcer{
		manager: manager,
		logger:  logger,
	}
}

// EnforceMCPServerCreation checks quota before creating an MCP server
func (e *Enforcer) EnforceMCPServerCreation(ctx context.Context, userID string) error {
	e.logger.DebugContext(ctx, "checking MCP server quota", "userID", userID)
	
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeMCPServer), 1); err != nil {
		e.logger.WarnContext(ctx, "MCP server quota exceeded", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeMCPServer), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to reserve MCP server quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	e.logger.DebugContext(ctx, "MCP server quota reserved", "userID", userID)
	return nil
}

// EnforceMCPServerDeletion releases quota when deleting an MCP server
func (e *Enforcer) EnforceMCPServerDeletion(ctx context.Context, userID string) error {
	e.logger.DebugContext(ctx, "releasing MCP server quota", "userID", userID)
	
	if err := e.manager.ReleaseQuota(ctx, userID, string(ResourceTypeMCPServer), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to release MCP server quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	return nil
}

// EnforceThreadCreation checks quota before creating a thread
func (e *Enforcer) EnforceThreadCreation(ctx context.Context, userID string) error {
	e.logger.DebugContext(ctx, "checking thread quota", "userID", userID)
	
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeThread), 1); err != nil {
		e.logger.WarnContext(ctx, "thread quota exceeded", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeThread), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to reserve thread quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	e.logger.DebugContext(ctx, "thread quota reserved", "userID", userID)
	return nil
}

// EnforceThreadDeletion releases quota when deleting a thread
func (e *Enforcer) EnforceThreadDeletion(ctx context.Context, userID string) error {
	e.logger.DebugContext(ctx, "releasing thread quota", "userID", userID)
	
	if err := e.manager.ReleaseQuota(ctx, userID, string(ResourceTypeThread), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to release thread quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	return nil
}

// EnforceKnowledgeSetCreation checks quota before creating a knowledge set
func (e *Enforcer) EnforceKnowledgeSetCreation(ctx context.Context, userID string) error {
	e.logger.DebugContext(ctx, "checking knowledge set quota", "userID", userID)
	
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeKnowledgeSet), 1); err != nil {
		e.logger.WarnContext(ctx, "knowledge set quota exceeded", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeKnowledgeSet), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to reserve knowledge set quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	e.logger.DebugContext(ctx, "knowledge set quota reserved", "userID", userID)
	return nil
}

// EnforceKnowledgeSetDeletion releases quota when deleting a knowledge set
func (e *Enforcer) EnforceKnowledgeSetDeletion(ctx context.Context, userID string) error {
	e.logger.DebugContext(ctx, "releasing knowledge set quota", "userID", userID)
	
	if err := e.manager.ReleaseQuota(ctx, userID, string(ResourceTypeKnowledgeSet), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to release knowledge set quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	return nil
}

// EnforceKnowledgeFileUpload checks quota before uploading a knowledge file
func (e *Enforcer) EnforceKnowledgeFileUpload(ctx context.Context, userID string, fileSizeBytes int64) error {
	e.logger.DebugContext(ctx, "checking knowledge file quota", 
		"userID", userID, 
		"fileSizeBytes", fileSizeBytes)
	
	// Check file count quota
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeKnowledgeFile), 1); err != nil {
		e.logger.WarnContext(ctx, "knowledge file count quota exceeded", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	// Check storage quota
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeStorage), fileSizeBytes); err != nil {
		e.logger.WarnContext(ctx, "storage quota exceeded", 
			"userID", userID, 
			"fileSizeBytes", fileSizeBytes,
			"error", err)
		return err
	}
	
	// Reserve both quotas
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeKnowledgeFile), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to reserve knowledge file quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeStorage), fileSizeBytes); err != nil {
		// Rollback file count reservation
		_ = e.manager.ReleaseQuota(ctx, userID, string(ResourceTypeKnowledgeFile), 1)
		
		e.logger.ErrorContext(ctx, "failed to reserve storage quota", 
			"userID", userID, 
			"fileSizeBytes", fileSizeBytes,
			"error", err)
		return err
	}
	
	e.logger.DebugContext(ctx, "knowledge file quota reserved", 
		"userID", userID, 
		"fileSizeBytes", fileSizeBytes)
	return nil
}

// EnforceKnowledgeFileDeletion releases quota when deleting a knowledge file
func (e *Enforcer) EnforceKnowledgeFileDeletion(ctx context.Context, userID string, fileSizeBytes int64) error {
	e.logger.DebugContext(ctx, "releasing knowledge file quota", 
		"userID", userID, 
		"fileSizeBytes", fileSizeBytes)
	
	// Release file count
	if err := e.manager.ReleaseQuota(ctx, userID, string(ResourceTypeKnowledgeFile), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to release knowledge file quota", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	// Release storage
	if err := e.manager.ReleaseQuota(ctx, userID, string(ResourceTypeStorage), fileSizeBytes); err != nil {
		e.logger.ErrorContext(ctx, "failed to release storage quota", 
			"userID", userID, 
			"fileSizeBytes", fileSizeBytes,
			"error", err)
		return err
	}
	
	return nil
}

// EnforceLLMUsage checks and reserves LLM quota (tokens and cost)
func (e *Enforcer) EnforceLLMUsage(ctx context.Context, userID string, tokens int64, costUSD float64) error {
	e.logger.DebugContext(ctx, "checking LLM usage quota", 
		"userID", userID, 
		"tokens", tokens, 
		"costUSD", costUSD)
	
	// Check token quota
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeLLMTokens), tokens); err != nil {
		e.logger.WarnContext(ctx, "LLM token quota exceeded", 
			"userID", userID, 
			"tokens", tokens,
			"error", err)
		return err
	}
	
	// Check cost quota (convert to cents for integer math)
	costCents := int64(costUSD * 100)
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeLLMCost), costCents); err != nil {
		e.logger.WarnContext(ctx, "LLM cost quota exceeded", 
			"userID", userID, 
			"costUSD", costUSD,
			"error", err)
		return err
	}
	
	// Reserve both quotas
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeLLMTokens), tokens); err != nil {
		e.logger.ErrorContext(ctx, "failed to reserve LLM token quota", 
			"userID", userID, 
			"tokens", tokens,
			"error", err)
		return err
	}
	
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeLLMCost), costCents); err != nil {
		// Rollback token reservation
		_ = e.manager.ReleaseQuota(ctx, userID, string(ResourceTypeLLMTokens), tokens)
		
		e.logger.ErrorContext(ctx, "failed to reserve LLM cost quota", 
			"userID", userID, 
			"costUSD", costUSD,
			"error", err)
		return err
	}
	
	e.logger.DebugContext(ctx, "LLM usage quota reserved", 
		"userID", userID, 
		"tokens", tokens, 
		"costUSD", costUSD)
	return nil
}

// EnforceAPIRequest checks rate limiting for API requests
func (e *Enforcer) EnforceAPIRequest(ctx context.Context, userID string) error {
	if err := e.manager.CheckQuota(ctx, userID, string(ResourceTypeAPIRequest), 1); err != nil {
		e.logger.WarnContext(ctx, "API rate limit exceeded", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	if err := e.manager.ReserveQuota(ctx, userID, string(ResourceTypeAPIRequest), 1); err != nil {
		e.logger.ErrorContext(ctx, "failed to track API request", 
			"userID", userID, 
			"error", err)
		return err
	}
	
	return nil
}

// GetUserQuotaStatus returns a human-readable quota status for a user
func (e *Enforcer) GetUserQuotaStatus(ctx context.Context, userID string) (string, error) {
	quota, err := e.manager.GetQuota(ctx, userID)
	if err != nil {
		return "", err
	}
	
	usage, err := e.manager.GetUsage(ctx, userID)
	if err != nil {
		return "", err
	}
	
	status := fmt.Sprintf("Quota Status for User: %s\n", userID)
	status += "================================\n"
	status += fmt.Sprintf("MCP Servers: %d / %d\n", usage.MCPServerCount, quota.MaxMCPServers)
	status += fmt.Sprintf("Threads: %d / %d\n", usage.ThreadCount, quota.MaxThreads)
	status += fmt.Sprintf("Knowledge Sets: %d / %d\n", usage.KnowledgeSetCount, quota.MaxKnowledgeSets)
	status += fmt.Sprintf("Knowledge Files: %d / %d\n", usage.KnowledgeFileCount, quota.MaxKnowledgeFiles)
	status += fmt.Sprintf("Storage: %.2f GB / %.2f GB\n", 
		float64(usage.TotalStorageBytes)/(1024*1024*1024), 
		float64(quota.MaxStorageBytes)/(1024*1024*1024))
	status += fmt.Sprintf("LLM Tokens (Today): %d / %d\n", usage.LLMTokensToday, quota.MaxLLMTokensPerDay)
	status += fmt.Sprintf("LLM Cost (Today): $%.2f / $%.2f\n", usage.LLMCostToday, quota.MaxLLMCostPerDay)
	status += fmt.Sprintf("LLM Cost (Month): $%.2f / $%.2f\n", usage.LLMCostThisMonth, quota.MaxLLMCostPerMonth)
	status += fmt.Sprintf("Requests (This Minute): %d / %d\n", usage.RequestsThisMinute, quota.MaxRequestsPerMinute)
	
	return status, nil
}

// IsQuotaExceededError checks if an error is a quota exceeded error
func IsQuotaExceededError(err error) bool {
	_, ok := err.(*QuotaExceededError)
	return ok
}
