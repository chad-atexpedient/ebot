# ebot TypeScript SDK

Official TypeScript/JavaScript client library for the [ebot platform](https://github.com/chad-atexpedient/ebot).

Works in Node.js, browsers, and edge runtimes (Cloudflare Workers, Vercel Edge, etc.).

## Features

- ✅ **TypeScript Native** - Full type safety and IntelliSense support
- ✅ **Modern** - Built with async/await, ESM, and latest standards
- ✅ **Universal** - Works in Node.js, browsers, and edge runtimes
- ✅ **Tree-Shakeable** - Only import what you need
- ✅ **Zero Dependencies** - Uses native `fetch` API
- ✅ **Comprehensive** - Coverage of all ebot APIs
- ✅ **Well-Tested** - High test coverage and battle-tested

## Installation

```bash
npm install @expedient/ebot
# or
yarn add @expedient/ebot
# or
pnpm add @expedient/ebot
```

## Quick Start

```typescript
import { EbotClient } from '@expedient/ebot';

const client = new EbotClient({
  apiKey: 'your-api-key'
});

// List MCP servers
const servers = await client.mcpServers.list();
console.log(servers);

// Create a thread
const thread = await client.threads.create({
  projectId: 'proj-123',
  title: 'My Conversation'
});

// Send a message
const response = await client.threads.sendMessage(
  thread.id,
  'Hello, ebot!'
);

// Get quota usage
const usage = await client.quotas.getUsage();
console.log(`Used ${usage.limits[0].used}/${usage.limits[0].limit}`);
```

## API Reference

### Client Configuration

```typescript
const client = new EbotClient({
  apiKey: 'your-api-key',
  baseUrl: 'https://api.expedient.cloud', // optional
  timeout: 30000, // optional, in milliseconds
  fetch: customFetch // optional, custom fetch implementation
});
```

### MCP Servers

```typescript
// List servers
const servers = await client.mcpServers.list();
const workspaceServers = await client.mcpServers.list('ws-123');

// Get server
const server = await client.mcpServers.get('server-id');

// Create server
const newServer = await client.mcpServers.create({
  name: 'My Server',
  manifest: {
    name: 'my-server',
    version: '1.0.0',
    tools: [...]
  },
  workspaceId: 'ws-123' // optional
});

// Update server
const updated = await client.mcpServers.update('server-id', {
  name: 'Updated Name'
});

// Delete server
await client.mcpServers.delete('server-id');
```

### Threads

```typescript
// List threads
const threads = await client.threads.list();
const projectThreads = await client.threads.list('proj-123');

// Create thread
const thread = await client.threads.create({
  projectId: 'proj-123',
  title: 'My Conversation' // optional
});

// Send message
const response = await client.threads.sendMessage(
  'thread-id',
  'Hello!'
);

// Delete thread
await client.threads.delete('thread-id');
```

### Quotas

```typescript
// Get quota usage
const usage = await client.quotas.getUsage();

// Check specific resource
const mcpLimit = usage.limits.find(l => l.resourceType === 'mcp_servers');
if (mcpLimit) {
  console.log(`Used: ${mcpLimit.used}/${mcpLimit.limit}`);
  console.log(`Percentage: ${(mcpLimit.used / mcpLimit.limit) * 100}%`);
}

// Update limits (admin only)
const updated = await client.quotas.updateLimits('user-id', {
  mcp_servers: 20,
  threads: 100
});
```

### Cost Management

```typescript
// Get cost report
const report = await client.costs.getReport({
  rangeDays: 7,
  userId: 'user-123', // optional
  workspaceId: 'ws-123' // optional
});

console.log(`Total: $${report.totalCostUsd.toFixed(2)}`);
console.log(`Tokens: ${report.totalTokens.toLocaleString()}`);

// By model
Object.entries(report.byModel).forEach(([model, cost]) => {
  console.log(`${model}: $${cost.toFixed(2)}`);
});

// Export CSV
const csv = await client.costs.exportCsv(30);
```

### Multi-Region

```typescript
// List regions
const regions = await client.regions.list();
regions.forEach(region => {
  console.log(`${region.name} (${region.location}): ${region.status}`);
});

// Set data residency
const policy = await client.regions.setDataResidency({
  workspaceId: 'ws-123',
  allowedRegions: ['us-east-1', 'us-west-2'],
  primaryRegion: 'us-east-1'
});
```

### Access Control

```typescript
// List policies
const policies = await client.accessControl.listPolicies();

// Create ABAC policy
const policy = await client.accessControl.createPolicy({
  name: 'Business Hours Only',
  rules: [{
    condition: 'request.time.hour >= 9 && request.time.hour <= 17',
    effect: 'allow'
  }],
  priority: 100 // optional
});

// Create service account
const sa = await client.accessControl.createServiceAccount({
  name: 'CI/CD Pipeline',
  description: 'For automated deployments',
  permissions: ['mcp:read', 'threads:write']
});
```

## Error Handling

```typescript
import {
  EbotError,
  AuthenticationError,
  QuotaExceededError,
  ResourceNotFoundError,
  ValidationError
} from '@expedient/ebot';

try {
  const server = await client.mcpServers.create({
    name: 'My Server',
    manifest: {...}
  });
} catch (error) {
  if (error instanceof AuthenticationError) {
    console.error('Invalid API key');
  } else if (error instanceof QuotaExceededError) {
    console.error('Quota exceeded:', error.resourceType);
  } else if (error instanceof ValidationError) {
    console.error('Validation failed:', error.field);
  } else if (error instanceof ResourceNotFoundError) {
    console.error('Resource not found');
  } else if (error instanceof EbotError) {
    console.error('API error:', error.message);
  }
}
```

## Advanced Usage

### React Integration

```tsx
import { EbotClient } from '@expedient/ebot';
import { useState, useEffect } from 'react';

function MCPServerList() {
  const [servers, setServers] = useState([]);
  
  useEffect(() => {
    const client = new EbotClient({ apiKey: process.env.EBOT_API_KEY });
    client.mcpServers.list().then(setServers);
  }, []);
  
  return (
    <ul>
      {servers.map(server => (
        <li key={server.id}>{server.name}</li>
      ))}
    </ul>
  );
}
```

### Next.js API Route

```typescript
// app/api/servers/route.ts
import { EbotClient } from '@expedient/ebot';
import { NextResponse } from 'next/server';

export async function GET() {
  const client = new EbotClient({
    apiKey: process.env.EBOT_API_KEY!
  });
  
  const servers = await client.mcpServers.list();
  return NextResponse.json(servers);
}
```

### Cloudflare Worker

```typescript
import { EbotClient } from '@expedient/ebot';

export default {
  async fetch(request: Request, env: any) {
    const client = new EbotClient({
      apiKey: env.EBOT_API_KEY
    });
    
    const servers = await client.mcpServers.list();
    return Response.json(servers);
  }
};
```

## TypeScript Types

The SDK is fully typed:

```typescript
import type {
  MCPServer,
  Thread,
  Agent,
  Workspace,
  QuotaUsage,
  CostReport,
  Region
} from '@expedient/ebot';

const server: MCPServer = await client.mcpServers.get('id');
const usage: QuotaUsage = await client.quotas.getUsage();
```

## Development

```bash
# Install dependencies
pnpm install

# Build
pnpm build

# Run tests
pnpm test

# Type check
pnpm typecheck

# Lint
pnpm lint

# Format
pnpm format
```

## Requirements

- Node.js 16+ (or any environment with `fetch` support)
- TypeScript 5+ (for development)

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Support

- **Documentation**: https://docs.expedient.cloud/ebot
- **GitHub**: https://github.com/chad-atexpedient/ebot
- **Issues**: https://github.com/chad-atexpedient/ebot/issues

## Related

- [Python SDK](../python/)
- [CLI Tool](../cli/)
- [API Documentation](https://docs.expedient.cloud/ebot/api)
