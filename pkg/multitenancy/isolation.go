package multitenancy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// IsolationLevel defines the level of tenant isolation
type IsolationLevel string

const (
	// IsolationLevelShared - Shared resources with logical separation
	IsolationLevelShared IsolationLevel = "shared"
	// IsolationLevelLogical - Logical separation with dedicated schemas
	IsolationLevelLogical IsolationLevel = "logical"
	// IsolationLevelPhysical - Physical separation with dedicated infrastructure
	IsolationLevelPhysical IsolationLevel = "physical"
)

// TenantIsolationManager manages tenant isolation
type TenantIsolationManager interface {
	// CreateTenant creates a new tenant with specified isolation
	CreateTenant(ctx context.Context, tenant *Tenant) error
	
	// GetTenantContext retrieves tenant-specific context
	GetTenantContext(ctx context.Context, tenantID string) (*TenantContext, error)
	
	// IsolateTenantData ensures data isolation for tenant
	IsolateTenantData(ctx context.Context, tenantID string) error
	
	// MigrateTenant migrates tenant to new region/isolation level
	MigrateTenant(ctx context.Context, tenantID string, opts MigrationOptions) error
	
	// DeleteTenant removes tenant and all data
	DeleteTenant(ctx context.Context, tenantID string) error
}

// Tenant represents a tenant in the system
type Tenant struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	IsolationLevel  IsolationLevel `json:"isolationLevel"`
	EncryptionKeyID string         `json:"encryptionKeyId"`
	DatabaseSchema  string         `json:"databaseSchema"`
	NetworkPolicy   string         `json:"networkPolicy"`
	Region          string         `json:"region"`
	Status          TenantStatus   `json:"status"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	
	// Resource limits
	MaxUsers      int   `json:"maxUsers"`
	MaxStorage    int64 `json:"maxStorage"`
	MaxMCPServers int   `json:"maxMcpServers"`
	
	// White-labeling
	BrandingConfig *BrandingConfig `json:"brandingConfig,omitempty"`
}

// TenantStatus represents tenant lifecycle status
type TenantStatus string

const (
	TenantStatusProvisioning TenantStatus = "provisioning"
	TenantStatusActive       TenantStatus = "active"
	TenantStatusSuspended    TenantStatus = "suspended"
	TenantStatusMigrating    TenantStatus = "migrating"
	TenantStatusDeleting     TenantStatus = "deleting"
	TenantStatusDeleted      TenantStatus = "deleted"
)

// TenantContext contains tenant-specific runtime context
type TenantContext struct {
	TenantID        string
	EncryptionKey   []byte
	DatabaseSchema  string
	NetworkPolicyID string
	IsolationLevel  IsolationLevel
	Region          string
	
	// Connection pools
	DatabasePool interface{}
	CachePool    interface{}
}

// BrandingConfig contains white-label configuration
type BrandingConfig struct {
	LogoURL         string            `json:"logoUrl"`
	PrimaryColor    string            `json:"primaryColor"`
	SecondaryColor  string            `json:"secondaryColor"`
	CompanyName     string            `json:"companyName"`
	CustomDomain    string            `json:"customDomain"`
	CustomCSS       string            `json:"customCss"`
	EmailTemplate   string            `json:"emailTemplate"`
	CustomMetadata  map[string]string `json:"customMetadata,omitempty"`
}

// MigrationOptions contains options for tenant migration
type MigrationOptions struct {
	TargetRegion        string
	TargetIsolationLevel IsolationLevel
	MigrateData         bool
	DowntimeAllowed     bool
	ValidationRequired  bool
}

// isolationManager is the default implementation
type isolationManager struct {
	mu              sync.RWMutex
	tenants         map[string]*Tenant
	contexts        map[string]*TenantContext
	encryptionKeys  map[string][]byte
}

// NewIsolationManager creates a new tenant isolation manager
func NewIsolationManager() TenantIsolationManager {
	return &isolationManager{
		tenants:        make(map[string]*Tenant),
		contexts:       make(map[string]*TenantContext),
		encryptionKeys: make(map[string][]byte),
	}
}

// CreateTenant creates a new tenant with isolation
func (m *isolationManager) CreateTenant(ctx context.Context, tenant *Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Validate tenant
	if tenant.ID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	if tenant.Name == "" {
		return fmt.Errorf("tenant name is required")
	}
	
	// Check if tenant already exists
	if _, exists := m.tenants[tenant.ID]; exists {
		return fmt.Errorf("tenant %s already exists", tenant.ID)
	}
	
	// Set defaults
	if tenant.IsolationLevel == "" {
		tenant.IsolationLevel = IsolationLevelLogical
	}
	if tenant.Status == "" {
		tenant.Status = TenantStatusProvisioning
	}
	
	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now
	
	// Generate encryption key
	encKey, err := generateEncryptionKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}
	
	keyID := fmt.Sprintf("key-%s", tenant.ID)
	tenant.EncryptionKeyID = keyID
	m.encryptionKeys[keyID] = encKey
	
	// Generate database schema name based on isolation level
	switch tenant.IsolationLevel {
	case IsolationLevelShared:
		tenant.DatabaseSchema = "public" // Shared schema
	case IsolationLevelLogical:
		tenant.DatabaseSchema = fmt.Sprintf("tenant_%s", sanitizeSchemaName(tenant.ID))
	case IsolationLevelPhysical:
		tenant.DatabaseSchema = fmt.Sprintf("tenant_%s_dedicated", sanitizeSchemaName(tenant.ID))
	}
	
	// Generate network policy
	tenant.NetworkPolicy = fmt.Sprintf("netpol-%s", tenant.ID)
	
	// Store tenant
	m.tenants[tenant.ID] = tenant
	
	// Create tenant context
	ctx = m.createTenantContext(tenant, encKey)
	
	// Mark as active
	tenant.Status = TenantStatusActive
	tenant.UpdatedAt = time.Now()
	
	return nil
}

// GetTenantContext retrieves tenant context
func (m *isolationManager) GetTenantContext(ctx context.Context, tenantID string) (*TenantContext, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	tenantCtx, exists := m.contexts[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant context not found for %s", tenantID)
	}
	
	return tenantCtx, nil
}

// IsolateTenantData ensures data isolation
func (m *isolationManager) IsolateTenantData(ctx context.Context, tenantID string) error {
	m.mu.RLock()
	tenant, exists := m.tenants[tenantID]
	m.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}
	
	// Apply isolation based on level
	switch tenant.IsolationLevel {
	case IsolationLevelShared:
		// Row-level security (RLS) in PostgreSQL
		return m.applyRowLevelSecurity(ctx, tenant)
		
	case IsolationLevelLogical:
		// Separate schema with dedicated tables
		return m.createDedicatedSchema(ctx, tenant)
		
	case IsolationLevelPhysical:
		// Separate database instance
		return m.provisionDedicatedDatabase(ctx, tenant)
	}
	
	return nil
}

// MigrateTenant migrates tenant to new configuration
func (m *isolationManager) MigrateTenant(ctx context.Context, tenantID string, opts MigrationOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}
	
	// Update status
	tenant.Status = TenantStatusMigrating
	tenant.UpdatedAt = time.Now()
	
	// Perform migration (simplified - production would be more complex)
	if opts.TargetRegion != "" && opts.TargetRegion != tenant.Region {
		tenant.Region = opts.TargetRegion
	}
	
	if opts.TargetIsolationLevel != "" && opts.TargetIsolationLevel != tenant.IsolationLevel {
		tenant.IsolationLevel = opts.TargetIsolationLevel
		// Re-provision with new isolation level
		if err := m.IsolateTenantData(ctx, tenantID); err != nil {
			return fmt.Errorf("failed to apply new isolation level: %w", err)
		}
	}
	
	// Mark as active
	tenant.Status = TenantStatusActive
	tenant.UpdatedAt = time.Now()
	
	return nil
}

// DeleteTenant removes tenant and all data
func (m *isolationManager) DeleteTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}
	
	// Update status
	tenant.Status = TenantStatusDeleting
	tenant.UpdatedAt = time.Now()
	
	// Delete tenant data based on isolation level
	switch tenant.IsolationLevel {
	case IsolationLevelShared:
		// Delete rows with WHERE tenant_id = ?
		// (implementation would be in data layer)
		
	case IsolationLevelLogical:
		// DROP SCHEMA tenant_xxx CASCADE
		// (implementation would be in data layer)
		
	case IsolationLevelPhysical:
		// DROP DATABASE tenant_xxx
		// (implementation would be in data layer)
	}
	
	// Remove encryption key
	delete(m.encryptionKeys, tenant.EncryptionKeyID)
	
	// Remove context
	delete(m.contexts, tenantID)
	
	// Mark as deleted
	tenant.Status = TenantStatusDeleted
	tenant.UpdatedAt = time.Now()
	
	return nil
}

// Helper functions

func (m *isolationManager) createTenantContext(tenant *Tenant, encKey []byte) context.Context {
	ctx := &TenantContext{
		TenantID:        tenant.ID,
		EncryptionKey:   encKey,
		DatabaseSchema:  tenant.DatabaseSchema,
		NetworkPolicyID: tenant.NetworkPolicy,
		IsolationLevel:  tenant.IsolationLevel,
		Region:          tenant.Region,
	}
	
	m.contexts[tenant.ID] = ctx
	return context.Background() // Would return context with values in production
}

func (m *isolationManager) applyRowLevelSecurity(ctx context.Context, tenant *Tenant) error {
	// In production, this would execute SQL like:
	// CREATE POLICY tenant_isolation ON table_name
	// USING (tenant_id = current_setting('app.current_tenant')::uuid);
	return nil
}

func (m *isolationManager) createDedicatedSchema(ctx context.Context, tenant *Tenant) error {
	// In production, this would execute SQL like:
	// CREATE SCHEMA tenant_xxx;
	// CREATE TABLE tenant_xxx.users (...);
	return nil
}

func (m *isolationManager) provisionDedicatedDatabase(ctx context.Context, tenant *Tenant) error {
	// In production, this would:
	// 1. Provision new RDS instance or PostgreSQL server
	// 2. Configure network isolation
	// 3. Set up replication if needed
	return nil
}

func generateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) // 256-bit key
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func sanitizeSchemaName(name string) string {
	// Remove special characters, keep alphanumeric and underscore
	sanitized := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			sanitized += string(c)
		}
	}
	return sanitized
}
