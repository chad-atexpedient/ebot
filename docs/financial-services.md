# Financial Services Compliance

Complete guide for PCI-DSS, SOX, trade surveillance, and audit trails in ebot.

## Features

- **PCI-DSS**: Card data detection, tokenization, CDE validation
- **SOX**: Change management, four-eyes principle, period locking
- **Audit Trail**: Immutable blockchain-like transaction records
- **Trade Surveillance**: 10 pattern detection algorithms

## Quick Start

```go
// PCI Compliance
pci := finance.NewPCICompliance(logger)
report, _ := pci.ScanForCardData(ctx, text)

// SOX Controls
sox := finance.NewSOXControls(logger)
sox.ValidateChange(ctx, change)

// Audit Trail
trail := finance.NewAuditTrail(logger)
trail.RecordTransaction(ctx, tx)

// Trade Surveillance
surveillance := finance.NewTradeSurveillance(logger)
alerts, _ := surveillance.MonitorTrade(ctx, trade)
```

See full documentation at: https://docs.expedient.cloud/ebot/financial
