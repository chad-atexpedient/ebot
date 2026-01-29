// Package finance provides financial services compliance features
package finance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PCICompliance manager for Payment Card Industry Data Security Standard
type PCICompliance interface {
	// Card data detection
	ScanForCardData(ctx context.Context, text string) (*CardDataReport, error)
	MaskCardData(ctx context.Context, text string) (string, error)
	TokenizeCardData(ctx context.Context, text string, method TokenizationMethod) (string, error)

	// Compliance validation
	ValidateCDE(ctx context.Context, scope CDEScope) (*CDEValidation, error)
	GenerateSAQ(ctx context.Context, orgID string) (*SAQReport, error)

	// Audit and reporting
	GetComplianceStatus(ctx context.Context, orgID string) (*PCIStatus, error)
}

// CardDataType represents types of payment card data
type CardDataType string

const (
	CardDataTypePAN            CardDataType = "primary_account_number"
	CardDataTypeCVV            CardDataType = "cvv"
	CardDataTypeExpiry         CardDataType = "expiration_date"
	CardDataTypeCardholderName CardDataType = "cardholder_name"
	CardDataTypeTrackData      CardDataType = "track_data"
	CardDataTypePIN            CardDataType = "pin"
	CardDataTypeBIN            CardDataType = "bank_identification_number"
	CardDataTypeIssuerID       CardDataType = "issuer_id"
	CardDataTypeServiceCode    CardDataType = "service_code"
	CardDataTypeSecurityCode   CardDataType = "security_code"
)

// TokenizationMethod defines how to tokenize card data
type TokenizationMethod string

const (
	TokenMethodFormatPreserving TokenizationMethod = "format_preserving"
	TokenMethodHash             TokenizationMethod = "hash"
	TokenMethodEncryption       TokenizationMethod = "encryption"
	TokenMethodRandom           TokenizationMethod = "random"
	TokenMethodVault            TokenizationMethod = "vault"
	TokenMethodMasking          TokenizationMethod = "masking"
)

// CardDataReport contains detected card data
type CardDataReport struct {
	ContainsCardData bool
	DataTypes        []CardDataType
	Locations        []CardDataLocation
	RiskLevel        RiskLevel
	PANCount         int
	CVVCount         int
	Recommendations  []string
}

type CardDataLocation struct {
	Type  CardDataType
	Start int
	End   int
	Value string
}

// CDEScope defines the Cardholder Data Environment scope
type CDEScope struct {
	Systems       []string
	Networks      []string
	Applications  []string
	Databases     []string
	FileStores    []string
	SegmentedFrom []string
}

// CDEValidation result
type CDEValidation struct {
	IsCompliant      bool
	Scope            CDEScope
	Violations       []string
	SegmentationOK   bool
	AccessControlsOK bool
	EncryptionOK     bool
	MonitoringOK     bool
	Recommendations  []string
}

// SAQReport is a Self-Assessment Questionnaire report
type SAQReport struct {
	Type            string // SAQ A, A-EP, B, B-IP, C, C-VT, D, P2PE
	Date            time.Time
	Organization    string
	ComplianceLevel string
	Requirements    []SAQRequirement
	AttestationOK   bool
	AOCGenerated    bool
}

type SAQRequirement struct {
	Number      string
	Description string
	Compliant   bool
	Evidence    []string
	Notes       string
}

// PCIStatus overall compliance status
type PCIStatus struct {
	OrganizationID   string
	LastAssessment   time.Time
	ComplianceLevel  string
	ValidUntil       time.Time
	QSAAttested      bool
	AOCOnFile        bool
	Violations       []string
	RemediationItems []string
}

type RiskLevel string

const (
	RiskLevelNone     RiskLevel = "none"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// pciComplianceManager implements PCICompliance
type pciComplianceManager struct {
	mu         sync.RWMutex
	patterns   map[CardDataType]*regexp.Regexp
	statuses   map[string]*PCIStatus
	tokenVault map[string]string
	logger     Logger
}

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// NewPCICompliance creates a new PCI compliance manager
func NewPCICompliance(logger Logger) PCICompliance {
	m := &pciComplianceManager{
		patterns:   make(map[CardDataType]*regexp.Regexp),
		statuses:   make(map[string]*PCIStatus),
		tokenVault: make(map[string]string),
		logger:     logger,
	}

	m.initPatterns()
	return m
}

func (m *pciComplianceManager) initPatterns() {
	// Primary Account Number (PAN) - matches card number formats
	// Luhn validation is performed separately after pattern matching
	m.patterns[CardDataTypePAN] = regexp.MustCompile(`\b(?:\d{4}[\s-]?){3}\d{4}\b`)

	// CVV/CVC/CSC - IMPROVED: Only match in context of card data
	// Must be preceded by CVV/CVC/CSC/Security Code keywords to avoid false positives
	m.patterns[CardDataTypeCVV] = regexp.MustCompile(`(?i)(?:cvv|cvc|csc|cvv2|cvc2|security\s*code|card\s*verification)[\s:]*(\d{3,4})\b`)

	// Expiration date (MM/YY, MM/YYYY, MM-YY, MMYY)
	m.patterns[CardDataTypeExpiry] = regexp.MustCompile(`\b(?:0[1-9]|1[0-2])[-/]?(?:\d{2}|\d{4})\b`)

	// Track data (starts with %)
	m.patterns[CardDataTypeTrackData] = regexp.MustCompile(`%[A-Z0-9]{1,19}\^[A-Z\s]{2,26}\^[0-9]{4}`)
}

// ValidateLuhn validates a PAN using the Luhn algorithm (ISO/IEC 7812-1)
// This is required by PCI DSS for proper PAN detection
func ValidateLuhn(pan string) bool {
	// Remove all non-digit characters
	clean := ""
	for _, r := range pan {
		if r >= '0' && r <= '9' {
			clean += string(r)
		}
	}

	// PAN must be between 13 and 19 digits
	if len(clean) < 13 || len(clean) > 19 {
		return false
	}

	// Luhn algorithm implementation
	sum := 0
	isSecond := false

	// Process from right to left
	for i := len(clean) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(clean[i]))

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	return sum%10 == 0
}

// GetCardIssuer identifies the card issuer based on BIN/IIN
func GetCardIssuer(pan string) string {
	clean := ""
	for _, r := range pan {
		if r >= '0' && r <= '9' {
			clean += string(r)
		}
	}

	if len(clean) < 6 {
		return "Unknown"
	}

	// Check card issuer based on BIN ranges
	prefix1 := clean[0:1]
	prefix2 := clean[0:2]
	prefix4 := clean[0:4]

	// Visa: starts with 4
	if prefix1 == "4" {
		return "Visa"
	}

	// Mastercard: 51-55 or 2221-2720
	if prefix2 >= "51" && prefix2 <= "55" {
		return "Mastercard"
	}
	if prefix4 >= "2221" && prefix4 <= "2720" {
		return "Mastercard"
	}

	// American Express: 34, 37
	if prefix2 == "34" || prefix2 == "37" {
		return "American Express"
	}

	// Discover: 6011, 644-649, 65
	if prefix4 == "6011" || prefix2 == "65" {
		return "Discover"
	}
	if len(clean) >= 3 {
		prefix3 := clean[0:3]
		if prefix3 >= "644" && prefix3 <= "649" {
			return "Discover"
		}
	}

	// JCB: 3528-3589
	if prefix4 >= "3528" && prefix4 <= "3589" {
		return "JCB"
	}

	// Diners Club: 36, 38, 300-305
	if prefix2 == "36" || prefix2 == "38" {
		return "Diners Club"
	}
	if len(clean) >= 3 {
		prefix3 := clean[0:3]
		if prefix3 >= "300" && prefix3 <= "305" {
			return "Diners Club"
		}
	}

	return "Unknown"
}

func (m *pciComplianceManager) ScanForCardData(ctx context.Context, text string) (*CardDataReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &CardDataReport{
		ContainsCardData: false,
		DataTypes:        []CardDataType{},
		Locations:        []CardDataLocation{},
		RiskLevel:        RiskLevelNone,
		Recommendations:  []string{},
	}

	// Scan for PANs with Luhn validation
	if pattern, ok := m.patterns[CardDataTypePAN]; ok {
		matches := pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			potentialPAN := text[match[0]:match[1]]

			// Validate using Luhn algorithm - only report if valid
			if ValidateLuhn(potentialPAN) {
				report.ContainsCardData = true
				if !containsType(report.DataTypes, CardDataTypePAN) {
					report.DataTypes = append(report.DataTypes, CardDataTypePAN)
				}

				location := CardDataLocation{
					Type:  CardDataTypePAN,
					Start: match[0],
					End:   match[1],
					Value: potentialPAN,
				}
				report.Locations = append(report.Locations, location)
				report.PANCount++
			}
		}
	}

	// Scan for CVV with contextual matching (improved pattern)
	if pattern, ok := m.patterns[CardDataTypeCVV]; ok {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			// match[2] and match[3] are the indices of the captured group (the actual CVV digits)
			if len(match) >= 4 && match[2] >= 0 && match[3] >= 0 {
				cvvValue := text[match[2]:match[3]]
				report.ContainsCardData = true
				if !containsType(report.DataTypes, CardDataTypeCVV) {
					report.DataTypes = append(report.DataTypes, CardDataTypeCVV)
				}

				location := CardDataLocation{
					Type:  CardDataTypeCVV,
					Start: match[2],
					End:   match[3],
					Value: cvvValue,
				}
				report.Locations = append(report.Locations, location)
				report.CVVCount++
			}
		}
	}

	// Scan for expiration dates (only if PAN found nearby for context)
	if report.PANCount > 0 {
		if pattern, ok := m.patterns[CardDataTypeExpiry]; ok {
			matches := pattern.FindAllStringIndex(text, -1)
			if len(matches) > 0 {
				report.ContainsCardData = true
				if !containsType(report.DataTypes, CardDataTypeExpiry) {
					report.DataTypes = append(report.DataTypes, CardDataTypeExpiry)
				}

				for _, match := range matches {
					location := CardDataLocation{
						Type:  CardDataTypeExpiry,
						Start: match[0],
						End:   match[1],
						Value: text[match[0]:match[1]],
					}
					report.Locations = append(report.Locations, location)
				}
			}
		}
	}

	// Scan for track data
	if pattern, ok := m.patterns[CardDataTypeTrackData]; ok {
		matches := pattern.FindAllStringIndex(text, -1)
		if len(matches) > 0 {
			report.ContainsCardData = true
			if !containsType(report.DataTypes, CardDataTypeTrackData) {
				report.DataTypes = append(report.DataTypes, CardDataTypeTrackData)
			}

			for _, match := range matches {
				location := CardDataLocation{
					Type:  CardDataTypeTrackData,
					Start: match[0],
					End:   match[1],
					Value: text[match[0]:match[1]],
				}
				report.Locations = append(report.Locations, location)
			}
		}
	}

	// Assess risk level
	if report.PANCount > 0 {
		report.RiskLevel = RiskLevelCritical
		report.Recommendations = append(report.Recommendations,
			"Immediately tokenize or encrypt all Primary Account Numbers",
			"Never store PANs in plain text",
			"Implement PCI DSS requirements 3.3 and 3.4",
		)
	}

	if report.CVVCount > 0 {
		report.RiskLevel = RiskLevelCritical
		report.Recommendations = append(report.Recommendations,
			"NEVER STORE CVV/CVC after authorization (PCI DSS Requirement 3.2.2)",
			"Purge all CVV data immediately",
		)
	}

	if len(report.DataTypes) > 0 && report.RiskLevel != RiskLevelCritical {
		report.RiskLevel = RiskLevelHigh
	}

	m.logger.Info("PCI scan completed", "cardDataFound", report.ContainsCardData, "riskLevel", report.RiskLevel)

	return report, nil
}

// containsType checks if a CardDataType slice contains a specific type
func containsType(types []CardDataType, t CardDataType) bool {
	for _, existing := range types {
		if existing == t {
			return true
		}
	}
	return false
}

func (m *pciComplianceManager) MaskCardData(ctx context.Context, text string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := text

	// Mask PAN - show only last 4 digits (only for Luhn-valid PANs)
	if pattern, ok := m.patterns[CardDataTypePAN]; ok {
		result = pattern.ReplaceAllStringFunc(result, func(pan string) string {
			// Only mask if it's a valid PAN
			if !ValidateLuhn(pan) {
				return pan
			}
			// Remove spaces/dashes
			clean := strings.ReplaceAll(strings.ReplaceAll(pan, " ", ""), "-", "")
			if len(clean) >= 4 {
				return strings.Repeat("*", len(clean)-4) + clean[len(clean)-4:]
			}
			return strings.Repeat("*", len(clean))
		})
	}

	// Mask CVV completely (using the improved contextual pattern)
	if pattern, ok := m.patterns[CardDataTypeCVV]; ok {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Find where the digits are and replace them
			digitPattern := regexp.MustCompile(`\d{3,4}$`)
			return digitPattern.ReplaceAllString(match, "***")
		})
	}

	// Mask expiration date
	if pattern, ok := m.patterns[CardDataTypeExpiry]; ok {
		result = pattern.ReplaceAllString(result, "**/**")
	}

	return result, nil
}

func (m *pciComplianceManager) TokenizeCardData(ctx context.Context, text string, method TokenizationMethod) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := text

	switch method {
	case TokenMethodFormatPreserving:
		// Format-preserving tokenization (maintains format)
		if pattern, ok := m.patterns[CardDataTypePAN]; ok {
			result = pattern.ReplaceAllStringFunc(result, func(pan string) string {
				// Only tokenize valid PANs
				if !ValidateLuhn(pan) {
					return pan
				}
				token := m.generateFormatPreservingToken(pan)
				m.tokenVault[token] = pan
				return token
			})
		}

	case TokenMethodHash:
		// One-way hash (irreversible)
		if pattern, ok := m.patterns[CardDataTypePAN]; ok {
			result = pattern.ReplaceAllStringFunc(result, func(pan string) string {
				// Only hash valid PANs
				if !ValidateLuhn(pan) {
					return pan
				}
				hash := sha256.Sum256([]byte(pan))
				return "tok_" + hex.EncodeToString(hash[:16])
			})
		}

	case TokenMethodMasking:
		// Simple masking
		return m.MaskCardData(ctx, text)

	default:
		return "", fmt.Errorf("unsupported tokenization method: %s", method)
	}

	m.logger.Info("Card data tokenized", "method", method)
	return result, nil
}

func (m *pciComplianceManager) generateFormatPreservingToken(pan string) string {
	// Simplified format-preserving tokenization
	// In production, use FPE (Format-Preserving Encryption) algorithms like FF3-1
	clean := strings.ReplaceAll(strings.ReplaceAll(pan, " ", ""), "-", "")
	if len(clean) < 10 {
		return clean
	}

	// Keep first 6 (BIN) and last 4 digits, tokenize middle
	prefix := clean[:6]
	suffix := clean[len(clean)-4:]
	middle := strings.Repeat("X", len(clean)-10)

	return prefix + middle + suffix
}

func (m *pciComplianceManager) ValidateCDE(ctx context.Context, scope CDEScope) (*CDEValidation, error) {
	validation := &CDEValidation{
		IsCompliant:      true,
		Scope:            scope,
		Violations:       []string{},
		SegmentationOK:   true,
		AccessControlsOK: true,
		EncryptionOK:     true,
		MonitoringOK:     true,
		Recommendations:  []string{},
	}

	// Validate segmentation
	if len(scope.SegmentedFrom) == 0 {
		validation.SegmentationOK = false
		validation.IsCompliant = false
		validation.Violations = append(validation.Violations,
			"CDE must be segmented from other networks (PCI DSS Requirement 1.2.1)")
		validation.Recommendations = append(validation.Recommendations,
			"Implement network segmentation using firewalls or VLANs",
			"Document network segmentation architecture")
	}

	// Validate scope completeness
	if len(scope.Systems) == 0 && len(scope.Applications) == 0 {
		validation.IsCompliant = false
		validation.Violations = append(validation.Violations,
			"CDE scope must include all systems and applications that store, process, or transmit cardholder data")
	}

	m.logger.Info("CDE validation completed", "compliant", validation.IsCompliant)

	return validation, nil
}

func (m *pciComplianceManager) GenerateSAQ(ctx context.Context, orgID string) (*SAQReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	report := &SAQReport{
		Type:            "SAQ D", // Most comprehensive
		Date:            time.Now(),
		Organization:    orgID,
		ComplianceLevel: "Unknown",
		Requirements:    m.getSAQRequirements(),
		AttestationOK:   false,
		AOCGenerated:    false,
	}

	// Check compliance for all requirements
	compliantCount := 0
	for _, req := range report.Requirements {
		if req.Compliant {
			compliantCount++
		}
	}

	if compliantCount == len(report.Requirements) {
		report.ComplianceLevel = "Compliant"
		report.AttestationOK = true
	} else if compliantCount > len(report.Requirements)/2 {
		report.ComplianceLevel = "Partially Compliant"
	} else {
		report.ComplianceLevel = "Non-Compliant"
	}

	m.logger.Info("SAQ generated", "type", report.Type, "level", report.ComplianceLevel)

	return report, nil
}

func (m *pciComplianceManager) getSAQRequirements() []SAQRequirement {
	return []SAQRequirement{
		{
			Number:      "1.1",
			Description: "Establish and implement firewall and router configuration standards",
			Compliant:   false,
			Evidence:    []string{},
			Notes:       "Requires firewall rules documentation and implementation",
		},
		{
			Number:      "2.1",
			Description: "Always change vendor-supplied defaults",
			Compliant:   false,
			Evidence:    []string{},
			Notes:       "Change default passwords and unnecessary services",
		},
		{
			Number:      "3.4",
			Description: "Render PAN unreadable",
			Compliant:   false,
			Evidence:    []string{},
			Notes:       "Use strong cryptography with associated key management",
		},
		{
			Number:      "4.1",
			Description: "Use strong cryptography for transmission over open networks",
			Compliant:   false,
			Evidence:    []string{},
			Notes:       "TLS 1.2 or higher required",
		},
		// Add more requirements as needed
	}
}

func (m *pciComplianceManager) GetComplianceStatus(ctx context.Context, orgID string) (*PCIStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[orgID]
	if !exists {
		status = &PCIStatus{
			OrganizationID:   orgID,
			LastAssessment:   time.Now(),
			ComplianceLevel:  "Not Assessed",
			ValidUntil:       time.Now().AddDate(1, 0, 0), // 1 year
			QSAAttested:      false,
			AOCOnFile:        false,
			Violations:       []string{},
			RemediationItems: []string{},
		}
	}

	return status, nil
}
