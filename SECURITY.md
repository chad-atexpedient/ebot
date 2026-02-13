# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | ✅ Active          |
| < main  | ❌ Not supported   |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in ebot, please report it responsibly.

### How to Report

1. **DO NOT** open a public GitHub issue for security vulnerabilities.
2. Email: **cloud-support@expedient.com** with subject line `[SECURITY] ebot vulnerability report`
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Suggested fix (if any)

### Response Timeline

| Stage | Timeline |
|-------|----------|
| Acknowledgment | Within 48 hours |
| Initial assessment | Within 5 business days |
| Fix for critical issues | Within 7 days |
| Fix for high issues | Within 30 days |
| Public disclosure | After fix is released |

### Scope

The following are in scope for security reports:
- SQL injection, XSS, CSRF vulnerabilities
- Authentication/authorization bypasses
- Data exposure or leakage
- Privilege escalation
- Denial of service (application-level)
- Cryptographic weaknesses
- Dependency vulnerabilities (critical/high severity)

### Out of Scope

- Social engineering attacks
- Physical attacks
- Vulnerabilities in third-party services
- Issues already reported in public issues

## Security Best Practices

When contributing to ebot, please follow these practices:

1. **Never commit secrets** — Use environment variables or secret managers
2. **Parameterize all queries** — Use `ScopeQueryParams()` for tenant-scoped queries
3. **Validate all input** — Use `ValidateTenantID()` and similar validation functions
4. **Pin dependencies** — Always use specific versions, never `@master` or `@latest`
5. **Review security scan results** — CI blocks on CRITICAL/HIGH vulnerabilities

## Acknowledgments

We thank the security community for helping keep ebot and its users safe.
