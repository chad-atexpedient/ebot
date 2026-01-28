# ebot Resource Quota System

## Overview

The ebot quota system provides comprehensive resource management and capacity planning capabilities to prevent resource exhaustion, ensure fair allocation, and control costs in multi-tenant environments.

## Features

- **Per-User Quotas**: Configurable limits for individual users
- **Per-Workspace Quotas**: Resource limits at the workspace level
- **Time-Based Quotas**: Daily and monthly limits for LLM usage
- **Rate Limiting**: Per-minute API request limits
- **Cost Controls**: LLM cost tracking and budgeting
- **Flexible Tiers**: Default, Power User, and Enterprise quota templates

## Resource Types

### Infrastructure Resources
- **MCP Servers**: Number of MCP servers a user can deploy
- **Threads**: Number of conversation threads
- **Messages**: Messages per thread
- **CPU**: CPU cores allocated
- **Memory**: Memory allocated (bytes)
- **Storage**: Total storage used (bytes)

### Knowledge Resources
- **Knowledge Sets**: Number of knowledge bases
- **Knowledge Files**: Number of files in knowledge bases
- **File Size**: Maximum individual file size
- **Total Knowledge Size**: Total size of all knowledge data

### Usage Limits
- **API Requests**: Requests per minute
- **Concurrent Requests**: Simultaneous requests
- **LLM Tokens**: Daily and monthly token limits
- **LLM Cost**: Daily and monthly spend limits (USD)

## Quota Tiers

### Default User
```go
MaxMCPServers:       10
MaxThreads:          100
MaxKnowledgeSets:    5
MaxKnowledgeFiles:   1000
MaxStorageBytes:     50 GB
MaxLLMCostPerDay:    $50
MaxLLMCostPerMonth:  $1000
```

### Power User
```go
MaxMCPServers:       50
MaxThreads:          500
MaxKnowledgeSets:    25
MaxKnowledgeFiles:   10000
MaxStorageBytes:     500 GB
MaxLLMCostPerDay:    $500
MaxLLMCostPerMonth:  $10000
```

### Enterprise
```go
All limits:          -1 (unlimited)
```

## Usage

### Checking Quotas

```go
import "github.com/chad-atexpedient/ebot/pkg/quota"

// Create manager
manager := quota.NewManager()

// Check if user can create an MCP server
err := manager.CheckQuota(ctx, userID, string(quota.ResourceTypeMCPServer), 1)
if err != nil {
    if quota.IsQuotaExceededError(err) {
        // Handle quota exceeded
        return fmt.Errorf("cannot create MCP server: %w", err)
    }
    return err
}
```

### Reserving Resources

```go
// Reserve quota when creating a resource
err := manager.ReserveQuota(ctx, userID, string(quota.ResourceTypeMCPServer), 1)
if err != nil {
    return err
}

// If creation fails, release the reserved quota
defer func() {
    if creationFailed {
        _ = manager.ReleaseQuota(ctx, userID, string(quota.ResourceTypeMCPServer), 1)
    }
}()
```

### Using the Enforcer

```go
import "github.com/chad-atexpedient/ebot/pkg/quota"

// Create enforcer
manager := quota.NewManager()
enforcer := quota.NewEnforcer(manager, logger)

// Enforce MCP server creation
if err := enforcer.EnforceMCPServerCreation(ctx, userID); err != nil {
    return err
}

// On successful creation, quota is already reserved
// On deletion, release the quota
defer func() {
    if deleted {
        _ = enforcer.EnforceMCPServerDeletion(ctx, userID)
    }
}()
```

### Tracking LLM Usage

```go
// Before making an LLM call
tokens := int64(estimatedTokens)
costUSD := calculateCost(model, tokens)

if err := enforcer.EnforceLLMUsage(ctx, userID, tokens, costUSD); err != nil {
    return err
}

// Usage is automatically tracked
// Daily and monthly counters are reset automatically
```

### Checking User Status

```go
// Get human-readable quota status
status, err := enforcer.GetUserQuotaStatus(ctx, userID)
if err != nil {
    return err
}

fmt.Println(status)
// Output:
// Quota Status for User: user-123
// ================================
// MCP Servers: 5 / 10
// Threads: 23 / 100
// Knowledge Sets: 2 / 5
// Storage: 12.50 GB / 50.00 GB
// LLM Tokens (Today): 45000 / 1000000
// LLM Cost (Today): $12.50 / $50.00
// ...
```

## API Integration

### Handler Example

```go
func (h *Handler) CreateMCPServer(w http.ResponseWriter, r *http.Request) {
    userID := getUserFromContext(r.Context())
    
    // Enforce quota
    if err := h.quotaEnforcer.EnforceMCPServerCreation(r.Context(), userID); err != nil {
        if quota.IsQuotaExceededError(err) {
            http.Error(w, err.Error(), http.StatusForbidden)
            return
        }
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    
    // Create the MCP server
    server, err := h.createServer(r.Context(), userID)
    if err != nil {
        // Release quota on failure
        _ = h.quotaEnforcer.EnforceMCPServerDeletion(r.Context(), userID)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return success
    json.NewEncoder(w).Encode(server)
}
```

## Configuration

### Setting Custom Quotas

```go
// Create custom quota for a specific user
customQuota := &quota.ResourceQuota{
    MaxMCPServers:       25,
    MaxThreads:          250,
    MaxKnowledgeSets:    10,
    MaxStorageBytes:     100 * 1024 * 1024 * 1024, // 100 GB
    MaxLLMCostPerDay:    100.0, // $100/day
    MaxLLMCostPerMonth:  2000.0, // $2000/month
}

err := manager.UpdateQuota(ctx, userID, customQuota)
if err != nil {
    return err
}
```

### Getting Usage Information

```go
// Get current usage
usage, err := manager.GetUsage(ctx, userID)
if err != nil {
    return err
}

fmt.Printf("User has created %d MCP servers\n", usage.MCPServerCount)
fmt.Printf("User has used %.2f GB of storage\n", 
    float64(usage.TotalStorageBytes)/(1024*1024*1024))
fmt.Printf("User has spent $%.2f on LLM today\n", usage.LLMCostToday)
```

## Time-Based Resets

The quota system automatically resets time-based counters:

- **Per-Minute**: API request counters reset every minute
- **Daily**: LLM token and cost counters reset at midnight UTC
- **Monthly**: Monthly LLM counters reset on the 1st of each month

## Best Practices

### 1. Check Before Reserve
Always check quota before attempting to reserve resources:

```go
// Check first
if err := manager.CheckQuota(ctx, userID, resourceType, amount); err != nil {
    return err
}

// Then reserve
if err := manager.ReserveQuota(ctx, userID, resourceType, amount); err != nil {
    return err
}
```

### 2. Release on Failure
Always release reserved quota if resource creation fails:

```go
// Reserve quota
if err := manager.ReserveQuota(ctx, userID, resourceType, 1); err != nil {
    return err
}

// Create resource
resource, err := createResource(ctx)
if err != nil {
    // Release on failure
    _ = manager.ReleaseQuota(ctx, userID, resourceType, 1)
    return err
}
```

### 3. Use the Enforcer
The enforcer handles check + reserve atomically and provides better logging:

```go
// Prefer this
if err := enforcer.EnforceMCPServerCreation(ctx, userID); err != nil {
    return err
}

// Over manual check + reserve
if err := manager.CheckQuota(ctx, userID, ...); err != nil {
    return err
}
if err := manager.ReserveQuota(ctx, userID, ...); err != nil {
    return err
}
```

### 4. Monitor Usage
Regularly monitor usage patterns to:
- Detect quota exhaustion trends
- Identify users hitting limits
- Plan capacity upgrades
- Optimize cost allocation

## Error Handling

```go
err := manager.CheckQuota(ctx, userID, resourceType, amount)
if err != nil {
    if quotaErr, ok := err.(*quota.QuotaExceededError); ok {
        // Quota exceeded - show user-friendly message
        log.Printf("User %s exceeded quota for %s: requested %d, available %d",
            quotaErr.UserID,
            quotaErr.ResourceType,
            quotaErr.Requested,
            quotaErr.Available)
        
        // Return appropriate HTTP status
        return http.StatusForbidden
    }
    
    // Other error
    return http.StatusInternalServerError
}
```

## Database Integration

The current implementation uses in-memory storage. For production:

1. **Add Database Backend**: Implement quota storage in PostgreSQL
2. **Add Caching**: Cache frequently accessed quotas
3. **Add Metrics**: Export quota metrics to Prometheus
4. **Add Alerts**: Alert on quota threshold violations

Example database schema:

```sql
CREATE TABLE user_quotas (
    user_id VARCHAR(255) PRIMARY KEY,
    max_mcp_servers INT NOT NULL,
    max_threads INT NOT NULL,
    max_knowledge_sets INT NOT NULL,
    max_storage_bytes BIGINT NOT NULL,
    max_llm_cost_per_day DECIMAL(10,2) NOT NULL,
    max_llm_cost_per_month DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE user_usage (
    user_id VARCHAR(255) PRIMARY KEY,
    mcp_server_count INT NOT NULL DEFAULT 0,
    thread_count INT NOT NULL DEFAULT 0,
    knowledge_set_count INT NOT NULL DEFAULT 0,
    total_storage_bytes BIGINT NOT NULL DEFAULT 0,
    llm_cost_today DECIMAL(10,2) NOT NULL DEFAULT 0,
    llm_cost_this_month DECIMAL(10,2) NOT NULL DEFAULT 0,
    last_updated TIMESTAMP NOT NULL
);
```

## Future Enhancements

1. **Workspace-Level Quotas**: Aggregate quotas across workspace users
2. **Dynamic Quotas**: Adjust quotas based on usage patterns
3. **Quota Requests**: Allow users to request quota increases
4. **Quota Notifications**: Email/Slack alerts when approaching limits
5. **Historical Tracking**: Track quota usage over time
6. **Cost Forecasting**: Predict future costs based on usage trends
7. **Burst Allowance**: Temporary quota increases for short periods

## Metrics

The quota system should export these metrics to Prometheus:

- `ebot_quota_usage{user_id, resource_type}`: Current usage
- `ebot_quota_limit{user_id, resource_type}`: Configured limit
- `ebot_quota_exceeded_total{user_id, resource_type}`: Quota exceeded counter
- `ebot_quota_utilization{user_id, resource_type}`: Usage as percentage of limit

## Troubleshooting

### Users Hitting Limits
1. Check current usage: `manager.GetUsage(ctx, userID)`
2. Check configured quota: `manager.GetQuota(ctx, userID)`
3. Review usage patterns in logs
4. Consider increasing quota or upgrading tier

### Quota Not Releasing
1. Ensure `ReleaseQuota` is called on resource deletion
2. Check for failed cleanup operations
3. Review audit logs for orphaned resources
4. Manually reset usage if needed

### Performance Issues
1. Add database indexes on user_id, workspace_id
2. Implement quota caching (Redis)
3. Batch quota checks where possible
4. Use connection pooling for database

## Support

For questions or issues with the quota system:
- File an issue: https://github.com/chad-atexpedient/ebot/issues
- Email: cloud-support@expedient.com
