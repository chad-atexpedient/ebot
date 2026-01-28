"""
ebot API Client

Provides both synchronous and asynchronous clients for the ebot API.
"""

import asyncio
from typing import Optional, Dict, Any, List
from datetime import datetime
import httpx

from .exceptions import EbotError, AuthenticationError, QuotaExceededError, ResourceNotFoundError
from .resources import MCPServers, Threads, Agents, Workspaces, Quotas, Costs, Regions, AccessControl
from .types import MCPServer, Thread, Agent, Workspace


class AsyncEbotClient:
    """
    Async client for the ebot API.
    
    Args:
        api_key: Your ebot API key
        base_url: Base URL for the ebot API (default: https://api.expedient.cloud)
        timeout: Request timeout in seconds (default: 30)
        
    Example:
        >>> client = AsyncEbotClient(api_key="sk-...")
        >>> servers = await client.mcp_servers.list()
        >>> for server in servers:
        ...     print(server.name)
    """
    
    def __init__(
        self,
        api_key: str,
        base_url: str = "https://api.expedient.cloud",
        timeout: int = 30,
        **kwargs
    ):
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        
        # Create async HTTP client
        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            timeout=timeout,
            headers={
                "Authorization": f"Bearer {api_key}",
                "User-Agent": "ebot-python-sdk/0.1.0",
                "Content-Type": "application/json",
            },
            **kwargs
        )
        
        # Initialize resource managers
        self.mcp_servers = MCPServers(self)
        self.threads = Threads(self)
        self.agents = Agents(self)
        self.workspaces = Workspaces(self)
        self.quotas = Quotas(self)
        self.costs = Costs(self)
        self.regions = Regions(self)
        self.access_control = AccessControl(self)
    
    async def request(
        self,
        method: str,
        path: str,
        **kwargs
    ) -> Any:
        """
        Make an HTTP request to the ebot API.
        
        Args:
            method: HTTP method (GET, POST, PUT, DELETE)
            path: API path (e.g., "/api/mcp-servers")
            **kwargs: Additional arguments for httpx.request
            
        Returns:
            Parsed JSON response
            
        Raises:
            AuthenticationError: Invalid API key
            QuotaExceededError: Quota limit reached
            ResourceNotFoundError: Resource not found
            EbotError: Other API errors
        """
        try:
            response = await self._client.request(method, path, **kwargs)
            
            # Handle error responses
            if response.status_code == 401:
                raise AuthenticationError("Invalid API key")
            elif response.status_code == 429:
                raise QuotaExceededError("Quota exceeded")
            elif response.status_code == 404:
                raise ResourceNotFoundError(f"Resource not found: {path}")
            elif response.status_code >= 400:
                error_data = response.json() if response.content else {}
                raise EbotError(
                    f"API error: {response.status_code}",
                    status_code=response.status_code,
                    response=error_data
                )
            
            # Return parsed JSON
            if response.content:
                return response.json()
            return None
            
        except httpx.HTTPError as e:
            raise EbotError(f"HTTP error: {str(e)}") from e
    
    async def close(self):
        """Close the HTTP client."""
        await self._client.aclose()
    
    async def __aenter__(self):
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()


class EbotClient:
    """
    Synchronous client for the ebot API.
    
    This is a wrapper around AsyncEbotClient that runs async methods
    synchronously using asyncio.
    
    Args:
        api_key: Your ebot API key
        base_url: Base URL for the ebot API
        timeout: Request timeout in seconds
        
    Example:
        >>> client = EbotClient(api_key="sk-...")
        >>> servers = client.mcp_servers.list()
        >>> for server in servers:
        ...     print(server.name)
    """
    
    def __init__(self, api_key: str, **kwargs):
        self._async_client = AsyncEbotClient(api_key, **kwargs)
        self._loop = None
    
    def _get_loop(self):
        """Get or create event loop."""
        if self._loop is None:
            try:
                self._loop = asyncio.get_event_loop()
            except RuntimeError:
                self._loop = asyncio.new_event_loop()
                asyncio.set_event_loop(self._loop)
        return self._loop
    
    def _run_async(self, coro):
        """Run async coroutine synchronously."""
        loop = self._get_loop()
        return loop.run_until_complete(coro)
    
    def request(self, method: str, path: str, **kwargs) -> Any:
        """Make a synchronous HTTP request."""
        return self._run_async(self._async_client.request(method, path, **kwargs))
    
    @property
    def mcp_servers(self):
        return self._async_client.mcp_servers
    
    @property
    def threads(self):
        return self._async_client.threads
    
    @property
    def agents(self):
        return self._async_client.agents
    
    @property
    def workspaces(self):
        return self._async_client.workspaces
    
    @property
    def quotas(self):
        return self._async_client.quotas
    
    @property
    def costs(self):
        return self._async_client.costs
    
    @property
    def regions(self):
        return self._async_client.regions
    
    @property
    def access_control(self):
        return self._async_client.access_control
    
    def close(self):
        """Close the client."""
        self._run_async(self._async_client.close())
    
    def __enter__(self):
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
