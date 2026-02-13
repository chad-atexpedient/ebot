package multitenancy

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// validTenantIDPattern restricts tenant IDs to safe characters only.
// This prevents SQL injection when tenant IDs are used in query scoping.
var validTenantIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,62}[A-Za-z0-9]$`)

// ValidateTenantID checks whether a tenant ID contains only safe characters.
// Returns an error if the tenant ID could be used for injection attacks.
func ValidateTenantID(tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant ID cannot be empty")
	}
	if len(tenantID) > 64 {
		return fmt.Errorf("tenant ID exceeds maximum length of 64 characters")
	}
	if !validTenantIDPattern.MatchString(tenantID) {
		return fmt.Errorf("tenant ID contains invalid characters (allowed: alphanumeric, hyphens, underscores)")
	}
	return nil
}

// TenantResolver resolves tenant ID from HTTP requests
type TenantResolver interface {
	// ResolveTenant extracts tenant ID from request
	ResolveTenant(r *http.Request) (string, error)
}

// TenantMiddleware provides HTTP middleware for tenant context
type TenantMiddleware struct {
	resolver      TenantResolver
	tenantManager TenantManager
	allowPublic   []string // Paths that don't require tenant context
}

// NewTenantMiddleware creates new tenant middleware
func NewTenantMiddleware(resolver TenantResolver, manager TenantManager) *TenantMiddleware {
	return &TenantMiddleware{
		resolver:      resolver,
		tenantManager: manager,
		allowPublic: []string{
			"/health",
			"/metrics",
			"/auth/",
			"/api/public/",
		},
	}
}

// Middleware returns HTTP middleware that injects tenant context
func (m *TenantMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is public
		for _, prefix := range m.allowPublic {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Resolve tenant
		tenantID, err := m.resolver.ResolveTenant(r)
		if err != nil {
			http.Error(w, "Tenant not found", http.StatusNotFound)
			return
		}

		// Validate tenant ID format before any further processing
		if err := ValidateTenantID(tenantID); err != nil {
			http.Error(w, "Invalid tenant identifier", http.StatusBadRequest)
			return
		}

		// Verify tenant exists and is active
		tenant, err := m.tenantManager.GetTenant(r.Context(), tenantID)
		if err != nil {
			http.Error(w, "Tenant not found", http.StatusNotFound)
			return
		}

		if tenant.Status != TenantStatusActive {
			http.Error(w, "Tenant is not active", http.StatusForbidden)
			return
		}

		// Add tenant to context
		ctx := WithTenant(r.Context(), tenantID)

		// Set tenant header for downstream services
		w.Header().Set("X-Tenant-ID", tenantID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Resolution Strategies

// HeaderResolver resolves tenant from X-Tenant-ID header
type HeaderResolver struct {
	headerName string
}

// NewHeaderResolver creates a header-based resolver
func NewHeaderResolver(headerName string) *HeaderResolver {
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}
	return &HeaderResolver{headerName: headerName}
}

// ResolveTenant extracts tenant from header
func (r *HeaderResolver) ResolveTenant(req *http.Request) (string, error) {
	tenantID := req.Header.Get(r.headerName)
	if tenantID == "" {
		return "", fmt.Errorf("missing %s header", r.headerName)
	}
	return tenantID, nil
}

// SubdomainResolver resolves tenant from subdomain
type SubdomainResolver struct {
	baseDomain string
}

// NewSubdomainResolver creates a subdomain-based resolver
func NewSubdomainResolver(baseDomain string) *SubdomainResolver {
	return &SubdomainResolver{baseDomain: baseDomain}
}

// ResolveTenant extracts tenant from subdomain
func (r *SubdomainResolver) ResolveTenant(req *http.Request) (string, error) {
	host := req.Host

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Check if it's a subdomain of base domain
	if !strings.HasSuffix(host, "."+r.baseDomain) {
		return "", fmt.Errorf("invalid domain: %s", host)
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+r.baseDomain)
	if subdomain == "" || subdomain == "www" || subdomain == "api" {
		return "", fmt.Errorf("no tenant subdomain")
	}

	return subdomain, nil
}

// PathResolver resolves tenant from URL path
type PathResolver struct {
	pathPrefix string
}

// NewPathResolver creates a path-based resolver
func NewPathResolver(prefix string) *PathResolver {
	if prefix == "" {
		prefix = "/t/"
	}
	return &PathResolver{pathPrefix: prefix}
}

// ResolveTenant extracts tenant from path
func (r *PathResolver) ResolveTenant(req *http.Request) (string, error) {
	path := req.URL.Path

	if !strings.HasPrefix(path, r.pathPrefix) {
		return "", fmt.Errorf("path does not contain tenant prefix")
	}

	// Extract tenant ID from path
	remaining := strings.TrimPrefix(path, r.pathPrefix)
	parts := strings.SplitN(remaining, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("no tenant ID in path")
	}

	return parts[0], nil
}

// JWTResolver resolves tenant from JWT claims
type JWTResolver struct {
	claimName string
}

// NewJWTResolver creates a JWT-based resolver
func NewJWTResolver(claimName string) *JWTResolver {
	if claimName == "" {
		claimName = "tenant_id"
	}
	return &JWTResolver{claimName: claimName}
}

// ResolveTenant extracts tenant from JWT
func (r *JWTResolver) ResolveTenant(req *http.Request) (string, error) {
	if claims := req.Context().Value("jwt_claims"); claims != nil {
		if claimsMap, ok := claims.(map[string]interface{}); ok {
			if tenantID, ok := claimsMap[r.claimName].(string); ok {
				return tenantID, nil
			}
		}
	}

	return "", fmt.Errorf("no tenant claim in JWT")
}

// CompositeResolver tries multiple resolvers in order
type CompositeResolver struct {
	resolvers []TenantResolver
}

// NewCompositeResolver creates a resolver that tries multiple strategies
func NewCompositeResolver(resolvers ...TenantResolver) *CompositeResolver {
	return &CompositeResolver{resolvers: resolvers}
}

// ResolveTenant tries each resolver until one succeeds
func (r *CompositeResolver) ResolveTenant(req *http.Request) (string, error) {
	var lastErr error
	for _, resolver := range r.resolvers {
		tenantID, err := resolver.ResolveTenant(req)
		if err == nil && tenantID != "" {
			return tenantID, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no resolver could extract tenant ID")
}

// TenantScopedDB provides database queries scoped to tenant
type TenantScopedDB struct {
	tenantIDColumn string
}

// NewTenantScopedDB creates a new tenant-scoped DB helper
func NewTenantScopedDB() *TenantScopedDB {
	return &TenantScopedDB{
		tenantIDColumn: "tenant_id",
	}
}

// ScopeQuery adds tenant filter to a query using validated tenant IDs.
//
// SECURITY: The tenant ID is validated against a strict regex pattern before
// being interpolated into the query. However, callers SHOULD prefer using
// parameterized queries via ScopeQueryParams() whenever possible.
//
// WARNING: This method uses string interpolation for backward compatibility.
// For new code, use ScopeQueryParams() which returns the query and parameters
// separately for use with parameterized database drivers.
func (db *TenantScopedDB) ScopeQuery(ctx context.Context, query string) string {
	tenantID := GetTenantFromContext(ctx)
	if tenantID == "" {
		return query
	}

	// Validate tenant ID to prevent SQL injection.
	// If validation fails, return a query that matches nothing rather than
	// risking injection. This is a defense-in-depth measure — the middleware
	// should have already validated the tenant ID before it reaches context.
	if err := ValidateTenantID(tenantID); err != nil {
		// Return a query guaranteed to return no rows
		return "SELECT NULL WHERE 1=0"
	}

	// Add WHERE clause if not present
	if strings.Contains(strings.ToUpper(query), "WHERE") {
		return query + fmt.Sprintf(" AND %s = '%s'", db.tenantIDColumn, tenantID)
	}
	return query + fmt.Sprintf(" WHERE %s = '%s'", db.tenantIDColumn, tenantID)
}

// ScopeQueryParams returns a parameterized query and args for safe tenant scoping.
// This is the PREFERRED method for tenant-scoped queries as it uses parameterized
// queries that are immune to SQL injection regardless of input validation.
//
// Usage:
//
//	query, args := scopedDB.ScopeQueryParams(ctx, "SELECT * FROM resources", existingArgs)
//	rows, err := db.QueryContext(ctx, query, args...)
func (db *TenantScopedDB) ScopeQueryParams(ctx context.Context, query string, args []interface{}) (string, []interface{}) {
	tenantID := GetTenantFromContext(ctx)
	if tenantID == "" {
		return query, args
	}

	paramIndex := len(args) + 1
	args = append(args, tenantID)

	if strings.Contains(strings.ToUpper(query), "WHERE") {
		return query + fmt.Sprintf(" AND %s = $%d", db.tenantIDColumn, paramIndex), args
	}
	return query + fmt.Sprintf(" WHERE %s = $%d", db.tenantIDColumn, paramIndex), args
}

// GetTenantFilter returns a filter map for the current tenant
func (db *TenantScopedDB) GetTenantFilter(ctx context.Context) map[string]interface{} {
	tenantID := GetTenantFromContext(ctx)
	if tenantID == "" {
		return nil
	}
	return map[string]interface{}{
		db.tenantIDColumn: tenantID,
	}
}
