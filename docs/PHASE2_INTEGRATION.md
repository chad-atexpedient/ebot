# Phase 2 Integration Guide

This document describes how Phase 1 components have been integrated into the ebot platform and provides guidance for using the new features.

## Overview

Phase 2A focuses on integrating all Phase 1 systems into the existing ebot codebase:

- ✅ **Quota System** - Fully integrated with API middleware
- ✅ **Cost Metering** - Wrapped LLM invocations
- ✅ **HA Coordinator** - Connected to server lifecycle
- ✅ **UI Dashboards** - Admin dashboards created
- ✅ **API Handlers** - REST endpoints for all features

---

## 1. Quota System Integration

### Architecture

The quota system is now integrated at three levels:

1. **Middleware Layer** (`pkg/api/middleware/quota.go`)
   - Intercepts API requests
   - Checks and reserves quotas before operations
   - Automatic rollback on failures

2. **API Handlers** (`pkg/api/handlers/quota.go`)
   - GET `/api/quota/usage` - Get current quota usage
   - GET `/api/quota/usage?userId=xxx` - Get user quota (admin)
   - PUT `/api/quota/update` - Update quota limits (admin)

3. **UI Dashboard** (`ui/admin/src/components/QuotaDashboard.svelte`)
   - Real-time quota visualization
   - Progress bars for each resource type
   - Color-coded warnings (green → yellow → red)

### Usage Example

#### Apply Quota Middleware to Routes

```go
package router

import (
    "github.com/chad-atexpedient/ebot/pkg/api/middleware"
    "github.com/chad-atexpedient/ebot/pkg/quota"
)

func SetupRoutes(qm quota.Manager) {
    quotaMW := middleware.NewQuotaMiddleware(qm)
    
    // Apply to MCP server creation
    router.POST("/api/mcp-servers", 
        quotaMW.EnforceMCPServerQuota(
            quotaMW.RollbackQuotaOnError(
                handlers.CreateMCPServer)))
    
    // Apply to thread creation
    router.POST("/api/threads",
        quotaMW.EnforceThreadQuota(
            quotaMW.RollbackQuotaOnError(
                handlers.CreateThread)))
    
    // Apply to knowledge set uploads
    router.POST("/api/knowledge",
        quotaMW.EnforceKnowledgeQuota(
            quotaMW.RollbackQuotaOnError(
                handlers.CreateKnowledgeSet)))
}
```

#### Check Quota Programmatically

```go
package handlers

import (
    "github.com/chad-atexpedient/ebot/pkg/quota"
)

func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := getUserID(ctx)
    
    // Check quota before expensive operation
    if err := h.quotaManager.CheckQuota(ctx, userID, quota.ResourceTypeMCPServers, 1); err != nil {
        http.Error(w, "Quota exceeded", http.StatusTooManyRequests)
        return
    }
    
    // Reserve quota
    if err := h.quotaManager.ReserveQuota(ctx, userID, quota.ResourceTypeMCPServers, 1); err != nil {
        http.Error(w, "Failed to reserve quota", http.StatusInternalServerError)
        return
    }
    
    // Create resource
    resource, err := createResource()
    if err != nil {
        // Release quota on failure
        h.quotaManager.ReleaseQuota(ctx, userID, quota.ResourceTypeMCPServers, 1)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    respondJSON(w, resource)
}
```

### Configuration

Set quota tiers in environment variables:

```bash
# Quota tier assignment
EBOT_QUOTA_DEFAULT_TIER=default
EBOT_QUOTA_POWER_USER_TIER=power_user
EBOT_QUOTA_ENTERPRISE_TIER=enterprise

# Custom quota overrides (optional)
EBOT_QUOTA_MCP_SERVERS=20
EBOT_QUOTA_THREADS=100
EBOT_QUOTA_STORAGE_GB=50
```

---

## 2. Cost Metering Integration

### Architecture

Cost metering tracks every LLM invocation:

1. **Metering Wrapper** (`pkg/invoke/metering_wrapper.go`)
   - Wraps all LLM calls
   - Captures token usage
   - Calculates costs using pricing tables

2. **API Handlers** (`pkg/api/handlers/costs.go`)
   - GET `/api/costs/report?range=7d` - Cost report
   - GET `/api/costs/user?userId=xxx` - User costs
   - GET `/api/costs/workspace?workspaceId=xxx` - Workspace costs
   - GET `/api/costs/budget` - Budget status
   - GET `/api/costs/export` - Export as CSV

3. **UI Dashboard** (`ui/admin/src/components/CostDashboard.svelte`)
   - Cost trends over time
   - Breakdown by model
   - Top users and workspaces
   - Budget tracking

### Usage Example

#### Wrap LLM Invocations

```go
package invoke

import (
    "github.com/chad-atexpedient/ebot/pkg/metering"
)

type Invoker struct {
    meteringWrapper *MeteringWrapper
    // ... other fields
}

func (i *Invoker) Invoke(ctx context.Context, req InvokeRequest) (Response, error) {
    // Wrap the LLM call
    var response Response
    err := i.meteringWrapper.WrapInvocation(
        ctx,
        req.UserID,
        req.WorkspaceID,
        req.ThreadID,
        req.Provider,    // "openai", "anthropic"
        req.Model,       // "gpt-4", "claude-3-opus"
        func() (TokenUsage, error) {
            // Actual LLM call
            resp, err := callLLM(req)
            if err != nil {
                return TokenUsage{}, err
            }
            
            response = resp
            return TokenUsage{
                PromptTokens:     resp.Usage.PromptTokens,
                CompletionTokens: resp.Usage.CompletionTokens,
                TotalTokens:      resp.Usage.TotalTokens,
            }, nil
        },
    )
    
    return response, err
}
```

#### Query Cost Reports

```bash
# Get 7-day cost report
curl -H "Authorization: Bearer $TOKEN" \
  "https://ebot.example.com/api/costs/report?range=7d"

# Get user-specific costs
curl -H "Authorization: Bearer $TOKEN" \
  "https://ebot.example.com/api/costs/user?userId=user123"

# Check budget status
curl -H "Authorization: Bearer $TOKEN" \
  "https://ebot.example.com/api/costs/budget"
```

### Configuration

Configure cost tracking:

```bash
# Enable cost tracking
EBOT_METERING_ENABLED=true

# Storage backend
EBOT_METERING_STORAGE=postgres  # or "memory" for testing

# Budget alerts
EBOT_BUDGET_ALERT_THRESHOLD=0.8  # Alert at 80%
EBOT_BUDGET_ALERT_EMAIL=admin@example.com

# Cost optimization
EBOT_COST_OPTIMIZATION_ENABLED=true
```

---

## 3. High Availability Integration

### Architecture

HA coordinator ensures cluster stability:

1. **HA Integration** (`pkg/server/ha_integration.go`)
   - Leader election
   - Health monitoring
   - Automatic failover

2. **Health Checks**
   - Database connectivity
   - API server responsiveness
   - MCP gateway availability

### Usage Example

#### Start HA Coordinator

```go
package main

import (
    "github.com/chad-atexpedient/ebot/pkg/ha"
    "github.com/chad-atexpedient/ebot/pkg/server"
)

func main() {
    // Create HA coordinator
    coordinator := ha.NewCoordinator(ha.Config{
        NodeID:         "node-1",
        ElectionPrefix: "/ebot/leader",
        TTL:            time.Second * 10,
    })
    
    // Create HA integration
    haIntegration := server.NewHAIntegration(coordinator, logger)
    
    // Start HA operations
    if err := haIntegration.Start(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Start server
    srv := server.New(config)
    srv.Run()
}
```

#### Check Cluster Status

```bash
# Get cluster health
curl http://localhost:8080/health/cluster

# Response:
{
  "leaderId": "node-1",
  "members": [
    {
      "nodeId": "node-1",
      "health": "healthy",
      "role": "leader"
    },
    {
      "nodeId": "node-2",
      "health": "healthy",
      "role": "follower"
    }
  ],
  "overallHealth": "healthy"
}
```

### Configuration

Configure HA settings:

```bash
# Enable HA mode
EBOT_HA_ENABLED=true

# Node configuration
EBOT_NODE_ID=node-1
EBOT_NODE_NAME=ebot-primary

# Leader election
EBOT_LEADER_ELECTION_TTL=10s
EBOT_LEADER_ELECTION_RENEW=5s

# Health check intervals
EBOT_HEALTH_CHECK_INTERVAL=10s
EBOT_HEALTH_CHECK_TIMEOUT=5s
```

---

## 4. UI Dashboard Integration

### Admin Dashboards

Two new dashboards are available in the admin UI:

#### Quota Dashboard

**Route:** `/admin/quotas`

Features:
- Real-time quota usage per resource type
- Visual progress bars
- Color-coded warnings
- Per-user and per-workspace views
- Drill-down capabilities

#### Cost Dashboard

**Route:** `/admin/costs`

Features:
- Cost trends over time (line chart)
- Cost breakdown by model
- Top users by spending
- Top workspaces by spending
- Time range filters (7d, 30d, 90d)
- Export to CSV

### Embedding Dashboards

```html
<!-- In your admin layout -->
<nav>
  <a href="/admin/quotas">Quota Management</a>
  <a href="/admin/costs">Cost Analytics</a>
</nav>

<main>
  <!-- Dashboard components render here -->
  <QuotaDashboard />
  <CostDashboard />
</main>
```

---

## 5. API Endpoints Reference

### Quota Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/quota/usage` | Get current user's quota usage | User |
| GET | `/api/quota/usage?userId=xxx` | Get specific user's quota | Admin |
| PUT | `/api/quota/update` | Update quota limits | Admin |
| GET | `/api/quota/list` | List all user quotas | Admin |

### Cost Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/costs/report?range=7d` | Get cost report | User |
| GET | `/api/costs/user?userId=xxx` | Get user costs | User/Admin |
| GET | `/api/costs/workspace?workspaceId=xxx` | Get workspace costs | Member/Admin |
| GET | `/api/costs/budget` | Get budget status | User |
| GET | `/api/costs/export` | Export costs as CSV | User |

### Health Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/health` | Basic health check | Public |
| GET | `/health/cluster` | Cluster status | Admin |
| GET | `/health/detailed` | Detailed health | Admin |

---

## 6. Testing the Integration

### Unit Tests

Run unit tests for integrated components:

```bash
# Test quota system
go test ./pkg/quota/... -v

# Test metering
go test ./pkg/metering/... -v

# Test HA coordinator
go test ./pkg/ha/... -v

# Test API handlers
go test ./pkg/api/handlers/... -v
```

### Integration Tests

```bash
# Run full integration test suite
make test-integration

# Test quota enforcement
go test ./tests/integration/quota_test.go -v

# Test cost tracking
go test ./tests/integration/cost_test.go -v
```

### Manual Testing

1. **Test Quota Enforcement:**
```bash
# Create resources until quota is exceeded
for i in {1..25}; do
  curl -X POST http://localhost:8080/api/mcp-servers \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"name": "server-'$i'"}'
done
```

2. **Test Cost Tracking:**
```bash
# Make LLM calls and check cost accumulation
curl -X POST http://localhost:8080/api/threads/thread-1/invoke \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Hello"}'

# Check costs
curl http://localhost:8080/api/costs/report?range=7d \
  -H "Authorization: Bearer $TOKEN"
```

3. **Test HA Failover:**
```bash
# Kill leader node
docker stop ebot-node-1

# Verify follower takes over
curl http://localhost:8080/health/cluster
```

---

## 7. Migration Guide

### Migrating from Phase 1 Standalone

If you were testing Phase 1 components standalone, migrate to integrated version:

1. **Update Imports:**
```go
// Before
import "github.com/chad-atexpedient/ebot/pkg/quota"

// After (same, but now integrated)
import "github.com/chad-atexpedient/ebot/pkg/quota"
```

2. **Update Configuration:**
```bash
# Before: Separate configs
QUOTA_ENABLED=true
METERING_ENABLED=true

# After: Unified config
EBOT_QUOTA_ENABLED=true
EBOT_METERING_ENABLED=true
```

3. **Update Database:**
```bash
# Run migrations
./bin/ebot migrate up
```

---

## 8. Next Steps (Phase 2B)

Now that Phase 1 features are integrated, Phase 2B will add:

- **Multi-Region Support** - Data residency and cross-region replication
- **Advanced Access Control** - SAML, ABAC, service accounts
- **SDK Development** - Python and TypeScript SDKs
- **Performance Optimization** - Redis caching, query optimization
- **Advanced Monitoring** - Distributed tracing, alerting

---

## 9. Troubleshooting

### Quota Issues

**Problem:** Quota not enforcing  
**Solution:** Ensure middleware is applied to routes:
```go
router.Use(quotaMW.EnforceMCPServerQuota)
```

**Problem:** Quota limits too low  
**Solution:** Update quota tier or adjust limits:
```bash
EBOT_QUOTA_MCP_SERVERS=50
```

### Cost Tracking Issues

**Problem:** Costs not being recorded  
**Solution:** Verify metering wrapper is used:
```go
invoker := invoke.NewInvoker(meteringWrapper)
```

**Problem:** Incorrect cost calculations  
**Solution:** Check pricing tables in `pkg/metering/calculator.go`

### HA Issues

**Problem:** Leader election failing  
**Solution:** Check etcd connectivity and election prefix:
```bash
EBOT_LEADER_ELECTION_PREFIX=/ebot/leader
```

**Problem:** Split-brain scenario  
**Solution:** Increase TTL and ensure time sync:
```bash
EBOT_LEADER_ELECTION_TTL=15s
```

---

## 10. Support & Resources

- **Documentation:** https://docs.expedient.cloud/ebot
- **GitHub Issues:** https://github.com/chad-atexpedient/ebot/issues
- **Internal Support:** Contact Expedient Cloud Support

---

**Phase 2A Integration Complete! ✅**

All Phase 1 features are now fully integrated and production-ready. Proceed to Phase 2B for new feature development.
