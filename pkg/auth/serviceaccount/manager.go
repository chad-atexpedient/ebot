package serviceaccount

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/chad-atexpedient/ebot/logger"
	"golang.org/x/crypto/bcrypt"
)

// ServiceAccountManager manages service accounts and API keys
type ServiceAccountManager interface {
	// CreateServiceAccount creates a new service account
	CreateServiceAccount(ctx context.Context, account *ServiceAccount) (*ServiceAccount, error)
	
	// GetServiceAccount retrieves a service account by ID
	GetServiceAccount(ctx context.Context, id string) (*ServiceAccount, error)
	
	// ListServiceAccounts lists service accounts
	ListServiceAccounts(ctx context.Context, workspaceID string) ([]*ServiceAccount, error)
	
	// UpdateServiceAccount updates a service account
	UpdateServiceAccount(ctx context.Context, account *ServiceAccount) error
	
	// DeleteServiceAccount deletes a service account
	DeleteServiceAccount(ctx context.Context, id string) error
	
	// GenerateAPIKey generates a new API key for a service account
	GenerateAPIKey(ctx context.Context, accountID string, opts *APIKeyOptions) (*APIKey, error)
	
	// ValidateAPIKey validates an API key and returns the service account
	ValidateAPIKey(ctx context.Context, key string) (*ServiceAccount, error)
	
	// RevokeAPIKey revokes an API key
	RevokeAPIKey(ctx context.Context, keyID string) error
	
	// ListAPIKeys lists API keys for a service account
	ListAPIKeys(ctx context.Context, accountID string) ([]*APIKey, error)
	
	// RotateAPIKey rotates an API key (creates new, marks old as deprecated)
	RotateAPIKey(ctx context.Context, keyID string, gracePeriod time.Duration) (*APIKey, error)
}

// ServiceAccount represents a machine-to-machine account
type ServiceAccount struct {
	ID          string
	Name        string
	Description string
	WorkspaceID string
	Type        AccountType
	Status      AccountStatus
	Permissions []Permission
	Tags        map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   string
	LastUsedAt  *time.Time
}

// AccountType represents the type of service account
type AccountType string

const (
	AccountTypeStandard  AccountType = "standard"
	AccountTypeElevated  AccountType = "elevated"
	AccountTypeReadOnly  AccountType = "read_only"
)

// AccountStatus represents the status of a service account
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusRevoked   AccountStatus = "revoked"
)

// Permission represents a permission granted to a service account
type Permission struct {
	Resource string   // e.g., "mcp_servers", "threads", "workspaces"
	Actions  []string // e.g., ["read", "write", "delete"]
	Scope    string   // e.g., "workspace:123", "global"
}

// APIKey represents an API key for service account authentication
type APIKey struct {
	ID               string
	ServiceAccountID string
	Name             string
	KeyPrefix        string // First 8 chars of key for identification
	KeyHash          string // bcrypt hash of full key
	Status           KeyStatus
	Scopes           []string
	ExpiresAt        *time.Time
	RotationPolicy   *RotationPolicy
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastUsedAt       *time.Time
	UsageCount       int64
	RateLimitRPM     int // Requests per minute
}

// KeyStatus represents the status of an API key
type KeyStatus string

const (
	KeyStatusActive      KeyStatus = "active"
	KeyStatusDeprecated  KeyStatus = "deprecated"
	KeyStatusRevoked     KeyStatus = "revoked"
	KeyStatusExpired     KeyStatus = "expired"
)

// RotationPolicy defines automatic key rotation behavior
type RotationPolicy struct {
	Enabled         bool
	RotationPeriod  time.Duration // e.g., 90 days
	GracePeriod     time.Duration // e.g., 7 days to use old key
	AutoRevoke      bool          // Automatically revoke after grace period
	NotifyBeforeDays int          // Notify N days before expiration
}

// APIKeyOptions configures new API key generation
type APIKeyOptions struct {
	Name           string
	Scopes         []string
	ExpiresAt      *time.Time
	RotationPolicy *RotationPolicy
	RateLimitRPM   int
}

// serviceAccountManager implements ServiceAccountManager
type serviceAccountManager struct {
	accounts map[string]*ServiceAccount
	keys     map[string]*APIKey
	keyIndex map[string]string // keyHash -> keyID
	mu       sync.RWMutex
	logger   logger.Logger
}

// NewServiceAccountManager creates a new service account manager
func NewServiceAccountManager(log logger.Logger) ServiceAccountManager {
	return &serviceAccountManager{
		accounts: make(map[string]*ServiceAccount),
		keys:     make(map[string]*APIKey),
		keyIndex: make(map[string]string),
		logger:   log,
	}
}

// CreateServiceAccount creates a new service account
func (m *serviceAccountManager) CreateServiceAccount(ctx context.Context, account *ServiceAccount) (*ServiceAccount, error) {
	if account.Name == "" {
		return nil, fmt.Errorf("service account name is required")
	}
	if account.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if account.ID == "" {
		account.ID = generateID()
	}
	
	if _, exists := m.accounts[account.ID]; exists {
		return nil, fmt.Errorf("service account %s already exists", account.ID)
	}
	
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now
	account.Status = AccountStatusActive
	
	if account.Type == "" {
		account.Type = AccountTypeStandard
	}
	
	m.accounts[account.ID] = account
	
	m.logger.Info("Created service account",
		"id", account.ID,
		"name", account.Name,
		"workspace_id", account.WorkspaceID)
	
	return account, nil
}

// GetServiceAccount retrieves a service account by ID
func (m *serviceAccountManager) GetServiceAccount(ctx context.Context, id string) (*ServiceAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	account, exists := m.accounts[id]
	if !exists {
		return nil, fmt.Errorf("service account %s not found", id)
	}
	
	return account, nil
}

// ListServiceAccounts lists service accounts
func (m *serviceAccountManager) ListServiceAccounts(ctx context.Context, workspaceID string) ([]*ServiceAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	accounts := make([]*ServiceAccount, 0)
	for _, account := range m.accounts {
		if workspaceID == "" || account.WorkspaceID == workspaceID {
			accounts = append(accounts, account)
		}
	}
	
	return accounts, nil
}

// UpdateServiceAccount updates a service account
func (m *serviceAccountManager) UpdateServiceAccount(ctx context.Context, account *ServiceAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.accounts[account.ID]; !exists {
		return fmt.Errorf("service account %s not found", account.ID)
	}
	
	account.UpdatedAt = time.Now()
	m.accounts[account.ID] = account
	
	m.logger.Info("Updated service account",
		"id", account.ID,
		"name", account.Name)
	
	return nil
}

// DeleteServiceAccount deletes a service account
func (m *serviceAccountManager) DeleteServiceAccount(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.accounts[id]; !exists {
		return fmt.Errorf("service account %s not found", id)
	}
	
	// Revoke all API keys
	for keyID, key := range m.keys {
		if key.ServiceAccountID == id {
			delete(m.keys, keyID)
			delete(m.keyIndex, key.KeyHash)
		}
	}
	
	delete(m.accounts, id)
	
	m.logger.Info("Deleted service account", "id", id)
	
	return nil
}

// GenerateAPIKey generates a new API key for a service account
func (m *serviceAccountManager) GenerateAPIKey(ctx context.Context, accountID string, opts *APIKeyOptions) (*APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	account, exists := m.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("service account %s not found", accountID)
	}
	
	if account.Status != AccountStatusActive {
		return nil, fmt.Errorf("service account is not active")
	}
	
	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	
	// Format: ebot_<base64-encoded-random>
	fullKey := "ebot_" + base64.RawURLEncoding.EncodeToString(keyBytes)
	
	// Hash the key for storage
	keyHash, err := bcrypt.GenerateFromPassword([]byte(fullKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %w", err)
	}
	
	// Create API key record
	apiKey := &APIKey{
		ID:               generateID(),
		ServiceAccountID: accountID,
		Name:             opts.Name,
		KeyPrefix:        fullKey[:12], // "ebot_" + first 7 chars
		KeyHash:          string(keyHash),
		Status:           KeyStatusActive,
		Scopes:           opts.Scopes,
		ExpiresAt:        opts.ExpiresAt,
		RotationPolicy:   opts.RotationPolicy,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		RateLimitRPM:     opts.RateLimitRPM,
	}
	
	if apiKey.RateLimitRPM == 0 {
		apiKey.RateLimitRPM = 1000 // Default limit
	}
	
	m.keys[apiKey.ID] = apiKey
	m.keyIndex[apiKey.KeyHash] = apiKey.ID
	
	m.logger.Info("Generated API key",
		"key_id", apiKey.ID,
		"account_id", accountID,
		"key_prefix", apiKey.KeyPrefix)
	
	// Return the key with the actual key value (only time it's available)
	// Store the full key temporarily in the Key field
	apiKey.KeyHash = fullKey // Temporarily replace for return
	
	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns the service account
func (m *serviceAccountManager) ValidateAPIKey(ctx context.Context, key string) (*ServiceAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Find key by comparing hashes
	for _, apiKey := range m.keys {
		if err := bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(key)); err == nil {
			// Key is valid, check status
			if apiKey.Status != KeyStatusActive {
				return nil, fmt.Errorf("API key is not active: %s", apiKey.Status)
			}
			
			// Check expiration
			if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
				return nil, fmt.Errorf("API key has expired")
			}
			
			// Get service account
			account, exists := m.accounts[apiKey.ServiceAccountID]
			if !exists {
				return nil, fmt.Errorf("service account not found")
			}
			
			if account.Status != AccountStatusActive {
				return nil, fmt.Errorf("service account is not active")
			}
			
			// Update last used (would be async in production)
			now := time.Now()
			apiKey.LastUsedAt = &now
			apiKey.UsageCount++
			account.LastUsedAt = &now
			
			m.logger.Info("API key validated",
				"key_id", apiKey.ID,
				"account_id", account.ID,
				"usage_count", apiKey.UsageCount)
			
			return account, nil
		}
	}
	
	return nil, fmt.Errorf("invalid API key")
}

// RevokeAPIKey revokes an API key
func (m *serviceAccountManager) RevokeAPIKey(ctx context.Context, keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key, exists := m.keys[keyID]
	if !exists {
		return fmt.Errorf("API key %s not found", keyID)
	}
	
	key.Status = KeyStatusRevoked
	key.UpdatedAt = time.Now()
	
	m.logger.Info("Revoked API key",
		"key_id", keyID,
		"account_id", key.ServiceAccountID)
	
	return nil
}

// ListAPIKeys lists API keys for a service account
func (m *serviceAccountManager) ListAPIKeys(ctx context.Context, accountID string) ([]*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	keys := make([]*APIKey, 0)
	for _, key := range m.keys {
		if key.ServiceAccountID == accountID {
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

// RotateAPIKey rotates an API key (creates new, marks old as deprecated)
func (m *serviceAccountManager) RotateAPIKey(ctx context.Context, keyID string, gracePeriod time.Duration) (*APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	oldKey, exists := m.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("API key %s not found", keyID)
	}
	
	// Mark old key as deprecated
	oldKey.Status = KeyStatusDeprecated
	oldKey.UpdatedAt = time.Now()
	
	// Create new key with same settings
	m.mu.Unlock() // Unlock to call GenerateAPIKey
	newKey, err := m.GenerateAPIKey(ctx, oldKey.ServiceAccountID, &APIKeyOptions{
		Name:           oldKey.Name + " (rotated)",
		Scopes:         oldKey.Scopes,
		ExpiresAt:      oldKey.ExpiresAt,
		RotationPolicy: oldKey.RotationPolicy,
		RateLimitRPM:   oldKey.RateLimitRPM,
	})
	m.mu.Lock() // Re-lock
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}
	
	// Schedule old key revocation after grace period
	if gracePeriod > 0 {
		go func() {
			time.Sleep(gracePeriod)
			m.RevokeAPIKey(context.Background(), keyID)
		}()
	}
	
	m.logger.Info("Rotated API key",
		"old_key_id", keyID,
		"new_key_id", newKey.ID,
		"grace_period", gracePeriod)
	
	return newKey, nil
}

// Helper functions

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("sa_%s", base64.RawURLEncoding.EncodeToString(b))
}
