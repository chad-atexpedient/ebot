// Package finance provides immutable financial transaction audit trails
package finance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AuditTrail manages immutable financial transaction records
type AuditTrail interface {
	// Transaction recording
	RecordTransaction(ctx context.Context, tx *FinancialTransaction) error
	GetTransaction(ctx context.Context, txID string) (*FinancialTransaction, error)
	ListTransactions(ctx context.Context, filters TransactionFilters) ([]*FinancialTransaction, error)
	
	// Integrity verification
	VerifyIntegrity(ctx context.Context, txID string) (bool, error)
	VerifyChain(ctx context.Context) (bool, []string, error)
	
	// Retention and archival
	ArchiveTransactions(ctx context.Context, olderThan time.Time) error
	GetRetentionPolicy(ctx context.Context) (*RetentionPolicy, error)
}

type FinancialTransaction struct {
	ID string
	Timestamp time.Time
	Type string
	Amount float64
	Currency string
	FromAccount string
	ToAccount string
	Description string
	UserID string
	Metadata map[string]string
	
	// Blockchain-like chaining for immutability
	PreviousHash string
	Hash string
	
	// Retention
	RetentionYears int
	Archived bool
}

type TransactionFilters struct {
	StartDate time.Time
	EndDate time.Time
	Account string
	Type string
	MinAmount float64
	MaxAmount float64
	UserID string
}

type RetentionPolicy struct {
	DefaultYears int
	TransactionTypes map[string]int // Type-specific retention
	ArchiveAfterDays int
	PurgeAfterYears int
}

// auditTrailManager implements AuditTrail
type auditTrailManager struct {
	mu sync.RWMutex
	transactions []*FinancialTransaction
	txIndex map[string]*FinancialTransaction
	retentionPolicy *RetentionPolicy
	logger Logger
}

func NewAuditTrail(logger Logger) AuditTrail {
	return &auditTrailManager{
		transactions: []*FinancialTransaction{},
		txIndex: make(map[string]*FinancialTransaction),
		retentionPolicy: &RetentionPolicy{
			DefaultYears: 7, // Standard financial record retention
			TransactionTypes: map[string]int{
				"tax_related": 7,
				"payroll": 7,
				"contract": 10,
				"asset": 10,
			},
			ArchiveAfterDays: 90,
			PurgeAfterYears: 7,
		},
		logger: logger,
	}
}

func (a *auditTrailManager) RecordTransaction(ctx context.Context, tx *FinancialTransaction) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Get previous transaction hash for chaining
	if len(a.transactions) > 0 {
		last := a.transactions[len(a.transactions)-1]
		tx.PreviousHash = last.Hash
	} else {
		tx.PreviousHash = "genesis"
	}
	
	// Calculate hash for this transaction
	tx.Hash = a.calculateHash(tx)
	
	// Set retention based on type
	if years, ok := a.retentionPolicy.TransactionTypes[tx.Type]; ok {
		tx.RetentionYears = years
	} else {
		tx.RetentionYears = a.retentionPolicy.DefaultYears
	}
	
	// Store immutably
	a.transactions = append(a.transactions, tx)
	a.txIndex[tx.ID] = tx
	
	a.logger.Info("Transaction recorded in audit trail",
		"id", tx.ID,
		"type", tx.Type,
		"amount", tx.Amount,
		"hash", tx.Hash[:16])
	
	return nil
}

func (a *auditTrailManager) calculateHash(tx *FinancialTransaction) string {
	// Create deterministic hash from transaction data
	data := fmt.Sprintf("%s|%s|%s|%.2f|%s|%s|%s|%s|%s",
		tx.ID,
		tx.Timestamp.Format(time.RFC3339Nano),
		tx.Type,
		tx.Amount,
		tx.Currency,
		tx.FromAccount,
		tx.ToAccount,
		tx.UserID,
		tx.PreviousHash,
	)
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (a *auditTrailManager) GetTransaction(ctx context.Context, txID string) (*FinancialTransaction, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	tx, exists := a.txIndex[txID]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}
	
	return tx, nil
}

func (a *auditTrailManager) ListTransactions(ctx context.Context, filters TransactionFilters) ([]*FinancialTransaction, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	var result []*FinancialTransaction
	
	for _, tx := range a.transactions {
		if a.matchesFilters(tx, filters) {
			result = append(result, tx)
		}
	}
	
	return result, nil
}

func (a *auditTrailManager) matchesFilters(tx *FinancialTransaction, filters TransactionFilters) bool {
	if !filters.StartDate.IsZero() && tx.Timestamp.Before(filters.StartDate) {
		return false
	}
	if !filters.EndDate.IsZero() && tx.Timestamp.After(filters.EndDate) {
		return false
	}
	if filters.Account != "" && tx.FromAccount != filters.Account && tx.ToAccount != filters.Account {
		return false
	}
	if filters.Type != "" && tx.Type != filters.Type {
		return false
	}
	if tx.Amount < filters.MinAmount {
		return false
	}
	if filters.MaxAmount > 0 && tx.Amount > filters.MaxAmount {
		return false
	}
	if filters.UserID != "" && tx.UserID != filters.UserID {
		return false
	}
	return true
}

func (a *auditTrailManager) VerifyIntegrity(ctx context.Context, txID string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	tx, exists := a.txIndex[txID]
	if !exists {
		return false, fmt.Errorf("transaction not found: %s", txID)
	}
	
	// Recalculate hash
	expectedHash := a.calculateHash(tx)
	
	if tx.Hash != expectedHash {
		a.logger.Error("Transaction integrity violation detected", "txID", txID)
		return false, nil
	}
	
	return true, nil
}

func (a *auditTrailManager) VerifyChain(ctx context.Context) (bool, []string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	var violations []string
	
	for i, tx := range a.transactions {
		// Verify hash
		valid, err := a.VerifyIntegrity(ctx, tx.ID)
		if err != nil {
			return false, violations, err
		}
		if !valid {
			violations = append(violations, fmt.Sprintf("Transaction %s: hash mismatch", tx.ID))
		}
		
		// Verify chain linkage
		if i > 0 {
			prev := a.transactions[i-1]
			if tx.PreviousHash != prev.Hash {
				violations = append(violations, fmt.Sprintf("Transaction %s: chain broken", tx.ID))
			}
		}
	}
	
	if len(violations) > 0 {
		a.logger.Error("Audit trail integrity violations detected", "count", len(violations))
		return false, violations, nil
	}
	
	return true, nil, nil
}

func (a *auditTrailManager) ArchiveTransactions(ctx context.Context, olderThan time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	archivedCount := 0
	
	for _, tx := range a.transactions {
		if tx.Timestamp.Before(olderThan) && !tx.Archived {
			tx.Archived = true
			archivedCount++
			// In production: move to cold storage (S3 Glacier, etc.)
		}
	}
	
	a.logger.Info("Transactions archived", "count", archivedCount)
	
	return nil
}

func (a *auditTrailManager) GetRetentionPolicy(ctx context.Context) (*RetentionPolicy, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	return a.retentionPolicy, nil
}
