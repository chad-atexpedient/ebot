# ebot Developer Portal

## Overview

The ebot Developer Portal provides comprehensive documentation, interactive API reference, SDKs, and tools for developers building on the ebot platform.

## Getting Started

### Quick Start

1. **Get your API key**
   ```bash
   # From ebot UI: Settings → API Keys → Create New Key
   export EBOT_API_KEY="your-api-key-here"
   ```

2. **Install SDK**
   
   **Python:**
   ```bash
   pip install ebot-sdk
   ```
   
   **TypeScript:**
   ```bash
   npm install @expedient/ebot
   ```

3. **Make your first request**
   
   **Python:**
   ```python
   from ebot import AsyncEbotClient
   
   async with AsyncEbotClient(api_key="your-key") as client:
       servers = await client.mcp_servers.list()
       print(f"Found {len(servers)} MCP servers")
   ```
   
   **TypeScript:**
   ```typescript
   import { EbotClient } from '@expedient/ebot';
   
   const client = new EbotClient({ apiKey: 'your-key' });
   const servers = await client.mcpServers.list();
   console.log(`Found ${servers.length} MCP servers`);
   ```

## API Reference

### Base URL

```
Production:  https://api.ebot.expedient.cloud
Staging:     https://api-staging.ebot.expedient.cloud
Development: http://localhost:8080
```

### Authentication

All API requests require authentication via API key:

```bash
curl -H \"Authorization: Bearer your-api-key\" \\
  https://api.ebot.expedient.cloud/api/mcp-servers
```

### Rate Limits

| Tier | Requests/Minute | Burst |
|------|----------------|-------|
| Free | 60 | 100 |
| Pro | 600 | 1000 |
| Enterprise | Custom | Custom |

## Core API Endpoints

### MCP Servers

**List MCP Servers**
```http
GET /api/mcp-servers
```

**Create MCP Server**
```http
POST /api/mcp-servers
Content-Type: application/json

{
  \"name\": \"my-server\",
  \"manifest\": {
    \"name\": \"my-server\",
    \"version\": \"1.0.0\"
  }
}
```

### Threads

**Create Thread**
```http
POST /api/threads
Content-Type: application/json

{
  \"projectID\": \"project-123\",
  \"description\": \"My conversation\"
}
```

**Send Message**
```http
POST /api/threads/{threadID}/messages
Content-Type: application/json

{
  \"content\": \"Hello, how can you help me?\"
}
```

### Costs

**Get Cost Report**
```http
GET /api/costs/report?range=7d
```

**Response:**
```json
{
  \"totalCost\": 45.67,
  \"breakdown\": {
    \"gpt-4\": 30.50,
    \"claude-3-opus\": 15.17
  },
  \"period\": {
    \"start\": \"2026-01-21T00:00:00Z\",
    \"end\": \"2026-01-28T00:00:00Z\"
  }
}
```

### Quotas

**Check Usage**
```http
GET /api/quota/usage
```

**Response:**
```json
{
  \"user\": \"user-123\",
  \"mcpServers\": {
    \"current\": 5,
    \"limit\": 50
  },
  \"threads\": {
    \"current\": 120,
    \"limit\": 1000
  }
}
```

## SDKs

### Python SDK

**Installation:**
```bash
pip install ebot-sdk
```

**Full Documentation:** [Python SDK Guide](../sdk/python/README.md)

**Example:**
```python
from ebot import AsyncEbotClient

async def main():
    async with AsyncEbotClient(api_key=\"your-key\") as client:
        # Create MCP server
        server = await client.mcp_servers.create(
            name=\"weather-server\",
            manifest={\"name\": \"weather\", \"version\": \"1.0.0\"}
        )
        
        # Check quotas
        usage = await client.quotas.get_usage()
        print(f\"Using {usage.mcp_servers.current}/{usage.mcp_servers.limit} servers\")
        
        # Get costs
        costs = await client.costs.get_report(range_days=7)
        print(f\"Total cost: ${costs.total_cost:.2f}\")

if __name__ == \"__main__\":
    import asyncio
    asyncio.run(main())
```

### TypeScript SDK

**Installation:**
```bash
npm install @expedient/ebot
```

**Full Documentation:** [TypeScript SDK Guide](../sdk/typescript/README.md)

**Example:**
```typescript
import { EbotClient } from '@expedient/ebot';

async function main() {
  const client = new EbotClient({ apiKey: 'your-key' });
  
  // Create MCP server
  const server = await client.mcpServers.create({
    name: 'weather-server',
    manifest: { name: 'weather', version: '1.0.0' }
  });
  
  // Check quotas
  const usage = await client.quotas.getUsage();
  console.log(`Using ${usage.mcpServers.current}/${usage.mcpServers.limit} servers`);
  
  // Get costs
  const costs = await client.costs.getReport({ rangeDays: 7 });
  console.log(`Total cost: $${costs.totalCost.toFixed(2)}`);
}

main().catch(console.error);
```

## Interactive API Explorer

Visit the interactive API explorer at:
https://api.ebot.expedient.cloud/docs

Features:
- Try API calls directly from browser
- Auto-generated from OpenAPI 3.0 spec
- Code examples in multiple languages
- Real-time response inspection

## Webhooks

Subscribe to events in ebot:

**Available Events:**
- `mcp_server.created`
- `mcp_server.deleted`
- `thread.created`
- `thread.completed`
- `quota.exceeded`
- `cost.threshold_reached`

**Setup:**
```http
POST /api/webhooks
Content-Type: application/json

{
  \"url\": \"https://your-app.com/webhook\",
  \"events\": [\"mcp_server.created\", \"quota.exceeded\"],
  \"secret\": \"your-webhook-secret\"
}
```

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Invalid API key |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error |

## Best Practices

### API Keys
- ✅ Store API keys in environment variables
- ✅ Rotate keys regularly (every 90 days)
- ✅ Use different keys for dev/staging/prod
- ❌ Never commit keys to version control
- ❌ Never log keys in application code

### Error Handling
```python
from ebot import EbotClient, EbotError, RateLimitError

try:
    async with EbotClient(api_key=key) as client:
        result = await client.mcp_servers.list()
except RateLimitError as e:
    # Wait and retry
    await asyncio.sleep(e.retry_after)
except EbotError as e:
    logger.error(f\"API error: {e.message}\", status=e.status_code)
```

### Pagination
```python
# Get all results with pagination
all_servers = []
page = 1

while True:
    response = await client.mcp_servers.list(page=page, per_page=100)
    all_servers.extend(response.data)
    
    if not response.has_more:
        break
    page += 1
```

## Support

- **Documentation:** https://docs.expedient.cloud/ebot
- **GitHub Issues:** https://github.com/chad-atexpedient/ebot/issues
- **Email Support:** support@expedient.cloud
- **Enterprise Support:** Contact your account manager

## Changelog

### v1.0.0 (2026-01-28)
- Initial public release
- Python SDK v1.0.0
- TypeScript SDK v1.0.0
- Complete REST API
- Multi-region support
- HIPAA compliance
- PCI-DSS compliance

## Additional Resources

- [Architecture Overview](./architecture.md)
- [Security Guide](./security-hardening.md)
- [Multi-Tenancy Guide](./multi-tenancy.md)
- [GitOps Deployment](./gitops-deployment.md)
- [Performance Tuning](./performance.md)
