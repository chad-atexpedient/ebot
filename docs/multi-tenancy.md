# Multi-Tenancy in ebot

## Overview

ebot provides enterprise-grade multi-tenancy capabilities with three isolation levels, automated provisioning, white-labeling, and comprehensive tenant lifecycle management. This guide covers all aspects of implementing and managing multi-tenant deployments.

## Table of Contents

1. [Isolation Levels](#isolation-levels)
2. [Tenant Provisioning](#tenant-provisioning)
3. [White-Labeling](#white-labeling)
4. [Resource Management](#resource-management)
5. [Security Considerations](#security-considerations)
6. [API Reference](#api-reference)
7. [Best Practices](#best-practices)

---

## Isolation Levels

ebot supports three levels of tenant isolation to meet different security and compliance requirements:

### 1. Shared Isolation

**Use Case:** Small deployments, development/testing, cost-sensitive scenarios

**Characteristics:**
- Shared database with row-level security (RLS)
- Logical separation via `tenant_id` columns
- Shared infrastructure and resources
- Lowest cost, easiest to manage
- Suitable for non-sensitive data

**Implementation:**
```sql
-- PostgreSQL Row-Level Security
CREATE POLICY tenant_isolation ON users
  USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### 2. Logical Isolation (Recommended)

**Use Case:** Most enterprise deployments, GDPR compliance, moderate security requirements

**Characteristics:**
- Dedicated database schema per tenant
- Separate tables and indexes
- Shared database instance
- Per-tenant encryption keys
- Good balance of isolation and cost
- GDPR compliant

**Implementation:**
```sql
-- Dedicated schema
CREATE SCHEMA tenant_abc123;
CREATE TABLE tenant_abc123.users (...);
CREATE TABLE tenant_abc123.projects (...);
```

### 3. Physical Isolation

**Use Case:** High-security requirements, HIPAA, financial services, government

**Characteristics:**
- Dedicated database instance per tenant
- Complete network isolation
- Per-tenant infrastructure
- Highest security and compliance
- Higher cost and complexity
- Suitable for regulated industries

**Implementation:**
- Separate RDS instance or PostgreSQL server
- Dedicated VPC/network
- Separate encryption keys and certificates

---

## Tenant Provisioning

### Automated Provisioning

ebot provides fully automated tenant provisioning with progress tracking:

```go
import "github.com/chad-atexpedient/ebot/pkg/multitenancy"

// Create provisioning request
req := multitenancy.ProvisioningRequest{
    TenantName:      "Acme Corporation",
    IsolationLevel:  multitenancy.IsolationLevelLogical,
    Region:          "us-east-1",
    AdminEmail:      "admin@acme.com",
    
    // Resource limits
    MaxUsers:        100,
    MaxStorage:      100 * 1024 * 1024 * 1024, // 100 GB
    MaxMCPServers:   50,
    
    // Features
    EnableCustomDomain:  true,
    EnableWhiteLabeling: true,
    EnableAuditLogs:     true,
    
    // Branding
    BrandingConfig: &multitenancy.BrandingConfig{
        LogoURL:        "https://acme.com/logo.png",
        PrimaryColor:   "#0066CC",
        SecondaryColor: "#FF9900",
        CompanyName:    "Acme Corporation",
        CustomDomain:   "ebot.acme.com",
    },
}

// Provision tenant
tenant, err := provisioningMgr.ProvisionTenant(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Tenant %s provisioned successfully\n", tenant.ID)
```

### Provisioning Steps

The system executes these steps automatically:

1. **Create Tenant Record** - Register tenant in system
2. **Provision Database** - Create schema/database based on isolation level
3. **Setup Encryption** - Generate per-tenant encryption keys
4. **Configure Network Isolation** - Set up network policies
5. **Create Admin User** - Initial administrator account
6. **Apply Resource Quotas** - Set resource limits
7. **Setup Branding** (optional) - Apply white-label configuration
8. **Configure Custom Domain** (optional) - DNS and SSL setup
9. **Initialize Audit System** (optional) - Enable audit logging
10. **Send Welcome Email** - Notify administrator

### Monitoring Provisioning

```go
// Check provisioning status
status, err := provisioningMgr.GetProvisioningStatus(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", status.Status)
fmt.Printf("Progress: %d%%\n", status.Progress)
fmt.Printf("Current Step: %s\n", status.CurrentStep)

for _, step := range status.Steps {
    fmt.Printf("  - %s: %s\n", step.Name, step.Status)
}
```

---

## White-Labeling

### Configuration

Full white-labeling support for enterprise customers:

```go
branding := &multitenancy.BrandingConfig{
    // Visual identity
    LogoURL:        "https://cdn.example.com/logo.png",
    PrimaryColor:   "#1E40AF",
    SecondaryColor: "#F59E0B",
    CompanyName:    "Your Company",
    
    // Custom domain
    CustomDomain: "ai.yourcompany.com",
    
    // Custom styling
    CustomCSS: `
        .header { background: linear-gradient(to right, #1E40AF, #3B82F6); }
        .btn-primary { background-color: #1E40AF; }
    `,
    
    // Email templates
    EmailTemplate: "custom-template-v2",
    
    // Additional metadata
    CustomMetadata: map[string]string{
        "support_email": "support@yourcompany.com",
        "support_phone": "+1-800-123-4567",
        "terms_url":     "https://yourcompany.com/terms",
    },
}
```

### Custom Domains

1. **DNS Configuration:**
   ```
   CNAME ebot.yourcompany.com -> platform.ebot.expedient.cloud
   ```

2. **SSL Certificate:**
   - Automatically provisioned via Let's Encrypt
   - Or upload custom certificate

3. **Verification:**
   ```bash
   curl https://ebot.yourcompany.com/health
   # Should return 200 OK
   ```

---

## Resource Management

### Resource Quotas

Set per-tenant resource limits:

```go
// Apply quotas during provisioning
req := multitenancy.ProvisioningRequest{
    MaxUsers:      100,     // Maximum users in tenant
    MaxStorage:    100 * GB, // Maximum storage
    MaxMCPServers: 50,      // Maximum MCP servers
}

// Scale existing tenant
scaling := multitenancy.ScalingOptions{
    MaxUsers:      intPtr(200),
    MaxStorage:    int64Ptr(200 * GB),
    MaxMCPServers: intPtr(100),
}
provisioningMgr.ScaleTenant(ctx, tenantID, scaling)
```

### Usage Monitoring

```go
// Get tenant context
tenantCtx, err := isolationMgr.GetTenantContext(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}

// Check resource usage
usage := quotaMgr.GetUsage(ctx, tenantID)
fmt.Printf("Users: %d/%d\n", usage.CurrentUsers, usage.MaxUsers)
fmt.Printf("Storage: %s/%s\n", humanize.Bytes(usage.UsedStorage), humanize.Bytes(usage.MaxStorage))
fmt.Printf("MCP Servers: %d/%d\n", usage.ActiveServers, usage.MaxMCPServers)
```

---

## Security Considerations

### Data Isolation

**Shared Isolation:**
- Row-Level Security (RLS) in PostgreSQL
- Set `tenant_id` context for all queries
- Validate tenant access in application layer

**Logical Isolation:**
- Separate schema per tenant
- Schema-level permissions
- Per-tenant encryption keys
- No cross-tenant data access possible

**Physical Isolation:**
- Dedicated database instance
- Network segmentation
- Separate infrastructure
- Complete isolation guarantee

### Encryption

**Per-Tenant Keys:**
```go
// Automatic per-tenant encryption
tenant, _ := isolationMgr.CreateTenant(ctx, &multitenancy.Tenant{
    Name:           "Acme",
    IsolationLevel: multitenancy.IsolationLevelLogical,
})

// Encryption key automatically generated
fmt.Printf("Encryption Key ID: %s\n", tenant.EncryptionKeyID)
```

**Key Rotation:**
```go
// Rotate tenant encryption key (annual recommended)
err := keyMgr.RotateTenantKey(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}
```

### Network Isolation

**Kubernetes NetworkPolicy:**
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tenant-abc123
spec:
  podSelector:
    matchLabels:
      tenant: abc123
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          tenant: abc123
```

---

## API Reference

### REST Endpoints

#### Create Tenant
```http
POST /api/tenants
Content-Type: application/json

{
  "tenantName": "Acme Corporation",
  "isolationLevel": "logical",
  "region": "us-east-1",
  "adminEmail": "admin@acme.com",
  "maxUsers": 100,
  "maxStorage": 107374182400,
  "maxMcpServers": 50,
  "enableCustomDomain": true,
  "enableWhiteLabeling": true,
  "brandingConfig": {
    "logoUrl": "https://acme.com/logo.png",
    "primaryColor": "#0066CC",
    "companyName": "Acme Corporation",
    "customDomain": "ebot.acme.com"
  }
}
```

**Response:**
```json
{
  "id": "tenant-1234567890",
  "name": "Acme Corporation",
  "status": "provisioning",
  "createdAt": "2026-01-28T10:00:00Z"
}
```

#### Get Provisioning Status
```http
GET /api/tenants/{tenantId}/provisioning-status
```

**Response:**
```json
{
  "tenantId": "tenant-1234567890",
  "status": "active",
  "progress": 100,
  "currentStep": "Send Welcome Email",
  "steps": [
    {
      "name": "Create Tenant Record",
      "status": "completed",
      "duration": "0.5s"
    },
    {
      "name": "Provision Database",
      "status": "completed",
      "duration": "2.3s"
    }
  ]
}
```

#### Scale Tenant
```http
PUT /api/tenants/{tenantId}/scale
Content-Type: application/json

{
  "maxUsers": 200,
  "maxStorage": 214748364800,
  "maxMcpServers": 100
}
```

#### Delete Tenant
```http
DELETE /api/tenants/{tenantId}
```

---

## Best Practices

### Choosing Isolation Level

**Use Shared when:**
- Development/testing environments
- Cost is primary concern
- Data is non-sensitive
- Small number of tenants

**Use Logical when:**
- Production deployments
- GDPR compliance required
- Moderate security requirements
- Most enterprise use cases

**Use Physical when:**
- HIPAA compliance required
- Financial services
- Government/defense
- Highest security needs
- Customer explicitly requests dedicated infrastructure

### Performance Optimization

1. **Connection Pooling:**
   ```go
   // Configure per-tenant connection pools
   pool := &pgxpool.Config{
       MaxConns:          25,
       MinConns:          5,
       MaxConnLifetime:   time.Hour,
       MaxConnIdleTime:   30 * time.Minute,
   }
   ```

2. **Caching:**
   ```go
   // Cache tenant context to avoid repeated lookups
   tenantCtx := cache.GetOrSet(tenantID, func() interface{} {
       return isolationMgr.GetTenantContext(ctx, tenantID)
   })
   ```

3. **Indexes:**
   ```sql
   -- Always index tenant_id in shared isolation
   CREATE INDEX idx_users_tenant ON users(tenant_id);
   CREATE INDEX idx_projects_tenant ON projects(tenant_id);
   ```

### Monitoring

```go
// Track tenant metrics
metrics := []string{
    "tenant.users.count",
    "tenant.storage.bytes",
    "tenant.api.requests",
    "tenant.cost.monthly",
}

for _, metric := range metrics {
    value := monitor.GetMetric(tenantID, metric)
    fmt.Printf("%s: %v\n", metric, value)
}
```

### Disaster Recovery

```go
// Backup tenant data
backup, err := backupMgr.CreateTenantBackup(ctx, tenantID, backupOpts)

// Restore tenant
err = backupMgr.RestoreTenantBackup(ctx, tenantID, backupID)

// Cross-region replication
err = isolationMgr.MigrateTenant(ctx, tenantID, multitenancy.MigrationOptions{
    TargetRegion: "eu-west-1",
    MigrateData:  true,
})
```

---

## Troubleshooting

### Tenant Provisioning Fails

**Check provisioning status:**
```go
status, _ := provisioningMgr.GetProvisioningStatus(ctx, tenantID)
for _, step := range status.Steps {
    if step.Status == "failed" {
        fmt.Printf("Failed step: %s\n", step.Name)
        fmt.Printf("Error: %s\n", step.Error)
    }
}
```

**Common issues:**
- Database connection limits reached
- Network policy conflicts
- Insufficient permissions
- Resource quota exceeded

### Cross-Tenant Data Leakage

**Verify isolation:**
```sql
-- Test RLS policies
SET app.current_tenant = 'tenant-123';
SELECT * FROM users; -- Should only return tenant-123 data

SET app.current_tenant = 'tenant-456';
SELECT * FROM users; -- Should only return tenant-456 data
```

### Performance Issues

**Check tenant metrics:**
```bash
# Query latency per tenant
SELECT tenant_id, AVG(query_duration_ms)
FROM query_logs
WHERE timestamp > NOW() - INTERVAL '1 hour'
GROUP BY tenant_id
ORDER BY avg DESC;

# Connection count per tenant
SELECT tenant_id, COUNT(*)
FROM pg_stat_activity
GROUP BY tenant_id;
```

---

## Migration Guide

### From Single-Tenant to Multi-Tenant

1. **Add tenant_id columns:**
   ```sql
   ALTER TABLE users ADD COLUMN tenant_id UUID;
   ALTER TABLE projects ADD COLUMN tenant_id UUID;
   ```

2. **Create RLS policies:**
   ```sql
   ALTER TABLE users ENABLE ROW LEVEL SECURITY;
   CREATE POLICY tenant_isolation ON users
     USING (tenant_id = current_setting('app.current_tenant')::uuid);
   ```

3. **Update application code:**
   ```go
   // Set tenant context for all queries
   ctx = context.WithValue(ctx, "tenant_id", tenantID)
   ```

### From Shared to Logical Isolation

```go
// Migrate tenant to dedicated schema
err := isolationMgr.MigrateTenant(ctx, tenantID, multitenancy.MigrationOptions{
    TargetIsolationLevel: multitenancy.IsolationLevelLogical,
    MigrateData:          true,
    ValidationRequired:   true,
})
```

---

## Additional Resources

- [GDPR Compliance Guide](./compliance-gdpr.md)
- [HIPAA Compliance Guide](./healthcare-hipaa.md)
- [Performance Tuning](./performance.md)
- [API Reference](./api-reference.md)

---

**For support:** Contact Expedient Cloud Support
**Documentation:** https://docs.expedient.cloud/ebot/multi-tenancy
