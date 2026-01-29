# sdk/python/ebot/exceptions.py
"""Exception classes for the ebot SDK."""

from typing import Optional, Dict, List, Any


class EbotError(Exception):
    """Base exception for all ebot errors."""
    
    def __init__(
        self,
        message: str,
        status_code: Optional[int] = None,
        response: Optional[Dict[str, Any]] = None
    ):
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.response = response or {}
        self.request_id = self.response.get("request_id")
    
    def __str__(self) -> str:
        if self.status_code:
            return f"[{self.status_code}] {self.message}"
        return self.message
    
    def __repr__(self) -> str:
        return f"{self.__class__.__name__}(message={self.message!r}, status_code={self.status_code})"


class AuthenticationError(EbotError):
    """
    Invalid or expired API key.
    
    Raised when:
    - API key is missing or malformed
    - API key has been revoked
    - API key has expired
    """
    pass


class AuthorizationError(EbotError):
    """
    Insufficient permissions for the requested operation.
    
    Raised when:
    - User lacks required role/permission
    - Resource access is denied
    - ABAC policy denies the request
    """
    pass


class RateLimitError(EbotError):
    """
    Rate limit exceeded.
    
    Raised when:
    - Per-user rate limit exceeded
    - Per-IP rate limit exceeded
    - Global rate limit exceeded
    """
    
    def __init__(
        self,
        message: str,
        retry_after: Optional[int] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.retry_after = retry_after
    
    def __str__(self) -> str:
        base = super().__str__()
        if self.retry_after:
            return f"{base} (retry after {self.retry_after}s)"
        return base


class QuotaExceededError(EbotError):
    """
    Resource quota exceeded.
    
    Raised when:
    - Token quota exceeded
    - Storage quota exceeded
    - MCP server limit reached
    - User limit reached
    """
    
    def __init__(
        self,
        message: str,
        quota_type: Optional[str] = None,
        current_usage: Optional[float] = None,
        limit: Optional[float] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.quota_type = quota_type
        self.current_usage = current_usage
        self.limit = limit


class ValidationError(EbotError):
    """
    Request validation failed.
    
    Raised when:
    - Required fields are missing
    - Field values are invalid
    - Request format is incorrect
    """
    
    def __init__(
        self,
        message: str,
        errors: Optional[List[Dict[str, str]]] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.errors = errors or []
    
    def __str__(self) -> str:
        base = super().__str__()
        if self.errors:
            error_details = "; ".join(
                f"{e.get('field', 'unknown')}: {e.get('message', 'invalid')}"
                for e in self.errors
            )
            return f"{base} - {error_details}"
        return base


class NotFoundError(EbotError):
    """
    Resource not found.
    
    Raised when:
    - Requested resource doesn't exist
    - Resource has been deleted
    - Invalid resource ID format
    """
    
    def __init__(
        self,
        message: str,
        resource_type: Optional[str] = None,
        resource_id: Optional[str] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.resource_type = resource_type
        self.resource_id = resource_id


class ConflictError(EbotError):
    """
    Resource conflict.
    
    Raised when:
    - Resource already exists
    - Concurrent modification detected
    - Version mismatch
    """
    pass


class ServerError(EbotError):
    """
    Server-side error (5xx).
    
    Raised when:
    - Internal server error
    - Service temporarily unavailable
    - Backend service failure
    """
    pass


class NetworkError(EbotError):
    """
    Network connectivity error.
    
    Raised when:
    - Connection refused
    - DNS resolution failed
    - Network unreachable
    """
    pass


class TimeoutError(EbotError):
    """
    Request timed out.
    
    Raised when:
    - Connection timeout
    - Read timeout
    - Overall request timeout
    """
    
    def __init__(
        self,
        message: str,
        timeout_seconds: Optional[float] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.timeout_seconds = timeout_seconds


class CircuitBreakerError(EbotError):
    """
    Circuit breaker is open.
    
    Raised when:
    - Too many consecutive failures
    - Service is temporarily blocked
    """
    
    def __init__(
        self,
        message: str,
        recovery_time: Optional[float] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.recovery_time = recovery_time


def raise_for_status(status_code: int, response_data: Dict[str, Any]) -> None:
    """
    Raise appropriate exception based on HTTP status code.
    
    Args:
        status_code: HTTP response status code
        response_data: Parsed response JSON
        
    Raises:
        Appropriate EbotError subclass
    """
    message = response_data.get("message", f"HTTP {status_code}")
    
    if status_code == 400:
        raise ValidationError(
            message,
            errors=response_data.get("errors"),
            status_code=status_code,
            response=response_data
        )
    
    elif status_code == 401:
        raise AuthenticationError(
            message,
            status_code=status_code,
            response=response_data
        )
    
    elif status_code == 403:
        raise AuthorizationError(
            message,
            status_code=status_code,
            response=response_data
        )
    
    elif status_code == 404:
        raise NotFoundError(
            message,
            resource_type=response_data.get("resource_type"),
            resource_id=response_data.get("resource_id"),
            status_code=status_code,
            response=response_data
        )
    
    elif status_code == 409:
        raise ConflictError(
            message,
            status_code=status_code,
            response=response_data
        )
    
    elif status_code == 422:
        raise ValidationError(
            message,
            errors=response_data.get("errors"),
            status_code=status_code,
            response=response_data
        )
    
    elif status_code == 429:
        raise RateLimitError(
            message,
            retry_after=response_data.get("retry_after"),
            status_code=status_code,
            response=response_data
        )
    
    elif status_code == 402:
        raise QuotaExceededError(
            message,
            quota_type=response_data.get("quota_type"),
            current_usage=response_data.get("current_usage"),
            limit=response_data.get("limit"),
            status_code=status_code,
            response=response_data
        )
    
    elif 500 <= status_code < 600:
        raise ServerError(
            message,
            status_code=status_code,
            response=response_data
        )
    
    else:
        raise EbotError(
            message,
            status_code=status_code,
            response=response_data
        )
