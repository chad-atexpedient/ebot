package abac

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

	// GetCacheStats returns CEL program cache statistics
	GetCacheStats() CacheStats
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

// CacheStats contains CEL program cache statistics
type CacheStats struct {
	Size     int     `json:"size"`
	MaxSize  int     `json:"maxSize"`
	Hits     int64   `json:"hits"`
	Misses   int64   `json:"misses"`
	HitRate  float64 `json:"hitRate"`
	Evictions int64  `json:"evictions"`
}

// celProgramCache caches compiled CEL programs to avoid recompilation
type celProgramCache struct {
	programs  sync.Map // condition string -> *cachedProgram
	hits      int64
	misses    int64
	evictions int64
	maxSize   int
	mu        sync.Mutex
}

// cachedProgram represents a cached CEL program
type cachedProgram struct {
	program   cel.Program
	lastUsed  time.Time
	useCount  int64
}

// newCELProgramCache creates a new CEL program cache
func newCELProgramCache(maxSize int) *celProgramCache {
	if maxSize <= 0 {
		maxSize = 1000 // Default max size
	}
	return &celProgramCache{
		maxSize: maxSize,
	}
}

// Get retrieves a cached program
func (c *celProgramCache) Get(condition string) (cel.Program, bool) {
	if val, ok := c.programs.Load(condition); ok {
		cached := val.(*cachedProgram)
		cached.lastUsed = time.Now()
		atomic.AddInt64(&cached.useCount, 1)
		atomic.AddInt64(&c.hits, 1)
		return cached.program, true
	}
	atomic.AddInt64(&c.misses, 1)
	return nil, false
}

// Set stores a program in the cache
func (c *celProgramCache) Set(condition string, program cel.Program) {
	// Check if we need to evict
	c.mu.Lock()
	if c.size() >= c.maxSize {
		c.evictLRU()
	}
	c.mu.Unlock()

	c.programs.Store(condition, &cachedProgram{
		program:  program,
		lastUsed: time.Now(),
		useCount: 1,
	})
}

// Delete removes a program from the cache
func (c *celProgramCache) Delete(condition string) {
	c.programs.Delete(condition)
}

// size returns the current cache size
func (c *celProgramCache) size() int {
	size := 0
	c.programs.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	return size
}

// evictLRU evicts the least recently used entry
func (c *celProgramCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	c.programs.Range(func(key, value interface{}) bool {
		cached := value.(*cachedProgram)
		if oldestKey == "" || cached.lastUsed.Before(oldestTime) {
			oldestKey = key.(string)
			oldestTime = cached.lastUsed
		}
		return true
	})

	if oldestKey != "" {
		c.programs.Delete(oldestKey)
		atomic.AddInt64(&c.evictions, 1)
	}
}

// Stats returns cache statistics
func (c *celProgramCache) Stats() CacheStats {
	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	total := hits + misses

	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return CacheStats{
		Size:      c.size(),
		MaxSize:   c.maxSize,
		Hits:      hits,
		Misses:    misses,
		HitRate:   hitRate,
		Evictions: atomic.LoadInt64(&c.evictions),
	}
}

// Clear removes all entries from the cache
func (c *celProgramCache) Clear() {
	c.programs.Range(func(key, _ interface{}) bool {
		c.programs.Delete(key)
		return true
	})
}

// abacEngine implements ABACEngine
type abacEngine struct {
	policies     map[string]*Policy
	mu           sync.RWMutex
	logger       logger.Logger
	celEnv       *cel.Env
	programCache *celProgramCache // CEL program cache for performance
}

// NewABACEngine creates a new ABAC engine
func NewABACEngine(log logger.Logger) (ABACEngine, error) {
	return NewABACEngineWithCacheSize(log, 1000)
}

// NewABACEngineWithCacheSize creates a new ABAC engine with custom cache size
func NewABACEngineWithCacheSize(log logger.Logger, cacheSize int) (ABACEngine, error) {
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
		policies:     make(map[string]*Policy),
		logger:       log,
		celEnv:       env,
		programCache: newCELProgramCache(cacheSize),
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

// AddPolicy adds a new policy with pre-compilation of CEL expressions
func (e *abacEngine) AddPolicy(policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	// Pre-compile and cache the policy condition
	if policy.Condition != "" {
		if _, err := e.compileAndCache(policy.Condition); err != nil {
			return fmt.Errorf("invalid policy condition: %w", err)
		}
	}

	// Pre-compile and cache all rule conditions
	for _, rule := range policy.Rules {
		if rule.Condition != "" {
			if _, err := e.compileAndCache(rule.Condition); err != nil {
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

// RemovePolicy removes a policy by ID and invalidates its cache entries
func (e *abacEngine) RemovePolicy(policyID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	policy, exists := e.policies[policyID]
	if !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}

	// Invalidate cached programs for this policy
	e.invalidatePolicyCache(policy)

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

// UpdatePolicy updates an existing policy and refreshes its cache
func (e *abacEngine) UpdatePolicy(policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	old, exists := e.policies[policy.ID]
	if !exists {
		return fmt.Errorf("policy %s not found", policy.ID)
	}

	// Invalidate old cache entries
	e.invalidatePolicyCache(old)

	// Pre-compile and cache new conditions
	if policy.Condition != "" {
		if _, err := e.compileAndCacheInternal(policy.Condition); err != nil {
			return fmt.Errorf("invalid policy condition: %w", err)
		}
	}

	for _, rule := range policy.Rules {
		if rule.Condition != "" {
			if _, err := e.compileAndCacheInternal(rule.Condition); err != nil {
				return fmt.Errorf("invalid rule condition in rule %s: %w", rule.ID, err)
			}
		}
	}

	policy.UpdatedAt = time.Now()
	e.policies[policy.ID] = policy

	e.logger.Info("Updated policy",
		"policy_id", policy.ID,
		"name", policy.Name)

	return nil
}

// GetCacheStats returns CEL program cache statistics
func (e *abacEngine) GetCacheStats() CacheStats {
	return e.programCache.Stats()
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

// evaluateCondition evaluates a CEL condition using cached programs
// This is the key performance optimization - O(1) for cached conditions
func (e *abacEngine) evaluateCondition(condition string, evalContext map[string]interface{}) (bool, error) {
	// Try to get from cache first - O(1) lookup
	prg, ok := e.programCache.Get(condition)
	if !ok {
		// Cache miss - compile and cache
		var err error
		prg, err = e.compileAndCache(condition)
		if err != nil {
			return false, err
		}
	}

	// Evaluate the cached program
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

// compileAndCache compiles a CEL condition and caches the program
func (e *abacEngine) compileAndCache(condition string) (cel.Program, error) {
	// Parse the condition
	ast, issues := e.celEnv.Compile(condition)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile condition: %w", issues.Err())
	}

	// Create program
	prg, err := e.celEnv.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	// Cache the program
	e.programCache.Set(condition, prg)

	return prg, nil
}

// compileAndCacheInternal is used when lock is already held
func (e *abacEngine) compileAndCacheInternal(condition string) (cel.Program, error) {
	return e.compileAndCache(condition)
}

// invalidatePolicyCache removes cached programs for a policy
func (e *abacEngine) invalidatePolicyCache(policy *Policy) {
	if policy.Condition != "" {
		e.programCache.Delete(policy.Condition)
	}
	for _, rule := range policy.Rules {
		if rule.Condition != "" {
			e.programCache.Delete(rule.Condition)
		}
	}
}

// validateCondition validates a CEL condition without caching
func (e *abacEngine) validateCondition(condition string) error {
	_, issues := e.celEnv.Compile(condition)
	if issues != nil && issues.Err() != nil {
		return issues.Err()
	}
	return nil
}
