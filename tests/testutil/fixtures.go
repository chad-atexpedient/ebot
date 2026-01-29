// Package testutil provides testing utilities and fixtures for ebot tests
package testutil

import (
	"context"
	"time"
)

// Common test fixtures

// TestUser represents a standard test user
type TestUser struct {
	ID       string
	Email    string
	Name     string
	Groups   []string
	Roles    []string
	TenantID string
}

// NewTestUser creates a standard test user
func NewTestUser() *TestUser {
	return &TestUser{
		ID:       "user-test-123",
		Email:    "test@example.com",
		Name:     "Test User",
		Groups:   []string{"developers", "testers"},
		Roles:    []string{"user"},
		TenantID: "tenant-test-123",
	}
}

// NewTestAdmin creates a test admin user
func NewTestAdmin() *TestUser {
	return &TestUser{
		ID:       "user-admin-123",
		Email:    "admin@example.com",
		Name:     "Test Admin",
		Groups:   []string{"admins"},
		Roles:    []string{"admin", "user"},
		TenantID: "tenant-test-123",
	}
}

// TestWorkspace represents a standard test workspace
type TestWorkspace struct {
	ID       string
	Name     string
	TenantID string
	OwnerID  string
}

// NewTestWorkspace creates a standard test workspace
func NewTestWorkspace() *TestWorkspace {
	return &TestWorkspace{
		ID:       "workspace-test-123",
		Name:     "Test Workspace",
		TenantID: "tenant-test-123",
		OwnerID:  "user-test-123",
	}
}

// TestMCPServer represents a standard test MCP server
type TestMCPServer struct {
	ID          string
	Name        string
	WorkspaceID string
	Status      string
	Tools       []string
}

// NewTestMCPServer creates a standard test MCP server
func NewTestMCPServer() *TestMCPServer {
	return &TestMCPServer{
		ID:          "mcp-test-123",
		Name:        "Test MCP Server",
		WorkspaceID: "workspace-test-123",
		Status:      "running",
		Tools:       []string{"read_file", "write_file", "list_dir"},
	}
}

// TestThread represents a standard test conversation thread
type TestThread struct {
	ID          string
	Title       string
	WorkspaceID string
	UserID      string
	AgentID     string
}

// NewTestThread creates a standard test thread
func NewTestThread() *TestThread {
	return &TestThread{
		ID:          "thread-test-123",
		Title:       "Test Conversation",
		WorkspaceID: "workspace-test-123",
		UserID:      "user-test-123",
		AgentID:     "agent-test-123",
	}
}

// Test PHI Data

// PHITestCases contains test cases for PHI detection
var PHITestCases = []struct {
	Name        string
	Input       string
	ContainsPHI bool
	PHITypes    []string
}{
	{
		Name:        "SSN with dashes",
		Input:       "Patient SSN: 123-45-6789",
		ContainsPHI: true,
		PHITypes:    []string{"ssn"},
	},
	{
		Name:        "SSN without dashes",
		Input:       "SSN is 123456789",
		ContainsPHI: true,
		PHITypes:    []string{"ssn"},
	},
	{
		Name:        "Email address",
		Input:       "Contact: patient@hospital.com",
		ContainsPHI: true,
		PHITypes:    []string{"email"},
	},
	{
		Name:        "Phone number",
		Input:       "Call 555-123-4567 for appointment",
		ContainsPHI: true,
		PHITypes:    []string{"phone_number"},
	},
	{
		Name:        "Date of birth",
		Input:       "DOB: 01/15/1990",
		ContainsPHI: true,
		PHITypes:    []string{"date_of_birth"},
	},
	{
		Name:        "Medical record number",
		Input:       "MRN: ABC123456",
		ContainsPHI: true,
		PHITypes:    []string{"medical_record_number"},
	},
	{
		Name:        "No PHI",
		Input:       "The patient is doing well.",
		ContainsPHI: false,
		PHITypes:    nil,
	},
	{
		Name:        "Multiple PHI types",
		Input:       "Patient: Dr. John Smith, SSN: 123-45-6789, Email: john@hospital.com",
		ContainsPHI: true,
		PHITypes:    []string{"ssn", "email", "name"},
	},
}

// Test Card Data

// CardTestCases contains test cases for PCI card data detection
var CardTestCases = []struct {
	Name           string
	Input          string
	ContainsCard   bool
	IsValidLuhn    bool
	CardType       string
}{
	{
		Name:         "Valid Visa",
		Input:        "4111 1111 1111 1111",
		ContainsCard: true,
		IsValidLuhn:  true,
		CardType:     "Visa",
	},
	{
		Name:         "Valid Mastercard",
		Input:        "5500 0000 0000 0004",
		ContainsCard: true,
		IsValidLuhn:  true,
		CardType:     "Mastercard",
	},
	{
		Name:         "Valid Amex",
		Input:        "3400 0000 0000 009",
		ContainsCard: true,
		IsValidLuhn:  true,
		CardType:     "American Express",
	},
	{
		Name:         "Invalid Luhn",
		Input:        "4111 1111 1111 1112",
		ContainsCard: true,
		IsValidLuhn:  false,
		CardType:     "",
	},
	{
		Name:         "No card data",
		Input:        "Total: $19.99",
		ContainsCard: false,
		IsValidLuhn:  false,
		CardType:     "",
	},
	{
		Name:         "CVV with context",
		Input:        "CVV: 123",
		ContainsCard: true,
		IsValidLuhn:  false,
		CardType:     "cvv",
	},
}

// Test ABAC Policies

// TestPolicy represents a test ABAC policy
type TestPolicy struct {
	ID        string
	Name      string
	Effect    string
	Condition string
	Enabled   bool
}

// ABACTestPolicies contains test ABAC policies
var ABACTestPolicies = []TestPolicy{
	{
		ID:        "policy-allow-owner",
		Name:      "Allow owners",
		Effect:    "allow",
		Condition: `resource["owner"] == subject["id"]`,
		Enabled:   true,
	},
	{
		ID:        "policy-allow-admin",
		Name:      "Allow admins",
		Effect:    "allow",
		Condition: `"admin" in subject["roles"]`,
		Enabled:   true,
	},
	{
		ID:        "policy-deny-external",
		Name:      "Deny external",
		Effect:    "deny",
		Condition: `subject["type"] == "external"`,
		Enabled:   true,
	},
}

// Utility functions

// TestContext creates a context with timeout for tests
func TestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// TestContextWithValue creates a test context with a value
func TestContextWithValue(key, value interface{}) context.Context {
	return context.WithValue(context.Background(), key, value)
}

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr returns a pointer to an int64
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr returns a pointer to a bool
func BoolPtr(b bool) *bool {
	return &b
}
