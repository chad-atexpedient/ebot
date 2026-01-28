"""
ebot Python SDK

Official Python client for the ebot platform.
Provides async/sync interfaces for all ebot APIs.

Example:
    >>> from ebot import EbotClient
    >>> client = EbotClient(api_key="your-key")
    >>> servers = await client.mcp_servers.list()
"""

from .client import EbotClient, AsyncEbotClient
from .exceptions import (
    EbotError,
    AuthenticationError,
    QuotaExceededError,
    ResourceNotFoundError,
    ValidationError,
)
from .types import (
    MCPServer,
    Thread,
    Agent,
    Workspace,
    QuotaUsage,
    CostReport,
    Region,
)

__version__ = "0.1.0"
__all__ = [
    "EbotClient",
    "AsyncEbotClient",
    "EbotError",
    "AuthenticationError",
    "QuotaExceededError",
    "ResourceNotFoundError",
    "ValidationError",
    "MCPServer",
    "Thread",
    "Agent",
    "Workspace",
    "QuotaUsage",
    "CostReport",
    "Region",
]
