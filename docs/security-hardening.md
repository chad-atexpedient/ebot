# Security Hardening in ebot

## Overview

ebot implements comprehensive security hardening measures including advanced rate limiting, security headers, automated secrets rotation, input validation, and DDoS protection. This guide covers all security features and best practices.

## Table of Contents

1. [Rate Limiting & DDoS Protection](#rate-limiting--ddos-protection)
2. [Security Headers](#security-headers)
3. [Secrets Rotation](#secrets-rotation)
4. [Input Validation](#input-validation)
5. [Best Practices](#best-practices)
6. [API Reference](#api-reference)

---

## Rate Limiting & DDoS Protection

### Overview

ebot provides multi-layered rate limiting with automatic DDoS protection:

- **Global rate limits** - Platform-wide request caps
- **Per-IP limits** - Prevent abuse from single sources
- **Per-user limits** - Fair resource allocation
- **Per-endpoint limits** - Protect sensitive operations
- **Adaptive rate limiting** - Automatically adjust under load
- **IP blocking** - Temporary bans for suspicious activity

### Configuration

```go
import "github.com/chad-atexpedient/ebot/pkg/security"

config := &security.RateLimitConfig{
    // Global limits
    GlobalRequestsPerSecond: 10000,
    GlobalBurstSize:         20000,

    // Per-IP limits
    PerIPRequestsPerSecond: 100,
    PerIPBurstSize:         200,

    // Per-user limits
    PerUserRequestsPerSecond: 500,
    PerUserBurstSize:         1000,

    // Endpoint-specific limits
    EndpointLimits: map[string]security.EndpointLimit{
        "/api/auth/login": {
            RequestsPerSecond: 5,
            BurstSize:         10,
        },
        "/api/auth/register": {
            RequestsPerSecond: 2,
            BurstSize:         5,
        },
    },

    // DDoS protection
    EnableDDoSProtection:  true,
    SuspiciousIPThreshold: 500, // Requests/sec
    BlockDuration:         15 * time.Minute,

    // Adaptive rate limiting
    EnableAdaptive:     true,
    LoadThreshold:      0.8,  // Trigger at 80% load
    AdaptiveMultiplier: 0.5,  // Reduce limits by 50%
}

rateLimiter := security.NewRateLimiter(config, cacheManager)
```

### HTTP Middleware

```go
// Add rate limiting middleware to your HTTP server
getUserID := func(r *http.Request) string {
    // Extract user ID from token/session
    return r.Header.Get("X-User-ID")
}

http.Handle("/", rateLimiter.Middleware(getUserID)(yourHandler))
```

### Checking Rate Limits Programmatically

```go
result, err := rateLimiter.CheckLimit(request, userID)
if err != nil {
    log.Fatal(err)
}

if !result.Allowed {
    fmt.Printf("Rate limit exceeded: %s\n", result.Reason)
    fmt.Printf("Retry after: %s\n", result.RetryAfter)
    return
}

fmt.Printf("Remaining requests: %d\n", result.RemainingHits)
```

### Managing Blocked IPs

```go
// Get currently blocked IPs
blockedIPs := rateLimiter.GetBlockedIPs()
for ip, until := range blockedIPs {
    fmt.Printf("IP %s blocked until %s\n", ip, until)
}

// Manually unblock an IP
rateLimiter.UnblockIP("192.168.1.100")
```

### Adaptive Rate Limiting

```go
// Update system load (0.0-1.0)
// This triggers adaptive rate limiting if load > threshold
rateLimiter.UpdateSystemLoad(0.85)

// Get current system load
load := rateLimiter.GetSystemLoad()
fmt.Printf("Current load: %.2f%%\n", load*100)
```

### Response Headers

Rate limit information is included in response headers:

```
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1706457600
Retry-After: 60
```

---

## Security Headers

### Overview

ebot automatically sets comprehensive security headers on all responses:

- **Content-Security-Policy (CSP)** - Prevent XSS attacks
- **Strict-Transport-Security (HSTS)** - Enforce HTTPS
- **X-Frame-Options** - Prevent clickjacking
- **X-Content-Type-Options** - Prevent MIME sniffing
- **X-XSS-Protection** - Legacy XSS protection
- **Referrer-Policy** - Control referrer information
- **Permissions-Policy** - Control browser features
- **Cross-Origin policies** - Isolate cross-origin resources

### Default Configuration

```go
import "github.com/chad-atexpedient/ebot/pkg/security"

// Use secure defaults
config := security.DefaultSecurityHeaders()

// Or production-grade strict settings
config := security.ProductionSecurityHeaders()

// Or relaxed settings for development
config := security.DevelopmentSecurityHeaders()
```

### Custom Configuration

```go
config := &security.SecurityHeadersConfig{
    // Content Security Policy
    CSP: security.CSPConfig{
        Enable:       true,
        DefaultSrc:   []string{"'self'"},
        ScriptSrc:    []string{"'self'", "https://trusted-cdn.com"},
        StyleSrc:     []string{"'self'", "'unsafe-inline'"},
        ImgSrc:       []string{"'self'", "data:", "https:"},
        ConnectSrc:   []string{"'self'", "https://api.example.com"},
        FrameAncestors: []string{"'none'"},
        UpgradeInsecureRequests: true,
        BlockAllMixedContent:    true,
    },

    // HSTS
    EnableHSTS:             true,
    HSTSMaxAge:             31536000, // 1 year
    HSTSIncludeSubdomains:  true,
    HSTSPreload:            true,

    // Frame options
    FrameOptions: "DENY",

    // Other headers
    EnableNoSniff:       true,
    EnableXSSProtection: true,
    ReferrerPolicy:      "strict-origin-when-cross-origin",

    // Permissions Policy
    PermissionsPolicy: map[string][]string{
        "camera":      {},
        "microphone":  {},
        "geolocation": {},
    },

    // Cross-Origin policies
    CrossOriginOpenerPolicy:   "same-origin",
    CrossOriginResourcePolicy: "same-origin",
    CrossOriginEmbedderPolicy: "require-corp",
}
```

### HTTP Middleware

```go
// Add security headers middleware
middleware := security.SecurityHeadersMiddleware(config)
http.Handle("/", middleware(yourHandler))
```

### CSP Violation Reporting

```go
config.CSP.ReportURI = "https://your-domain.com/csp-report"

// Handle CSP reports
http.HandleFunc("/csp-report", func(w http.ResponseWriter, r *http.Request) {
    var report map[string]interface{}
    json.NewDecoder(r.Body).Decode(&report)
    log.Printf("CSP Violation: %+v", report)
})
```

---

## Secrets Rotation

### Overview

Automated credential rotation with configurable intervals and grace periods:

- **Automatic rotation** - Scheduled rotation of all secrets
- **Grace periods** - Old secrets remain valid during transition
- **Notifications** - Alerts before rotation due
- **Multiple secret types** - API keys, passwords, encryption keys, etc.

### Configuration

```go
import "github.com/chad-atexpedient/ebot/pkg/security"

config := &security.SecretsRotationConfig{
    // Rotation intervals
    APIKeyRotationInterval:        90 * 24 * time.Hour,  // 90 days
    DatabasePasswordInterval:      30 * 24 * time.Hour,  // 30 days
    EncryptionKeyInterval:         365 * 24 * time.Hour, // 1 year
    OAuthClientSecretInterval:     180 * 24 * time.Hour, // 6 months
    ServiceAccountKeyInterval:     90 * 24 * time.Hour,  // 90 days

    // Grace period (old secret still valid)
    GracePeriod: 7 * 24 * time.Hour, // 7 days

    // Notifications
    NotifyDaysBefore: 14,
    NotifyEmail:      "security@example.com",
    NotifyWebhook:    "https://hooks.slack.com/...",

    // Enable auto-rotation
    EnableAutoRotation: true,
}

manager := security.NewSecretsRotationManager(
    config,
    storage,   // Implement SecretsStorage interface
    notifier,  // Implement Notifier interface
    logger,
)
```

### Registering Secrets

```go
secret := &security.Secret{
    ID:   "api-key-123",
    Type: security.SecretTypeAPIKey,
    CurrentValue: "sk-current-secret-value",
    Metadata: map[string]string{
        "owner": "service-account-1",
        "scope": "read-write",
    },
}

err := manager.RegisterSecret(ctx, secret)
if err != nil {
    log.Fatal(err)
}
```

### Manual Rotation

```go
// Rotate a specific secret immediately
err := manager.RotateSecret(ctx, "api-key-123")
if err != nil {
    log.Fatal(err)
}
```

### Validating Secrets

```go
// Check if a secret value is valid
valid, err := manager.ValidateSecret(ctx, "api-key-123", providedValue)
if err != nil {
    log.Fatal(err)
}

if !valid {
    http.Error(w, "Invalid API key", http.StatusUnauthorized)
    return
}

// Both current and previous (in grace period) values are accepted
```

### Checking Rotation Status

```go
status, err := manager.GetRotationStatus(ctx, "api-key-123")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", status.Status) // ok, due_soon, overdue
fmt.Printf("Last rotated: %s\n", status.LastRotatedAt)
fmt.Printf("Next rotation: %s\n", status.NextRotationAt)
fmt.Printf("Days until rotation: %d\n", status.DaysUntilRotation)
fmt.Printf("In grace period: %v\n", status.InGracePeriod)
```

### Listing Secrets Due for Rotation

```go
secrets, err := manager.ListSecretsForRotation(ctx)
for _, secret := range secrets {
    fmt.Printf("Secret %s (%s): %d days until rotation\n",
        secret.SecretID,
        secret.Type,
        secret.DaysUntilRotation,
    )
}
```

### Implementing Storage Interface

```go
type PostgresSecretsStorage struct {
    db *sql.DB
}

func (s *PostgresSecretsStorage) Store(ctx context.Context, secret *security.Secret) error {
    query := `
        INSERT INTO secrets (id, type, current_value, previous_value, created_at, ...)
        VALUES ($1, $2, $3, $4, $5, ...)
        ON CONFLICT (id) DO UPDATE SET ...
    `
    _, err := s.db.ExecContext(ctx, query, secret.ID, secret.Type, ...)
    return err
}

func (s *PostgresSecretsStorage) Get(ctx context.Context, id string) (*security.Secret, error) {
    var secret security.Secret
    query := `SELECT * FROM secrets WHERE id = $1`
    err := s.db.QueryRowContext(ctx, query, id).Scan(&secret...)
    return &secret, err
}
```

### Implementing Notifier Interface

```go
type EmailNotifier struct {
    smtpHost string
    from     string
}

func (n *EmailNotifier) NotifyRotationDue(ctx context.Context, secret *security.Secret, daysUntil int) error {
    subject := fmt.Sprintf("Secret %s rotation due in %d days", secret.ID, daysUntil)
    body := fmt.Sprintf("Secret %s (%s) will be rotated in %d days", secret.ID, secret.Type, daysUntil)
    return n.sendEmail(subject, body)
}

func (n *EmailNotifier) NotifyRotationCompleted(ctx context.Context, secret *security.Secret) error {
    subject := fmt.Sprintf("Secret %s rotated successfully", secret.ID)
    body := fmt.Sprintf("Secret %s (%s) has been rotated. Grace period ends: %s",
        secret.ID, secret.Type, secret.GraceEndsAt)
    return n.sendEmail(subject, body)
}

func (n *EmailNotifier) NotifyRotationFailed(ctx context.Context, secret *security.Secret, err error) error {
    subject := fmt.Sprintf("SECRET ROTATION FAILED: %s", secret.ID)
    body := fmt.Sprintf("CRITICAL: Failed to rotate secret %s: %v", secret.ID, err)
    return n.sendEmail(subject, body)
}
```

---

## Input Validation

### Overview

Comprehensive input validation to prevent injection attacks and malformed data.

### Best Practices

1. **Validate all inputs** - Never trust user input
2. **Whitelist, don't blacklist** - Define what's allowed, not what's forbidden
3. **Type checking** - Enforce strict types
4. **Length limits** - Prevent buffer overflows and DoS
5. **Format validation** - Use regex patterns for structured data
6. **Sanitization** - Remove dangerous characters

### Example Validation

```go
import (
    "regexp"
    "strings"
)

// Email validation
func validateEmail(email string) error {
    if len(email) > 254 {
        return errors.New("email too long")
    }
    
    pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
    matched, _ := regexp.MatchString(pattern, email)
    if !matched {
        return errors.New("invalid email format")
    }
    
    return nil
}

// URL validation
func validateURL(url string) error {
    if len(url) > 2048 {
        return errors.New("URL too long")
    }
    
    parsed, err := url.Parse(url)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    
    // Only allow HTTPS
    if parsed.Scheme != "https" {
        return errors.New("only HTTPS URLs allowed")
    }
    
    return nil
}

// SQL injection prevention
func sanitizeInput(input string) string {
    // Use parameterized queries instead!
    // This is just an example of what NOT to do
    dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_"}
    result := input
    for _, d := range dangerous {
        result = strings.ReplaceAll(result, d, "")
    }
    return result
}

// Use parameterized queries
func getUserByEmail(db *sql.DB, email string) (*User, error) {
    if err := validateEmail(email); err != nil {
        return nil, err
    }
    
    query := "SELECT * FROM users WHERE email = $1"
    var user User
    err := db.QueryRow(query, email).Scan(&user...)
    return &user, err
}
```

---

## Best Practices

### 1. Defense in Depth

Implement multiple layers of security:

```go
// Layer 1: Rate limiting
http.Handle("/", rateLimiter.Middleware(getUserID)(

    // Layer 2: Security headers
    securityHeaders.Middleware(

        // Layer 3: Authentication
        authMiddleware(

            // Layer 4: Authorization
            authzMiddleware(

                // Layer 5: Input validation
                validationMiddleware(
                    yourHandler,
                ),
            ),
        ),
    ),
))
```

### 2. Fail Securely

```go
// Bad: Exposes information
if !authenticated {
    http.Error(w, "User not found in database", http.StatusUnauthorized)
}

// Good: Generic error message
if !authenticated {
    http.Error(w, "Authentication failed", http.StatusUnauthorized)
}
```

### 3. Least Privilege

```go
// Grant minimum necessary permissions
permissions := []string{"read:projects"}
// Don't grant: []string{"*"}
```

### 4. Secure Defaults

```go
// Bad: Insecure default
config := &Config{
    EnableSSL: false, // Requires opt-in
}

// Good: Secure default
config := &Config{
    EnableSSL: true, // Requires opt-out
}
```

### 5. Security Logging

```go
import "log/slog"

// Log security events
slog.Info("Authentication failed",
    "ip", clientIP,
    "username", username,
    "reason", "invalid_password",
)

// Log secrets rotation
slog.Info("Secret rotated",
    "secret_id", secretID,
    "type", secretType,
    "rotation_count", count,
)

// Log rate limit violations
slog.Warn("Rate limit exceeded",
    "ip", clientIP,
    "endpoint", path,
    "user_id", userID,
)
```

---

## API Reference

### Rate Limiter Endpoints

```http
GET /api/security/rate-limits/status
GET /api/security/blocked-ips
POST /api/security/blocked-ips/{ip}/unblock
PUT /api/security/system-load
```

### Secrets Rotation Endpoints

```http
GET /api/security/secrets/rotation-status
GET /api/security/secrets/{id}/status
POST /api/security/secrets/{id}/rotate
GET /api/security/secrets/due-for-rotation
```

### Example: Check Blocked IPs

```bash
curl -H "Authorization: Bearer $TOKEN" \
  https://ebot.example.com/api/security/blocked-ips
```

Response:
```json
{
  "blocked_ips": [
    {
      "ip": "203.0.113.45",
      "blocked_until": "2026-01-28T15:30:00Z",
      "reason": "suspicious_activity",
      "request_count": 1500
    }
  ]
}
```

### Example: Rotate Secret

```bash
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  https://ebot.example.com/api/security/secrets/api-key-123/rotate
```

Response:
```json
{
  "success": true,
  "secret_id": "api-key-123",
  "rotated_at": "2026-01-28T12:00:00Z",
  "next_rotation": "2026-04-28T12:00:00Z",
  "grace_period_ends": "2026-02-04T12:00:00Z"
}
```

---

## Compliance

These security features help meet various compliance requirements:

- **PCI-DSS** - Rate limiting, input validation, secrets rotation
- **HIPAA** - Access controls, audit logging, encryption
- **SOC 2** - Comprehensive security controls, incident response
- **GDPR** - Data protection, access controls
- **ISO 27001** - Information security management

---

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [CWE/SANS Top 25](https://www.sans.org/top25-software-errors/)

---

**For support:** Contact Expedient Cloud Security Team  
**Documentation:** https://docs.expedient.cloud/ebot/security
