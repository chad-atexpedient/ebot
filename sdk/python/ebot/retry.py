# sdk/python/ebot/retry.py
"""Retry logic with exponential backoff for the ebot SDK."""

import asyncio
import random
from functools import wraps
from typing import Callable, Type, Tuple, Optional, Any
import logging

logger = logging.getLogger(__name__)


class RetryConfig:
    """Configuration for retry behavior."""
    
    def __init__(
        self,
        max_retries: int = 3,
        base_delay: float = 1.0,
        max_delay: float = 60.0,
        exponential_base: float = 2.0,
        jitter: bool = True,
        retryable_exceptions: Tuple[Type[Exception], ...] = (),
        retryable_status_codes: Tuple[int, ...] = (429, 500, 502, 503, 504),
    ):
        """
        Initialize retry configuration.
        
        Args:
            max_retries: Maximum number of retry attempts
            base_delay: Initial delay between retries in seconds
            max_delay: Maximum delay between retries in seconds
            exponential_base: Base for exponential backoff calculation
            jitter: Whether to add random jitter to delays
            retryable_exceptions: Tuple of exception types that should trigger retry
            retryable_status_codes: HTTP status codes that should trigger retry
        """
        self.max_retries = max_retries
        self.base_delay = base_delay
        self.max_delay = max_delay
        self.exponential_base = exponential_base
        self.jitter = jitter
        self.retryable_exceptions = retryable_exceptions
        self.retryable_status_codes = retryable_status_codes


def calculate_delay(attempt: int, config: RetryConfig) -> float:
    """
    Calculate delay for a retry attempt using exponential backoff.
    
    Args:
        attempt: Current attempt number (0-indexed)
        config: Retry configuration
        
    Returns:
        Delay in seconds
    """
    # Exponential backoff: base_delay * (exponential_base ^ attempt)
    delay = config.base_delay * (config.exponential_base ** attempt)
    
    # Cap at max_delay
    delay = min(delay, config.max_delay)
    
    # Add jitter (±25% randomization)
    if config.jitter:
        jitter_range = delay * 0.25
        delay = delay + random.uniform(-jitter_range, jitter_range)
    
    return max(0, delay)


def should_retry(
    exception: Exception,
    config: RetryConfig,
    attempt: int,
    response_status: Optional[int] = None
) -> bool:
    """
    Determine if a request should be retried.
    
    Args:
        exception: The exception that was raised
        config: Retry configuration
        attempt: Current attempt number
        response_status: HTTP response status code if available
        
    Returns:
        True if the request should be retried
    """
    # Check if we've exceeded max retries
    if attempt >= config.max_retries:
        return False
    
    # Check if status code is retryable
    if response_status is not None and response_status in config.retryable_status_codes:
        return True
    
    # Check if exception type is retryable
    if isinstance(exception, config.retryable_exceptions):
        return True
    
    # Check for common network exceptions
    from .exceptions import NetworkError, ServerError, RateLimitError
    if isinstance(exception, (NetworkError, ServerError)):
        return True
    
    # RateLimitError should always be retried (with respect to Retry-After)
    if isinstance(exception, RateLimitError):
        return True
    
    return False


def with_retry(config: Optional[RetryConfig] = None):
    """
    Decorator that adds retry logic to async functions.
    
    Args:
        config: Retry configuration (uses defaults if not provided)
        
    Returns:
        Decorated function with retry logic
    """
    if config is None:
        config = RetryConfig()
    
    def decorator(func: Callable) -> Callable:
        @wraps(func)
        async def wrapper(*args, **kwargs) -> Any:
            last_exception = None
            
            for attempt in range(config.max_retries + 1):
                try:
                    return await func(*args, **kwargs)
                except Exception as e:
                    last_exception = e
                    
                    # Get status code if available
                    status_code = getattr(e, 'status_code', None)
                    
                    if not should_retry(e, config, attempt, status_code):
                        raise
                    
                    # Calculate delay
                    delay = calculate_delay(attempt, config)
                    
                    # Check for Retry-After header
                    retry_after = getattr(e, 'retry_after', None)
                    if retry_after is not None:
                        delay = max(delay, float(retry_after))
                    
                    logger.warning(
                        f"Request failed (attempt {attempt + 1}/{config.max_retries + 1}), "
                        f"retrying in {delay:.2f}s: {e}"
                    )
                    
                    await asyncio.sleep(delay)
            
            # If we get here, all retries failed
            raise last_exception
        
        return wrapper
    return decorator


def with_retry_sync(config: Optional[RetryConfig] = None):
    """
    Decorator that adds retry logic to synchronous functions.
    
    Args:
        config: Retry configuration (uses defaults if not provided)
        
    Returns:
        Decorated function with retry logic
    """
    import time
    
    if config is None:
        config = RetryConfig()
    
    def decorator(func: Callable) -> Callable:
        @wraps(func)
        def wrapper(*args, **kwargs) -> Any:
            last_exception = None
            
            for attempt in range(config.max_retries + 1):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    last_exception = e
                    
                    status_code = getattr(e, 'status_code', None)
                    
                    if not should_retry(e, config, attempt, status_code):
                        raise
                    
                    delay = calculate_delay(attempt, config)
                    
                    retry_after = getattr(e, 'retry_after', None)
                    if retry_after is not None:
                        delay = max(delay, float(retry_after))
                    
                    logger.warning(
                        f"Request failed (attempt {attempt + 1}/{config.max_retries + 1}), "
                        f"retrying in {delay:.2f}s: {e}"
                    )
                    
                    time.sleep(delay)
            
            raise last_exception
        
        return wrapper
    return decorator


class RetryState:
    """Tracks retry state for a request."""
    
    def __init__(self, config: RetryConfig):
        self.config = config
        self.attempt = 0
        self.total_delay = 0.0
        self.last_exception: Optional[Exception] = None
    
    def should_retry(self, exception: Exception, status_code: Optional[int] = None) -> bool:
        """Check if another retry should be attempted."""
        self.last_exception = exception
        return should_retry(exception, self.config, self.attempt, status_code)
    
    def get_delay(self) -> float:
        """Get the delay before the next retry."""
        delay = calculate_delay(self.attempt, self.config)
        self.total_delay += delay
        self.attempt += 1
        return delay
    
    def reset(self):
        """Reset the retry state."""
        self.attempt = 0
        self.total_delay = 0.0
        self.last_exception = None
