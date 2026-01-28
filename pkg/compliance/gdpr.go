package compliance

import (
	"context"
	"fmt"
	"time"
)

// GDPRCompliance handles GDPR compliance requirements
type GDPRCompliance interface {
	// ExportUserData exports all data for a user (Right to Data Portability)
	ExportUserData(ctx context.Context, userID string) (*DataExport, error)
	
	// DeleteUserData deletes all data for a user (Right to be Forgotten)
	DeleteUserData(ctx context.Context, userID string) error
	
	// RecordConsent records user consent for data processing
	RecordConsent(ctx context.Context, consent UserConsent) error
	
	// GetConsent retrieves user consent records
	GetConsent(ctx context.Context, userID string) (*UserConsent, error)
	
	// RevokeConsent revokes user consent
	RevokeConsent(ctx context.Context, userID string, consentType ConsentType) error
	
	// GetProcessingRecords gets data processing records for auditing
	GetProcessingRecords(ctx context.Context, userID string) ([]ProcessingRecord, error)
}

// DataExport contains exported user data in portable format
type DataExport struct {
	UserID      string
	ExportedAt  time.Time
	Format      string // JSON, CSV, XML
	Data        map[string]interface{}
	Metadata    ExportMetadata
}

// ExportMetadata contains metadata about the data export
type ExportMetadata struct {
	TotalRecords     int
	DataCategories   []string
	RetentionPeriods map[string]string
	LegalBasis       string
}

// UserConsent represents user consent for data processing
type UserConsent struct {
	UserID      string
	ConsentType ConsentType
	Granted     bool
	GrantedAt   time.Time
	RevokedAt   *time.Time
	Purpose     string
	LegalBasis  string
	Version     string // Consent form version
}

// ConsentType defines different types of consent
type ConsentType string

const (
	ConsentTypeMarketing      ConsentType = "marketing"
	ConsentTypeAnalytics      ConsentType = "analytics"
	ConsentTypeDataProcessing ConsentType = "data_processing"
	ConsentTypeDataSharing    ConsentType = "data_sharing"
	ConsentTypeThirdParty     ConsentType = "third_party"
)

// ProcessingRecord documents data processing activities
type ProcessingRecord struct {
	ID              string
	UserID          string
	Activity        string
	Purpose         string
	LegalBasis      string
	DataCategories  []string
	Recipients      []string
	RetentionPeriod string
	ProcessedAt     time.Time
	ProcessedBy     string
}

// gdprCompliance implements GDPRCompliance
type gdprCompliance struct {
	// In production, these would be actual database/storage connections
	userStore    UserDataStore
	consentStore ConsentStore
	auditStore   AuditStore
}

// UserDataStore provides access to user data across systems
type UserDataStore interface {
	GetAllUserData(ctx context.Context, userID string) (map[string]interface{}, error)
	DeleteAllUserData(ctx context.Context, userID string) error
}

// ConsentStore manages consent records
type ConsentStore interface {
	SaveConsent(ctx context.Context, consent UserConsent) error
	GetConsent(ctx context.Context, userID string) (*UserConsent, error)
	UpdateConsent(ctx context.Context, consent UserConsent) error
}

// AuditStore manages processing records
type AuditStore interface {
	RecordProcessing(ctx context.Context, record ProcessingRecord) error
	GetProcessingRecords(ctx context.Context, userID string) ([]ProcessingRecord, error)
}

// NewGDPRCompliance creates a new GDPR compliance handler
func NewGDPRCompliance(userStore UserDataStore, consentStore ConsentStore, auditStore AuditStore) GDPRCompliance {
	return &gdprCompliance{
		userStore:    userStore,
		consentStore: consentStore,
		auditStore:   auditStore,
	}
}

// ExportUserData exports all data for a user
func (g *gdprCompliance) ExportUserData(ctx context.Context, userID string) (*DataExport, error) {
	// Retrieve all user data
	data, err := g.userStore.GetAllUserData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user data: %w", err)
	}
	
	// Get consent records
	consent, err := g.consentStore.GetConsent(ctx, userID)
	if err != nil {
		// Log error but continue
		consent = nil
	}
	
	// Get processing records
	records, err := g.auditStore.GetProcessingRecords(ctx, userID)
	if err != nil {
		// Log error but continue
		records = nil
	}
	
	// Build export
	export := &DataExport{
		UserID:     userID,
		ExportedAt: time.Now(),
		Format:     "JSON",
		Data: map[string]interface{}{
			"user_data":          data,
			"consent_records":    consent,
			"processing_records": records,
		},
		Metadata: ExportMetadata{
			TotalRecords:   len(data),
			DataCategories: g.extractDataCategories(data),
			LegalBasis:     "Legitimate Interest / Consent",
		},
	}
	
	// Record the export in audit log
	g.auditStore.RecordProcessing(ctx, ProcessingRecord{
		UserID:     userID,
		Activity:   "data_export",
		Purpose:    "GDPR Right to Data Portability",
		LegalBasis: "Article 20 GDPR",
		ProcessedAt: time.Now(),
	})
	
	return export, nil
}

// DeleteUserData deletes all data for a user
func (g *gdprCompliance) DeleteUserData(ctx context.Context, userID string) error {
	// Verify user has right to deletion
	// (Some data may need to be retained for legal reasons)
	
	// Record deletion request before deleting
	g.auditStore.RecordProcessing(ctx, ProcessingRecord{
		UserID:      userID,
		Activity:    "data_deletion_requested",
		Purpose:     "GDPR Right to be Forgotten",
		LegalBasis:  "Article 17 GDPR",
		ProcessedAt: time.Now(),
	})
	
	// Delete user data
	if err := g.userStore.DeleteAllUserData(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user data: %w", err)
	}
	
	// Record successful deletion
	g.auditStore.RecordProcessing(ctx, ProcessingRecord{
		UserID:      userID,
		Activity:    "data_deletion_completed",
		Purpose:     "GDPR Right to be Forgotten",
		LegalBasis:  "Article 17 GDPR",
		ProcessedAt: time.Now(),
	})
	
	return nil
}

// RecordConsent records user consent
func (g *gdprCompliance) RecordConsent(ctx context.Context, consent UserConsent) error {
	consent.GrantedAt = time.Now()
	
	if err := g.consentStore.SaveConsent(ctx, consent); err != nil {
		return fmt.Errorf("failed to save consent: %w", err)
	}
	
	// Record in audit log
	g.auditStore.RecordProcessing(ctx, ProcessingRecord{
		UserID:      consent.UserID,
		Activity:    "consent_granted",
		Purpose:     consent.Purpose,
		LegalBasis:  "Article 6(1)(a) GDPR - Consent",
		ProcessedAt: time.Now(),
	})
	
	return nil
}

// GetConsent retrieves user consent
func (g *gdprCompliance) GetConsent(ctx context.Context, userID string) (*UserConsent, error) {
	return g.consentStore.GetConsent(ctx, userID)
}

// RevokeConsent revokes user consent
func (g *gdprCompliance) RevokeConsent(ctx context.Context, userID string, consentType ConsentType) error {
	consent, err := g.consentStore.GetConsent(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get consent: %w", err)
	}
	
	now := time.Now()
	consent.Granted = false
	consent.RevokedAt = &now
	
	if err := g.consentStore.UpdateConsent(ctx, *consent); err != nil {
		return fmt.Errorf("failed to update consent: %w", err)
	}
	
	// Record in audit log
	g.auditStore.RecordProcessing(ctx, ProcessingRecord{
		UserID:      userID,
		Activity:    "consent_revoked",
		Purpose:     string(consentType),
		LegalBasis:  "Article 7(3) GDPR - Right to Withdraw",
		ProcessedAt: time.Now(),
	})
	
	return nil
}

// GetProcessingRecords gets data processing records
func (g *gdprCompliance) GetProcessingRecords(ctx context.Context, userID string) ([]ProcessingRecord, error) {
	return g.auditStore.GetProcessingRecords(ctx, userID)
}

// extractDataCategories extracts data categories from user data
func (g *gdprCompliance) extractDataCategories(data map[string]interface{}) []string {
	categories := []string{}
	seen := make(map[string]bool)
	
	for key := range data {
		category := g.categorizeData(key)
		if !seen[category] {
			categories = append(categories, category)
			seen[category] = true
		}
	}
	
	return categories
}

// categorizeData categorizes data fields
func (g *gdprCompliance) categorizeData(field string) string {
	// Simple categorization - in production, this would be more sophisticated
	switch {
	case field == "email" || field == "phone" || field == "address":
		return "Contact Information"
	case field == "name" || field == "user_id":
		return "Identity Information"
	case field == "password" || field == "api_key":
		return "Authentication Information"
	case field == "threads" || field == "messages":
		return "User Generated Content"
	case field == "usage" || field == "analytics":
		return "Usage Data"
	default:
		return "Other"
	}
}

// DataRetentionPolicy defines how long data should be retained
type DataRetentionPolicy struct {
	DataCategory    string
	RetentionPeriod time.Duration
	LegalBasis      string
	AutoDelete      bool
}

// DefaultRetentionPolicies returns common GDPR retention policies
func DefaultRetentionPolicies() []DataRetentionPolicy {
	return []DataRetentionPolicy{
		{
			DataCategory:    "Contact Information",
			RetentionPeriod: 2 * 365 * 24 * time.Hour, // 2 years
			LegalBasis:      "Legitimate Interest",
			AutoDelete:      true,
		},
		{
			DataCategory:    "User Generated Content",
			RetentionPeriod: 5 * 365 * 24 * time.Hour, // 5 years
			LegalBasis:      "Contract Performance",
			AutoDelete:      false, // Manual review required
		},
		{
			DataCategory:    "Audit Logs",
			RetentionPeriod: 7 * 365 * 24 * time.Hour, // 7 years
			LegalBasis:      "Legal Obligation",
			AutoDelete:      true,
		},
		{
			DataCategory:    "Analytics Data",
			RetentionPeriod: 90 * 24 * time.Hour, // 90 days
			LegalBasis:      "Consent",
			AutoDelete:      true,
		},
	}
}

// DataProtectionImpactAssessment (DPIA) for high-risk processing
type DataProtectionImpactAssessment struct {
	ID                string
	ProcessingActivity string
	DataCategories    []string
	RiskLevel         RiskLevel
	Mitigations       []string
	ReviewedBy        string
	ReviewedAt        time.Time
	NextReview        time.Time
}

// PerformDPIA performs a data protection impact assessment
func PerformDPIA(activity string, dataCategories []string) *DataProtectionImpactAssessment {
	// Simplified DPIA - in production, this would be more thorough
	riskLevel := RiskLevelLow
	
	// Check for high-risk data categories
	highRiskCategories := map[string]bool{
		"medical":   true,
		"financial": true,
		"biometric": true,
		"children":  true,
	}
	
	for _, category := range dataCategories {
		if highRiskCategories[category] {
			riskLevel = RiskLevelHigh
			break
		}
	}
	
	mitigations := []string{
		"Encryption at rest and in transit",
		"Access controls and authentication",
		"Regular security audits",
		"Data minimization",
		"Pseudonymization where possible",
	}
	
	if riskLevel == RiskLevelHigh {
		mitigations = append(mitigations,
			"Additional security measures",
			"Regular DPIA reviews",
			"Consultation with DPO",
		)
	}
	
	return &DataProtectionImpactAssessment{
		ProcessingActivity: activity,
		DataCategories:     dataCategories,
		RiskLevel:          riskLevel,
		Mitigations:        mitigations,
		ReviewedAt:         time.Now(),
		NextReview:         time.Now().Add(365 * 24 * time.Hour), // Annual review
	}
}
