package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// SecretsRotationConfig configures automatic secrets rotation
type SecretsRotationConfig struct {
	// Rotation intervals
	APIKeyRotationInterval        time.Duration
	DatabasePasswordInterval      time.Duration
	EncryptionKeyInterval         time.Duration
	OAuthClientSecretInterval     time.Duration
	ServiceAccountKeyInterval     time.Duration

	// Grace periods (old secrets remain valid)
	GracePeriod time.Duration

	// Notification settings
	NotifyDaysBefore int
	NotifyEmail      string
	NotifyWebhook    string

	// Auto-rotation enabled
	EnableAutoRotation bool
}

// DefaultSecretsRotationConfig returns secure defaults
func DefaultSecretsRotationConfig() *SecretsRotationConfig {
	return &SecretsRotationConfig{
		APIKeyRotationInterval:        90 * 24 * time.Hour,  // 90 days
		DatabasePasswordInterval:      30 * 24 * time.Hour,  // 30 days
		EncryptionKeyInterval:         365 * 24 * time.Hour, // 1 year
		OAuthClientSecretInterval:     180 * 24 * time.Hour, // 6 months
		ServiceAccountKeyInterval:     90 * 24 * time.Hour,  // 90 days
		GracePeriod:                   7 * 24 * time.Hour,   // 7 days
		NotifyDaysBefore:              14,
		EnableAutoRotation:            true,
	}
}

// SecretsRotationManager manages automatic credential rotation
type SecretsRotationManager struct {
	config   *SecretsRotationConfig
	secrets  sync.Map // map[string]*Secret
	storage  SecretsStorage
	notifier Notifier
	logger   *slog.Logger
	stopCh   chan struct{}
}

// Secret represents a rotatable secret
type Secret struct {
	ID             string
	Type           SecretType
	CurrentValue   string
	PreviousValue  string
	CreatedAt      time.Time
	LastRotatedAt  time.Time
	NextRotationAt time.Time
	RotationCount  int
	GraceEndsAt    time.Time
	Metadata       map[string]string
	mu             sync.RWMutex
}

// SecretType defines the type of secret
type SecretType string

const (
	SecretTypeAPIKey              SecretType = "api_key"
	SecretTypeDatabasePassword    SecretType = "database_password"
	SecretTypeEncryptionKey       SecretType = "encryption_key"
	SecretTypeOAuthClientSecret   SecretType = "oauth_client_secret"
	SecretTypeServiceAccountKey   SecretType = "service_account_key"
	SecretTypeJWTSigningKey       SecretType = "jwt_signing_key"
	SecretTypeSAMLSigningKey      SecretType = "saml_signing_key"
)

// SecretsStorage interface for persisting secrets
type SecretsStorage interface {
	Store(ctx context.Context, secret *Secret) error
	Get(ctx context.Context, id string) (*Secret, error)
	List(ctx context.Context, secretType SecretType) ([]*Secret, error)
	Delete(ctx context.Context, id string) error
}

// Notifier interface for rotation notifications
type Notifier interface {
	NotifyRotationDue(ctx context.Context, secret *Secret, daysUntil int) error
	NotifyRotationCompleted(ctx context.Context, secret *Secret) error
	NotifyRotationFailed(ctx context.Context, secret *Secret, err error) error
}

// NewSecretsRotationManager creates a new secrets rotation manager
func NewSecretsRotationManager(
	config *SecretsRotationConfig,
	storage SecretsStorage,
	notifier Notifier,
	logger *slog.Logger,
) *SecretsRotationManager {
	if config == nil {
		config = DefaultSecretsRotationConfig()
	}

	srm := &SecretsRotationManager{
		config:   config,
		storage:  storage,
		notifier: notifier,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}

	// Start rotation monitor if auto-rotation enabled
	if config.EnableAutoRotation {
		go srm.monitorRotations()
	}

	return srm
}

// RegisterSecret registers a secret for automatic rotation
func (srm *SecretsRotationManager) RegisterSecret(ctx context.Context, secret *Secret) error {
	now := time.Now()

	secret.CreatedAt = now
	secret.LastRotatedAt = now
	secret.NextRotationAt = srm.calculateNextRotation(secret.Type, now)

	// Store in memory and persistent storage
	srm.secrets.Store(secret.ID, secret)
	if err := srm.storage.Store(ctx, secret); err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}

	srm.logger.Info("Secret registered for rotation",
		"id", secret.ID,
		"type", secret.Type,
		"next_rotation", secret.NextRotationAt,
	)

	return nil
}

// RotateSecret manually rotates a secret
func (srm *SecretsRotationManager) RotateSecret(ctx context.Context, secretID string) error {
	val, ok := srm.secrets.Load(secretID)
	if !ok {
		return fmt.Errorf("secret %s not found", secretID)
	}

	secret := val.(*Secret)
	return srm.rotateSecret(ctx, secret)
}

// rotateSecret performs the actual rotation
func (srm *SecretsRotationManager) rotateSecret(ctx context.Context, secret *Secret) error {
	secret.mu.Lock()
	defer secret.mu.Unlock()

	srm.logger.Info("Rotating secret",
		"id", secret.ID,
		"type", secret.Type,
		"rotation_count", secret.RotationCount,
	)

	// Generate new secret value
	newValue, err := srm.generateSecretValue(secret.Type)
	if err != nil {
		srm.notifier.NotifyRotationFailed(ctx, secret, err)
		return fmt.Errorf("failed to generate new secret: %w", err)
	}

	// Move current to previous
	secret.PreviousValue = secret.CurrentValue
	secret.CurrentValue = newValue
	secret.LastRotatedAt = time.Now()
	secret.NextRotationAt = srm.calculateNextRotation(secret.Type, secret.LastRotatedAt)
	secret.GraceEndsAt = secret.LastRotatedAt.Add(srm.config.GracePeriod)
	secret.RotationCount++

	// Persist the change
	if err := srm.storage.Store(ctx, secret); err != nil {
		return fmt.Errorf("failed to store rotated secret: %w", err)
	}

	// Notify success
	if err := srm.notifier.NotifyRotationCompleted(ctx, secret); err != nil {
		srm.logger.Warn("Failed to send rotation notification", "error", err)
	}

	srm.logger.Info("Secret rotated successfully",
		"id", secret.ID,
		"type", secret.Type,
		"next_rotation", secret.NextRotationAt,
	)

	return nil
}

// generateSecretValue generates a cryptographically secure secret
func (srm *SecretsRotationManager) generateSecretValue(secretType SecretType) (string, error) {
	var length int

	switch secretType {
	case SecretTypeAPIKey:
		length = 32
	case SecretTypeDatabasePassword:
		length = 32
	case SecretTypeEncryptionKey:
		length = 32
	case SecretTypeOAuthClientSecret:
		length = 32
	case SecretTypeServiceAccountKey:
		length = 64
	case SecretTypeJWTSigningKey:
		length = 64
	case SecretTypeSAMLSigningKey:
		length = 64
	default:
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// calculateNextRotation calculates when the secret should be rotated next
func (srm *SecretsRotationManager) calculateNextRotation(secretType SecretType, from time.Time) time.Time {
	var interval time.Duration

	switch secretType {
	case SecretTypeAPIKey:
		interval = srm.config.APIKeyRotationInterval
	case SecretTypeDatabasePassword:
		interval = srm.config.DatabasePasswordInterval
	case SecretTypeEncryptionKey:
		interval = srm.config.EncryptionKeyInterval
	case SecretTypeOAuthClientSecret:
		interval = srm.config.OAuthClientSecretInterval
	case SecretTypeServiceAccountKey:
		interval = srm.config.ServiceAccountKeyInterval
	default:
		interval = 90 * 24 * time.Hour // Default 90 days
	}

	return from.Add(interval)
}

// monitorRotations continuously checks for secrets needing rotation
func (srm *SecretsRotationManager) monitorRotations() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			srm.checkAndRotate(ctx)
		case <-srm.stopCh:
			return
		}
	}
}

// checkAndRotate checks all secrets and rotates those that are due
func (srm *SecretsRotationManager) checkAndRotate(ctx context.Context) {
	now := time.Now()

	srm.secrets.Range(func(key, value interface{}) bool {
		secret := value.(*Secret)

		// Check if rotation is due
		if now.After(secret.NextRotationAt) {
			if err := srm.rotateSecret(ctx, secret); err != nil {
				srm.logger.Error("Failed to rotate secret",
					"id", secret.ID,
					"error", err,
				)
			}
			return true
		}

		// Check if notification is needed
		daysUntil := int(time.Until(secret.NextRotationAt).Hours() / 24)
		if daysUntil <= srm.config.NotifyDaysBefore && daysUntil > 0 {
			if err := srm.notifier.NotifyRotationDue(ctx, secret, daysUntil); err != nil {
				srm.logger.Warn("Failed to send rotation notification",
					"id", secret.ID,
					"error", err,
				)
			}
		}

		return true
	})
}

// GetSecret returns the current value of a secret
func (srm *SecretsRotationManager) GetSecret(ctx context.Context, secretID string) (string, error) {
	val, ok := srm.secrets.Load(secretID)
	if !ok {
		// Try loading from storage
		secret, err := srm.storage.Get(ctx, secretID)
		if err != nil {
			return "", fmt.Errorf("secret not found: %w", err)
		}
		srm.secrets.Store(secretID, secret)
		return secret.CurrentValue, nil
	}

	secret := val.(*Secret)
	secret.mu.RLock()
	defer secret.mu.RUnlock()

	return secret.CurrentValue, nil
}

// ValidateSecret checks if a secret value is valid (current or in grace period)
func (srm *SecretsRotationManager) ValidateSecret(ctx context.Context, secretID, value string) (bool, error) {
	val, ok := srm.secrets.Load(secretID)
	if !ok {
		// Try loading from storage
		secret, err := srm.storage.Get(ctx, secretID)
		if err != nil {
			return false, fmt.Errorf("secret not found: %w", err)
		}
		srm.secrets.Store(secretID, secret)
		val = secret
	}

	secret := val.(*Secret)
	secret.mu.RLock()
	defer secret.mu.RUnlock()

	// Check current value
	if value == secret.CurrentValue {
		return true, nil
	}

	// Check previous value (if within grace period)
	now := time.Now()
	if value == secret.PreviousValue && now.Before(secret.GraceEndsAt) {
		srm.logger.Info("Secret validated using previous value (grace period)",
			"id", secretID,
			"grace_ends", secret.GraceEndsAt,
		)
		return true, nil
	}

	return false, nil
}

// GetRotationStatus returns the rotation status for a secret
func (srm *SecretsRotationManager) GetRotationStatus(ctx context.Context, secretID string) (*RotationStatus, error) {
	val, ok := srm.secrets.Load(secretID)
	if !ok {
		return nil, fmt.Errorf("secret %s not found", secretID)
	}

	secret := val.(*Secret)
	secret.mu.RLock()
	defer secret.mu.RUnlock()

	now := time.Now()
	daysUntilRotation := int(time.Until(secret.NextRotationAt).Hours() / 24)

	status := &RotationStatus{
		SecretID:          secret.ID,
		Type:              secret.Type,
		LastRotatedAt:     secret.LastRotatedAt,
		NextRotationAt:    secret.NextRotationAt,
		RotationCount:     secret.RotationCount,
		DaysUntilRotation: daysUntilRotation,
		InGracePeriod:     now.Before(secret.GraceEndsAt),
		GraceEndsAt:       secret.GraceEndsAt,
	}

	if daysUntilRotation <= 0 {
		status.Status = "overdue"
	} else if daysUntilRotation <= srm.config.NotifyDaysBefore {
		status.Status = "due_soon"
	} else {
		status.Status = "ok"
	}

	return status, nil
}

// Stop stops the rotation monitor
func (srm *SecretsRotationManager) Stop() {
	close(srm.stopCh)
}

// RotationStatus contains the rotation status of a secret
type RotationStatus struct {
	SecretID          string
	Type              SecretType
	Status            string // ok, due_soon, overdue
	LastRotatedAt     time.Time
	NextRotationAt    time.Time
	RotationCount     int
	DaysUntilRotation int
	InGracePeriod     bool
	GraceEndsAt       time.Time
}

// ListSecretsForRotation returns all secrets due for rotation
func (srm *SecretsRotationManager) ListSecretsForRotation(ctx context.Context) ([]*RotationStatus, error) {
	var statuses []*RotationStatus

	srm.secrets.Range(func(key, value interface{}) bool {
		secret := value.(*Secret)
		status, err := srm.GetRotationStatus(ctx, secret.ID)
		if err != nil {
			srm.logger.Error("Failed to get rotation status", "id", secret.ID, "error", err)
			return true
		}

		if status.Status != "ok" {
			statuses = append(statuses, status)
		}

		return true
	})

	return statuses, nil
}
