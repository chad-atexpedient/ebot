/**
 * ebot SDK Error Classes
 * @packageDocumentation
 */

/**
 * Base error class for all ebot SDK errors
 */
export class EbotError extends Error {
  /** HTTP status code (if applicable) */
  statusCode?: number;
  /** Raw response data */
  response?: any;
  /** Request ID for debugging */
  requestId?: string;

  constructor(
    message: string,
    statusCode?: number,
    response?: any,
    requestId?: string
  ) {
    super(message);
    this.name = 'EbotError';
    this.statusCode = statusCode;
    this.response = response;
    this.requestId = requestId;

    // Maintains proper stack trace for where error was thrown
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, EbotError);
    }
  }

  /**
   * Returns a string representation of the error for logging
   */
  toString(): string {
    let str = `${this.name}: ${this.message}`;
    if (this.statusCode) {
      str += ` (HTTP ${this.statusCode})`;
    }
    if (this.requestId) {
      str += ` [Request ID: ${this.requestId}]`;
    }
    return str;
  }

  /**
   * Converts the error to a JSON object
   */
  toJSON(): Record<string, any> {
    return {
      name: this.name,
      message: this.message,
      statusCode: this.statusCode,
      requestId: this.requestId,
      response: this.response,
    };
  }
}

/**
 * Error thrown when authentication fails (401)
 */
export class AuthenticationError extends EbotError {
  constructor(
    message: string = 'Authentication failed',
    statusCode: number = 401,
    response?: any,
    requestId?: string
  ) {
    super(message, statusCode, response, requestId);
    this.name = 'AuthenticationError';
  }
}

/**
 * Error thrown when rate limit is exceeded (429)
 */
export class RateLimitError extends EbotError {
  /** Seconds to wait before retrying */
  retryAfter: number;
  /** Number of remaining requests */
  remaining?: number;
  /** Unix timestamp when the rate limit resets */
  resetAt?: number;

  constructor(
    message: string = 'Rate limit exceeded',
    statusCode: number = 429,
    response?: any,
    requestId?: string,
    retryAfter: number = 60,
    remaining?: number,
    resetAt?: number
  ) {
    super(message, statusCode, response, requestId);
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
    this.remaining = remaining;
    this.resetAt = resetAt;
  }

  toJSON(): Record<string, any> {
    return {
      ...super.toJSON(),
      retryAfter: this.retryAfter,
      remaining: this.remaining,
      resetAt: this.resetAt,
    };
  }
}

/**
 * Error thrown when request validation fails (400, 422)
 */
export class ValidationError extends EbotError {
  /** Detailed validation errors */
  errors: ValidationErrorDetail[];

  constructor(
    message: string = 'Validation failed',
    statusCode: number = 400,
    response?: any,
    requestId?: string,
    errors: ValidationErrorDetail[] = []
  ) {
    super(message, statusCode, response, requestId);
    this.name = 'ValidationError';
    this.errors = errors;
  }

  toJSON(): Record<string, any> {
    return {
      ...super.toJSON(),
      errors: this.errors,
    };
  }
}

/**
 * Validation error detail
 */
export interface ValidationErrorDetail {
  /** Field that failed validation */
  field: string;
  /** Error message */
  message: string;
  /** Error code */
  code?: string;
}

/**
 * Error thrown when a resource is not found (404)
 */
export class NotFoundError extends EbotError {
  /** Type of resource that was not found */
  resourceType?: string;
  /** ID of the resource that was not found */
  resourceId?: string;

  constructor(
    message: string = 'Resource not found',
    statusCode: number = 404,
    response?: any,
    requestId?: string,
    resourceType?: string,
    resourceId?: string
  ) {
    super(message, statusCode, response, requestId);
    this.name = 'NotFoundError';
    this.resourceType = resourceType;
    this.resourceId = resourceId;
  }
}

/**
 * Error thrown when access is forbidden (403)
 */
export class ForbiddenError extends EbotError {
  /** Action that was attempted */
  action?: string;
  /** Resource that access was denied to */
  resource?: string;

  constructor(
    message: string = 'Access forbidden',
    statusCode: number = 403,
    response?: any,
    requestId?: string,
    action?: string,
    resource?: string
  ) {
    super(message, statusCode, response, requestId);
    this.name = 'ForbiddenError';
    this.action = action;
    this.resource = resource;
  }
}

/**
 * Error thrown when quota is exceeded (402)
 */
export class QuotaExceededError extends EbotError {
  /** Type of quota that was exceeded */
  quotaType?: string;
  /** Current usage */
  currentUsage?: number;
  /** Quota limit */
  limit?: number;

  constructor(
    message: string = 'Quota exceeded',
    statusCode: number = 402,
    response?: any,
    requestId?: string,
    quotaType?: string,
    currentUsage?: number,
    limit?: number
  ) {
    super(message, statusCode, response, requestId);
    this.name = 'QuotaExceededError';
    this.quotaType = quotaType;
    this.currentUsage = currentUsage;
    this.limit = limit;
  }
}

/**
 * Error thrown for server errors (5xx)
 */
export class ServerError extends EbotError {
  constructor(
    message: string = 'Server error',
    statusCode: number = 500,
    response?: any,
    requestId?: string
  ) {
    super(message, statusCode, response, requestId);
    this.name = 'ServerError';
  }
}

/**
 * Error thrown for network-related issues
 */
export class NetworkError extends EbotError {
  /** Original error that caused this */
  cause?: Error;

  constructor(
    message: string = 'Network error',
    cause?: Error
  ) {
    super(message);
    this.name = 'NetworkError';
    this.cause = cause;
  }
}

/**
 * Error thrown when a request times out
 */
export class TimeoutError extends EbotError {
  /** Timeout duration in milliseconds */
  timeout: number;

  constructor(
    message: string = 'Request timed out',
    timeout: number = 30000
  ) {
    super(message);
    this.name = 'TimeoutError';
    this.timeout = timeout;
  }
}

/**
 * Type guard to check if an error is an EbotError
 */
export function isEbotError(error: unknown): error is EbotError {
  return error instanceof EbotError;
}

/**
 * Type guard to check if an error is retryable
 */
export function isRetryableError(error: unknown): boolean {
  if (error instanceof RateLimitError) return true;
  if (error instanceof ServerError) return true;
  if (error instanceof NetworkError) return true;
  if (error instanceof TimeoutError) return true;
  return false;
}
