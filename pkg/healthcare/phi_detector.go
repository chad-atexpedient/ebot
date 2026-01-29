package healthcare

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// PHIType represents different types of Protected Health Information
type PHIType string

const (
	PHITypeMedicalRecordNumber PHIType = "medical_record_number"
	PHITypeHealthPlanNumber    PHIType = "health_plan_number"
	PHITypeAccountNumber       PHIType = "account_number"
	PHITypeCertificateNumber   PHIType = "certificate_number"
	PHITypeLicenseNumber       PHIType = "license_number"
	PHITypeVehicleIdentifier   PHIType = "vehicle_identifier"
	PHITypeDeviceIdentifier    PHIType = "device_identifier"
	PHITypeWebURL              PHIType = "web_url"
	PHITypeIPAddress           PHIType = "ip_address"
	PHITypeBiometricIdentifier PHIType = "biometric_identifier"
	PHITypeFacePhoto           PHIType = "face_photo"
	PHITypeSSN                 PHIType = "ssn"
	PHITypeEmail               PHIType = "email"
	PHITypePhoneNumber         PHIType = "phone_number"
	PHITypeFaxNumber           PHIType = "fax_number"
	PHITypeDateOfBirth         PHIType = "date_of_birth"
	PHITypeName                PHIType = "name"
	PHITypeAddress             PHIType = "address"
	PHITypeDiagnosisCode       PHIType = "diagnosis_code"
	PHITypeProcedureCode       PHIType = "procedure_code"
	PHITypeAge                 PHIType = "age"
)

// PHILocation represents where PHI was found in the text
type PHILocation struct {
	Start  int     `json:"start"`
	End    int     `json:"end"`
	Text   string  `json:"text"`
	Type   PHIType `json:"type"`
	Score  float64 `json:"score"` // Confidence score 0.0-1.0
	Reason string  `json:"reason"`
}

// PHIReport contains detection results
type PHIReport struct {
	ContainsPHI   bool          `json:"contains_phi"`
	PHITypes      []PHIType     `json:"phi_types"`
	Locations     []PHILocation `json:"locations"`
	RiskLevel     RiskLevel     `json:"risk_level"`
	DetectionTime time.Duration `json:"detection_time"`
	RedactedText  string        `json:"redacted_text,omitempty"`
	TotalPHICount int           `json:"total_phi_count"`
	HighRiskCount int           `json:"high_risk_count"`
}

// RiskLevel indicates the severity of PHI exposure
type RiskLevel string

const (
	RiskLevelNone     RiskLevel = "none"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// PHIDetector interface for PHI detection
type PHIDetector interface {
	// ScanForPHI scans text for PHI
	ScanForPHI(ctx context.Context, text string) (*PHIReport, error)

	// RedactPHI redacts PHI from text
	RedactPHI(ctx context.Context, text string) (string, error)

	// ValidateHIPAACompliance checks if text is HIPAA compliant
	ValidateHIPAACompliance(ctx context.Context, text string) (bool, []string, error)
}

// phiDetectorImpl implements PHIDetector
type phiDetectorImpl struct {
	patterns map[PHIType]*regexp.Regexp
	mu       sync.RWMutex
}

// NewPHIDetector creates a new PHI detector
func NewPHIDetector() PHIDetector {
	d := &phiDetectorImpl{
		patterns: make(map[PHIType]*regexp.Regexp),
	}

	// Initialize regex patterns for each PHI type
	d.initializePatterns()

	return d
}

// initializePatterns sets up regex patterns for PHI detection
func (d *phiDetectorImpl) initializePatterns() {
	d.patterns[PHITypeSSN] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b|\b\d{9}\b`)
	d.patterns[PHITypeEmail] = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	d.patterns[PHITypePhoneNumber] = regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b|\(\d{3}\)\s*\d{3}[-.]?\d{4}`)
	d.patterns[PHITypeFaxNumber] = regexp.MustCompile(`(?i)\bfax:?\s*\d{3}[-.]?\d{3}[-.]?\d{4}\b`)
	d.patterns[PHITypeDateOfBirth] = regexp.MustCompile(`\b(0[1-9]|1[0-2])/(0[1-9]|[12][0-9]|3[01])/(19|20)\d{2}\b`)
	d.patterns[PHITypeMedicalRecordNumber] = regexp.MustCompile(`(?i)\bMRN:?\s*[A-Z0-9]{6,12}\b|\bmedical\s+record\s+#?\s*[A-Z0-9]{6,12}\b`)
	d.patterns[PHITypeHealthPlanNumber] = regexp.MustCompile(`(?i)\b(health\s+plan|insurance)\s+#?\s*[A-Z0-9]{6,15}\b`)
	d.patterns[PHITypeIPAddress] = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	d.patterns[PHITypeDiagnosisCode] = regexp.MustCompile(`(?i)\b(ICD-10|ICD-9):?\s*[A-Z0-9.]{3,7}\b`)
	d.patterns[PHITypeProcedureCode] = regexp.MustCompile(`(?i)\b(CPT|HCPCS):?\s*[0-9]{4,5}[A-Z]?\b`)

	// Address pattern (simplified)
	d.patterns[PHITypeAddress] = regexp.MustCompile(`(?i)\b\d+\s+[A-Z][a-z]+\s+(Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Lane|Ln|Drive|Dr|Court|Ct)\b`)

	// Name pattern (simplified - looks for Title + Name patterns)
	d.patterns[PHITypeName] = regexp.MustCompile(`\b(Mr|Mrs|Ms|Dr|Prof)\.?\s+[A-Z][a-z]+\s+[A-Z][a-z]+\b`)
}

// scanInternal performs the actual PHI scanning without generating redacted text.
// This is the core scanning logic that avoids recursion by not calling RedactPHI.
func (d *phiDetectorImpl) scanInternal(text string) []PHILocation {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var locations []PHILocation

	for phiType, pattern := range d.patterns {
		matches := pattern.FindAllStringIndex(text, -1)

		for _, match := range matches {
			location := PHILocation{
				Start:  match[0],
				End:    match[1],
				Text:   text[match[0]:match[1]],
				Type:   phiType,
				Score:  d.calculateConfidenceScore(phiType, text[match[0]:match[1]]),
				Reason: fmt.Sprintf("Matched pattern for %s", phiType),
			}
			locations = append(locations, location)
		}
	}

	return locations
}

// applyRedactions applies redactions to text based on PHI locations.
// Locations are sorted in descending order by start position to avoid offset issues.
func (d *phiDetectorImpl) applyRedactions(text string, locations []PHILocation) string {
	if len(locations) == 0 {
		return text
	}

	// Create a copy to avoid modifying the original slice
	sortedLocations := make([]PHILocation, len(locations))
	copy(sortedLocations, locations)

	// Sort by start position descending using sort.Slice (O(n log n))
	sort.Slice(sortedLocations, func(i, j int) bool {
		return sortedLocations[i].Start > sortedLocations[j].Start
	})

	// Redact from end to start to avoid offset issues
	redacted := text
	for _, location := range sortedLocations {
		redactionText := d.getRedactionText(location.Type)
		redacted = redacted[:location.Start] + redactionText + redacted[location.End:]
	}

	return redacted
}

// ScanForPHI scans text for PHI and returns a comprehensive report.
// This method is safe from recursion as it uses scanInternal and applyRedactions.
func (d *phiDetectorImpl) ScanForPHI(ctx context.Context, text string) (*PHIReport, error) {
	startTime := time.Now()

	// Use internal scan to avoid recursion
	locations := d.scanInternal(text)

	report := &PHIReport{
		ContainsPHI:   len(locations) > 0,
		PHITypes:      []PHIType{},
		Locations:     locations,
		RiskLevel:     RiskLevelNone,
		TotalPHICount: len(locations),
	}

	// Build PHI types list and count high-risk items
	seenTypes := make(map[PHIType]bool)
	for _, loc := range locations {
		if !seenTypes[loc.Type] {
			report.PHITypes = append(report.PHITypes, loc.Type)
			seenTypes[loc.Type] = true
		}

		if d.isHighRiskPHI(loc.Type) {
			report.HighRiskCount++
		}
	}

	// Calculate risk level
	report.RiskLevel = d.calculateRiskLevel(report)

	// Generate redacted text using applyRedactions (no recursion)
	if report.ContainsPHI {
		report.RedactedText = d.applyRedactions(text, locations)
	}

	report.DetectionTime = time.Since(startTime)

	return report, nil
}

// RedactPHI redacts PHI from text and returns the redacted string.
// This method is safe from recursion as it uses scanInternal directly.
func (d *phiDetectorImpl) RedactPHI(ctx context.Context, text string) (string, error) {
	// Use internal scan to avoid recursion (don't call ScanForPHI here)
	locations := d.scanInternal(text)

	if len(locations) == 0 {
		return text, nil
	}

	return d.applyRedactions(text, locations), nil
}

// ValidateHIPAACompliance checks if text is HIPAA compliant
func (d *phiDetectorImpl) ValidateHIPAACompliance(ctx context.Context, text string) (bool, []string, error) {
	report, err := d.ScanForPHI(ctx, text)
	if err != nil {
		return false, nil, err
	}

	violations := []string{}

	if report.ContainsPHI {
		violations = append(violations, fmt.Sprintf("Text contains %d instances of PHI", report.TotalPHICount))

		if report.HighRiskCount > 0 {
			violations = append(violations, fmt.Sprintf("Text contains %d high-risk PHI identifiers", report.HighRiskCount))
		}

		for _, phiType := range report.PHITypes {
			violations = append(violations, fmt.Sprintf("PHI type detected: %s", phiType))
		}
	}

	return len(violations) == 0, violations, nil
}

// calculateConfidenceScore calculates confidence that this is PHI
func (d *phiDetectorImpl) calculateConfidenceScore(phiType PHIType, text string) float64 {
	// High confidence for structured data
	switch phiType {
	case PHITypeSSN, PHITypeMedicalRecordNumber, PHITypeHealthPlanNumber:
		return 0.95
	case PHITypeEmail, PHITypePhoneNumber, PHITypeDateOfBirth:
		return 0.90
	case PHITypeDiagnosisCode, PHITypeProcedureCode:
		return 0.85
	case PHITypeAddress, PHITypeName:
		return 0.70 // Lower confidence, more false positives
	default:
		return 0.80
	}
}

// isHighRiskPHI determines if PHI type is high risk
func (d *phiDetectorImpl) isHighRiskPHI(phiType PHIType) bool {
	switch phiType {
	case PHITypeSSN, PHITypeMedicalRecordNumber, PHITypeHealthPlanNumber,
		PHITypeBiometricIdentifier, PHITypeFacePhoto:
		return true
	default:
		return false
	}
}

// calculateRiskLevel determines overall risk level
func (d *phiDetectorImpl) calculateRiskLevel(report *PHIReport) RiskLevel {
	if !report.ContainsPHI {
		return RiskLevelNone
	}

	// Critical if high-risk PHI found
	if report.HighRiskCount > 0 {
		return RiskLevelCritical
	}

	// Risk based on quantity
	if report.TotalPHICount >= 10 {
		return RiskLevelHigh
	} else if report.TotalPHICount >= 5 {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

// getRedactionText returns the redaction text for a PHI type
func (d *phiDetectorImpl) getRedactionText(phiType PHIType) string {
	typeStr := strings.ToUpper(string(phiType))
	typeStr = strings.ReplaceAll(typeStr, "_", " ")
	return fmt.Sprintf("[REDACTED %s]", typeStr)
}

// De-identification functions

// DeidentificationMethod represents different de-identification techniques
type DeidentificationMethod string

const (
	MethodRedaction      DeidentificationMethod = "redaction"
	MethodGeneralization DeidentificationMethod = "generalization"
	MethodPerturbation   DeidentificationMethod = "perturbation"
	MethodSuppression    DeidentificationMethod = "suppression"
)

// DeidentifyOptions configures de-identification
type DeidentifyOptions struct {
	Method            DeidentificationMethod
	PreserveStructure bool  // Keep data structure (e.g., date format)
	ReplacementSeed   int64 // For consistent pseudonymization
}

// Deidentify applies de-identification to text
func (d *phiDetectorImpl) Deidentify(ctx context.Context, text string, opts DeidentifyOptions) (string, error) {
	switch opts.Method {
	case MethodRedaction:
		return d.RedactPHI(ctx, text)
	case MethodGeneralization:
		return d.generalizePHI(ctx, text)
	case MethodSuppression:
		return "", nil // Complete suppression
	default:
		return d.RedactPHI(ctx, text)
	}
}

// generalizePHI generalizes PHI (e.g., specific dates to year only)
func (d *phiDetectorImpl) generalizePHI(ctx context.Context, text string) (string, error) {
	// Use internal scan to avoid recursion
	locations := d.scanInternal(text)

	if len(locations) == 0 {
		return text, nil
	}

	// Sort by start position descending
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Start > locations[j].Start
	})

	generalized := text
	for _, location := range locations {
		var replacement string

		switch location.Type {
		case PHITypeDateOfBirth:
			// Replace with year only
			replacement = "[YEAR ONLY]"
		case PHITypeAddress:
			// Replace with city/state only
			replacement = "[CITY, STATE]"
		case PHITypeAge:
			// Replace with age range
			replacement = "[AGE RANGE: XX-XX]"
		default:
			replacement = d.getRedactionText(location.Type)
		}

		generalized = generalized[:location.Start] + replacement + generalized[location.End:]
	}

	return generalized, nil
}
