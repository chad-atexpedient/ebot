package multitenancy

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"regexp"
)

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
		// Check for custom domain
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
	// This would integrate with your JWT validation
	// For now, check for a pre-extracted claim in context

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

// tenantIDPattern constrains tenant IDs to safe characters to prevent SQL injection
var tenantIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// TenantScopedDB provides database queries scoped to tenant.
//
// WARNING: ScopeQuery performs string-based query rewriting and is only made
// safe by strict validation of tenant IDs. New code SHOULD prefer using
// parameterized queries or database row-level security where possible.
type TenantScopedDB struct {
	tenantIDColumn string
}

// NewTenantScopedDB creates a new tenant-scoped DB helper
func NewTenantScopedDB() *TenantScopedDB {
	return &TenantScopedDB{
		tenantIDColumn: "tenant_id",
	}
}

// ScopeQuery adds tenant filter to a query.
//
// NOTE: This helper is intentionally conservative to avoid SQL injection.
// - It only allows tenant IDs that match [A-Za-z0-9_-]+
// - If the tenant ID is invalid, it returns a query that matches no rows.
//
// New code SHOULD prefer parameterized queries or database-enforced
// row-level security instead of string concatenation.
func (db *TenantScopedDB) ScopeQuery(ctx context.Context, query string) string {
	tenantID := GetTenantFromContext(ctx)
	if tenantID == "" {
		return query
	}

	// Validate tenant ID to avoid SQL injection
	if !tenantIDPattern.MatchString(tenantID) {
		// Invalid tenant ID → ensure no rows are returned instead of leaking data.
		upper := strings.ToUpper(query)
		if strings.Contains(upper, "WHERE") {
			return query + " AND 1=0"
		}
		return query + " WHERE 1=0"
	}

	// Add WHERE clause if not present
	upper := strings.ToUpper(query)
	if strings.Contains(upper, "WHERE") {
		return query + fmt.Sprintf(" AND %s = '%s'", db.tenantIDColumn, tenantID)
	}
	return query + fmt.Sprintf(" WHERE %s = '%s'", db.tenantIDColumn, tenantID)
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
