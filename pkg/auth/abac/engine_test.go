package abac

import (
	"context"
	"testing"
	"time"
)

// mockLogger implements logger.Logger for testing
type mockLogger struct {
	messages []string
}

func (l *mockLogger) Info(msg string, args ...interface{})  { l.messages = append(l.messages, msg) }
func (l *mockLogger) Error(msg string, args ...interface{}) { l.messages = append(l.messages, msg) }
func (l *mockLogger) Warn(msg string, args ...interface{})  { l.messages = append(l.messages, msg) }
func (l *mockLogger) Debug(msg string, args ...interface{}) { l.messages = append(l.messages, msg) }

func newMockLogger() *mockLogger {
	return &mockLogger{messages: []string{}}
}

func TestNewABACEngine(t *testing.T) {
	logger := newMockLogger()
	engine, err := NewABACEngine(logger)
	if err != nil {
		t.Fatalf("NewABACEngine failed: %v", err)
	}
	if engine == nil {
		t.Fatal("NewABACEngine returned nil")
	}
}

func TestAddPolicy(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	policy := &Policy{
		ID:        "policy-1",
		Name:      "Test Policy",
		Effect:    EffectAllow,
		Condition: `subject.id != ""`,
		Enabled:   true,
	}

	err := engine.AddPolicy(policy)
	if err != nil {
		t.Fatalf("AddPolicy failed: %v", err)
	}

	// Verify policy was added
	retrieved, err := engine.GetPolicy("policy-1")
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}
	if retrieved.Name != "Test Policy" {
		t.Errorf("Expected policy name 'Test Policy', got '%s'", retrieved.Name)
	}
}

func TestAddPolicy_InvalidID(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	policy := &Policy{
		ID:   "",
		Name: "No ID Policy",
	}

	err := engine.AddPolicy(policy)
	if err == nil {
		t.Error("Expected error for policy without ID")
	}
}

func TestAddPolicy_InvalidCondition(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	policy := &Policy{
		ID:        "policy-invalid",
		Name:      "Invalid Condition",
		Condition: "this is not valid CEL !!!", // Invalid CEL
		Enabled:   true,
	}

	err := engine.AddPolicy(policy)
	if err == nil {
		t.Error("Expected error for invalid CEL condition")
	}
}

func TestAddPolicy_Duplicate(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	policy := &Policy{
		ID:      "policy-dup",
		Name:    "Duplicate Policy",
		Enabled: true,
	}

	_ = engine.AddPolicy(policy)
	err := engine.AddPolicy(policy)
	if err == nil {
		t.Error("Expected error for duplicate policy ID")
	}
}

func TestRemovePolicy(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	policy := &Policy{
		ID:      "policy-remove",
		Name:    "To Be Removed",
		Enabled: true,
	}

	_ = engine.AddPolicy(policy)

	err := engine.RemovePolicy("policy-remove")
	if err != nil {
		t.Fatalf("RemovePolicy failed: %v", err)
	}

	_, err = engine.GetPolicy("policy-remove")
	if err == nil {
		t.Error("Expected error when getting removed policy")
	}
}

func TestRemovePolicy_NotFound(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	err := engine.RemovePolicy("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent policy")
	}
}

func TestListPolicies(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add multiple policies
	for i := 1; i <= 3; i++ {
		policy := &Policy{
			ID:      "policy-" + string(rune('0'+i)),
			Name:    "Policy " + string(rune('0'+i)),
			Enabled: true,
		}
		_ = engine.AddPolicy(policy)
	}

	policies, err := engine.ListPolicies()
	if err != nil {
		t.Fatalf("ListPolicies failed: %v", err)
	}

	if len(policies) != 3 {
		t.Errorf("Expected 3 policies, got %d", len(policies))
	}
}

func TestUpdatePolicy(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	policy := &Policy{
		ID:      "policy-update",
		Name:    "Original Name",
		Enabled: true,
	}
	_ = engine.AddPolicy(policy)

	updatedPolicy := &Policy{
		ID:      "policy-update",
		Name:    "Updated Name",
		Enabled: false,
	}

	err := engine.UpdatePolicy(updatedPolicy)
	if err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}

	retrieved, _ := engine.GetPolicy("policy-update")
	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got '%s'", retrieved.Name)
	}
	if retrieved.Enabled {
		t.Error("Expected policy to be disabled")
	}
}

func TestEvaluate_AllowPolicy(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add allow policy
	policy := &Policy{
		ID:        "allow-all",
		Name:      "Allow All",
		Effect:    EffectAllow,
		Condition: "true",
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	request := &AuthorizationRequest{
		Subject: &Subject{
			ID:     "user-1",
			Type:   "user",
			Groups: []string{"developers"},
			Roles:  []string{"developer"},
		},
		Resource: &Resource{
			ID:    "resource-1",
			Type:  "document",
			Owner: "user-1",
		},
		Action:  "read",
		Context: map[string]interface{}{},
	}

	decision, err := engine.Evaluate(context.Background(), request)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if !decision.Allowed {
		t.Error("Expected request to be allowed")
	}
}

func TestEvaluate_DenyPolicy(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add deny policy
	policy := &Policy{
		ID:        "deny-all",
		Name:      "Deny All",
		Effect:    EffectDeny,
		Condition: "true",
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	request := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "user-1",
			Type: "user",
		},
		Resource: &Resource{
			ID:   "resource-1",
			Type: "document",
		},
		Action: "delete",
	}

	decision, err := engine.Evaluate(context.Background(), request)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Allowed {
		t.Error("Expected request to be denied")
	}
}

func TestEvaluate_DenyTakesPrecedence(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add allow policy
	allowPolicy := &Policy{
		ID:        "allow-read",
		Name:      "Allow Read",
		Effect:    EffectAllow,
		Condition: "true",
		Enabled:   true,
		Priority:  1,
	}
	_ = engine.AddPolicy(allowPolicy)

	// Add deny policy
	denyPolicy := &Policy{
		ID:        "deny-specific",
		Name:      "Deny Specific",
		Effect:    EffectDeny,
		Condition: "true",
		Enabled:   true,
		Priority:  2,
	}
	_ = engine.AddPolicy(denyPolicy)

	request := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "user-1",
			Type: "user",
		},
		Resource: &Resource{
			ID:   "resource-1",
			Type: "document",
		},
		Action: "read",
	}

	decision, err := engine.Evaluate(context.Background(), request)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Deny should take precedence
	if decision.Allowed {
		t.Error("Expected deny to take precedence over allow")
	}
}

func TestEvaluate_NoMatchingPolicies(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add policy with condition that won't match
	policy := &Policy{
		ID:        "specific-policy",
		Name:      "Specific Policy",
		Effect:    EffectAllow,
		Condition: `action == "specific-action"`,
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	request := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "user-1",
			Type: "user",
		},
		Resource: &Resource{
			ID:   "resource-1",
			Type: "document",
		},
		Action: "different-action",
	}

	decision, err := engine.Evaluate(context.Background(), request)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Default deny when no policies match
	if decision.Allowed {
		t.Error("Expected default deny when no policies match")
	}
}

func TestEvaluate_DisabledPolicy(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add disabled policy
	policy := &Policy{
		ID:        "disabled-policy",
		Name:      "Disabled Policy",
		Effect:    EffectAllow,
		Condition: "true",
		Enabled:   false, // Disabled
	}
	_ = engine.AddPolicy(policy)

	request := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "user-1",
			Type: "user",
		},
		Resource: &Resource{
			ID:   "resource-1",
			Type: "document",
		},
		Action: "read",
	}

	decision, err := engine.Evaluate(context.Background(), request)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Disabled policy should not be evaluated
	if decision.Allowed {
		t.Error("Disabled policy should not be evaluated")
	}
}

func TestEvaluate_SubjectCondition(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Policy that checks subject ID
	policy := &Policy{
		ID:        "check-subject",
		Name:      "Check Subject",
		Effect:    EffectAllow,
		Condition: `subject.id == "admin-user"`,
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	// Request from admin user
	adminRequest := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "admin-user",
			Type: "user",
		},
		Resource: &Resource{
			ID:   "resource-1",
			Type: "document",
		},
		Action: "admin",
	}

	decision, _ := engine.Evaluate(context.Background(), adminRequest)
	if !decision.Allowed {
		t.Error("Expected admin user to be allowed")
	}

	// Request from regular user
	regularRequest := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "regular-user",
			Type: "user",
		},
		Resource: &Resource{
			ID:   "resource-1",
			Type: "document",
		},
		Action: "admin",
	}

	decision, _ = engine.Evaluate(context.Background(), regularRequest)
	if decision.Allowed {
		t.Error("Expected regular user to be denied")
	}
}

func TestEvaluate_ResourceOwnerCondition(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Policy that checks if user is owner
	policy := &Policy{
		ID:        "owner-access",
		Name:      "Owner Access",
		Effect:    EffectAllow,
		Condition: `subject.id == resource.owner`,
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	// Request from owner
	ownerRequest := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "user-123",
			Type: "user",
		},
		Resource: &Resource{
			ID:    "resource-1",
			Type:  "document",
			Owner: "user-123",
		},
		Action: "edit",
	}

	decision, _ := engine.Evaluate(context.Background(), ownerRequest)
	if !decision.Allowed {
		t.Error("Expected owner to be allowed")
	}

	// Request from non-owner
	nonOwnerRequest := &AuthorizationRequest{
		Subject: &Subject{
			ID:   "user-456",
			Type: "user",
		},
		Resource: &Resource{
			ID:    "resource-1",
			Type:  "document",
			Owner: "user-123",
		},
		Action: "edit",
	}

	decision, _ = engine.Evaluate(context.Background(), nonOwnerRequest)
	if decision.Allowed {
		t.Error("Expected non-owner to be denied")
	}
}

func TestEvaluate_ActionCondition(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Policy that only allows read actions
	policy := &Policy{
		ID:        "read-only",
		Name:      "Read Only",
		Effect:    EffectAllow,
		Condition: `action == "read"`,
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	// Read action
	readRequest := &AuthorizationRequest{
		Subject:  &Subject{ID: "user-1", Type: "user"},
		Resource: &Resource{ID: "resource-1", Type: "document"},
		Action:   "read",
	}

	decision, _ := engine.Evaluate(context.Background(), readRequest)
	if !decision.Allowed {
		t.Error("Expected read action to be allowed")
	}

	// Write action
	writeRequest := &AuthorizationRequest{
		Subject:  &Subject{ID: "user-1", Type: "user"},
		Resource: &Resource{ID: "resource-1", Type: "document"},
		Action:   "write",
	}

	decision, _ = engine.Evaluate(context.Background(), writeRequest)
	if decision.Allowed {
		t.Error("Expected write action to be denied")
	}
}

func TestCELProgramCache(t *testing.T) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add policy with condition
	policy := &Policy{
		ID:        "cached-policy",
		Name:      "Cached Policy",
		Effect:    EffectAllow,
		Condition: `subject.id != ""`,
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	request := &AuthorizationRequest{
		Subject:  &Subject{ID: "user-1", Type: "user"},
		Resource: &Resource{ID: "resource-1", Type: "document"},
		Action:   "read",
	}

	// Evaluate multiple times to test caching
	for i := 0; i < 100; i++ {
		_, err := engine.Evaluate(context.Background(), request)
		if err != nil {
			t.Fatalf("Evaluate failed on iteration %d: %v", i, err)
		}
	}

	// Get cache stats (if available)
	impl := engine.(*abacEngine)
	stats := impl.programCache.Stats()
	
	// After 100 evaluations with 1 unique condition, hit rate should be high
	if stats.Hits < 99 {
		t.Errorf("Expected high cache hit rate, got %d hits", stats.Hits)
	}
}

func BenchmarkEvaluate(b *testing.B) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add a policy
	policy := &Policy{
		ID:        "benchmark-policy",
		Name:      "Benchmark Policy",
		Effect:    EffectAllow,
		Condition: `subject.id != "" && action == "read"`,
		Enabled:   true,
	}
	_ = engine.AddPolicy(policy)

	request := &AuthorizationRequest{
		Subject: &Subject{
			ID:     "user-1",
			Type:   "user",
			Groups: []string{"developers"},
			Roles:  []string{"developer"},
		},
		Resource: &Resource{
			ID:    "resource-1",
			Type:  "document",
			Owner: "user-2",
		},
		Action:    "read",
		Context:   map[string]interface{}{},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Evaluate(context.Background(), request)
	}
}

func BenchmarkEvaluate_ManyPolicies(b *testing.B) {
	logger := newMockLogger()
	engine, _ := NewABACEngine(logger)

	// Add many policies
	for i := 0; i < 100; i++ {
		policy := &Policy{
			ID:        "policy-" + string(rune(i)),
			Name:      "Policy " + string(rune(i)),
			Effect:    EffectAllow,
			Condition: `subject.id != ""`,
			Enabled:   true,
		}
		_ = engine.AddPolicy(policy)
	}

	request := &AuthorizationRequest{
		Subject:  &Subject{ID: "user-1", Type: "user"},
		Resource: &Resource{ID: "resource-1", Type: "document"},
		Action:   "read",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Evaluate(context.Background(), request)
	}
}
