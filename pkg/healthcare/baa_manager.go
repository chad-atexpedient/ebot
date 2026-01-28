package healthcare

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// BAAStatus represents the status of a Business Associate Agreement
type BAAStatus string

const (
	BAAStatusDraft     BAAStatus = "draft"
	BAAStatusActive    BAAStatus = "active"
	BAAStatusExpired   BAAStatus = "expired"
	BAAStatusTerminated BAAStatus = "terminated"
	BAAStatusRenewal   BAAStatus = "renewal"
)

// BAA represents a Business Associate Agreement
type BAA struct {
	ID                   string            `json:"id"`
	OrganizationID       string            `json:"organization_id"`
	OrganizationName     string            `json:"organization_name"`
	BusinessAssociateName string           `json:"business_associate_name"`
	Status               BAAStatus         `json:"status"`
	SignedDate           time.Time         `json:"signed_date"`
	EffectiveDate        time.Time         `json:"effective_date"`
	ExpirationDate       time.Time         `json:"expiration_date"`
	AutoRenew            bool              `json:"auto_renew"`
	DocumentHash         string            `json:"document_hash"`
	DocumentURL          string            `json:"document_url"`
	Scope                []string          `json:"scope"` // PHI types covered
	ServicesProvided     []string          `json:"services_provided"`
	ComplianceFramework  []string          `json:"compliance_framework"` // HIPAA, HITECH, etc.
	AuditRequired        bool              `json:"audit_required"`
	LastAuditDate        *time.Time        `json:"last_audit_date,omitempty"`
	NextAuditDate        *time.Time        `json:"next_audit_date,omitempty"`
	BreachNotification   BreachNotificationConfig `json:"breach_notification"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// BreachNotificationConfig defines breach notification requirements
type BreachNotificationConfig struct {
	Required          bool          `json:"required"`
	NotificationWindow time.Duration `json:"notification_window"` // e.g., 60 days
	ContactEmail      string        `json:"contact_email"`
	ContactPhone      string        `json:"contact_phone"`
	EscalationPath    []string      `json:"escalation_path"`
}

// BAAManager manages Business Associate Agreements
type BAAManager interface {
	// CreateBAA creates a new BAA
	CreateBAA(ctx context.Context, baa *BAA) error
	
	// GetBAA retrieves a BAA by ID
	GetBAA(ctx context.Context, baaID string) (*BAA, error)
	
	// ListBAAs lists all BAAs
	ListBAAs(ctx context.Context, filters BAAFilters) ([]*BAA, error)
	
	// UpdateBAA updates an existing BAA
	UpdateBAA(ctx context.Context, baa *BAA) error
	
	// TerminateBAA terminates a BAA
	TerminateBAA(ctx context.Context, baaID string, reason string) error
	
	// RenewBAA renews an expiring BAA
	RenewBAA(ctx context.Context, baaID string, newExpirationDate time.Time) error
	
	// ValidateBAA validates BAA is active and covers required scope
	ValidateBAA(ctx context.Context, organizationID string, requiredScope []string) (bool, error)
	
	// GetExpiringBAAs returns BAAs expiring within specified duration
	GetExpiringBAAs(ctx context.Context, within time.Duration) ([]*BAA, error)
	
	// RecordAudit records an audit event
	RecordAudit(ctx context.Context, baaID string, auditResult AuditResult) error
}

// BAAFilters for filtering BAAs
type BAAFilters struct {
	OrganizationID string
	Status         BAAStatus
	ExpiresAfter   *time.Time
	ExpiresBefore  *time.Time
}

// AuditResult represents the result of a BAA audit
type AuditResult struct {
	AuditDate     time.Time         `json:"audit_date"`
	AuditorName   string            `json:"auditor_name"`
	Findings      []string          `json:"findings"`
	Compliant     bool              `json:"compliant"`
	Recommendations []string        `json:"recommendations"`
	NextAuditDate time.Time         `json:"next_audit_date"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// baaManagerImpl implements BAAManager
type baaManagerImpl struct {
	baas map[string]*BAA
	mu   sync.RWMutex
}

// NewBAAManager creates a new BAA manager
func NewBAAManager() BAAManager {
	return &baaManagerImpl{
		baas: make(map[string]*BAA),
	}
}

// CreateBAA creates a new BAA
func (m *baaManagerImpl) CreateBAA(ctx context.Context, baa *BAA) error {
	if baa.ID == "" {
		baa.ID = m.generateBAAID(baa)
	}
	
	// Validate required fields
	if err := m.validateBAA(baa); err != nil {
		return fmt.Errorf("invalid BAA: %w", err)
	}
	
	// Set timestamps
	now := time.Now()
	baa.CreatedAt = now
	baa.UpdatedAt = now
	
	// Calculate next audit date if required
	if baa.AuditRequired && baa.NextAuditDate == nil {
		nextAudit := baa.EffectiveDate.AddDate(1, 0, 0) // Annual audit
		baa.NextAuditDate = &nextAudit
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.baas[baa.ID]; exists {
		return fmt.Errorf("BAA with ID %s already exists", baa.ID)
	}
	
	m.baas[baa.ID] = baa
	
	return nil
}

// GetBAA retrieves a BAA by ID
func (m *baaManagerImpl) GetBAA(ctx context.Context, baaID string) (*BAA, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	baa, exists := m.baas[baaID]
	if !exists {
		return nil, fmt.Errorf("BAA not found: %s", baaID)
	}
	
	// Update status if expired
	if baa.Status == BAAStatusActive && time.Now().After(baa.ExpirationDate) {
		baa.Status = BAAStatusExpired
	}
	
	return baa, nil
}

// ListBAAs lists all BAAs with optional filters
func (m *baaManagerImpl) ListBAAs(ctx context.Context, filters BAAFilters) ([]*BAA, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := []*BAA{}
	
	for _, baa := range m.baas {
		// Apply filters
		if filters.OrganizationID != "" && baa.OrganizationID != filters.OrganizationID {
			continue
		}
		
		if filters.Status != "" && baa.Status != filters.Status {
			continue
		}
		
		if filters.ExpiresAfter != nil && baa.ExpirationDate.Before(*filters.ExpiresAfter) {
			continue
		}
		
		if filters.ExpiresBefore != nil && baa.ExpirationDate.After(*filters.ExpiresBefore) {
			continue
		}
		
		result = append(result, baa)
	}
	
	return result, nil
}

// UpdateBAA updates an existing BAA
func (m *baaManagerImpl) UpdateBAA(ctx context.Context, baa *BAA) error {
	if err := m.validateBAA(baa); err != nil {
		return fmt.Errorf("invalid BAA: %w", err)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	existing, exists := m.baas[baa.ID]
	if !exists {
		return fmt.Errorf("BAA not found: %s", baa.ID)
	}
	
	// Preserve creation timestamp
	baa.CreatedAt = existing.CreatedAt
	baa.UpdatedAt = time.Now()
	
	m.baas[baa.ID] = baa
	
	return nil
}

// TerminateBAA terminates a BAA
func (m *baaManagerImpl) TerminateBAA(ctx context.Context, baaID string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	baa, exists := m.baas[baaID]
	if !exists {
		return fmt.Errorf("BAA not found: %s", baaID)
	}
	
	baa.Status = BAAStatusTerminated
	baa.UpdatedAt = time.Now()
	
	if baa.Metadata == nil {
		baa.Metadata = make(map[string]string)
	}
	baa.Metadata["termination_reason"] = reason
	baa.Metadata["termination_date"] = time.Now().Format(time.RFC3339)
	
	return nil
}

// RenewBAA renews an expiring BAA
func (m *baaManagerImpl) RenewBAA(ctx context.Context, baaID string, newExpirationDate time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	baa, exists := m.baas[baaID]
	if !exists {
		return fmt.Errorf("BAA not found: %s", baaID)
	}
	
	if baa.Status == BAAStatusTerminated {
		return fmt.Errorf("cannot renew terminated BAA")
	}
	
	baa.ExpirationDate = newExpirationDate
	baa.Status = BAAStatusActive
	baa.UpdatedAt = time.Now()
	
	if baa.Metadata == nil {
		baa.Metadata = make(map[string]string)
	}
	baa.Metadata["renewal_date"] = time.Now().Format(time.RFC3339)
	
	return nil
}

// ValidateBAA validates BAA is active and covers required scope
func (m *baaManagerImpl) ValidateBAA(ctx context.Context, organizationID string, requiredScope []string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Find active BAA for organization
	var activeBaa *BAA
	for _, baa := range m.baas {
		if baa.OrganizationID == organizationID && baa.Status == BAAStatusActive {
			// Check not expired
			if time.Now().Before(baa.ExpirationDate) {
				activeBaa = baa
				break
			}
		}
	}
	
	if activeBaa == nil {
		return false, fmt.Errorf("no active BAA found for organization %s", organizationID)
	}
	
	// Validate scope coverage
	scopeMap := make(map[string]bool)
	for _, s := range activeBaa.Scope {
		scopeMap[s] = true
	}
	
	for _, required := range requiredScope {
		if !scopeMap[required] {
			return false, fmt.Errorf("BAA does not cover required scope: %s", required)
		}
	}
	
	return true, nil
}

// GetExpiringBAAs returns BAAs expiring within specified duration
func (m *baaManagerImpl) GetExpiringBAAs(ctx context.Context, within time.Duration) ([]*BAA, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	expirationThreshold := time.Now().Add(within)
	result := []*BAA{}
	
	for _, baa := range m.baas {
		if baa.Status == BAAStatusActive && baa.ExpirationDate.Before(expirationThreshold) {
			result = append(result, baa)
		}
	}
	
	return result, nil
}

// RecordAudit records an audit event
func (m *baaManagerImpl) RecordAudit(ctx context.Context, baaID string, auditResult AuditResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	baa, exists := m.baas[baaID]
	if !exists {
		return fmt.Errorf("BAA not found: %s", baaID)
	}
	
	baa.LastAuditDate = &auditResult.AuditDate
	baa.NextAuditDate = &auditResult.NextAuditDate
	baa.UpdatedAt = time.Now()
	
	if baa.Metadata == nil {
		baa.Metadata = make(map[string]string)
	}
	baa.Metadata["last_audit_compliant"] = fmt.Sprintf("%t", auditResult.Compliant)
	baa.Metadata["last_audit_findings"] = fmt.Sprintf("%d", len(auditResult.Findings))
	
	return nil
}

// Helper functions

func (m *baaManagerImpl) generateBAAID(baa *BAA) string {
	data := fmt.Sprintf("%s-%s-%d", baa.OrganizationID, baa.BusinessAssociateName, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "baa-" + hex.EncodeToString(hash[:8])
}

func (m *baaManagerImpl) validateBAA(baa *BAA) error {
	if baa.OrganizationID == "" {
		return fmt.Errorf("organization ID is required")
	}
	
	if baa.OrganizationName == "" {
		return fmt.Errorf("organization name is required")
	}
	
	if baa.BusinessAssociateName == "" {
		return fmt.Errorf("business associate name is required")
	}
	
	if baa.EffectiveDate.IsZero() {
		return fmt.Errorf("effective date is required")
	}
	
	if baa.ExpirationDate.IsZero() {
		return fmt.Errorf("expiration date is required")
	}
	
	if baa.ExpirationDate.Before(baa.EffectiveDate) {
		return fmt.Errorf("expiration date must be after effective date")
	}
	
	if len(baa.Scope) == 0 {
		return fmt.Errorf("scope is required")
	}
	
	if baa.BreachNotification.Required && baa.BreachNotification.ContactEmail == "" {
		return fmt.Errorf("breach notification contact email is required")
	}
	
	return nil
}
