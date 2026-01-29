package finance

import (
	"context"
	"testing"
	"time"
)

type testLogger struct{}

func (l *testLogger) Info(msg string, args ...interface{})  {}
func (l *testLogger) Error(msg string, args ...interface{}) {}
func (l *testLogger) Warn(msg string, args ...interface{})  {}

func TestAESGCMEncryptor(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Test encrypt/decrypt roundtrip
	plaintext := []byte("4532015112830366")

	ciphertext, nonce, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("Ciphertext should not be empty")
	}

	if len(nonce) == 0 {
		t.Error("Nonce should not be empty")
	}

	// Verify ciphertext is different from plaintext
	if string(ciphertext) == string(plaintext) {
		t.Error("Ciphertext should be different from plaintext")
	}

	// Decrypt
	decrypted, err := encryptor.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}
}

func TestAESGCMEncryptor_InvalidKey(t *testing.T) {
	// Key too short
	_, err := NewAESGCMEncryptor([]byte("short"))
	if err == nil {
		t.Error("Expected error for short key")
	}

	// Key too long
	longKey := make([]byte, 64)
	_, err = NewAESGCMEncryptor(longKey)
	if err == nil {
		t.Error("Expected error for long key")
	}
}

func TestAESGCMEncryptor_KeyRotation(t *testing.T) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)

	oldKeyID := encryptor.GetKeyID()

	// Rotate key
	err := encryptor.RotateKey()
	if err != nil {
		t.Fatalf("Key rotation failed: %v", err)
	}

	newKeyID := encryptor.GetKeyID()

	if oldKeyID == newKeyID {
		t.Error("Key ID should change after rotation")
	}
}

func TestSecureMemoryVault_StoreAndRetrieve(t *testing.T) {
	config := DefaultVaultConfig()
	vault, err := NewSecureMemoryVault(config, &testLogger{})
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	ctx := context.Background()
	token := "tok_test123"
	value := "4532015112830366"

	// Store
	err = vault.Store(ctx, token, value, StoreOptions{})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Retrieve
	retrieved, err := vault.Retrieve(ctx, token)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if retrieved != value {
		t.Errorf("Retrieved value mismatch: got %s, want %s", retrieved, value)
	}
}

func TestSecureMemoryVault_TokenNotFound(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()

	_, err := vault.Retrieve(ctx, "nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func TestSecureMemoryVault_TTL(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()
	token := "tok_expiring"
	value := "4532015112830366"

	// Store with very short TTL
	err := vault.Store(ctx, token, value, StoreOptions{
		TTL: 1 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	_, err = vault.Retrieve(ctx, token)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestSecureMemoryVault_Delete(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()
	token := "tok_delete"
	value := "4532015112830366"

	// Store
	vault.Store(ctx, token, value, StoreOptions{})

	// Delete
	err := vault.Delete(ctx, token)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should not exist
	if vault.Exists(ctx, token) {
		t.Error("Token should not exist after delete")
	}
}

func TestSecureMemoryVault_Exists(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()
	token := "tok_exists"

	// Should not exist initially
	if vault.Exists(ctx, token) {
		t.Error("Token should not exist initially")
	}

	// Store
	vault.Store(ctx, token, "value", StoreOptions{})

	// Should exist now
	if !vault.Exists(ctx, token) {
		t.Error("Token should exist after store")
	}
}

func TestSecureMemoryVault_KeyRotation(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()

	// Store multiple values
	tokens := map[string]string{
		"tok_1": "4532015112830366",
		"tok_2": "5425233430109903",
		"tok_3": "374245455400126",
	}

	for token, value := range tokens {
		vault.Store(ctx, token, value, StoreOptions{})
	}

	// Rotate key
	err := vault.RotateKey(ctx)
	if err != nil {
		t.Fatalf("Key rotation failed: %v", err)
	}

	// Verify all values can still be retrieved
	for token, expectedValue := range tokens {
		retrieved, err := vault.Retrieve(ctx, token)
		if err != nil {
			t.Errorf("Failed to retrieve %s after rotation: %v", token, err)
			continue
		}
		if retrieved != expectedValue {
			t.Errorf("Value mismatch for %s after rotation: got %s, want %s", token, retrieved, expectedValue)
		}
	}
}

func TestSecureMemoryVault_Stats(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()

	// Store some values
	vault.Store(ctx, "tok_1", "value1", StoreOptions{})
	vault.Store(ctx, "tok_2", "value2", StoreOptions{})

	// Access one value
	vault.Retrieve(ctx, "tok_1")
	vault.Retrieve(ctx, "tok_1")

	stats, err := vault.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.TotalEntries)
	}

	if stats.TotalAccesses != 2 {
		t.Errorf("Expected 2 accesses, got %d", stats.TotalAccesses)
	}
}

func TestSecureMemoryVault_Cleanup(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()

	// Store with short TTL
	vault.Store(ctx, "tok_expired", "value", StoreOptions{TTL: 1 * time.Millisecond})
	// Store without TTL (permanent)
	vault.Store(ctx, "tok_permanent", "value", StoreOptions{})

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Run cleanup
	err := vault.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Expired should be gone
	if vault.Exists(ctx, "tok_expired") {
		t.Error("Expired token should be cleaned up")
	}

	// Permanent should still exist
	if !vault.Exists(ctx, "tok_permanent") {
		t.Error("Permanent token should still exist")
	}
}

func TestSecureMemoryVault_AuditLog(t *testing.T) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})

	ctx := context.Background()

	// Perform operations
	vault.Store(ctx, "tok_audit", "value", StoreOptions{})
	vault.Retrieve(ctx, "tok_audit")
	vault.Delete(ctx, "tok_audit")

	// Check audit log
	auditLog := vault.GetAuditLog()

	if len(auditLog) < 3 {
		t.Errorf("Expected at least 3 audit entries, got %d", len(auditLog))
	}

	// Verify operations are logged
	operations := make(map[string]bool)
	for _, entry := range auditLog {
		operations[entry.Operation] = true
	}

	if !operations["store"] {
		t.Error("Store operation not logged")
	}
	if !operations["retrieve"] {
		t.Error("Retrieve operation not logged")
	}
	if !operations["delete"] {
		t.Error("Delete operation not logged")
	}
}

func TestSecureTokenizer_RandomToken(t *testing.T) {
	tokenizer := NewSecureTokenizer([]byte("test-hmac-key"))

	token1 := tokenizer.GenerateRandomToken()
	token2 := tokenizer.GenerateRandomToken()

	if token1 == token2 {
		t.Error("Random tokens should be unique")
	}

	if len(token1) < 10 {
		t.Error("Token should have reasonable length")
	}

	if token1[:4] != "tok_" {
		t.Error("Token should start with tok_ prefix")
	}
}

func TestSecureTokenizer_DeterministicToken(t *testing.T) {
	tokenizer := NewSecureTokenizer([]byte("test-hmac-key"))

	value := "4532015112830366"

	token1 := tokenizer.GenerateDeterministicToken(value)
	token2 := tokenizer.GenerateDeterministicToken(value)

	// Same input should produce same token
	if token1 != token2 {
		t.Error("Deterministic tokens should be identical for same input")
	}

	// Different input should produce different token
	token3 := tokenizer.GenerateDeterministicToken("different")
	if token1 == token3 {
		t.Error("Different inputs should produce different tokens")
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tok_abcdefghijklmnop", "tok_abcd****"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "12345678****"},
	}

	for _, tt := range tests {
		result := maskToken(tt.input)
		if result != tt.expected {
			t.Errorf("maskToken(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// Benchmark tests

func BenchmarkAESGCMEncrypt(b *testing.B) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)
	plaintext := []byte("4532015112830366")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = encryptor.Encrypt(plaintext)
	}
}

func BenchmarkAESGCMDecrypt(b *testing.B) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)
	plaintext := []byte("4532015112830366")
	ciphertext, nonce, _ := encryptor.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(ciphertext, nonce)
	}
}

func BenchmarkVaultStoreRetrieve(b *testing.B) {
	config := DefaultVaultConfig()
	vault, _ := NewSecureMemoryVault(config, &testLogger{})
	ctx := context.Background()
	value := "4532015112830366"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := "tok_bench"
		_ = vault.Store(ctx, token, value, StoreOptions{})
		_, _ = vault.Retrieve(ctx, token)
	}
}
