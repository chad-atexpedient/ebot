package compliance

import (
	"context"
	"regexp"
	"strings"
)

// PIIType represents different types of personally identifiable information
type PIIType string

const (
	PIITypeSSN           PIIType = "ssn"
	PIITypeCreditCard    PIIType = "credit_card"
	PIITypeEmail         PIIType = "email"
	PIITypePhoneNumber   PIIType = "phone_number"
	PIITypeMedicalID     PIIType = "medical_id"
	PIITypeIPAddress     PIIType = "ip_address"
	PIITypeDriverLicense PIIType = "driver_license"
	PIITypePassport      PIIType = "passport"
	PIITypeDateOfBirth   PIIType = "date_of_birth"
)

// PIILocation represents where PII was found in text
type PIILocation struct {
	Type      PIIType
	Start     int
	End       int
	MatchText string
	Confidence float64 // 0.0 to 1.0
}

// PIIReport contains the results of PII detection
type PIIReport struct {
	ContainsPII bool
	PIITypes    []PIIType
	Locations   []PIILocation
	RiskLevel   RiskLevel
}

// RiskLevel represents the severity of PII exposure
type RiskLevel string

const (
	RiskLevelNone     RiskLevel = "none"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// PIIDetector scans text for personally identifiable information
type PIIDetector interface {
	// ScanForPII scans text and returns detected PII
	ScanForPII(ctx context.Context, text string) (*PIIReport, error)
	
	// RedactPII redacts PII from text
	RedactPII(ctx context.Context, text string) (string, error)
	
	// ValidatePII checks if specific text matches a PII pattern
	ValidatePII(ctx context.Context, text string, piiType PIIType) bool
}

// piiDetector implements PIIDetector
type piiDetector struct {
	patterns map[PIIType]*regexp.Regexp
}

// NewPIIDetector creates a new PII detector
func NewPIIDetector() PIIDetector {
	return &piiDetector{
		patterns: initPatterns(),
	}
}

// initPatterns initializes regex patterns for PII detection
func initPatterns() map[PIIType]*regexp.Regexp {
	return map[PIIType]*regexp.Regexp{
		// US Social Security Number: 123-45-6789 or 123456789
		PIITypeSSN: regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`),
		
		// Credit Card: 16-digit numbers with optional spaces/dashes
		PIITypeCreditCard: regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
		
		// Email: standard email format
		PIITypeEmail: regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		
		// US Phone Number: various formats
		PIITypePhoneNumber: regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?([0-9]{3})\)?[-.\s]?([0-9]{3})[-.\s]?([0-9]{4})\b`),
		
		// IP Address: IPv4
		PIITypeIPAddress: regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`),
		
		// Medical Record Number: alphanumeric, various formats
		PIITypeMedicalID: regexp.MustCompile(`\b(?:MRN|Medical Record|Patient ID)[:\s]+([A-Z0-9]{6,12})\b`),
		
		// Date of Birth: MM/DD/YYYY or MM-DD-YYYY
		PIITypeDateOfBirth: regexp.MustCompile(`\b(?:DOB|Date of Birth)[:\s]+(\d{1,2}[-/]\d{1,2}[-/]\d{2,4})\b`),
	}
}

// ScanForPII scans text and returns detected PII
func (d *piiDetector) ScanForPII(ctx context.Context, text string) (*PIIReport, error) {
	report := &PIIReport{
		ContainsPII: false,
		PIITypes:    []PIIType{},
		Locations:   []PIILocation{},
		RiskLevel:   RiskLevelNone,
	}
	
	seenTypes := make(map[PIIType]bool)
	
	for piiType, pattern := range d.patterns {
		matches := pattern.FindAllStringIndex(text, -1)
		
		if len(matches) > 0 {
			report.ContainsPII = true
			
			if !seenTypes[piiType] {
				report.PIITypes = append(report.PIITypes, piiType)
				seenTypes[piiType] = true
			}
			
			for _, match := range matches {
				location := PIILocation{
					Type:       piiType,
					Start:      match[0],
					End:        match[1],
					MatchText:  text[match[0]:match[1]],
					Confidence: d.calculateConfidence(piiType, text[match[0]:match[1]]),
				}
				report.Locations = append(report.Locations, location)
			}
		}
	}
	
	// Calculate risk level based on types and quantity
	report.RiskLevel = d.calculateRiskLevel(report)
	
	return report, nil
}

// RedactPII redacts PII from text
func (d *piiDetector) RedactPII(ctx context.Context, text string) (string, error) {
	report, err := d.ScanForPII(ctx, text)
	if err != nil {
		return text, err
	}
	
	if !report.ContainsPII {
		return text, nil
	}
	
	// Sort locations by start position in reverse order
	// This ensures we don't mess up indices when replacing
	locations := report.Locations
	for i := 0; i < len(locations)-1; i++ {
		for j := i + 1; j < len(locations); j++ {
			if locations[i].Start < locations[j].Start {
				locations[i], locations[j] = locations[j], locations[i]
			}
		}
	}
	
	result := text
	for _, loc := range locations {
		redaction := d.getRedactionString(loc.Type)
		result = result[:loc.Start] + redaction + result[loc.End:]
	}
	
	return result, nil
}

// ValidatePII checks if specific text matches a PII pattern
func (d *piiDetector) ValidatePII(ctx context.Context, text string, piiType PIIType) bool {
	pattern, exists := d.patterns[piiType]
	if !exists {
		return false
	}
	
	return pattern.MatchString(text)
}

// calculateConfidence determines confidence level for detected PII
func (d *piiDetector) calculateConfidence(piiType PIIType, match string) float64 {
	// Basic confidence calculation
	// In production, this would use more sophisticated validation
	
	switch piiType {
	case PIITypeSSN:
		// Validate SSN format more strictly
		if len(strings.ReplaceAll(strings.ReplaceAll(match, "-", ""), " ", "")) == 9 {
			return 0.9
		}
		return 0.7
		
	case PIITypeCreditCard:
		// Could implement Luhn algorithm here
		if len(strings.ReplaceAll(strings.ReplaceAll(match, "-", ""), " ", "")) == 16 {
			return 0.85
		}
		return 0.6
		
	case PIITypeEmail:
		// Email regex is pretty reliable
		return 0.95
		
	case PIITypePhoneNumber:
		// Phone numbers can have false positives
		return 0.8
		
	case PIITypeIPAddress:
		// Validate IP ranges
		return 0.85
		
	case PIITypeMedicalID:
		// Context-dependent
		return 0.7
		
	default:
		return 0.5
	}
}

// calculateRiskLevel determines the risk level based on detected PII
func (d *piiDetector) calculateRiskLevel(report *PIIReport) RiskLevel {
	if !report.ContainsPII {
		return RiskLevelNone
	}
	
	highRiskTypes := map[PIIType]bool{
		PIITypeSSN:        true,
		PIITypeCreditCard: true,
		PIITypeMedicalID:  true,
		PIITypePassport:   true,
	}
	
	mediumRiskTypes := map[PIIType]bool{
		PIITypeDriverLicense: true,
		PIITypeDateOfBirth:   true,
	}
	
	hasHighRisk := false
	hasMediumRisk := false
	count := len(report.Locations)
	
	for _, piiType := range report.PIITypes {
		if highRiskTypes[piiType] {
			hasHighRisk = true
		} else if mediumRiskTypes[piiType] {
			hasMediumRisk = true
		}
	}
	
	// Determine risk level
	if hasHighRisk && count > 5 {
		return RiskLevelCritical
	} else if hasHighRisk {
		return RiskLevelHigh
	} else if hasMediumRisk || count > 10 {
		return RiskLevelMedium
	} else {
		return RiskLevelLow
	}
}

// getRedactionString returns the appropriate redaction string for a PII type
func (d *piiDetector) getRedactionString(piiType PIIType) string {
	switch piiType {
	case PIITypeSSN:
		return "[SSN-REDACTED]"
	case PIITypeCreditCard:
		return "[CARD-REDACTED]"
	case PIITypeEmail:
		return "[EMAIL-REDACTED]"
	case PIITypePhoneNumber:
		return "[PHONE-REDACTED]"
	case PIITypeMedicalID:
		return "[MEDICAL-ID-REDACTED]"
	case PIITypeIPAddress:
		return "[IP-REDACTED]"
	case PIITypeDriverLicense:
		return "[DL-REDACTED]"
	case PIITypePassport:
		return "[PASSPORT-REDACTED]"
	case PIITypeDateOfBirth:
		return "[DOB-REDACTED]"
	default:
		return "[PII-REDACTED]"
	}
}

// PHIDetector specifically detects Protected Health Information for HIPAA
type PHIDetector struct {
	piiDetector PIIDetector
}

// NewPHIDetector creates a new PHI detector
func NewPHIDetector() *PHIDetector {
	return &PHIDetector{
		piiDetector: NewPIIDetector(),
	}
}

// ScanForPHI scans text for protected health information
func (p *PHIDetector) ScanForPHI(ctx context.Context, text string) (*PIIReport, error) {
	// PHI includes all PII plus medical-specific information
	report, err := p.piiDetector.ScanForPII(ctx, text)
	if err != nil {
		return nil, err
	}
	
	// Add medical-specific patterns
	medicalPatterns := []string{
		`\b(?:diagnosis|diagnosed with|condition)[:\s]+([A-Za-z\s]+)\b`,
		`\b(?:medication|prescription|drug)[:\s]+([A-Za-z\s]+)\b`,
		`\b(?:procedure|surgery|operation)[:\s]+([A-Za-z\s]+)\b`,
	}
	
	for _, pattern := range medicalPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringIndex(text, -1)
		
		if len(matches) > 0 {
			report.ContainsPII = true
			report.RiskLevel = RiskLevelHigh // PHI is always high risk
			
			for _, match := range matches {
				location := PIILocation{
					Type:       PIITypeMedicalID,
					Start:      match[0],
					End:        match[1],
					MatchText:  text[match[0]:match[1]],
					Confidence: 0.8,
				}
				report.Locations = append(report.Locations, location)
			}
		}
	}
	
	return report, nil
}

// RedactPHI redacts protected health information
func (p *PHIDetector) RedactPHI(ctx context.Context, text string) (string, error) {
	return p.piiDetector.RedactPII(ctx, text)
}
