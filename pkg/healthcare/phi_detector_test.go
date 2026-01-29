package healthcare

import (
	"context"
	"strings"
	"testing"
)

func TestNewPHIDetector(t *testing.T) {
	detector := NewPHIDetector()
	if detector == nil {
		t.Fatal("NewPHIDetector returned nil")
	}
}

func TestScanForPHI_NoRecursion(t *testing.T) {
	// This test verifies that ScanForPHI and RedactPHI don't cause infinite recursion
	detector := NewPHIDetector()
	ctx := context.Background()

	// Text with PHI that would trigger redaction
	text := "Patient SSN: 123-45-6789, Email: patient@hospital.com"

	// This should complete without stack overflow
	report, err := detector.ScanForPHI(ctx, text)
	if err != nil {
		t.Fatalf("ScanForPHI failed: %v", err)
	}

	if !report.ContainsPHI {
		t.Error("Expected to detect PHI")
	}

	// Verify redacted text was generated (this proves no recursion)
	if report.RedactedText == "" {
		t.Error("Expected redacted text to be generated")
	}

	if report.RedactedText == text {
		t.Error("Redacted text should be different from original")
	}
}

func TestRedactPHI_NoRecursion(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "Contact Dr. John Smith at 555-123-4567"

	// This should complete without stack overflow
	redacted, err := detector.RedactPHI(ctx, text)
	if err != nil {
		t.Fatalf("RedactPHI failed: %v", err)
	}

	if redacted == text {
		t.Error("Expected text to be redacted")
	}
}

func TestScanForPHI_DetectsSSN(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		wantPHI  bool
		wantType PHIType
	}{
		{
			name:     "SSN with dashes",
			input:    "Patient SSN: 123-45-6789",
			wantPHI:  true,
			wantType: PHITypeSSN,
		},
		{
			name:     "SSN without dashes",
			input:    "SSN is 123456789",
			wantPHI:  true,
			wantType: PHITypeSSN,
		},
		{
			name:    "No SSN",
			input:   "This text has no sensitive data",
			wantPHI: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := detector.ScanForPHI(ctx, tt.input)
			if err != nil {
				t.Fatalf("ScanForPHI failed: %v", err)
			}

			if report.ContainsPHI != tt.wantPHI {
				t.Errorf("ContainsPHI = %v, want %v", report.ContainsPHI, tt.wantPHI)
			}

			if tt.wantPHI && tt.wantType != "" {
				found := false
				for _, phiType := range report.PHITypes {
					if phiType == tt.wantType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find PHI type %s, got types: %v", tt.wantType, report.PHITypes)
				}
			}
		})
	}
}

func TestScanForPHI_DetectsEmail(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "Contact: patient@hospital.com"
	report, err := detector.ScanForPHI(ctx, text)
	if err != nil {
		t.Fatalf("ScanForPHI failed: %v", err)
	}

	if !report.ContainsPHI {
		t.Error("Expected to detect PHI (email)")
	}

	found := false
	for _, phiType := range report.PHITypes {
		if phiType == PHITypeEmail {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find email PHI type, got: %v", report.PHITypes)
	}
}

func TestScanForPHI_DetectsPhoneNumber(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{"Format 1", "Call 555-123-4567"},
		{"Format 2", "Call 5551234567"},
		{"Format 3", "Call (555) 123-4567"},
		{"Format 4", "Call 555.123.4567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := detector.ScanForPHI(ctx, tt.input)
			if err != nil {
				t.Fatalf("ScanForPHI failed: %v", err)
			}

			if !report.ContainsPHI {
				t.Errorf("Expected to detect phone number in: %s", tt.input)
			}
		})
	}
}

func TestScanForPHI_DetectsDateOfBirth(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "DOB: 01/15/1990"
	report, err := detector.ScanForPHI(ctx, text)
	if err != nil {
		t.Fatalf("ScanForPHI failed: %v", err)
	}

	if !report.ContainsPHI {
		t.Error("Expected to detect date of birth")
	}

	found := false
	for _, phiType := range report.PHITypes {
		if phiType == PHITypeDateOfBirth {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find date of birth PHI type, got: %v", report.PHITypes)
	}
}

func TestScanForPHI_DetectsMRN(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{"MRN with colon", "MRN: ABC123456"},
		{"MRN without colon", "MRN ABC123456"},
		{"Medical record", "medical record #ABC123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := detector.ScanForPHI(ctx, tt.input)
			if err != nil {
				t.Fatalf("ScanForPHI failed: %v", err)
			}

			if !report.ContainsPHI {
				t.Errorf("Expected to detect MRN in: %s", tt.input)
			}
		})
	}
}

func TestRedactPHI_RedactsAllTypes(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "Patient SSN: 123-45-6789, Email: test@example.com, Phone: 555-123-4567"
	redacted, err := detector.RedactPHI(ctx, text)
	if err != nil {
		t.Fatalf("RedactPHI failed: %v", err)
	}

	// Verify original sensitive data is not present
	if strings.Contains(redacted, "123-45-6789") {
		t.Error("SSN should be redacted")
	}
	if strings.Contains(redacted, "test@example.com") {
		t.Error("Email should be redacted")
	}
	if strings.Contains(redacted, "555-123-4567") {
		t.Error("Phone should be redacted")
	}

	// Verify redaction markers are present
	if !strings.Contains(redacted, "[REDACTED") {
		t.Error("Expected redaction markers in output")
	}
}

func TestRedactPHI_NoChange_WhenNoPHI(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "This is a clean text with no PHI"
	redacted, err := detector.RedactPHI(ctx, text)
	if err != nil {
		t.Fatalf("RedactPHI failed: %v", err)
	}

	if redacted != text {
		t.Errorf("Text without PHI should remain unchanged, got: %s", redacted)
	}
}

func TestValidateHIPAACompliance_Clean(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "This document contains no protected health information."
	compliant, violations, err := detector.ValidateHIPAACompliance(ctx, text)
	if err != nil {
		t.Fatalf("ValidateHIPAACompliance failed: %v", err)
	}

	if !compliant {
		t.Errorf("Expected compliant text, got violations: %v", violations)
	}
	if len(violations) > 0 {
		t.Errorf("Expected no violations, got: %v", violations)
	}
}

func TestValidateHIPAACompliance_WithPHI(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "Patient John Doe, SSN: 123-45-6789, was admitted on 01/15/2024"
	compliant, violations, err := detector.ValidateHIPAACompliance(ctx, text)
	if err != nil {
		t.Fatalf("ValidateHIPAACompliance failed: %v", err)
	}

	if compliant {
		t.Error("Expected non-compliant result for text with PHI")
	}
	if len(violations) == 0 {
		t.Error("Expected violations to be reported")
	}
}

func TestRiskLevel_Critical(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	// SSN is high-risk PHI
	text := "SSN: 123-45-6789"
	report, err := detector.ScanForPHI(ctx, text)
	if err != nil {
		t.Fatalf("ScanForPHI failed: %v", err)
	}

	if report.RiskLevel != RiskLevelCritical {
		t.Errorf("Expected Critical risk level for SSN, got: %s", report.RiskLevel)
	}
}

func TestRiskLevel_None(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := "No PHI here"
	report, err := detector.ScanForPHI(ctx, text)
	if err != nil {
		t.Fatalf("ScanForPHI failed: %v", err)
	}

	if report.RiskLevel != RiskLevelNone {
		t.Errorf("Expected None risk level, got: %s", report.RiskLevel)
	}
}

func TestScanForPHI_EmptyText(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	report, err := detector.ScanForPHI(ctx, "")
	if err != nil {
		t.Fatalf("ScanForPHI failed on empty text: %v", err)
	}

	if report.ContainsPHI {
		t.Error("Empty text should not contain PHI")
	}
	if report.TotalPHICount != 0 {
		t.Errorf("Expected 0 PHI count, got: %d", report.TotalPHICount)
	}
}

func TestScanForPHI_MultiplePHI(t *testing.T) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := `
		Patient: Dr. John Smith
		SSN: 123-45-6789
		SSN2: 987-65-4321
		Email: patient@example.com
		Phone: 555-123-4567
		DOB: 01/15/1990
		MRN: ABC123456
	`

	report, err := detector.ScanForPHI(ctx, text)
	if err != nil {
		t.Fatalf("ScanForPHI failed: %v", err)
	}

	if report.TotalPHICount < 5 {
		t.Errorf("Expected at least 5 PHI instances, got: %d", report.TotalPHICount)
	}

	if len(report.PHITypes) < 4 {
		t.Errorf("Expected at least 4 different PHI types, got: %d types: %v", len(report.PHITypes), report.PHITypes)
	}
}

func BenchmarkScanForPHI(b *testing.B) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := `
		Patient: Dr. John Smith
		SSN: 123-45-6789
		Email: patient@example.com
		Phone: 555-123-4567
		DOB: 01/15/1990
		Address: 123 Main Street
		MRN: ABC123456
		This is additional text to make the benchmark more realistic.
		The patient presented with symptoms of common cold.
		No prior medical history of significance.
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.ScanForPHI(ctx, text)
	}
}

func BenchmarkRedactPHI(b *testing.B) {
	detector := NewPHIDetector()
	ctx := context.Background()

	text := `
		Patient: Dr. John Smith
		SSN: 123-45-6789
		Email: patient@example.com
		Phone: 555-123-4567
		DOB: 01/15/1990
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.RedactPHI(ctx, text)
	}
}
