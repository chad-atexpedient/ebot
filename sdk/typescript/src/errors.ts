/**
 * Error classes for the ebot TypeScript SDK.
 */

export class EbotError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public response?: any
  ) {
    super(message);
    this.name = 'EbotError';
    Object.setPrototypeOf(this, EbotError.prototype);
  }
}

export class AuthenticationError extends EbotError {
  constructor(message: string = 'Authentication failed') {
    super(message, 401);
    this.name = 'AuthenticationError';
    Object.setPrototypeOf(this, AuthenticationError.prototype);
  }
}

export class QuotaExceededError extends EbotError {
  constructor(
    message: string = 'Quota exceeded',
    public resourceType?: string,
    public limit?: number,
    public used?: number
  ) {
    super(message, 429);
    this.name = 'QuotaExceededError';
    Object.setPrototypeOf(this, QuotaExceededError.prototype);
  }
}

export class ResourceNotFoundError extends EbotError {
  constructor(message: string) {
    super(message, 404);
    this.name = 'ResourceNotFoundError';
    Object.setPrototypeOf(this, ResourceNotFoundError.prototype);
  }
}

export class ValidationError extends EbotError {
  constructor(
    message: string,
    public field?: string
  ) {
    super(message, 400);
    this.name = 'ValidationError';
    Object.setPrototypeOf(this, ValidationError.prototype);
  }
}

export class RateLimitError extends EbotError {
  constructor(
    message: string = 'Rate limit exceeded',
    public retryAfter?: number
  ) {
    super(message, 429);
    this.name = 'RateLimitError';
    Object.setPrototypeOf(this, RateLimitError.prototype);
  }
}
