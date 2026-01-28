# Multi-Region Support & Data Residency

## Overview

ebot's multi-region support enables organizations to deploy across multiple geographic regions while maintaining compliance with data residency requirements (GDPR, data sovereignty laws, etc.).

## Key Features

### ✅ **Regional Deployment**
- Deploy ebot in multiple geographic regions
- Each region operates independently
- Global control plane for coordination

### ✅ **Data Residency Policies**
- Enforce where data can be stored
- User-level and workspace-level policies
- Compliance framework support (GDPR, HIPAA, SOC 2)

### ✅ **Cross-Region Replication**
- Optional data replication across regions
- Configurable replication lag
- Automatic failover support

### ✅ **Region-Aware Routing**
- Automatic routing to correct region
- Policy-based routing decisions
- Latency-optimized routing

## Architecture

### Deployment Models

#### **1. Single Region (Default)**
```
┌─────────────────┐
│   ebot Region   │
│   (us-east-1)   │
│                 │
│  ┌───────────┐  │
│  │ Control   │  │
│  │ Plane     │  │
│  └───────────┘  │
│  ┌───────────┐  │
│  │ Data      │  │
│  │ Plane     │  │
│  └───────────┘  │
└─────────────────┘
```

#### **2. Multi-Region with Global Control Plane**
```
┌──────────────────────────────────────────┐
│       Global Control Plane               │
│   (Region Manager, Policy Engine)        │
└──────────────────────────────────────────┘
           │              │
    ┌──────┴────┐   ┌────┴──────┐
    │           │   │           │
┌───▼──────┐  ┌─▼────────┐  ┌──▼───────┐
│ us-east-1│  │ eu-west-1│  │ ap-south-1│
│          │  │          │  │           │
│ Data     │  │ Data     │  │ Data      │
│ Plane    │  │ Plane    │  │ Plane     │
└──────────┘  └──────────┘  └───────────┘
```

#### **3. Multi-Region with Replication**
```
┌────────────┐           ┌────────────┐
│ us-east-1  │◄─────────►│ us-west-2  │
│ (Primary)  │ Replicate │ (Replica)  │
└────────────┘           └────────────┘
      │
      │ Replicate
      ▼
┌────────────┐
│ eu-west-1  │
│ (Replica)  │
└────────────┘
```

## Configuration

### Enable Multi-Region

```bash
# Environment variables
export EBOT_MULTIREGION_ENABLED=true
export EBOT_CURRENT_REGION=us-east-1
export EBOT_CONTROL_PLANE_ENDPOINT=https://control.ebot.expedient.cloud
export EBOT_REPLICATION_ENABLED=true
export EBOT_REPLICATION_INTERVAL=5m
```

### Register a Region

```go
package main

import (
    "context"
    "github.com/chad-atexpedient/ebot/pkg/region"
)

func registerRegion() error {
    manager := region.NewManager(config, logger)
    
    region := &region.Region{
        ID:          "us-east-1",
        Name:        "us-east-1",
        DisplayName: "US East (Virginia)",
        Location:    "Northern Virginia, USA",
        Endpoint:    "https://us-east-1.ebot.expedient.cloud",
        Status:      region.RegionStatusActive,
        Capabilities: []string{
            "mcp-hosting",
            "knowledge-storage",
            "model-inference",
        },
        Metadata: map[string]string{
            "cloud":     "aws",
            "zone":      "us-east-1a",
            "certified": "SOC2,HIPAA",
        },
    }
    
    return manager.RegisterRegion(context.Background(), region)
}
```

### Create Data Residency Policy

```go
policy := &region.DataResidencyPolicy{
    ID:             "policy-gdpr-eu",
    Name:           "GDPR EU Data Residency",
    WorkspaceID:    "workspace-123",
    PrimaryRegion:  "eu-west-1",
    AllowedRegions: []string{"eu-west-1", "eu-central-1"},
    CrossRegionReplication: true,
    ReplicationRegions: []string{"eu-central-1"},
    ComplianceFrameworks: []string{"GDPR", "SOC2"},
    DataExportRestrictions: []string{"us-east-1"}, // No export to US
}

manager.CreatePolicy(context.Background(), policy)
```

## API Reference

### List Regions

```bash
GET /api/regions
```

**Response:**
```json
{
  "regions": [
    {
      "id": "us-east-1",
      "name": "us-east-1",
      "displayName": "US East (Virginia)",
      "location": "Northern Virginia, USA",
      "status": "active",
      "capabilities": ["mcp-hosting", "knowledge-storage"],
      "metadata": {
        "cloud": "aws",
        "certified": "SOC2,HIPAA"
      }
    }
  ],
  "count": 1
}
```

### Get User's Allowed Regions

```bash
GET /api/regions/user?userId=user123
```

**Response:**
```json
{
  "regions": [
    {
      "id": "eu-west-1",
      "displayName": "EU West (Ireland)",
      "status": "active"
    }
  ],
  "count": 1
}
```

### Create Data Residency Policy

```bash
POST /api/data-residency/policies
Content-Type: application/json

{
  "id": "policy-123",
  "name": "EU Data Residency",
  "workspaceId": "workspace-abc",
  "primaryRegion": "eu-west-1",
  "allowedRegions": ["eu-west-1", "eu-central-1"],
  "crossRegionReplication": true,
  "replicationRegions": ["eu-central-1"],
  "complianceFrameworks": ["GDPR"]
}
```

### Route a Request

```bash
POST /api/regions/route
Content-Type: application/json

{
  "userId": "user123",
  "resourceType": "mcp-server",
  "resourceId": "server-456"
}
```

**Response:**
```json
{
  "targetRegion": "eu-west-1",
  "reason": "user-policy",
  "policyApplied": "user123",
  "latency": "2.3ms"
}
```

### Validate Operation

```bash
POST /api/regions/validate
Content-Type: application/json

{
  "userId": "user123",
  "resourceId": "server-456",
  "operation": "export",
  "targetRegion": "us-east-1"
}
```

**Response (Denied):**
```json
{
  "allowed": false,
  "reason": "operation violates data residency policy"
}
```

### Initiate Replication

```bash
POST /api/regions/replication
Content-Type: application/json

{
  "resourceId": "server-456",
  "sourceRegion": "eu-west-1",
  "targetRegion": "eu-central-1"
}
```

## Kubernetes Resources

### DataResidency CRD

```yaml
apiVersion: ebot.expedient.ai/v1
kind: DataResidency
metadata:
  name: gdpr-policy
  namespace: default
spec:
  workspaceID: workspace-123
  primaryRegion: eu-west-1
  allowedRegions:
    - eu-west-1
    - eu-central-1
  crossRegionReplication: true
  replicationRegions:
    - eu-central-1
  complianceFrameworks:
    - GDPR
    - SOC2
  retentionPolicy:
    retentionPeriodDays: 365
    deleteAfterRetention: false
    archiveBeforeDelete: true
    archiveRegion: eu-west-1
  encryptionRequired: true
```

### RegionConfig CRD

```yaml
apiVersion: ebot.expedient.ai/v1
kind: RegionConfig
metadata:
  name: us-east-1
  namespace: default
spec:
  id: us-east-1
  displayName: "US East (Virginia)"
  location: "Northern Virginia, USA"
  endpoint: "https://us-east-1.ebot.expedient.cloud"
  capabilities:
    - mcp-hosting
    - knowledge-storage
    - model-inference
  databaseEndpoint: "postgres://db.us-east-1.internal:5432/ebot"
  storageBucket: "ebot-us-east-1"
  encryptionKeyID: "arn:aws:kms:us-east-1:123456789:key/abc"
  complianceCertifications:
    - SOC2
    - HIPAA
  metadata:
    cloud: "aws"
    zone: "us-east-1a"
```

## Use Cases

### 1. GDPR Compliance - EU Data Residency

**Requirement:** EU user data must stay in EU regions only.

```go
policy := &region.DataResidencyPolicy{
    ID:             "gdpr-eu",
    Name:           "GDPR EU Users",
    UserID:         "eu-user-123",
    PrimaryRegion:  "eu-west-1",
    AllowedRegions: []string{"eu-west-1", "eu-central-1", "eu-north-1"},
    DataExportRestrictions: []string{"us-east-1", "ap-south-1"},
    ComplianceFrameworks: []string{"GDPR"},
}
```

### 2. HIPAA - Healthcare Data in US

**Requirement:** Patient data must stay in HIPAA-certified US regions.

```go
policy := &region.DataResidencyPolicy{
    ID:             "hipaa-healthcare",
    Name:           "HIPAA Healthcare Data",
    WorkspaceID:    "healthcare-workspace",
    PrimaryRegion:  "us-east-1",
    AllowedRegions: []string{"us-east-1", "us-west-2"}, // HIPAA certified
    ComplianceFrameworks: []string{"HIPAA", "SOC2"},
    EncryptionRequired: true,
}
```

### 3. High Availability with Replication

**Requirement:** Data replicated across multiple regions for DR.

```go
policy := &region.DataResidencyPolicy{
    ID:                     "ha-global",
    Name:                   "Global HA Policy",
    WorkspaceID:            "enterprise-workspace",
    PrimaryRegion:          "us-east-1",
    AllowedRegions:         []string{"us-east-1", "us-west-2", "eu-west-1"},
    CrossRegionReplication: true,
    ReplicationRegions:     []string{"us-west-2", "eu-west-1"},
}
```

## Best Practices

### ✅ **1. Plan Your Regional Strategy**
- Identify where your users are located
- Understand compliance requirements
- Choose regions accordingly

### ✅ **2. Use Data Residency Policies**
- Define policies at workspace or user level
- Enforce compliance frameworks
- Regularly audit policy adherence

### ✅ **3. Enable Replication for Critical Data**
- Replicate to at least one other region
- Monitor replication lag
- Test failover procedures

### ✅ **4. Monitor Regional Health**
- Track region status
- Set up alerts for degraded regions
- Have fallback regions configured

### ✅ **5. Test Cross-Region Operations**
- Validate routing decisions
- Test operation validations
- Verify replication works

## Troubleshooting

### Issue: "Region not found"

**Cause:** Region not registered or misspelled ID.

**Solution:**
```bash
# List all regions
curl http://localhost:8080/api/regions

# Register missing region
curl -X POST http://localhost:8080/api/regions \
  -H "Content-Type: application/json" \
  -d '{"id":"eu-west-1",...}'
```

### Issue: "Policy violation"

**Cause:** Operation violates data residency policy.

**Solution:**
1. Check the policy: `GET /api/data-residency/policies/{resourceId}`
2. Verify target region is in `allowedRegions`
3. Check for export restrictions

### Issue: "Replication lagging"

**Cause:** Network issues or high load.

**Solution:**
```bash
# Check replication status
curl http://localhost:8080/api/regions/replication/server-123

# Reduce replication interval
export EBOT_REPLICATION_INTERVAL=10m
```

## Performance Considerations

### Latency
- Cross-region latency: 50-200ms
- Same-region latency: <10ms
- Use regional routing to minimize latency

### Storage
- Per-region storage costs apply
- Replication increases storage 2-3x
- Consider compression and deduplication

### Bandwidth
- Cross-region transfer costs
- Optimize replication frequency
- Use incremental replication

## Security

### Encryption
- Data encrypted at rest in each region
- TLS 1.3 for inter-region communication
- Separate encryption keys per region

### Access Control
- Region-aware access policies
- Validate user region access
- Audit cross-region operations

## Compliance Certifications

### Supported Frameworks
- **GDPR** (General Data Protection Regulation)
- **HIPAA** (Health Insurance Portability and Accountability Act)
- **SOC 2** (Service Organization Control 2)
- **ISO 27001** (Information Security Management)
- **FedRAMP** (Federal Risk and Authorization Management Program)

### Per-Region Certifications

| Region       | SOC 2 | HIPAA | GDPR | ISO 27001 | FedRAMP |
|--------------|-------|-------|------|-----------|---------|
| us-east-1    | ✅    | ✅    | ✅   | ✅        | ✅      |
| us-west-2    | ✅    | ✅    | ✅   | ✅        | ✅      |
| eu-west-1    | ✅    | ✅    | ✅   | ✅        | ❌      |
| eu-central-1 | ✅    | ❌    | ✅   | ✅        | ❌      |
| ap-south-1   | ✅    | ❌    | ✅   | ✅        | ❌      |

## Next Steps

1. **Deploy Multi-Region:** Follow the [Deployment Guide](deployment-multiregion.md)
2. **Configure Policies:** See [Policy Configuration](policy-configuration.md)
3. **Monitor Regions:** Set up [Regional Monitoring](monitoring.md)
4. **Test Failover:** Review [DR Procedures](disaster-recovery.md)

## Support

For questions or issues:
- Documentation: https://docs.expedient.cloud/ebot/multi-region
- Support: Contact Expedient Cloud Support
- GitHub Issues: https://github.com/chad-atexpedient/ebot/issues
