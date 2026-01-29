package multitenancy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TenantManager provides tenant lifecycle management
type TenantManager interface {
	// CreateTenant creates a new tenant
	CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error)

	// GetTenant retrieves a tenant by ID
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)

	// UpdateTenant updates tenant settings
	UpdateTenant(ctx context.Context, tenantID string, req UpdateTenantRequest) (*Tenant, error)

	// DeleteTenant removes a tenant and all data
	DeleteTenant(ctx context.Context, tenantID string) error

	// ListTenants lists tenants with optional filtering
	ListTenants(ctx context.Context, opts ListOptions) ([]*Tenant, error)

	// SuspendTenant suspends a tenant
	SuspendTenant(ctx context.Context, tenantID string, reason string) error

	// ActivateTenant activates a suspended tenant
	ActivateTenant(ctx context.Context, tenantID string) error

	// GetTenantUsage retrieves usage statistics
	GetTenantUsage(ctx context.Context, tenantID string) (*TenantUsage, error)
}

// CreateTenantRequest contains parameters for creating a tenant
type CreateTenantRequest struct {
	Name           string            `json:"name"`
	DisplayName    string            `json:"displayName"`
	IsolationLevel IsolationLevel    `json:"isolationLevel"`
	Region         string            `json:"region"`
	Plan           string            `json:"plan"`
	MaxUsers       int               `json:"maxUsers"`
	MaxStorage     int64             `json:"maxStorage"`
	MaxMCPServers  int               `json:"maxMcpServers"`
	AdminEmail     string            `json:"adminEmail"`
	CustomDomain   string            `json:"customDomain,omitempty"`
	Branding       *BrandingConfig   `json:"branding,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// UpdateTenantRequest contains parameters for updating a tenant
type UpdateTenantRequest struct {
	DisplayName   *string           `json:"displayName,omitempty"`
	Plan          *string           `json:"plan,omitempty"`
	MaxUsers      *int              `json:"maxUsers,omitempty"`
	MaxStorage    *int64            `json:"maxStorage,omitempty"`
	MaxMCPServers *int              `json:"maxMcpServers,omitempty"`
	CustomDomain  *string           `json:"customDomain,omitempty"`
	Branding      *BrandingConfig   `json:"branding,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// ListOptions contains options for listing tenants
type ListOptions struct {
	Offset   int
	Limit    int
	Status   TenantStatus
	Region   string
	SortBy   string
	SortDesc bool
}

// TenantUsage contains usage statistics for a tenant
type TenantUsage struct {
	TenantID       string    `json:"tenantId"`
	Period         time.Time `json:"period"`
	ActiveUsers    int64     `json:"activeUsers"`
	TotalUsers     int64     `json:"totalUsers"`
	APIRequests    int64     `json:"apiRequests"`
	TokensUsed     int64     `json:"tokensUsed"`
	StorageBytes   int64     `json:"storageBytes"`
	ComputeSeconds float64   `json:"computeSeconds"`
	MCPServers     int       `json:"mcpServers"`
	KnowledgeBases int       `json:"knowledgeBases"`
}

// tenantManager is the default implementation
type tenantManager struct {
	mu              sync.RWMutex
	tenants         map[string]*Tenant
	isolationMgr    TenantIsolationManager
	provisioner     *TenantProvisioner
	usageTracker    map[string]*TenantUsage
}

// NewTenantManager creates a new tenant manager
func NewTenantManager(isolationMgr TenantIsolationManager) TenantManager {
	return &tenantManager{
		tenants:      make(map[string]*Tenant),
		isolationMgr: isolationMgr,
		provisioner:  NewTenantProvisioner(),
		usageTracker: make(map[string]*TenantUsage),
	}
}

// CreateTenant creates a new tenant
func (m *tenantManager) CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate request
	if req.Name == "" {
		return nil, fmt.Errorf("tenant name is required")
	}
	if req.AdminEmail == "" {
		return nil, fmt.Errorf("admin email is required")
	}

	// Generate tenant ID
	tenantID := generateTenantID(req.Name)

	// Check for duplicates
	if _, exists := m.tenants[tenantID]; exists {
		return nil, fmt.Errorf("tenant with name %s already exists", req.Name)
	}

	// Set defaults
	if req.IsolationLevel == "" {
		req.IsolationLevel = IsolationLevelLogical
	}
	if req.MaxUsers == 0 {
		req.MaxUsers = 100 // Default max users
	}
	if req.MaxStorage == 0 {
		req.MaxStorage = 10 * 1024 * 1024 * 1024 // 10GB default
	}
	if req.MaxMCPServers == 0 {
		req.MaxMCPServers = 10 // Default MCP servers
	}

	now := time.Now()

	// Create tenant object
	tenant := &Tenant{
		ID:             tenantID,
		Name:           req.Name,
		IsolationLevel: req.IsolationLevel,
		Region:         req.Region,
		Status:         TenantStatusProvisioning,
		MaxUsers:       req.MaxUsers,
		MaxStorage:     req.MaxStorage,
		MaxMCPServers:  req.MaxMCPServers,
		BrandingConfig: req.Branding,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Provision tenant using isolation manager
	if err := m.isolationMgr.CreateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to provision tenant: %w", err)
	}

	// Run provisioning workflow
	if err := m.provisioner.ProvisionTenant(ctx, tenant); err != nil {
		// Rollback isolation
		m.isolationMgr.DeleteTenant(ctx, tenantID)
		return nil, fmt.Errorf("provisioning failed: %w", err)
	}

	// Store tenant
	m.tenants[tenantID] = tenant

	// Initialize usage tracking
	m.usageTracker[tenantID] = &TenantUsage{
		TenantID: tenantID,
		Period:   now,
	}

	return tenant, nil
}

// GetTenant retrieves a tenant by ID
func (m *tenantManager) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}

	return tenant, nil
}

// UpdateTenant updates tenant settings
func (m *tenantManager) UpdateTenant(ctx context.Context, tenantID string, req UpdateTenantRequest) (*Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}

	// Apply updates
	if req.DisplayName != nil {
		tenant.Name = *req.DisplayName
	}
	if req.MaxUsers != nil {
		tenant.MaxUsers = *req.MaxUsers
	}
	if req.MaxStorage != nil {
		tenant.MaxStorage = *req.MaxStorage
	}
	if req.MaxMCPServers != nil {
		tenant.MaxMCPServers = *req.MaxMCPServers
	}
	if req.Branding != nil {
		tenant.BrandingConfig = req.Branding
	}
	if req.CustomDomain != nil && tenant.BrandingConfig != nil {
		tenant.BrandingConfig.CustomDomain = *req.CustomDomain
	}

	tenant.UpdatedAt = time.Now()

	return tenant, nil
}

// DeleteTenant removes a tenant and all data
func (m *tenantManager) DeleteTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	// Check if tenant can be deleted
	if tenant.Status == TenantStatusDeleting {
		return fmt.Errorf("tenant %s is already being deleted", tenantID)
	}

	// Mark as deleting
	tenant.Status = TenantStatusDeleting
	tenant.UpdatedAt = time.Now()

	// Delete via isolation manager (handles data cleanup)
	if err := m.isolationMgr.DeleteTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to delete tenant data: %w", err)
	}

	// Remove from memory
	delete(m.tenants, tenantID)
	delete(m.usageTracker, tenantID)

	return nil
}

// ListTenants lists tenants with optional filtering
func (m *tenantManager) ListTenants(ctx context.Context, opts ListOptions) ([]*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect matching tenants
	var result []*Tenant
	for _, tenant := range m.tenants {
		// Filter by status
		if opts.Status != "" && tenant.Status != opts.Status {
			continue
		}
		// Filter by region
		if opts.Region != "" && tenant.Region != opts.Region {
			continue
		}
		result = append(result, tenant)
	}

	// Apply pagination
	start := opts.Offset
	if start > len(result) {
		return []*Tenant{}, nil
	}

	end := start + opts.Limit
	if opts.Limit == 0 || end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// SuspendTenant suspends a tenant
func (m *tenantManager) SuspendTenant(ctx context.Context, tenantID string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	if tenant.Status == TenantStatusSuspended {
		return fmt.Errorf("tenant %s is already suspended", tenantID)
	}

	tenant.Status = TenantStatusSuspended
	tenant.UpdatedAt = time.Now()

	return nil
}

// ActivateTenant activates a suspended tenant
func (m *tenantManager) ActivateTenant(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	if tenant.Status != TenantStatusSuspended {
		return fmt.Errorf("tenant %s is not suspended", tenantID)
	}

	tenant.Status = TenantStatusActive
	tenant.UpdatedAt = time.Now()

	return nil
}

// GetTenantUsage retrieves usage statistics
func (m *tenantManager) GetTenantUsage(ctx context.Context, tenantID string) (*TenantUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usage, exists := m.usageTracker[tenantID]
	if !exists {
		return nil, fmt.Errorf("no usage data for tenant %s", tenantID)
	}

	return usage, nil
}

// Helper functions

func generateTenantID(name string) string {
	// Generate a URL-safe tenant ID from name
	sanitized := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			sanitized += string(c)
		} else if c >= 'A' && c <= 'Z' {
			sanitized += string(c + 32) // lowercase
		} else if c == ' ' || c == '-' || c == '_' {
			sanitized += "-"
		}
	}

	// Add timestamp suffix for uniqueness
	return fmt.Sprintf("%s-%d", sanitized, time.Now().Unix())
}

// TenantContextKey is the context key for tenant ID
type TenantContextKey struct{}

// GetTenantFromContext retrieves tenant ID from context
func GetTenantFromContext(ctx context.Context) string {
	if v := ctx.Value(TenantContextKey{}); v != nil {
		return v.(string)
	}
	return ""
}

// WithTenant adds tenant ID to context
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantContextKey{}, tenantID)
}
