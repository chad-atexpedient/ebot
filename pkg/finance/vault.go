// Package finance provides financial services compliance features
package finance

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// Common errors
var (
	ErrTokenNotFound     = errors.New("token not found in vault")
	ErrTokenExpired      = errors.New("token has expired")
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrInvalidKey        = errors.New("invalid encryption key")
	ErrKeyRotationFailed = errors.New("key rotation failed")
	ErrUnauthorized      = errors.New("unauthorized vault access")
)

// TokenVault defines the interface for secure token storage
type TokenVault interface {
	// Store stores a token-to-value mapping with encryption
	Store(ctx context.Context, token, value string, opts StoreOptions) error

	// Retrieve retrieves and decrypts the original value for a token
	Retrieve(ctx context.Context, token string) (string, error)

	// Delete removes a token from the vault
	Delete(ctx context.Context, token string) error

	// Exists checks if a token exists
	Exists(ctx context.Context, token string) bool

	// RotateKey rotates the encryption key
	RotateKey(ctx context.Context) error

	// Stats returns vault statistics
	Stats(ctx context.Context) (*VaultStats, error)

	// Cleanup removes expired entries
	Cleanup(ctx context.Context) error
}

// StoreOptions configures how a value is stored
type StoreOptions struct {
	TTL           time.Duration     // Time to live (0 = never expires)
	Metadata      map[string]string // Optional metadata
	ForceEncrypt  bool              // Force encryption even if already encrypted
}

// VaultStats contains vault statistics
type VaultStats struct {
	TotalEntries   int64
	ExpiredEntries int64
	TotalAccesses  int64
	LastRotation   time.Time
	CurrentKeyID   string
}

// VaultEntry represents an encrypted vault entry
type VaultEntry struct {
	Token          string
	EncryptedValue []byte
	KeyID          string
	Nonce          []byte
	CreatedAt      time.Time
	ExpiresAt      *time.Time
	AccessCount    int64
	LastAccessedAt *time.Time
	Metadata       map[string]string
}

// AuditEntry represents a vault access audit log entry
type AuditEntry struct {
	ID          string
	Timestamp   time.Time
	UserID      string
	Operation   string // "store", "retrieve", "delete", "rotate"
	TokenMasked string
	Success     bool
	ErrorMsg    string
	IPAddress   string
	UserAgent   string
}

// Encryptor provides encryption/decryption capabilities
type Encryptor interface {
	Encrypt(plaintext []byte) (ciphertext []byte, nonce []byte, err error)
	Decrypt(ciphertext []byte, nonce []byte) (plaintext []byte, err error)
	GetKeyID() string
	RotateKey() error
}

// AESGCMEncryptor implements AES-256-GCM encryption
type AESGCMEncryptor struct {
	key   []byte
	keyID string
	mu    sync.RWMutex
}

// NewAESGCMEncryptor creates a new AES-256-GCM encryptor
func NewAESGCMEncryptor(key []byte) (*AESGCMEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: key must be 32 bytes for AES-256", ErrInvalidKey)
	}

	// Generate key ID from hash of key
	hash := sha256.Sum256(key)
	keyID := hex.EncodeToString(hash[:8])

	return &AESGCMEncryptor{
		key:   key,
		keyID: keyID,
	}, nil
}

// GenerateKey generates a new random 256-bit key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("%w: failed to generate nonce: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *AESGCMEncryptor) Decrypt(ciphertext []byte, nonce []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// GetKeyID returns the current key ID
func (e *AESGCMEncryptor) GetKeyID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.keyID
}

// RotateKey generates and sets a new encryption key
func (e *AESGCMEncryptor) RotateKey() error {
	newKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyRotationFailed, err)
	}

	hash := sha256.Sum256(newKey)
	newKeyID := hex.EncodeToString(hash[:8])

	e.mu.Lock()
	e.key = newKey
	e.keyID = newKeyID
	e.mu.Unlock()

	return nil
}

// SecureMemoryVault implements TokenVault with encrypted in-memory storage
// This is suitable for development/testing; production should use DatabaseVault
type SecureMemoryVault struct {
	entries     map[string]*VaultEntry
	encryptor   Encryptor
	auditLog    []AuditEntry
	lastRotation time.Time
	mu          sync.RWMutex
	logger      Logger
}

// VaultConfig configures the vault
type VaultConfig struct {
	EncryptionKey     []byte        // 32-byte key for AES-256
	DefaultTTL        time.Duration // Default TTL for entries
	CleanupInterval   time.Duration // How often to run cleanup
	EnableAuditLog    bool
	MaxAuditLogSize   int
}

// DefaultVaultConfig returns default configuration
func DefaultVaultConfig() VaultConfig {
	key, _ := GenerateKey()
	return VaultConfig{
		EncryptionKey:   key,
		DefaultTTL:      24 * time.Hour,
		CleanupInterval: time.Hour,
		EnableAuditLog:  true,
		MaxAuditLogSize: 10000,
	}
}

// NewSecureMemoryVault creates a new encrypted in-memory vault
func NewSecureMemoryVault(config VaultConfig, logger Logger) (*SecureMemoryVault, error) {
	encryptor, err := NewAESGCMEncryptor(config.EncryptionKey)
	if err != nil {
		return nil, err
	}

	vault := &SecureMemoryVault{
		entries:      make(map[string]*VaultEntry),
		encryptor:    encryptor,
		auditLog:     make([]AuditEntry, 0),
		lastRotation: time.Now(),
		logger:       logger,
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go vault.cleanupLoop(config.CleanupInterval)
	}

	return vault, nil
}

// Store stores an encrypted token-value mapping
func (v *SecureMemoryVault) Store(ctx context.Context, token, value string, opts StoreOptions) error {
	// Encrypt the value
	ciphertext, nonce, err := v.encryptor.Encrypt([]byte(value))
	if err != nil {
		v.logAudit(ctx, "store", token, false, err.Error())
		return err
	}

	entry := &VaultEntry{
		Token:          token,
		EncryptedValue: ciphertext,
		KeyID:          v.encryptor.GetKeyID(),
		Nonce:          nonce,
		CreatedAt:      time.Now(),
		AccessCount:    0,
		Metadata:       opts.Metadata,
	}

	if opts.TTL > 0 {
		expiresAt := time.Now().Add(opts.TTL)
		entry.ExpiresAt = &expiresAt
	}

	v.mu.Lock()
	v.entries[token] = entry
	v.mu.Unlock()

	v.logAudit(ctx, "store", token, true, "")
	return nil
}

// Retrieve retrieves and decrypts a value by token
func (v *SecureMemoryVault) Retrieve(ctx context.Context, token string) (string, error) {
	v.mu.Lock()
	entry, exists := v.entries[token]
	if !exists {
		v.mu.Unlock()
		v.logAudit(ctx, "retrieve", token, false, "not found")
		return "", ErrTokenNotFound
	}

	// Check expiration
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		delete(v.entries, token)
		v.mu.Unlock()
		v.logAudit(ctx, "retrieve", token, false, "expired")
		return "", ErrTokenExpired
	}

	// Update access statistics
	entry.AccessCount++
	now := time.Now()
	entry.LastAccessedAt = &now
	v.mu.Unlock()

	// Decrypt the value
	plaintext, err := v.encryptor.Decrypt(entry.EncryptedValue, entry.Nonce)
	if err != nil {
		v.logAudit(ctx, "retrieve", token, false, err.Error())
		return "", err
	}

	v.logAudit(ctx, "retrieve", token, true, "")
	return string(plaintext), nil
}

// Delete removes a token from the vault
func (v *SecureMemoryVault) Delete(ctx context.Context, token string) error {
	v.mu.Lock()
	_, exists := v.entries[token]
	if exists {
		delete(v.entries, token)
	}
	v.mu.Unlock()

	if !exists {
		v.logAudit(ctx, "delete", token, false, "not found")
		return ErrTokenNotFound
	}

	v.logAudit(ctx, "delete", token, true, "")
	return nil
}

// Exists checks if a token exists and is not expired
func (v *SecureMemoryVault) Exists(ctx context.Context, token string) bool {
	v.mu.RLock()
	entry, exists := v.entries[token]
	v.mu.RUnlock()

	if !exists {
		return false
	}

	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return false
	}

	return true
}

// RotateKey rotates the encryption key and re-encrypts all entries
func (v *SecureMemoryVault) RotateKey(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Store old encryptor for decryption
	oldEncryptor := v.encryptor

	// Create new encryptor with rotated key
	if err := v.encryptor.RotateKey(); err != nil {
		v.logAudit(ctx, "rotate", "", false, err.Error())
		return err
	}

	// Re-encrypt all entries with new key
	for token, entry := range v.entries {
		// Skip expired entries
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			continue
		}

		// Decrypt with old key
		plaintext, err := oldEncryptor.Decrypt(entry.EncryptedValue, entry.Nonce)
		if err != nil {
			// Log but continue with other entries
			if v.logger != nil {
				v.logger.Error("failed to decrypt entry during rotation", "token", maskToken(token), "error", err)
			}
			continue
		}

		// Encrypt with new key
		ciphertext, nonce, err := v.encryptor.Encrypt(plaintext)
		if err != nil {
			if v.logger != nil {
				v.logger.Error("failed to encrypt entry during rotation", "token", maskToken(token), "error", err)
			}
			continue
		}

		// Update entry
		entry.EncryptedValue = ciphertext
		entry.Nonce = nonce
		entry.KeyID = v.encryptor.GetKeyID()
	}

	v.lastRotation = time.Now()
	v.logAudit(ctx, "rotate", "", true, "")

	if v.logger != nil {
		v.logger.Info("key rotation completed", "newKeyID", v.encryptor.GetKeyID())
	}

	return nil
}

// Stats returns vault statistics
func (v *SecureMemoryVault) Stats(ctx context.Context) (*VaultStats, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	now := time.Now()
	var totalAccesses int64
	var expiredCount int64

	for _, entry := range v.entries {
		totalAccesses += entry.AccessCount
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			expiredCount++
		}
	}

	return &VaultStats{
		TotalEntries:   int64(len(v.entries)),
		ExpiredEntries: expiredCount,
		TotalAccesses:  totalAccesses,
		LastRotation:   v.lastRotation,
		CurrentKeyID:   v.encryptor.GetKeyID(),
	}, nil
}

// Cleanup removes expired entries
func (v *SecureMemoryVault) Cleanup(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	var removed int

	for token, entry := range v.entries {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			delete(v.entries, token)
			removed++
		}
	}

	if v.logger != nil && removed > 0 {
		v.logger.Info("vault cleanup completed", "entriesRemoved", removed)
	}

	return nil
}

// cleanupLoop runs periodic cleanup
func (v *SecureMemoryVault) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		_ = v.Cleanup(context.Background())
	}
}

// logAudit logs an audit entry
func (v *SecureMemoryVault) logAudit(ctx context.Context, operation, token string, success bool, errorMsg string) {
	entry := AuditEntry{
		ID:          generateAuditID(),
		Timestamp:   time.Now(),
		Operation:   operation,
		TokenMasked: maskToken(token),
		Success:     success,
		ErrorMsg:    errorMsg,
	}

	// TODO: Extract user info from context
	// entry.UserID = auth.UserIDFromContext(ctx)
	// entry.IPAddress = GetClientIPFromContext(ctx)

	v.mu.Lock()
	v.auditLog = append(v.auditLog, entry)

	// Trim audit log if too large
	if len(v.auditLog) > 10000 {
		v.auditLog = v.auditLog[len(v.auditLog)-10000:]
	}
	v.mu.Unlock()
}

// GetAuditLog returns the audit log
func (v *SecureMemoryVault) GetAuditLog() []AuditEntry {
	v.mu.RLock()
	defer v.mu.RUnlock()

	log := make([]AuditEntry, len(v.auditLog))
	copy(log, v.auditLog)
	return log
}

// Helper functions

// maskToken masks a token for logging (shows first 8 chars)
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:8] + "****"
}

// generateAuditID generates a unique audit entry ID
func generateAuditID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// SecureTokenizer generates secure tokens
type SecureTokenizer struct {
	hmacKey []byte
}

// NewSecureTokenizer creates a new secure tokenizer
func NewSecureTokenizer(hmacKey []byte) *SecureTokenizer {
	return &SecureTokenizer{hmacKey: hmacKey}
}

// GenerateRandomToken generates a cryptographically random token
func (t *SecureTokenizer) GenerateRandomToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "tok_" + base64.URLEncoding.EncodeToString(b)
}

// GenerateDeterministicToken generates a deterministic token from input
// This allows the same value to always produce the same token
func (t *SecureTokenizer) GenerateDeterministicToken(value string) string {
	h := sha256.New()
	h.Write(t.hmacKey)
	h.Write([]byte(value))
	hash := h.Sum(nil)
	return "tok_" + hex.EncodeToString(hash[:16])
}
