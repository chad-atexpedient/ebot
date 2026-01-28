# ebot Healthcare & HIPAA Compliance Guide

## Overview

ebot provides comprehensive **HIPAA (Health Insurance Portability and Accountability Act)** compliance features for healthcare organizations. This guide covers PHI protection, BAA management, and breach notification automation.

## Table of Contents

1. [HIPAA Compliance Features](#hipaa-compliance-features)
2. [PHI Detection & Protection](#phi-detection--protection)
3. [Business Associate Agreements](#business-associate-agreements)
4. [Breach Notification](#breach-notification)
5. [Configuration](#configuration)
6. [Best Practices](#best-practices)
7. [Audit & Reporting](#audit--reporting)

---

## HIPAA Compliance Features

ebot supports healthcare organizations with:

- ✅ **PHI Detection** - Automatically detect 18+ types of Protected Health Information
- ✅ **Data Redaction** - Redact or de-identify PHI automatically
- ✅ **BAA Management** - Track and manage Business Associate Agreements
- ✅ **Breach Notification** - Automated breach notification per HIPAA rules
- ✅ **Audit Logs** - Comprehensive audit trails for all PHI access
- ✅ **Access Controls** - Role-based access with minimum necessary enforcement
- ✅ **Encryption** - At-rest and in-transit encryption for all PHI

---

## PHI Detection & Protection

### Supported PHI Types

ebot can detect 18 HIPAA identifiers plus additional healthcare-specific data:

1. **Names** - Patient, provider, guarantor names
2. **Geographic Subdivisions** - Addresses, cities, states, ZIP codes
3. **Dates** - Birth dates, admission/discharge dates, death dates
4. **Phone Numbers** - All telephone numbers
5. **Fax Numbers** - Fax machine numbers
6. **Email Addresses** - Electronic mail addresses
7. **Social Security Numbers** - SSN detection
8. **Medical Record Numbers** - MRN, patient IDs
9. **Health Plan Numbers** - Insurance member IDs
10. **Account Numbers** - Financial account numbers
11. **Certificate/License Numbers** - Professional licenses
12. **Vehicle Identifiers** - License plates, VINs
13. **Device Identifiers** - Medical device serial numbers
14. **Web URLs** - Universal resource locators
15. **IP Addresses** - Internet protocol addresses
16. **Biometric Identifiers** - Fingerprints, voice prints
17. **Face Photos** - Photographic images
18. **Diagnosis Codes** - ICD-10, ICD-9 codes
19. **Procedure Codes** - CPT, HCPCS codes

### Usage Examples

#### Detect PHI in Text

```go
package main

import (
    "context"
    "fmt"
    "github.com/chad-atexpedient/ebot/pkg/healthcare"
)

func main() {
    detector := healthcare.NewPHIDetector()
    
    text := `
    Patient: John Doe
    DOB: 05/15/1980
    SSN: 123-45-6789
    MRN: MRN-987654
    Phone: 555-123-4567
    Email: john.doe@example.com
    Address: 123 Main Street, Boston, MA 02101
    
    Diagnosis: ICD-10: E11.9 (Type 2 Diabetes)
    Procedure: CPT: 99213 (Office visit)
    `
    
    report, err := detector.ScanForPHI(context.Background(), text)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Contains PHI: %v\n", report.ContainsPHI)
    fmt.Printf("Total PHI Count: %d\n", report.TotalPHICount)
    fmt.Printf("Risk Level: %s\n", report.RiskLevel)
    fmt.Printf("PHI Types Found: %v\n", report.PHITypes)
    
    for _, location := range report.Locations {
        fmt.Printf("  - %s: %s (confidence: %.2f)\n", 
            location.Type, location.Text, location.Score)
    }
}
```

Output:
```
Contains PHI: true
Total PHI Count: 9
Risk Level: critical
PHI Types Found: [name date_of_birth ssn medical_record_number phone_number email address diagnosis_code procedure_code]
  - name: John Doe (confidence: 0.70)
  - date_of_birth: 05/15/1980 (confidence: 0.90)
  - ssn: 123-45-6789 (confidence: 0.95)
  - medical_record_number: MRN-987654 (confidence: 0.95)
  ...
```

#### Redact PHI

```go
redacted, err := detector.RedactPHI(context.Background(), text)
if err != nil {
    panic(err)
}

fmt.Println(redacted)
```

Output:
```
Patient: [REDACTED NAME]
DOB: [REDACTED DATE OF BIRTH]
SSN: [REDACTED SSN]
MRN: [REDACTED MEDICAL RECORD NUMBER]
Phone: [REDACTED PHONE NUMBER]
Email: [REDACTED EMAIL]
Address: [REDACTED ADDRESS]

Diagnosis: [REDACTED DIAGNOSIS CODE]
Procedure: [REDACTED PROCEDURE CODE]
```

#### De-identification (Generalization)

```go
opts := healthcare.DeidentifyOptions{
    Method:            healthcare.MethodGeneralization,
    PreserveStructure: true,
}

deidentified, err := detector.Deidentify(context.Background(), text, opts)
```

Output:
```
Patient: [NAME]
DOB: [YEAR ONLY: 1980]
SSN: [REDACTED SSN]
MRN: [REDACTED MEDICAL RECORD NUMBER]
Phone: [PHONE NUMBER]
Email: [EMAIL]
Address: [CITY, STATE: Boston, MA]

Diagnosis: [DIAGNOSIS CODE]
Procedure: [PROCEDURE CODE]
```

---

## Business Associate Agreements

### BAA Management

Track and manage Business Associate Agreements to ensure HIPAA compliance.

#### Create a BAA

```go
package main

import (
    "context"
    "time"
    "github.com/chad-atexpedient/ebot/pkg/healthcare"
)

func main() {
    baaManager := healthcare.NewBAAManager()
    
    baa := &healthcare.BAA{
        OrganizationID:        "org-123",
        OrganizationName:      "City Hospital",
        BusinessAssociateName: "Cloud EHR Provider Inc.",
        Status:                healthcare.BAAStatusActive,
        SignedDate:            time.Now().AddDate(0, -1, 0), // 1 month ago
        EffectiveDate:         time.Now().AddDate(0, -1, 0),
        ExpirationDate:        time.Now().AddDate(3, 0, 0), // 3 years
        AutoRenew:             true,
        DocumentURL:           "https://example.com/baas/123.pdf",
        Scope: []string{
            "medical_record_number",
            "name",
            "date_of_birth",
            "diagnosis_code",
            "procedure_code",
        },
        ServicesProvided: []string{
            "Electronic Health Records",
            "Data Analytics",
            "Billing Services",
        },
        ComplianceFramework: []string{
            "HIPAA",
            "HITECH",
        },
        AuditRequired: true,
        BreachNotification: healthcare.BreachNotificationConfig{
            Required:           true,
            NotificationWindow: 24 * time.Hour, // 24 hours
            ContactEmail:       "security@cloudehr.example.com",
            ContactPhone:       "1-800-555-0100",
        },
    }
    
    err := baaManager.CreateBAA(context.Background(), baa)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("BAA created: %s\n", baa.ID)
}
```

#### Validate BAA Coverage

```go
// Check if organization has valid BAA for specific PHI types
valid, err := baaManager.ValidateBAA(
    context.Background(),
    "org-123",
    []string{"medical_record_number", "diagnosis_code"},
)

if !valid {
    fmt.Println("No valid BAA covering required scope")
}
```

#### Monitor Expiring BAAs

```go
// Get BAAs expiring in next 90 days
expiring, err := baaManager.GetExpiringBAAs(
    context.Background(),
    90 * 24 * time.Hour, // 90 days
)

for _, baa := range expiring {
    fmt.Printf("BAA %s expires on %s\n", baa.ID, baa.ExpirationDate)
    
    // Auto-renew if enabled
    if baa.AutoRenew {
        newExpiration := baa.ExpirationDate.AddDate(3, 0, 0) // +3 years
        err := baaManager.RenewBAA(context.Background(), baa.ID, newExpiration)
        if err == nil {
            fmt.Printf("BAA %s renewed until %s\n", baa.ID, newExpiration)
        }
    }
}
```

---

## Breach Notification

### Automated Breach Notification

ebot automates HIPAA breach notification requirements:

- **< 500 individuals**: 60-day notification window
- **≥ 500 individuals**: Immediate notification + HHS + Media
- **Business Associates**: Per BAA requirements

#### Report a Breach

```go
package main

import (
    "context"
    "time"
    "github.com/chad-atexpedient/ebot/pkg/healthcare"
)

func main() {
    baaManager := healthcare.NewBAAManager()
    notifier := healthcare.NewBreachNotifier(baaManager)
    
    breach := &healthcare.Breach{
        OrganizationID:      "org-123",
        Type:                healthcare.BreachTypeUnauthorizedAccess,
        Severity:            healthcare.BreachSeverityHigh,
        DiscoveredAt:        time.Now(),
        Description:         "Unauthorized employee accessed patient records",
        AffectedIndividuals: 250,
        PHITypesCompromised: []healthcare.PHIType{
            healthcare.PHITypeName,
            healthcare.PHITypeDateOfBirth,
            healthcare.PHITypeMedicalRecordNumber,
            healthcare.PHITypeDiagnosisCode,
        },
        IncidentCommander:   "security-team@hospital.example.com",
        MitigationSteps: []string{
            "Terminated employee access immediately",
            "Reviewed all access logs for the past 90 days",
            "Enhanced access controls and monitoring",
            "Mandatory security training for all staff",
        },
    }
    
    // Report the breach
    err := notifier.ReportBreach(context.Background(), breach)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Breach reported: %s\n", breach.ID)
    fmt.Printf("Notification required: %v\n", breach.NotificationRequired)
    
    if breach.NotificationDeadline != nil {
        fmt.Printf("Notification deadline: %s\n", breach.NotificationDeadline)
    }
    
    // Send notifications
    err = notifier.SendNotifications(context.Background(), breach.ID)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Notifications sent: %d\n", len(breach.NotificationsSent))
}
```

#### Breach Notification Rules

| Individuals Affected | Notification Required | Timeline | Recipients |
|---------------------|----------------------|----------|-----------|
| < 500 | Yes | 60 days | Individuals |
| ≥ 500 | Yes | 60 days | Individuals, HHS, Media |
| Any (high-risk PHI) | Yes | 60 days | Individuals |
| Low probability | No | N/A | N/A |

---

## Configuration

### Environment Variables

```bash
# HIPAA Compliance Settings
EBOT_HIPAA_ENABLED=true
EBOT_HIPAA_PHI_DETECTION_ENABLED=true
EBOT_HIPAA_AUTO_REDACTION=true
EBOT_HIPAA_BREACH_NOTIFICATION_ENABLED=true

# BAA Settings
EBOT_HIPAA_BAA_REQUIRED=true
EBOT_HIPAA_BAA_AUDIT_INTERVAL=365d

# Data Retention (minimum necessary)
EBOT_HIPAA_PHI_RETENTION_DAYS=2555  # 7 years (default)
EBOT_HIPAA_AUTO_DELETE_EXPIRED_PHI=true

# Access Controls
EBOT_HIPAA_MINIMUM_NECESSARY_ENFORCEMENT=true
EBOT_HIPAA_ACCESS_LOG_RETENTION_DAYS=2555
```

### Kubernetes Configuration

```yaml
# chart/values.yaml
hipaa:
  enabled: true
  
  phi:
    detectionEnabled: true
    autoRedaction: true
    retentionDays: 2555  # 7 years
    
  baa:
    required: true
    auditInterval: 365d
    
  breach:
    notificationEnabled: true
    autoNotification: true
    
  audit:
    enabled: true
    retentionDays: 2555
    immutable: true
    
  encryption:
    atRest: true
    inTransit: true
    algorithm: "AES-256-GCM"
```

---

## Best Practices

### 1. Minimum Necessary Principle

Only access PHI that is minimally necessary for the task:

```go
// Good: Request specific fields only
request := &EHRRequest{
    Fields: []string{"patient_id", "diagnosis_code"},
}

// Bad: Request all PHI
request := &EHRRequest{
    Fields: []string{"*"}, // Avoid wildcard requests
}
```

### 2. Access Logging

Log all PHI access:

```go
// Automatic logging in ebot
auditLogger.LogPHIAccess(ctx, PHIAccessEvent{
    UserID:       currentUser.ID,
    ResourceType: "patient_record",
    ResourceID:   "patient-123",
    Action:       "read",
    PHITypes:     []string{"medical_record_number", "diagnosis_code"},
    Justification: "Patient care - review diagnosis",
})
```

### 3. Regular BAA Audits

```go
// Schedule annual BAA audits
if baa.AuditRequired {
    audits, err := scheduleAnnualAudit(baa.ID)
    if err != nil {
        log.Error("Failed to schedule audit", "baa", baa.ID)
    }
}
```

### 4. Breach Response Plan

Have a documented breach response plan:

1. **Detect** - Monitor for unauthorized access
2. **Contain** - Immediately revoke access
3. **Assess** - Determine scope and risk
4. **Notify** - Follow HIPAA notification timelines
5. **Remediate** - Fix vulnerabilities
6. **Document** - Maintain detailed records

### 5. Staff Training

- Conduct annual HIPAA training
- Document all training sessions
- Test understanding with assessments
- Maintain training records for 6 years

---

## Audit & Reporting

### Generate Compliance Report

```go
report := healthcare.GenerateHIPAAComplianceReport(ctx, HIPAAReportOptions{
    StartDate: time.Now().AddDate(0, -12, 0), // Last 12 months
    EndDate:   time.Now(),
    IncludeBreaches: true,
    IncludeBAAs: true,
    IncludePHIAccess: true,
})

fmt.Printf("Compliance Score: %d%%\n", report.ComplianceScore)
fmt.Printf("Total Breaches: %d\n", report.TotalBreaches)
fmt.Printf("Active BAAs: %d\n", report.ActiveBAAs)
fmt.Printf("PHI Access Events: %d\n", report.PHIAccessEvents)
```

### Export Audit Logs

```bash
# Export PHI access logs
curl -X GET "http://localhost:8080/api/hipaa/audit-logs?start=2025-01-01&end=2025-12-31" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Accept: application/json" \
  > phi-audit-logs-2025.json
```

---

## API Reference

### PHI Detection API

```bash
# Scan text for PHI
POST /api/hipaa/scan
Content-Type: application/json

{
  "text": "Patient John Doe, DOB: 05/15/1980, SSN: 123-45-6789"
}

# Response
{
  "contains_phi": true,
  "phi_types": ["name", "date_of_birth", "ssn"],
  "risk_level": "critical",
  "total_phi_count": 3,
  "redacted_text": "Patient [REDACTED NAME], DOB: [REDACTED DATE OF BIRTH], SSN: [REDACTED SSN]"
}
```

### BAA Management API

```bash
# Create BAA
POST /api/hipaa/baas

# List BAAs
GET /api/hipaa/baas?status=active

# Get expiring BAAs
GET /api/hipaa/baas/expiring?within=90d

# Renew BAA
POST /api/hipaa/baas/{id}/renew
```

### Breach Notification API

```bash
# Report breach
POST /api/hipaa/breaches

# Send notifications
POST /api/hipaa/breaches/{id}/notify

# Get pending notifications
GET /api/hipaa/breaches/pending
```

---

## Compliance Checklist

- [ ] BAAs signed with all business associates
- [ ] PHI detection enabled on all inputs
- [ ] Auto-redaction configured
- [ ] Breach notification process documented
- [ ] Annual HIPAA training completed
- [ ] Access controls implemented (minimum necessary)
- [ ] Audit logs enabled and immutable
- [ ] Encryption enabled (at rest + in transit)
- [ ] Regular security risk assessments conducted
- [ ] Incident response plan tested

---

## Support

For HIPAA compliance questions:
- **Documentation**: https://docs.expedient.cloud/ebot/healthcare
- **Support**: healthcare-compliance@expedient.cloud
- **Emergency**: 1-800-EXPEDIENT

---

**Disclaimer**: This guide provides technical implementation details. Consult with legal and compliance professionals to ensure full HIPAA compliance for your specific use case.
