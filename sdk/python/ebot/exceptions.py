"""
Exception classes for the ebot Python SDK.
"""

from typing import Optional, Dict, Any


class EbotError(Exception):
    """Base exception for all ebot SDK errors."""
    
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
    
    def __str__(self):
        if self.status_code:
            return f"{self.message} (status: {self.status_code})"
        return self.message


class AuthenticationError(EbotError):
    """Raised when API key is invalid or missing."""
    pass


class QuotaExceededError(EbotError):
    """Raised when a quota limit is exceeded."""
    
    def __init__(
        self,
        message: str = "Quota exceeded",
        resource_type: Optional[str] = None,
        limit: Optional[int] = None,
        used: Optional[int] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.resource_type = resource_type
        self.limit = limit
        self.used = used
    
    def __str__(self):
        if self.resource_type and self.limit and self.used:
            return f"{self.message}: {self.resource_type} limit {self.limit}, used {self.used}"
        return self.message


class ResourceNotFoundError(EbotError):
    """Raised when a resource is not found."""
    pass


class ValidationError(EbotError):
    """Raised when request validation fails."""
    
    def __init__(
        self,
        message: str,
        field: Optional[str] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.field = field
    
    def __str__(self):
        if self.field:
            return f"{self.message} (field: {self.field})"
        return self.message


class RateLimitError(EbotError):
    """Raised when rate limit is exceeded."""
    
    def __init__(
        self,
        message: str = "Rate limit exceeded",
        retry_after: Optional[int] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.retry_after = retry_after
    
    def __str__(self):
        if self.retry_after:
            return f"{self.message}. Retry after {self.retry_after} seconds"
        return self.message


class RegionError(EbotError):
    """Raised when region operation fails."""
    pass


class ComplianceError(EbotError):
    """Raised when compliance policy is violated."""
    
    def __init__(
        self,
        message: str,
        policy_type: Optional[str] = None,
        **kwargs
    ):
        super().__init__(message, **kwargs)
        self.policy_type = policy_type


class TimeoutError(EbotError):
    """Raised when request times out."""
    pass


class NetworkError(EbotError):
    """Raised when network error occurs."""
    pass
