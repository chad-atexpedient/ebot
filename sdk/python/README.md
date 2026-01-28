# ebot Python SDK

Official Python client library for the [ebot platform](https://github.com/chad-atexpedient/ebot).

## Features

- ✅ **Async & Sync** - Full support for both async/await and synchronous code
- ✅ **Type Hints** - Complete type annotations for better IDE support
- ✅ **Auto-Retry** - Automatic retries with exponential backoff
- ✅ **Rate Limiting** - Built-in rate limit handling
- ✅ **Comprehensive** - Coverage of all ebot APIs
- ✅ **Well-Tested** - High test coverage and battle-tested
- ✅ **Easy to Use** - Intuitive API design

## Installation

```bash
pip install ebot
```

## Quick Start

### Async Usage (Recommended)

```python
import asyncio
from ebot import AsyncEbotClient

async def main():
    # Initialize client
    client = AsyncEbotClient(api_key="your-api-key")
    
    # List MCP servers
    servers = await client.mcp_servers.list()
    for server in servers:
        print(f"Server: {server.name}, Status: {server.status}")
    
    # Create a new thread
    thread = await client.threads.create(project_id="proj-123")
    
    # Send a message
    response = await client.threads.send_message(
        thread_id=thread.id,
        content="Hello, ebot!"
    )
    
    # Get quota usage
    usage = await client.quotas.get_usage()
    print(f"MCP Servers: {usage.get_limit('mcp_servers').used}/{usage.get_limit('mcp_servers').limit}")
    
    # Close the client
    await client.close()

asyncio.run(main())
```

### Sync Usage

```python
from ebot import EbotClient

# Initialize client
client = EbotClient(api_key="your-api-key")

# List MCP servers
servers = client.mcp_servers.list()
for server in servers:
    print(f"Server: {server.name}")

# Create a thread
thread = client.threads.create(project_id="proj-123")

# Close the client
client.close()
```

### Context Manager

```python
# Async
async with AsyncEbotClient(api_key="your-key") as client:
    servers = await client.mcp_servers.list()

# Sync
with EbotClient(api_key="your-key") as client:
    servers = client.mcp_servers.list()
```

## API Reference

### MCP Servers

```python
# List servers
servers = await client.mcp_servers.list()
servers = await client.mcp_servers.list(workspace_id="ws-123")

# Get server
server = await client.mcp_servers.get("server-id")

# Create server
server = await client.mcp_servers.create(
    name="My Server",
    manifest={
        "name": "my-server",
        "version": "1.0.0",
        "tools": [...]
    }
)

# Update server
server = await client.mcp_servers.update(
    "server-id",
    name="Updated Name"
)

# Delete server
await client.mcp_servers.delete("server-id")
```

### Threads

```python
# List threads
threads = await client.threads.list()
threads = await client.threads.list(project_id="proj-123")

# Create thread
thread = await client.threads.create(
    project_id="proj-123",
    title="My Conversation"
)

# Send message
response = await client.threads.send_message(
    thread_id="thread-id",
    content="Hello!"
)

# Delete thread
await client.threads.delete("thread-id")
```

### Quotas

```python
# Get quota usage
usage = await client.quotas.get_usage()

# Check specific resource
mcp_limit = usage.get_limit("mcp_servers")
print(f"Used: {mcp_limit.used}/{mcp_limit.limit}")
print(f"Percentage: {mcp_limit.percentage_used:.1f}%")
print(f"Exceeded: {mcp_limit.is_exceeded}")

# Update limits (admin only)
usage = await client.quotas.update_limits(
    user_id="user-123",
    limits={
        "mcp_servers": 20,
        "threads": 100
    }
)
```

### Cost Management

```python
# Get cost report
report = await client.costs.get_report(range_days=7)
print(f"Total cost: ${report.total_cost_usd:.2f}")
print(f"Total tokens: {report.total_tokens:,}")

# By model
for model, cost in report.by_model.items():
    print(f"{model}: ${cost:.2f}")

# Export CSV
csv_data = await client.costs.export_csv(range_days=30)
```

### Multi-Region

```python
# List regions
regions = await client.regions.list()
for region in regions:
    print(f"{region.name} ({region.location}): {region.status}")

# Set data residency
policy = await client.regions.set_data_residency(
    workspace_id="ws-123",
    allowed_regions=["us-east-1", "us-west-2"],
    primary_region="us-east-1"
)
```

### Access Control

```python
# List policies
policies = await client.access_control.list_policies()

# Create ABAC policy
policy = await client.access_control.create_policy(
    name="Business Hours Only",
    rules=[
        {
            "condition": "request.time.hour >= 9 && request.time.hour <= 17",
            "effect": "allow"
        }
    ],
    priority=100
)

# Create service account
sa = await client.access_control.create_service_account(
    name="CI/CD Pipeline",
    description="For automated deployments",
    permissions=["mcp:read", "threads:write"]
)
```

## Error Handling

```python
from ebot import (
    EbotError,
    AuthenticationError,
    QuotaExceededError,
    ResourceNotFoundError,
    ValidationError
)

try:
    server = await client.mcp_servers.create(
        name="My Server",
        manifest={...}
    )
except AuthenticationError:
    print("Invalid API key")
except QuotaExceededError as e:
    print(f"Quota exceeded: {e.resource_type}")
except ValidationError as e:
    print(f"Validation failed: {e.field}")
except ResourceNotFoundError:
    print("Resource not found")
except EbotError as e:
    print(f"API error: {e.message}")
```

## Advanced Usage

### Custom Base URL

```python
client = AsyncEbotClient(
    api_key="your-key",
    base_url="https://custom.domain.com"
)
```

### Custom Timeout

```python
client = AsyncEbotClient(
    api_key="your-key",
    timeout=60  # 60 seconds
)
```

### Type Checking

The SDK is fully typed and works great with mypy:

```python
from ebot import AsyncEbotClient
from ebot.types import MCPServer

client: AsyncEbotClient = AsyncEbotClient(api_key="key")
server: MCPServer = await client.mcp_servers.get("id")
```

## Development

```bash
# Clone repository
git clone https://github.com/chad-atexpedient/ebot.git
cd ebot/sdk/python

# Install development dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Run type checking
mypy ebot

# Format code
black ebot
ruff ebot
```

## Requirements

- Python 3.8+
- httpx >= 0.24.0
- typing-extensions >= 4.0.0

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Support

- **Documentation**: https://docs.expedient.cloud/ebot
- **GitHub**: https://github.com/chad-atexpedient/ebot
- **Issues**: https://github.com/chad-atexpedient/ebot/issues

## Related

- [TypeScript SDK](../typescript/)
- [CLI Tool](../cli/)
- [API Documentation](https://docs.expedient.cloud/ebot/api)
