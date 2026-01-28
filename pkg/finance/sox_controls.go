// Package finance provides Sarbanes-Oxley (SOX) compliance controls
package finance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SOXControls manages SOX compliance requirements
type SOXControls interface {
	// Change management
	ValidateChange(ctx context.Context, change *FinancialChange) error
	RecordChange(ctx context.Context, change *FinancialChange) error
	GetChangeHistory(ctx context.Context, filters ChangeFilters) ([]*FinancialChange, error)
	
	// Separation of duties
	EnforceSeparationOfDuties(ctx context.Context, userID, action string) error
	DefineRole(ctx context.Context, role *SOXRole) error
	
	// Audit and reporting
	GenerateAuditReport(ctx context.Context, period TimePeriod) (*SOXAuditReport, error)
	LockPeriod(ctx context.Context, period TimePeriod) error
}

type FinancialChange struct {
	ID string
	Type string // journal_entry, account_adjustment, close_period, etc.
	Description string
	Amount float64
	Account string
	InitiatorID string
	InitiatorName string
	ApproverID string
	ApproverName string
	Timestamp time.Time
	Approved bool
	Evidence []string
	Reason string
}

type ChangeFilters struct {
	StartDate time.Time
	EndDate time.Time
	InitiatorID string
	Type string
	MinAmount float64
}

type SOXRole struct {
	ID string
	Name string
	Permissions []string
	IncompatibleRoles []string // For separation of duties
	Users []string
}

type TimePeriod struct {
	StartDate time.Time
	EndDate time.Time
	FiscalYear int
	Quarter int
	Month int
	Closed bool
}

type SOXAuditReport struct {
	Period TimePeriod
	GeneratedAt time.Time
	TotalChanges int
	Violations []ControlViolation
	Compliance float64 // Percentage
	Recommendations []string
}

type ControlViolation struct {
	ChangeID string
	ViolationType string
	Severity string
	Description string
	Remediation string
}

// soxControlsManager implements SOXControls
type soxControlsManager struct {
	mu sync.RWMutex
	changes map[string]*FinancialChange
	roles map[string]*SOXRole
	lockedPeriods map[string]bool
	logger Logger
}

func NewSOXControls(logger Logger) SOXControls {
	return &soxControlsManager{
		changes: make(map[string]*FinancialChange),
		roles: make(map[string]*SOXRole),
		lockedPeriods: make(map[string]bool),
		logger: logger,
	}
}

func (s *soxControlsManager) ValidateChange(ctx context.Context, change *FinancialChange) error {
	// Four-eyes principle: initiator and approver must be different
	if change.InitiatorID == change.ApproverID {
		return fmt.Errorf("SOX violation: initiator and approver must be different (four-eyes principle)")
	}
	
	// Check if period is locked
	periodKey := s.getPeriodKey(change.Timestamp)
	if s.lockedPeriods[periodKey] {
		return fmt.Errorf("SOX violation: cannot modify transactions in locked period %s", periodKey)
	}
	
	// Validate evidence for significant changes
	if change.Amount >= 10000 && len(change.Evidence) == 0 {
		return fmt.Errorf("SOX violation: changes >= $10,000 require supporting evidence")
	}
	
	return nil
}

func (s *soxControlsManager) RecordChange(ctx context.Context, change *FinancialChange) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if err := s.ValidateChange(ctx, change); err != nil {
		return err
	}
	
	s.changes[change.ID] = change
	s.logger.Info("Financial change recorded", "id", change.ID, "type", change.Type, "amount", change.Amount)
	
	return nil
}

func (s *soxControlsManager) GetChangeHistory(ctx context.Context, filters ChangeFilters) ([]*FinancialChange, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*FinancialChange
	
	for _, change := range s.changes {
		if s.matchesFilters(change, filters) {
			result = append(result, change)
		}
	}
	
	return result, nil
}

func (s *soxControlsManager) matchesFilters(change *FinancialChange, filters ChangeFilters) bool {
	if !filters.StartDate.IsZero() && change.Timestamp.Before(filters.StartDate) {
		return false
	}
	if !filters.EndDate.IsZero() && change.Timestamp.After(filters.EndDate) {
		return false
	}
	if filters.InitiatorID != "" && change.InitiatorID != filters.InitiatorID {
		return false
	}
	if filters.Type != "" && change.Type != filters.Type {
		return false
	}
	if change.Amount < filters.MinAmount {
		return false
	}
	return true
}

func (s *soxControlsManager) EnforceSeparationOfDuties(ctx context.Context, userID, action string) error {
	// Check if user has conflicting roles
	userRoles := s.getUserRoles(userID)
	
	for _, role := range userRoles {
		for _, incompatible := range role.IncompatibleRoles {
			if s.userHasRole(userID, incompatible) {
				return fmt.Errorf("SOX violation: user has incompatible roles %s and %s", role.Name, incompatible)
			}
		}
	}
	
	return nil
}

func (s *soxControlsManager) DefineRole(ctx context.Context, role *SOXRole) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.roles[role.ID] = role
	s.logger.Info("SOX role defined", "role", role.Name)
	
	return nil
}

func (s *soxControlsManager) GenerateAuditReport(ctx context.Context, period TimePeriod) (*SOXAuditReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	report := &SOXAuditReport{
		Period: period,
		GeneratedAt: time.Now(),
		Violations: []ControlViolation{},
		Recommendations: []string{},
	}
	
	// Analyze all changes in period
	for _, change := range s.changes {
		if s.inPeriod(change.Timestamp, period) {
			report.TotalChanges++
			
			// Check for violations
			if change.InitiatorID == change.ApproverID {
				report.Violations = append(report.Violations, ControlViolation{
					ChangeID: change.ID,
					ViolationType: "separation_of_duties",
					Severity: "high",
					Description: "Same person initiated and approved change",
					Remediation: "Implement four-eyes principle for all changes",
				})
			}
		}
	}
	
	// Calculate compliance
	if report.TotalChanges > 0 {
		report.Compliance = float64(report.TotalChanges-len(report.Violations)) / float64(report.TotalChanges) * 100
	} else {
		report.Compliance = 100
	}
	
	s.logger.Info("SOX audit report generated", "period", period.FiscalYear, "compliance", report.Compliance)
	
	return report, nil
}

func (s *soxControlsManager) LockPeriod(ctx context.Context, period TimePeriod) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	periodKey := fmt.Sprintf("%d-Q%d", period.FiscalYear, period.Quarter)
	s.lockedPeriods[periodKey] = true
	
	s.logger.Info("Period locked for SOX compliance", "period", periodKey)
	
	return nil
}

func (s *soxControlsManager) getPeriodKey(t time.Time) string {
	year := t.Year()
	quarter := (int(t.Month()) - 1) / 3 + 1
	return fmt.Sprintf("%d-Q%d", year, quarter)
}

func (s *soxControlsManager) inPeriod(t time.Time, period TimePeriod) bool {
	return !t.Before(period.StartDate) && !t.After(period.EndDate)
}

func (s *soxControlsManager) getUserRoles(userID string) []*SOXRole {
	var roles []*SOXRole
	for _, role := range s.roles {
		for _, user := range role.Users {
			if user == userID {
				roles = append(roles, role)
			}
		}
	}
	return roles
}

func (s *soxControlsManager) userHasRole(userID, roleName string) bool {
	for _, role := range s.getUserRoles(userID) {
		if role.Name == roleName {
			return true
		}
	}
	return false
}
