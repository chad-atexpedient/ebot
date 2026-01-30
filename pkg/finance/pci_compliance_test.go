package finance

import (
	"context"
	"strings"
	"testing"
)

// mockLogger implements Logger for testing
type mockLogger struct {
	messages []string
}

func (l *mockLogger) Info(msg string, args ...interface{})  { l.messages = append(l.messages, msg) }
func (l *mockLogger) Error(msg string, args ...interface{}) { l.messages = append(l.messages, msg) }
func (l *mockLogger) Warn(msg string, args ...interface{})  { l.messages = append(l.messages, msg) }

func newMockLogger() *mockLogger {
	return &mockLogger{messages: []string{}}
}

func TestNewPCICompliance(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	if pci == nil {
		t.Fatal("NewPCICompliance returned nil")
	}
}

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		name     string
		pan      string
		expected bool
	}{
		// Valid test card numbers
		{"Visa valid", "4111111111111111", true},
		{"MasterCard valid", "5500000000000004", true},
		{"Amex valid", "340000000000009", true},
		{"Discover valid", "6011000000000004", true},
		{"JCB valid", "3530111333300000", true},
		
		// With formatting
		{"Visa with dashes", "4111-1111-1111-1111", true},
		{"Visa with spaces", "4111 1111 1111 1111", true},
		
		// Invalid numbers
		{"Invalid checksum", "4111111111111112", false},
		{"Too short", "411111", false},
		{"Too long", "41111111111111111111", false},
		{"All zeros", "0000000000000000", true}, // Passes Luhn but invalid in practice
		{"Random invalid", "1234567890123456", false},
		
		// Edge cases
		{"Empty string", "", false},
		{"Letters only", "abcdefghijklmnop", false},
		{"Mixed alphanumeric", "4111abcd11111111", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateLuhn(tt.pan)
			if result != tt.expected {
				t.Errorf("ValidateLuhn(%s) = %v, want %v", tt.pan, result, tt.expected)
			}
		})
	}
}

func TestGetCardIssuer(t *testing.T) {
	tests := []struct {
		name     string
		pan      string
		expected string
	}{
		// Visa
		{"Visa 4", "4111111111111111", "Visa"},
		{"Visa formatted", "4111-1111-1111-1111", "Visa"},
		
		// Mastercard
		{"MasterCard 51", "5111111111111111", "Mastercard"},
		{"MasterCard 52", "5211111111111111", "Mastercard"},
		{"MasterCard 53", "5311111111111111", "Mastercard"},
		{"MasterCard 54", "5411111111111111", "Mastercard"},
		{"MasterCard 55", "5511111111111111", "Mastercard"},
		{"MasterCard 2221", "2221111111111111", "Mastercard"},
		{"MasterCard 2720", "2720111111111111", "Mastercard"},
		
		// American Express
		{"Amex 34", "341111111111111", "American Express"},
		{"Amex 37", "371111111111111", "American Express"},
		
		// Discover
		{"Discover 6011", "6011111111111111", "Discover"},
		{"Discover 65", "6511111111111111", "Discover"},
		{"Discover 644", "6441111111111111", "Discover"},
		
		// JCB
		{"JCB 3528", "3528111111111111", "JCB"},
		{"JCB 3589", "3589111111111111", "JCB"},
		
		// Diners Club
		{"Diners 36", "3611111111111111", "Diners Club"},
		{"Diners 38", "3811111111111111", "Diners Club"},
		{"Diners 300", "3001111111111111", "Diners Club"},
		
		// Unknown
		{"Unknown prefix", "9911111111111111", "Unknown"},
		{"Short number", "41111", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCardIssuer(tt.pan)
			if result != tt.expected {
				t.Errorf("GetCardIssuer(%s) = %s, want %s", tt.pan, result, tt.expected)
			}
		})
	}
}

func TestScanForCardData_DetectsPAN(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		shouldFind  bool
		expectedPAN int
	}{
		{
			name:        "Valid Visa with Luhn",
			input:       "Card number: 4111111111111111",
			shouldFind:  true,
			expectedPAN: 1,
		},
		{
			name:        "Visa with dashes",
			input:       "Card: 4111-1111-1111-1111",
			shouldFind:  true,
			expectedPAN: 1,
		},
		{
			name:        "Visa with spaces",
			input:       "Card: 4111 1111 1111 1111",
			shouldFind:  true,
			expectedPAN: 1,
		},
		{
			name:        "Multiple PANs",
			input:       "Card1: 4111111111111111 Card2: 5500000000000004",
			shouldFind:  true,
			expectedPAN: 2,
		},
		{
			name:        "Invalid Luhn - should NOT detect",
			input:       "Card: 4111111111111112",
			shouldFind:  false,
			expectedPAN: 0,
		},
		{
			name:        "No card data",
			input:       "This is a normal text without any card data",
			shouldFind:  false,
			expectedPAN: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := pci.ScanForCardData(ctx, tt.input)
			if err != nil {
				t.Fatalf("ScanForCardData failed: %v", err)
			}

			if report.ContainsCardData != tt.shouldFind {
				t.Errorf("ContainsCardData = %v, want %v", report.ContainsCardData, tt.shouldFind)
			}

			if report.PANCount != tt.expectedPAN {
				t.Errorf("PANCount = %d, want %d", report.PANCount, tt.expectedPAN)
			}
		})
	}
}

func TestScanForCardData_DetectsCVV(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		shouldFind  bool
		expectedCVV int
	}{
		{
			name:        "CVV with keyword",
			input:       "CVV: 123",
			shouldFind:  true,
			expectedCVV: 1,
		},
		{
			name:        "CVC with keyword",
			input:       "CVC: 456",
			shouldFind:  true,
			expectedCVV: 1,
		},
		{
			name:        "Security code",
			input:       "Security code: 789",
			shouldFind:  true,
			expectedCVV: 1,
		},
		{
			name:        "4-digit Amex CVV",
			input:       "CVV2: 1234",
			shouldFind:  true,
			expectedCVV: 1,
		},
		{
			name:        "Standalone number - should NOT match",
			input:       "There are 123 items",
			shouldFind:  false,
			expectedCVV: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := pci.ScanForCardData(ctx, tt.input)
			if err != nil {
				t.Fatalf("ScanForCardData failed: %v", err)
			}

			if report.CVVCount != tt.expectedCVV {
				t.Errorf("CVVCount = %d, want %d", report.CVVCount, tt.expectedCVV)
			}
		})
	}
}

func TestScanForCardData_RiskLevel(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	tests := []struct {
		name          string
		input         string
		expectedLevel RiskLevel
	}{
		{
			name:          "No card data",
			input:         "Normal text",
			expectedLevel: RiskLevelNone,
		},
		{
			name:          "PAN present - critical",
			input:         "Card: 4111111111111111",
			expectedLevel: RiskLevelCritical,
		},
		{
			name:          "CVV present - critical",
			input:         "CVV: 123",
			expectedLevel: RiskLevelCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := pci.ScanForCardData(ctx, tt.input)
			if err != nil {
				t.Fatalf("ScanForCardData failed: %v", err)
			}

			if report.RiskLevel != tt.expectedLevel {
				t.Errorf("RiskLevel = %s, want %s", report.RiskLevel, tt.expectedLevel)
			}
		})
	}
}

func TestScanForCardData_Recommendations(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// PAN present
	report, _ := pci.ScanForCardData(ctx, "Card: 4111111111111111")
	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations when PAN is present")
	}

	// CVV present
	report, _ = pci.ScanForCardData(ctx, "CVV: 123")
	foundCVVWarning := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "NEVER STORE CVV") {
			foundCVVWarning = true
			break
		}
	}
	if !foundCVVWarning {
		t.Error("Expected CVV warning in recommendations")
	}
}

func TestMaskCardData(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Mask PAN",
			input:    "Card: 4111111111111111",
			expected: "Card: ************1111",
		},
		{
			name:     "Keep non-PAN text",
			input:    "Normal text",
			expected: "Normal text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pci.MaskCardData(ctx, tt.input)
			if err != nil {
				t.Fatalf("MaskCardData failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("MaskCardData() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestMaskCardData_OnlyMasksValidPANs(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Invalid Luhn should NOT be masked
	input := "Card: 4111111111111112" // Invalid checksum
	result, _ := pci.MaskCardData(ctx, input)
	
	// Should remain unchanged since Luhn validation fails
	if result != input {
		t.Errorf("Invalid PAN should not be masked, got: %s", result)
	}
}

func TestTokenizeCardData_Hash(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	input := "Card: 4111111111111111"
	result, err := pci.TokenizeCardData(ctx, input, TokenMethodHash)
	if err != nil {
		t.Fatalf("TokenizeCardData failed: %v", err)
	}

	// Should contain token prefix
	if !strings.Contains(result, "tok_") {
		t.Errorf("Expected hash token with tok_ prefix, got: %s", result)
	}

	// Should not contain original PAN
	if strings.Contains(result, "4111111111111111") {
		t.Error("Original PAN should not be in tokenized output")
	}
}

func TestTokenizeCardData_FormatPreserving(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	input := "Card: 4111111111111111"
	result, err := pci.TokenizeCardData(ctx, input, TokenMethodFormatPreserving)
	if err != nil {
		t.Fatalf("TokenizeCardData failed: %v", err)
	}

	// Should keep BIN (first 6) and last 4
	if !strings.Contains(result, "411111") {
		t.Errorf("Expected BIN to be preserved, got: %s", result)
	}
	if !strings.Contains(result, "1111") {
		t.Errorf("Expected last 4 to be preserved, got: %s", result)
	}
}

func TestTokenizeCardData_Masking(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	input := "Card: 4111111111111111"
	result, err := pci.TokenizeCardData(ctx, input, TokenMethodMasking)
	if err != nil {
		t.Fatalf("TokenizeCardData failed: %v", err)
	}

	// Should have asterisks
	if !strings.Contains(result, "****") {
		t.Errorf("Expected masked output, got: %s", result)
	}
}

func TestTokenizeCardData_InvalidMethod(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	_, err := pci.TokenizeCardData(ctx, "Card: 4111111111111111", "invalid_method")
	if err == nil {
		t.Error("Expected error for invalid tokenization method")
	}
}

func TestValidateCDE(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Valid CDE scope
	validScope := CDEScope{
		Systems:       []string{"payment-server"},
		Applications:  []string{"payment-app"},
		SegmentedFrom: []string{"corporate-network"},
	}

	result, err := pci.ValidateCDE(ctx, validScope)
	if err != nil {
		t.Fatalf("ValidateCDE failed: %v", err)
	}

	if !result.IsCompliant {
		t.Error("Expected valid CDE scope to be compliant")
	}
	if !result.SegmentationOK {
		t.Error("Expected segmentation to be OK")
	}
}

func TestValidateCDE_NoSegmentation(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// CDE without segmentation
	scope := CDEScope{
		Systems:      []string{"payment-server"},
		Applications: []string{"payment-app"},
		// Missing SegmentedFrom
	}

	result, _ := pci.ValidateCDE(ctx, scope)

	if result.IsCompliant {
		t.Error("Expected non-compliant without segmentation")
	}
	if result.SegmentationOK {
		t.Error("Expected segmentation to fail")
	}
	if len(result.Violations) == 0 {
		t.Error("Expected violations to be reported")
	}
}

func TestValidateCDE_EmptyScope(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Empty CDE scope
	scope := CDEScope{}

	result, _ := pci.ValidateCDE(ctx, scope)

	if result.IsCompliant {
		t.Error("Expected non-compliant for empty scope")
	}
}

func TestGenerateSAQ(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	report, err := pci.GenerateSAQ(ctx, "org-123")
	if err != nil {
		t.Fatalf("GenerateSAQ failed: %v", err)
	}

	if report.Organization != "org-123" {
		t.Errorf("Expected organization org-123, got %s", report.Organization)
	}

	if report.Type == "" {
		t.Error("Expected SAQ type to be set")
	}

	if len(report.Requirements) == 0 {
		t.Error("Expected SAQ requirements")
	}
}

func TestGetComplianceStatus(t *testing.T) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	status, err := pci.GetComplianceStatus(ctx, "org-456")
	if err != nil {
		t.Fatalf("GetComplianceStatus failed: %v", err)
	}

	if status.OrganizationID != "org-456" {
		t.Errorf("Expected organization org-456, got %s", status.OrganizationID)
	}

	// New organization should be "Not Assessed"
	if status.ComplianceLevel != "Not Assessed" {
		t.Errorf("Expected 'Not Assessed', got %s", status.ComplianceLevel)
	}
}

func BenchmarkValidateLuhn(b *testing.B) {
	pan := "4111111111111111"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateLuhn(pan)
	}
}

func BenchmarkScanForCardData(b *testing.B) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	text := `
		Customer payment information:
		Card Number: 4111 1111 1111 1111
		Expiry: 12/25
		CVV: 123
		Name: John Doe
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pci.ScanForCardData(ctx, text)
	}
}

func BenchmarkMaskCardData(b *testing.B) {
	logger := newMockLogger()
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	text := "Card: 4111111111111111"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pci.MaskCardData(ctx, text)
	}
}
