package abac

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chad-atexpedient/ebot/logger"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// ABACEngine evaluates attribute-based access control policies
type ABACEngine interface {
	// Evaluate checks if a request is allowed based on policies
	Evaluate(ctx context.Context, request *AuthorizationRequest) (*AuthorizationDecision, error)
	
	// AddPolicy adds a new policy
	AddPolicy(policy *Policy) error
	
	// RemovePolicy removes a policy by ID
	RemovePolicy(policyID string) error
	
	// GetPolicy retrieves a policy by ID
	GetPolicy(policyID string) (*Policy, error)
	
	// ListPolicies lists all policies
	ListPolicies() ([]*Policy, error)
	
	// UpdatePolicy updates an existing policy
	UpdatePolicy(policy *Policy) error
}

// AuthorizationRequest represents an access control request
type AuthorizationRequest struct {
	// Subject is the entity making the request
	Subject *Subject
	
	// Resource is the target resource
	Resource *Resource
	
	// Action is the operation being performed
	Action string
	
	// Context contains additional request context
	Context map[string]interface{}
	
	// Timestamp of the request
	Timestamp time.Time
}

// Subject represents the entity making a request
type Subject struct {
	ID         string
	Type       string // "user", "service_account", "group"
	Attributes map[string]interface{}
	Groups     []string
	Roles      []string
}

// Resource represents the target resource
type Resource struct {
	ID         string
	Type       string // "mcp_server", "thread", "workspace", etc.
	Attributes map[string]interface{}
	Owner      string
	Tags       []string
}

// AuthorizationDecision represents the result of policy evaluation
type AuthorizationDecision struct {
	Allowed       bool
	Reason        string
	AppliedPolicy string
	Obligations   []Obligation
	Advice        []string
}

// Obligation represents an action that must be taken
type Obligation struct {
	ID          string
	Type        string
	Description string
	Parameters  map[string]interface{}
}

// Policy represents an ABAC policy
type Policy struct {
	ID          string
	Name        string
	Description string
	Effect      Effect
	Priority    int
	Rules       []Rule
	Condition   string // CEL expression
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Rule represents a policy rule
type Rule struct {
	ID          string
	Description string
	Condition   string // CEL expression
	Effect      Effect
}

// Effect represents the policy effect
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// abacEngine implements ABACEngine
type abacEngine struct {
	policies   map[string]*Policy
	mu         sync.RWMutex
	logger     logger.Logger
	celEnv     *cel.Env
}

// NewABACEngine creates a new ABAC engine
func NewABACEngine(log logger.Logger) (ABACEngine, error) {
	// Create CEL environment with custom declarations
	env, err := cel.NewEnv(
		cel.Declarations(
			// Subject declarations
			decls.NewVar("subject", decls.NewMapType(decls.String, decls.Any)),
			decls.NewVar("subject.id", decls.String),
			decls.NewVar("subject.type", decls.String),
			decls.NewVar("subject.groups", decls.NewListType(decls.String)),
			decls.NewVar("subject.roles", decls.NewListType(decls.String)),
			
			// Resource declarations
			decls.NewVar("resource", decls.NewMapType(decls.String, decls.Any)),
			decls.NewVar("resource.id", decls.String),
			decls.NewVar("resource.type", decls.String),
			decls.NewVar("resource.owner", decls.String),
			decls.NewVar("resource.tags", decls.NewListType(decls.String)),
			
			// Action declaration
			decls.NewVar("action", decls.String),
			
			// Context declarations
			decls.NewVar("context", decls.NewMapType(decls.String, decls.Any)),
			decls.NewVar("context.time", decls.Timestamp),
			decls.NewVar("context.ip", decls.String),
			decls.NewVar("context.location", decls.String),
			
			// Helper functions
			decls.NewFunction("has_role",
				decls.NewOverload("has_role_string",
					[]*expr.Type{decls.String},
					decls.Bool)),
			decls.NewFunction("has_group",
				decls.NewOverload("has_group_string",
					[]*expr.Type{decls.String},
					decls.Bool)),
			decls.NewFunction("is_owner",
				decls.NewOverload("is_owner",
					[]*expr.Type{},
					decls.Bool)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}
	
	return &abacEngine{
		policies: make(map[string]*Policy),
		logger:   log,
		celEnv:   env,
	}, nil
}

// Evaluate checks if a request is allowed based on policies
func (e *abacEngine) Evaluate(ctx context.Context, request *AuthorizationRequest) (*AuthorizationDecision, error) {
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}
	
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// Build evaluation context
	evalContext := e.buildEvaluationContext(request)
	
	// Track applied policies and their effects
	var allowPolicies []*Policy
	var denyPolicies []*Policy
	
	// Evaluate all enabled policies
	for _, policy := range e.policies {
		if !policy.Enabled {
			continue
		}
		
		// Evaluate policy condition
		allowed, err := e.evaluatePolicy(policy, evalContext)
		if err != nil {
			e.logger.Error("Failed to evaluate policy",
				"policy_id", policy.ID,
				"error", err)
			continue
		}
		
		if allowed {
			if policy.Effect == EffectAllow {
				allowPolicies = append(allowPolicies, policy)
			} else {
				denyPolicies = append(denyPolicies, policy)
			}
		}
	}
	
	// Decision logic: Deny takes precedence over allow
	decision := &AuthorizationDecision{
		Allowed: false,
		Reason:  "No matching policies",
	}
	
	if len(denyPolicies) > 0 {
		// Deny if any deny policy matches
		decision.Allowed = false
		decision.Reason = fmt.Sprintf("Denied by policy: %s", denyPolicies[0].Name)
		decision.AppliedPolicy = denyPolicies[0].ID
	} else if len(allowPolicies) > 0 {
		// Allow if any allow policy matches and no deny
		decision.Allowed = true
		decision.Reason = fmt.Sprintf("Allowed by policy: %s", allowPolicies[0].Name)
		decision.AppliedPolicy = allowPolicies[0].ID
	}
	
	e.logger.Info("Policy evaluation complete",
		"subject", request.Subject.ID,
		"resource", request.Resource.ID,
		"action", request.Action,
		"allowed", decision.Allowed,
		"reason", decision.Reason)
	
	return decision, nil
}

// AddPolicy adds a new policy
func (e *abacEngine) AddPolicy(policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	
	// Validate CEL expression
	if policy.Condition != "" {
		if err := e.validateCondition(policy.Condition); err != nil {
			return fmt.Errorf("invalid policy condition: %w", err)
		}
	}
	
	// Validate rules
	for _, rule := range policy.Rules {
		if rule.Condition != "" {
			if err := e.validateCondition(rule.Condition); err != nil {
				return fmt.Errorf("invalid rule condition in rule %s: %w", rule.ID, err)
			}
		}
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.policies[policy.ID]; exists {
		return fmt.Errorf("policy %s already exists", policy.ID)
	}
	
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	e.policies[policy.ID] = policy
	
	e.logger.Info("Added policy",
		"policy_id", policy.ID,
		"name", policy.Name)
	
	return nil
}

// RemovePolicy removes a policy by ID
func (e *abacEngine) RemovePolicy(policyID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.policies[policyID]; !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}
	
	delete(e.policies, policyID)
	
	e.logger.Info("Removed policy", "policy_id", policyID)
	
	return nil
}

// GetPolicy retrieves a policy by ID
func (e *abacEngine) GetPolicy(policyID string) (*Policy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	policy, exists := e.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy %s not found", policyID)
	}
	
	return policy, nil
}

// ListPolicies lists all policies
func (e *abacEngine) ListPolicies() ([]*Policy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	policies := make([]*Policy, 0, len(e.policies))
	for _, policy := range e.policies {
		policies = append(policies, policy)
	}
	
	return policies, nil
}

// UpdatePolicy updates an existing policy
func (e *abacEngine) UpdatePolicy(policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}
	
	// Validate CEL expression
	if policy.Condition != "" {
		if err := e.validateCondition(policy.Condition); err != nil {
			return fmt.Errorf("invalid policy condition: %w", err)
		}
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.policies[policy.ID]; !exists {
		return fmt.Errorf("policy %s not found", policy.ID)
	}
	
	policy.UpdatedAt = time.Now()
	e.policies[policy.ID] = policy
	
	e.logger.Info("Updated policy",
		"policy_id", policy.ID,
		"name", policy.Name)
	
	return nil
}

// Helper methods

func (e *abacEngine) buildEvaluationContext(request *AuthorizationRequest) map[string]interface{} {
	return map[string]interface{}{
		"subject": map[string]interface{}{
			"id":         request.Subject.ID,
			"type":       request.Subject.Type,
			"attributes": request.Subject.Attributes,
			"groups":     request.Subject.Groups,
			"roles":      request.Subject.Roles,
		},
		"resource": map[string]interface{}{
			"id":         request.Resource.ID,
			"type":       request.Resource.Type,
			"attributes": request.Resource.Attributes,
			"owner":      request.Resource.Owner,
			"tags":       request.Resource.Tags,
		},
		"action": request.Action,
		"context": map[string]interface{}{
			"time":     request.Timestamp,
			"ip":       request.Context["ip"],
			"location": request.Context["location"],
		},
		// Helper functions
		"has_role": func(role string) bool {
			for _, r := range request.Subject.Roles {
				if r == role {
					return true
				}
			}
			return false
		},
		"has_group": func(group string) bool {
			for _, g := range request.Subject.Groups {
				if g == group {
					return true
				}
			}
			return false
		},
		"is_owner": func() bool {
			return request.Subject.ID == request.Resource.Owner
		},
	}
}

func (e *abacEngine) evaluatePolicy(policy *Policy, evalContext map[string]interface{}) (bool, error) {
	// Check policy condition first
	if policy.Condition != "" {
		result, err := e.evaluateCondition(policy.Condition, evalContext)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	
	// If no rules, the policy condition alone determines the result
	if len(policy.Rules) == 0 {
		return true, nil
	}
	
	// Evaluate rules
	for _, rule := range policy.Rules {
		if rule.Condition == "" {
			continue
		}
		
		result, err := e.evaluateCondition(rule.Condition, evalContext)
		if err != nil {
			return false, err
		}
		
		if result {
			// Rule matched, return based on rule effect
			// If rule effect matches policy effect, it's a match
			return rule.Effect == policy.Effect, nil
		}
	}
	
	return false, nil
}

func (e *abacEngine) evaluateCondition(condition string, evalContext map[string]interface{}) (bool, error) {
	// Parse the condition
	ast, issues := e.celEnv.Compile(condition)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile condition: %w", issues.Err())
	}
	
	// Create program
	prg, err := e.celEnv.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create program: %w", err)
	}
	
	// Evaluate
	out, _, err := prg.Eval(evalContext)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition: %w", err)
	}
	
	// Convert result to boolean
	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("condition did not evaluate to boolean")
	}
	
	return result, nil
}

func (e *abacEngine) validateCondition(condition string) error {
	_, issues := e.celEnv.Compile(condition)
	if issues != nil && issues.Err() != nil {
		return issues.Err()
	}
	return nil
}
