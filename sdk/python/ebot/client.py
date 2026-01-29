# sdk/python/ebot/client.py
"""HTTP client with retry logic and connection pooling for the ebot SDK."""

import asyncio
import logging
from typing import Optional, Dict, Any, List, Callable, AsyncIterator
from contextlib import asynccontextmanager

import httpx

from .retry import RetryConfig, RetryState, calculate_delay, should_retry
from .exceptions import (
    EbotError,
    NetworkError,
    TimeoutError,
    raise_for_status,
)

logger = logging.getLogger(__name__)


class AsyncEbotClient:
    """
    Async HTTP client for the ebot API with retry logic and connection pooling.
    
    Features:
    - Automatic retry with exponential backoff
    - Connection pooling with HTTP/2 support
    - Request/response hooks for logging and metrics
    - Configurable timeouts
    - Automatic error handling
    
    Example:
        ```python
        async with AsyncEbotClient(api_key="sk-...") as client:
            servers = await client.request("GET", "/api/mcp-servers")
        ```
    """
    
    def __init__(
        self,
        api_key: str,
        base_url: str = "https://api.expedient.cloud",
        timeout: float = 30.0,
        max_connections: int = 100,
        max_keepalive_connections: int = 20,
        keepalive_expiry: float = 5.0,
        retry_config: Optional[RetryConfig] = None,
        request_hooks: Optional[List[Callable]] = None,
        response_hooks: Optional[List[Callable]] = None,
        **kwargs
    ):
        """
        Initialize the async ebot client.
        
        Args:
            api_key: API key for authentication
            base_url: Base URL for the API
            timeout: Default request timeout in seconds
            max_connections: Maximum number of connections in the pool
            max_keepalive_connections: Maximum keepalive connections
            keepalive_expiry: Keepalive connection expiry in seconds
            retry_config: Configuration for retry behavior
            request_hooks: List of functions to call before each request
            response_hooks: List of functions to call after each response
            **kwargs: Additional arguments passed to httpx.AsyncClient
        """
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.retry_config = retry_config or RetryConfig()
        self._request_hooks = request_hooks or []
        self._response_hooks = response_hooks or []
        
        # Configure connection limits
        limits = httpx.Limits(
            max_connections=max_connections,
            max_keepalive_connections=max_keepalive_connections,
            keepalive_expiry=keepalive_expiry,
        )
        
        # Configure timeouts
        timeout_config = httpx.Timeout(
            connect=5.0,
            read=timeout,
            write=10.0,
            pool=5.0,
        )
        
        # Default headers
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": "ebot-python-sdk/1.0.0",
        }
        
        # Create the httpx client
        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            limits=limits,
            timeout=timeout_config,
            headers=headers,
            http2=True,  # Enable HTTP/2
            **kwargs
        )
        
        self._closed = False
    
    async def __aenter__(self):
        """Async context manager entry."""
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        await self.close()
    
    async def close(self):
        """Close the client and release resources."""
        if not self._closed:
            await self._client.aclose()
            self._closed = True
    
    async def request(
        self,
        method: str,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        json: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        timeout: Optional[float] = None,
        retry_config: Optional[RetryConfig] = None,
    ) -> Dict[str, Any]:
        """
        Make an HTTP request with automatic retry.
        
        Args:
            method: HTTP method (GET, POST, PUT, DELETE, etc.)
            path: API path (e.g., "/api/mcp-servers")
            params: Query parameters
            json: JSON body data
            headers: Additional headers
            timeout: Request-specific timeout override
            retry_config: Request-specific retry config override
            
        Returns:
            Parsed JSON response
            
        Raises:
            EbotError: On API errors
            NetworkError: On network failures
            TimeoutError: On request timeout
        """
        config = retry_config or self.retry_config
        state = RetryState(config)
        
        # Build request kwargs
        request_kwargs: Dict[str, Any] = {
            "method": method,
            "url": path,
        }
        if params:
            request_kwargs["params"] = params
        if json:
            request_kwargs["json"] = json
        if headers:
            request_kwargs["headers"] = headers
        if timeout:
            request_kwargs["timeout"] = timeout
        
        # Call request hooks
        for hook in self._request_hooks:
            await self._call_hook(hook, method, path, request_kwargs)
        
        while True:
            try:
                response = await self._client.request(**request_kwargs)
                
                # Call response hooks
                for hook in self._response_hooks:
                    await self._call_hook(hook, response)
                
                # Handle error responses
                if response.status_code >= 400:
                    try:
                        error_data = response.json()
                    except Exception:
                        error_data = {"message": response.text}
                    
                    # Add retry-after from header if present
                    retry_after = response.headers.get("Retry-After")
                    if retry_after:
                        error_data["retry_after"] = int(retry_after)
                    
                    raise_for_status(response.status_code, error_data)
                
                # Return successful response
                if response.status_code == 204:
                    return {}
                return response.json()
                
            except httpx.ConnectError as e:
                exc = NetworkError(f"Connection failed: {e}")
                if not state.should_retry(exc):
                    raise exc
                    
            except httpx.TimeoutException as e:
                exc = TimeoutError(f"Request timed out: {e}")
                if not state.should_retry(exc):
                    raise exc
                    
            except EbotError as e:
                if not state.should_retry(e, e.status_code):
                    raise
            
            # Calculate delay and wait before retry
            delay = state.get_delay()
            logger.warning(
                f"Request to {path} failed (attempt {state.attempt}), "
                f"retrying in {delay:.2f}s"
            )
            await asyncio.sleep(delay)
    
    async def _call_hook(self, hook: Callable, *args, **kwargs):
        """Call a hook function, handling both sync and async hooks."""
        result = hook(*args, **kwargs)
        if asyncio.iscoroutine(result):
            await result
    
    # Convenience methods
    
    async def get(self, path: str, **kwargs) -> Dict[str, Any]:
        """Make a GET request."""
        return await self.request("GET", path, **kwargs)
    
    async def post(self, path: str, **kwargs) -> Dict[str, Any]:
        """Make a POST request."""
        return await self.request("POST", path, **kwargs)
    
    async def put(self, path: str, **kwargs) -> Dict[str, Any]:
        """Make a PUT request."""
        return await self.request("PUT", path, **kwargs)
    
    async def patch(self, path: str, **kwargs) -> Dict[str, Any]:
        """Make a PATCH request."""
        return await self.request("PATCH", path, **kwargs)
    
    async def delete(self, path: str, **kwargs) -> Dict[str, Any]:
        """Make a DELETE request."""
        return await self.request("DELETE", path, **kwargs)
    
    # Streaming support
    
    @asynccontextmanager
    async def stream(
        self,
        method: str,
        path: str,
        **kwargs
    ) -> AsyncIterator[httpx.Response]:
        """
        Stream a response for large data or SSE.
        
        Example:
            ```python
            async with client.stream("GET", "/api/logs") as response:
                async for line in response.aiter_lines():
                    print(line)
            ```
        """
        async with self._client.stream(method, path, **kwargs) as response:
            yield response
    
    async def stream_sse(
        self,
        path: str,
        **kwargs
    ) -> AsyncIterator[Dict[str, Any]]:
        """
        Stream Server-Sent Events.
        
        Yields:
            Parsed SSE event data as dictionaries
        """
        import json
        
        async with self.stream("GET", path, **kwargs) as response:
            async for line in response.aiter_lines():
                if line.startswith("data: "):
                    data = line[6:]  # Remove "data: " prefix
                    if data.strip():
                        try:
                            yield json.loads(data)
                        except json.JSONDecodeError:
                            yield {"raw": data}


class SyncEbotClient:
    """
    Synchronous HTTP client for the ebot API.
    
    For use in non-async contexts. Wraps AsyncEbotClient.
    """
    
    def __init__(self, *args, **kwargs):
        """Initialize the sync client with same args as AsyncEbotClient."""
        self._async_client = AsyncEbotClient(*args, **kwargs)
    
    def __enter__(self):
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
    
    def close(self):
        """Close the client."""
        asyncio.get_event_loop().run_until_complete(
            self._async_client.close()
        )
    
    def request(self, *args, **kwargs) -> Dict[str, Any]:
        """Make a synchronous request."""
        return asyncio.get_event_loop().run_until_complete(
            self._async_client.request(*args, **kwargs)
        )
    
    def get(self, path: str, **kwargs) -> Dict[str, Any]:
        return self.request("GET", path, **kwargs)
    
    def post(self, path: str, **kwargs) -> Dict[str, Any]:
        return self.request("POST", path, **kwargs)
    
    def put(self, path: str, **kwargs) -> Dict[str, Any]:
        return self.request("PUT", path, **kwargs)
    
    def patch(self, path: str, **kwargs) -> Dict[str, Any]:
        return self.request("PATCH", path, **kwargs)
    
    def delete(self, path: str, **kwargs) -> Dict[str, Any]:
        return self.request("DELETE", path, **kwargs)
