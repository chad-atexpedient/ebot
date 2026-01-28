# Access Control Guide

This guide covers ebot's advanced access control features including SAML 2.0, ABAC policies, and service accounts.

## Table of Contents

1. [Overview](#overview)
2. [SAML 2.0 Authentication](#saml-20-authentication)
3. [ABAC Policy Engine](#abac-policy-engine)
4. [Service Accounts](#service-accounts)
5. [API Reference](#api-reference)
6. [Examples](#examples)
7. [Best Practices](#best-practices)

---

## Overview

ebot provides enterprise-grade access control through three main components:

1. **SAML 2.0** - Enterprise Single Sign-On (SSO)
2. **ABAC** - Attribute-Based Access Control with CEL expressions
3. **Service Accounts** - Machine-to-machine authentication with API keys

### Key Features

- ✅ **SAML 2.0 SSO** - Azure AD, Okta, Google Workspace integration
- ✅ **Policy-Based Access** - Fine-grained control with CEL expressions
- ✅ **Service Accounts** - API key management with rotation
- ✅ **Federated Identity** - External identity provider support
- ✅ **Audit Trails** - Complete logging of authentication and authorization
- ✅ **API Key Rotation** - Automatic and manual key rotation
- ✅ **Rate Limiting** - Per-key request rate limits

---

## SAML 2.0 Authentication

### Configuration

Configure SAML 2.0 authentication in your ebot deployment:

```yaml
# config.yaml
saml:
  enabled: true
  entityID: "https://ebot.expedient.cloud"
  acsUrl: "https://ebot.expedient.cloud/auth/saml/acs"
  idpSSOUrl: "https://idp.example.com/sso"
  idpMetadataURL: "https://idp.example.com/metadata"
  signAuthnRequests: true
  forceAuthn: false
  attributeMappings:
    email: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
    first_name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"
    last_name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"
    groups: "http://schemas.xmlsoap.org/claims/Group"
```

### Identity Provider Setup

#### Azure AD

1. **Register Application:**
   ```
   Azure Portal → Azure Active Directory → Enterprise Applications → New Application
   ```

2. **Configure SAML:**
   - Identifier (Entity ID): `https://ebot.expedient.cloud`
   - Reply URL: `https://ebot.expedient.cloud/auth/saml/acs`

3. **Attribute Mappings:**
   ```
   user.mail → email
   user.givenname → first_name
   user.surname → last_name
   user.groups → groups
   ```

4. **Download Certificate:**
   - Download the Base64 certificate from Azure AD
   - Configure in ebot SAML settings

#### Okta

1. **Create SAML Application:**
   ```
   Okta Admin → Applications → Create App Integration → SAML 2.0
   ```

2. **Configure URLs:**
   - Single sign on URL: `https://ebot.expedient.cloud/auth/saml/acs`
   - Audience URI: `https://ebot.expedient.cloud`

3. **Attribute Statements:**
   ```
   email: user.email
   first_name: user.firstName
   last_name: user.lastName
   groups: user.groups
   ```

#### Google Workspace

1. **Add Custom SAML App:**
   ```
   Google Admin → Apps → Web and mobile apps → Add custom SAML app
   ```

2. **Service Provider Details:**
   - ACS URL: `https://ebot.expedient.cloud/auth/saml/acs`
   - Entity ID: `https://ebot.expedient.cloud`

3. **Attribute Mapping:**
   ```
   Primary email → email
   First name → first_name
   Last name → last_name
   ```

### SAML Login Flow

1. User navigates to: `https://ebot.expedient.cloud/auth/saml/login`
2. ebot generates AuthnRequest and redirects to IdP
3. User authenticates at IdP
4. IdP sends SAML assertion to ACS endpoint
5. ebot validates assertion and creates session
6. User is redirected to ebot dashboard

### Testing SAML

```bash
# Get SAML metadata
curl https://ebot.expedient.cloud/auth/saml/metadata

# Initiate SSO flow
curl -i https://ebot.expedient.cloud/auth/saml/login
```

---

## ABAC Policy Engine

### Policy Structure

ABAC policies use CEL (Common Expression Language) for flexible, expressive rules:

```json
{
  "id": "policy_001",
  "name": "Admin Full Access",
  "description": "Administrators have full access to all resources",
  "effect": "allow",
  "enabled": true,
  "priority": 100,
  "rules": [
    {
      "id": "rule_001",
      "description": "User has admin role",
      "condition": "subject.roles.contains('admin')",
      "effect": "allow"
    }
  ]
}
```

### CEL Expressions

Available variables in CEL expressions:

#### Subject (User)
- `subject.id` - User ID
- `subject.type` - User type ("user", "service_account", "group")
- `subject.roles` - List of roles
- `subject.groups` - List of groups
- `subject.attributes` - Custom attributes map

#### Resource
- `resource.id` - Resource ID
- `resource.type` - Resource type ("mcp_server", "thread", "workspace")
- `resource.owner` - Owner ID
- `resource.tags` - List of tags
- `resource.attributes` - Custom attributes map

#### Action
- `action` - Action being performed ("read", "write", "delete", "execute")

#### Context
- `context.time` - Request timestamp
- `context.ip` - Client IP address
- `context.location` - Geographic location
- `context.*` - Any custom context

#### Helper Functions
- `has_role(role)` - Check if subject has role
- `has_group(group)` - Check if subject is in group
- `is_owner()` - Check if subject owns resource

### Example Policies

#### 1. Owner-Only Access
```json
{
  "name": "Owner Only Write",
  "effect": "allow",
  "rules": [{
    "condition": "action == 'write' && is_owner()",
    "effect": "allow"
  }]
}
```

#### 2. Time-Based Access
```json
{
  "name": "Business Hours Only",
  "effect": "allow",
  "rules": [{
    "condition": "context.time.getHours() >= 9 && context.time.getHours() < 17",
    "effect": "allow"
  }]
}
```

#### 3. Group-Based Access
```json
{
  "name": "Engineering Team Access",
  "effect": "allow",
  "rules": [{
    "condition": "has_group('engineering') && resource.tags.contains('dev')",
    "effect": "allow"
  }]
}
```

#### 4. Multi-Condition Policy
```json
{
  "name": "Sensitive Data Access",
  "effect": "allow",
  "rules": [{
    "condition": "has_role('data_scientist') && resource.attributes.classification == 'public' || has_role('admin')",
    "effect": "allow"
  }]
}
```

#### 5. Deny Policy (Takes Precedence)
```json
{
  "name": "Block Suspended Users",
  "effect": "deny",
  "rules": [{
    "condition": "subject.attributes.status == 'suspended'",
    "effect": "deny"
  }]
}
```

### Policy Evaluation

Policies are evaluated in priority order:
1. **Deny policies** take precedence over allow
2. Higher priority (lower number) evaluated first
3. First matching policy wins
4. Default is deny if no policy matches

### Creating Policies via API

```bash
curl -X POST https://ebot.expedient.cloud/api/policies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "My Policy",
    "description": "Custom access policy",
    "effect": "allow",
    "enabled": true,
    "priority": 100,
    "rules": [{
      "description": "Admins only",
      "condition": "has_role(\"admin\")",
      "effect": "allow"
    }]
  }'
```

---

## Service Accounts

### Overview

Service accounts provide machine-to-machine authentication using API keys.

### Account Types

1. **Standard** - Normal access with configured permissions
2. **Elevated** - Enhanced permissions for critical operations
3. **Read-Only** - Read-only access to resources

### Creating Service Accounts

#### Via API

```bash
curl -X POST https://ebot.expedient.cloud/api/service-accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "CI/CD Pipeline",
    "description": "Service account for automated deployments",
    "workspace_id": "ws_123",
    "type": "standard",
    "permissions": [
      {
        "resource": "mcp_servers",
        "actions": ["read", "write"],
        "scope": "workspace:ws_123"
      }
    ]
  }'
```

#### Via UI

1. Navigate to **Admin → Access Control → Service Accounts**
2. Click **Create Service Account**
3. Fill in details and permissions
4. Click **Create**

### API Key Management

#### Generating Keys

```bash
curl -X POST https://ebot.expedient.cloud/api/service-accounts/sa_123/api-keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Production Key",
    "scopes": ["read", "write"],
    "expiresAt": "2027-01-01T00:00:00Z",
    "rateLimitRPM": 1000,
    "rotationPolicy": {
      "enabled": true,
      "rotationPeriod": "90d",
      "gracePeriod": "7d",
      "autoRevoke": true,
      "notifyBeforeDays": 7
    }
  }'
```

Response:
```json
{
  "id": "key_abc123",
  "key": "ebot_aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890",
  "key_prefix": "ebot_aBcDeFg",
  "expires_at": "2027-01-01T00:00:00Z",
  "warning": "Save this key securely. It won't be shown again."
}
```

#### Using API Keys

```bash
# Authenticate with API key
curl https://ebot.expedient.cloud/api/mcp-servers \
  -H "Authorization: Bearer ebot_aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890"
```

#### Key Rotation

**Manual Rotation:**
```bash
curl -X POST https://ebot.expedient.cloud/api/api-keys/key_abc123/rotate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "gracePeriodDays": 7
  }'
```

**Automatic Rotation:**
- Configured via `rotationPolicy` when creating key
- Keys auto-rotate after `rotationPeriod`
- Old key remains valid during `gracePeriod`
- Notifications sent `notifyBeforeDays` before rotation

#### Revoking Keys

```bash
curl -X DELETE https://ebot.expedient.cloud/api/api-keys/key_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

### Key Security

- ✅ Keys are bcrypt-hashed in storage
- ✅ Full key shown only once at creation
- ✅ Per-key rate limiting
- ✅ Automatic expiration
- ✅ Usage tracking and audit logs
- ✅ IP-based restrictions (optional)

---

## API Reference

### ABAC Policies

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/policies` | Create policy |
| GET | `/api/policies` | List all policies |
| GET | `/api/policies/:id` | Get policy by ID |
| PUT | `/api/policies/:id` | Update policy |
| DELETE | `/api/policies/:id` | Delete policy |
| POST | `/api/policies/evaluate` | Evaluate authorization request |

### SAML

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/auth/saml/login` | Initiate SSO |
| POST | `/auth/saml/acs` | Assertion Consumer Service |
| GET | `/auth/saml/metadata` | Get SAML metadata XML |

### Service Accounts

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/service-accounts` | Create service account |
| GET | `/api/service-accounts` | List service accounts |
| GET | `/api/service-accounts/:id` | Get service account |
| PUT | `/api/service-accounts/:id` | Update service account |
| DELETE | `/api/service-accounts/:id` | Delete service account |
| POST | `/api/service-accounts/:id/api-keys` | Generate API key |
| GET | `/api/service-accounts/:id/api-keys` | List API keys |
| DELETE | `/api/api-keys/:id` | Revoke API key |
| POST | `/api/api-keys/:id/rotate` | Rotate API key |

---

## Examples

### Complete Setup Example

```bash
#!/bin/bash

# 1. Configure SAML
cat > saml-config.yaml <<EOF
saml:
  enabled: true
  entityID: "https://ebot.example.com"
  acsUrl: "https://ebot.example.com/auth/saml/acs"
  idpSSOUrl: "https://idp.example.com/sso"
EOF

# 2. Create ABAC Policy
curl -X POST https://ebot.example.com/api/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Engineering Full Access",
    "effect": "allow",
    "enabled": true,
    "priority": 100,
    "rules": [{
      "condition": "has_group(\"engineering\")",
      "effect": "allow"
    }]
  }'

# 3. Create Service Account
SA_RESPONSE=$(curl -X POST https://ebot.example.com/api/service-accounts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD",
    "workspace_id": "ws_prod",
    "type": "standard"
  }')

SA_ID=$(echo $SA_RESPONSE | jq -r '.id')

# 4. Generate API Key
KEY_RESPONSE=$(curl -X POST https://ebot.example.com/api/service-accounts/$SA_ID/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production Key",
    "expiresAt": "2027-01-01T00:00:00Z",
    "rateLimitRPM": 1000
  }')

API_KEY=$(echo $KEY_RESPONSE | jq -r '.key')
echo "API Key: $API_KEY"

# 5. Test API Key
curl https://ebot.example.com/api/mcp-servers \
  -H "Authorization: Bearer $API_KEY"
```

---

## Best Practices

### SAML

1. **Always use HTTPS** for SAML endpoints
2. **Enable signature validation** on assertions
3. **Rotate certificates** annually
4. **Test SSO flow** in staging first
5. **Configure attribute mappings** carefully
6. **Monitor SAML errors** in audit logs

### ABAC Policies

1. **Start with deny-by-default** approach
2. **Use explicit policies** rather than implicit
3. **Test policies** before enabling in production
4. **Document policy intent** in description field
5. **Use priority** to control evaluation order
6. **Audit policy changes** regularly
7. **Keep conditions simple** and readable

### Service Accounts

1. **Use least-privilege** permissions
2. **Rotate API keys** regularly (90 days recommended)
3. **Set expiration dates** on all keys
4. **Monitor key usage** and revoke unused keys
5. **Use separate accounts** for different services
6. **Enable automatic rotation** with grace periods
7. **Store keys securely** (vault, secrets manager)
8. **Never commit keys** to version control
9. **Set rate limits** appropriate to use case
10. **Audit service account** activity

### General

- ✅ Enable multi-factor authentication (MFA) for admins
- ✅ Regular access reviews (quarterly)
- ✅ Implement least-privilege access
- ✅ Monitor authentication failures
- ✅ Set up alerts for suspicious activity
- ✅ Document access control policies
- ✅ Train users on security best practices

---

## Troubleshooting

### SAML Issues

**Problem:** SAML login redirects to error page

**Solutions:**
1. Verify IdP metadata URL is accessible
2. Check entity ID matches exactly
3. Ensure ACS URL is correct
4. Validate certificate is not expired
5. Check IdP logs for specific errors

### ABAC Policy Issues

**Problem:** Policy not evaluating as expected

**Solutions:**
1. Test CEL expression in policy evaluator
2. Check variable names and syntax
3. Verify rule priority order
4. Review deny policies (they take precedence)
5. Check policy is enabled

### API Key Issues

**Problem:** API key authentication fails

**Solutions:**
1. Verify key is not expired
2. Check key is not revoked
3. Ensure correct Authorization header format
4. Verify service account is active
5. Check rate limit not exceeded

---

## Support

For additional help:
- **Documentation:** https://docs.expedient.cloud/ebot
- **Support:** Contact Expedient Cloud Support
- **GitHub Issues:** https://github.com/chad-atexpedient/ebot/issues

---

**Last Updated:** January 28, 2026  
**Version:** Phase 2C
