package finance

import (
	"context"
	"strings"
	"testing"
)

// mockLogger implements Logger for testing
type mockLogger struct{}

func (m *mockLogger) Info(msg string, args ...interface{})  {}
func (m *mockLogger) Error(msg string, args ...interface{}) {}
func (m *mockLogger) Warn(msg string, args ...interface{})  {}

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		name     string
		pan      string
		expected bool
	}{
		// Valid test card numbers (from various card networks for testing)
		{"Valid Visa", "4532015112830366", true},
		{"Valid Visa with spaces", "4532 0151 1283 0366", true},
		{"Valid Visa with dashes", "4532-0151-1283-0366", true},
		{"Valid Mastercard", "5425233430109903", true},
		{"Valid Amex", "374245455400126", true},
		{"Valid Discover", "6011000990139424", true},

		// Invalid numbers
		{"Invalid checksum", "4532015112830367", false},
		{"Too short", "453201511", false},
		{"Too long", "45320151128303661234", false},
		{"Invalid - all zeros", "0000000000000000", true}, // Actually passes Luhn!
		{"Random invalid", "1234567890123456", false},
		{"Invalid modified", "4532015112830365", false},
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
		{"Visa", "4532015112830366", "Visa"},
		{"Mastercard 51", "5125233430109903", "Mastercard"},
		{"Mastercard 55", "5525233430109903", "Mastercard"},
		{"Mastercard 2221", "2221000000000009", "Mastercard"},
		{"American Express 34", "340000000000009", "American Express"},
		{"American Express 37", "370000000000002", "American Express"},
		{"Discover 6011", "6011000990139424", "Discover"},
		{"Discover 65", "6500000000000002", "Discover"},
		{"JCB", "3530111333300000", "JCB"},
		{"Diners Club 36", "36000000000008", "Diners Club"},
		{"Unknown", "9999999999999999", "Unknown"},
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

func TestScanForCardData_ValidPAN(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Use a valid test PAN (passes Luhn)
	text := "Customer card: 4532015112830366"

	report, err := pci.ScanForCardData(ctx, text)
	if err != nil {
		t.Fatalf("ScanForCardData failed: %v", err)
	}

	if !report.ContainsCardData {
		t.Error("Expected to detect card data")
	}

	if report.PANCount != 1 {
		t.Errorf("Expected 1 PAN, got %d", report.PANCount)
	}

	if report.RiskLevel != RiskLevelCritical {
		t.Errorf("Expected Critical risk level, got %s", report.RiskLevel)
	}
}

func TestScanForCardData_InvalidPAN_NotDetected(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Use an invalid PAN (fails Luhn) - should NOT be detected
	text := "Customer card: 1234567890123456"

	report, err := pci.ScanForCardData(ctx, text)
	if err != nil {
		t.Fatalf("ScanForCardData failed: %v", err)
	}

	if report.PANCount != 0 {
		t.Errorf("Expected 0 PANs for invalid number, got %d", report.PANCount)
	}
}

func TestScanForCardData_CVVWithContext(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	tests := []struct {
		name         string
		text         string
		expectCVV    bool
		expectedCount int
	}{
		{
			name:         "CVV with keyword",
			text:         "Card CVV: 123",
			expectCVV:    true,
			expectedCount: 1,
		},
		{
			name:         "CVC with keyword",
			text:         "CVC: 456",
			expectCVV:    true,
			expectedCount: 1,
		},
		{
			name:         "Security code keyword",
			text:         "Security code: 789",
			expectCVV:    true,
			expectedCount: 1,
		},
		{
			name:         "CVV2 keyword",
			text:         "CVV2 321",
			expectCVV:    true,
			expectedCount: 1,
		},
		{
			name:         "Random 3-digit number - NO detection",
			text:         "Order quantity: 123 items",
			expectCVV:    false,
			expectedCount: 0,
		},
		{
			name:         "Phone number - NO detection",
			text:         "Call 555-1234 for support",
			expectCVV:    false,
			expectedCount: 0,
		},
		{
			name:         "Building number - NO detection",
			text:         "Located at building 456",
			expectCVV:    false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := pci.ScanForCardData(ctx, tt.text)
			if err != nil {
				t.Fatalf("ScanForCardData failed: %v", err)
			}

			if (report.CVVCount > 0) != tt.expectCVV {
				t.Errorf("CVV detection mismatch: got count=%d, expected detection=%v", report.CVVCount, tt.expectCVV)
			}

			if report.CVVCount != tt.expectedCount {
				t.Errorf("Expected %d CVVs, got %d", tt.expectedCount, report.CVVCount)
			}
		})
	}
}

func TestScanForCardData_NoFalsePositives(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Text that should NOT trigger false positives
	texts := []string{
		"The year 2024 was great",
		"Order ID: 1234567890123456", // 16 digits but fails Luhn
		"Phone: 555-123-4567",
		"Building 123, Floor 456",
		"Product code: 9876543210987654", // 16 digits but fails Luhn
		"The answer is 42",
		"Temperature: 123 degrees",
	}

	for _, text := range texts {
		t.Run(text[:20], func(t *testing.T) {
			report, err := pci.ScanForCardData(ctx, text)
			if err != nil {
				t.Fatalf("ScanForCardData failed: %v", err)
			}

			if report.PANCount > 0 {
				t.Errorf("False positive PAN detected in: %s", text)
			}
			if report.CVVCount > 0 {
				t.Errorf("False positive CVV detected in: %s", text)
			}
		})
	}
}

func TestMaskCardData(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	text := "Card: 4532015112830366, CVV: 123"
	masked, err := pci.MaskCardData(ctx, text)
	if err != nil {
		t.Fatalf("MaskCardData failed: %v", err)
	}

	// PAN should be masked, showing only last 4 digits
	if strings.Contains(masked, "4532015112830366") {
		t.Error("PAN should be masked")
	}

	if !strings.Contains(masked, "0366") {
		t.Error("Last 4 digits should be visible")
	}

	// CVV should be masked
	if strings.Contains(masked, "123") && !strings.Contains(masked, "***") {
		t.Error("CVV should be masked")
	}
}

func TestMaskCardData_InvalidPAN_NotMasked(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Invalid PAN should not be masked
	text := "Number: 1234567890123456"
	masked, err := pci.MaskCardData(ctx, text)
	if err != nil {
		t.Fatalf("MaskCardData failed: %v", err)
	}

	// Invalid PAN should remain unchanged
	if !strings.Contains(masked, "1234567890123456") {
		t.Error("Invalid PAN should not be masked")
	}
}

func TestTokenizeCardData_FormatPreserving(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	text := "Card: 4532015112830366"
	tokenized, err := pci.TokenizeCardData(ctx, text, TokenMethodFormatPreserving)
	if err != nil {
		t.Fatalf("TokenizeCardData failed: %v", err)
	}

	// Original PAN should not be present
	if strings.Contains(tokenized, "4532015112830366") {
		t.Error("Original PAN should be tokenized")
	}

	// BIN (first 6) and last 4 should be preserved
	if !strings.Contains(tokenized, "453201") {
		t.Error("BIN should be preserved in format-preserving tokenization")
	}

	if !strings.Contains(tokenized, "0366") {
		t.Error("Last 4 digits should be preserved")
	}
}

func TestTokenizeCardData_Hash(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	text := "Card: 4532015112830366"
	tokenized, err := pci.TokenizeCardData(ctx, text, TokenMethodHash)
	if err != nil {
		t.Fatalf("TokenizeCardData failed: %v", err)
	}

	// Should contain token prefix
	if !strings.Contains(tokenized, "tok_") {
		t.Error("Hash tokenization should produce tok_ prefix")
	}

	// Original PAN should not be present
	if strings.Contains(tokenized, "4532015112830366") {
		t.Error("Original PAN should be hashed")
	}
}

func TestValidateCDE(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	// Test with proper segmentation
	scope := CDEScope{
		Systems:       []string{"payment-server"},
		Applications:  []string{"payment-api"},
		SegmentedFrom: []string{"corporate-network"},
	}

	validation, err := pci.ValidateCDE(ctx, scope)
	if err != nil {
		t.Fatalf("ValidateCDE failed: %v", err)
	}

	if !validation.SegmentationOK {
		t.Error("Segmentation should be OK with SegmentedFrom defined")
	}

	// Test without segmentation
	scopeNoSeg := CDEScope{
		Systems:      []string{"payment-server"},
		Applications: []string{"payment-api"},
	}

	validationNoSeg, err := pci.ValidateCDE(ctx, scopeNoSeg)
	if err != nil {
		t.Fatalf("ValidateCDE failed: %v", err)
	}

	if validationNoSeg.SegmentationOK {
		t.Error("Segmentation should fail without SegmentedFrom")
	}

	if validationNoSeg.IsCompliant {
		t.Error("Should not be compliant without segmentation")
	}
}

func TestGenerateSAQ(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	report, err := pci.GenerateSAQ(ctx, "test-org")
	if err != nil {
		t.Fatalf("GenerateSAQ failed: %v", err)
	}

	if report.Organization != "test-org" {
		t.Errorf("Expected org 'test-org', got '%s'", report.Organization)
	}

	if report.Type != "SAQ D" {
		t.Errorf("Expected SAQ type 'SAQ D', got '%s'", report.Type)
	}

	if len(report.Requirements) == 0 {
		t.Error("SAQ should have requirements")
	}
}

func TestGetComplianceStatus(t *testing.T) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()

	status, err := pci.GetComplianceStatus(ctx, "new-org")
	if err != nil {
		t.Fatalf("GetComplianceStatus failed: %v", err)
	}

	if status.OrganizationID != "new-org" {
		t.Errorf("Expected org 'new-org', got '%s'", status.OrganizationID)
	}

	if status.ComplianceLevel != "Not Assessed" {
		t.Errorf("New org should be 'Not Assessed', got '%s'", status.ComplianceLevel)
	}
}

// Benchmark tests
func BenchmarkValidateLuhn(b *testing.B) {
	pan := "4532015112830366"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateLuhn(pan)
	}
}

func BenchmarkScanForCardData(b *testing.B) {
	logger := &mockLogger{}
	pci := NewPCICompliance(logger)
	ctx := context.Background()
	text := "Payment details: Card 4532015112830366, CVV: 123, Exp: 12/25"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pci.ScanForCardData(ctx, text)
	}
}
