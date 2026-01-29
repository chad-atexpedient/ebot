/**
 * ebot TypeScript SDK Client
 * @packageDocumentation
 */

import { EbotError, AuthenticationError, RateLimitError, ValidationError, NetworkError, ServerError } from './errors';
import type { RequestOptions, ClientConfig, RetryConfig } from './types';

/**
 * Default retry configuration
 */
const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 30000,
  retryableStatusCodes: [429, 500, 502, 503, 504],
};

/**
 * Default client configuration
 */
const DEFAULT_CONFIG: Partial<ClientConfig> = {
  baseUrl: 'https://api.expedient.cloud',
  timeout: 30000,
  retryConfig: DEFAULT_RETRY_CONFIG,
};

/**
 * HTTP client for ebot API with retry logic and error handling
 */
export class EbotClient {
  private apiKey: string;
  private baseUrl: string;
  private timeout: number;
  private retryConfig: RetryConfig;
  private tenantId?: string;
  private headers: Record<string, string>;

  constructor(config: ClientConfig) {
    if (!config.apiKey) {
      throw new Error('API key is required');
    }

    this.apiKey = config.apiKey;
    this.baseUrl = config.baseUrl || DEFAULT_CONFIG.baseUrl!;
    this.timeout = config.timeout || DEFAULT_CONFIG.timeout!;
    this.retryConfig = { ...DEFAULT_RETRY_CONFIG, ...config.retryConfig };
    this.tenantId = config.tenantId;
    this.headers = {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${this.apiKey}`,
      'User-Agent': 'ebot-sdk-typescript/1.0.0',
      ...config.headers,
    };

    if (this.tenantId) {
      this.headers['X-Tenant-ID'] = this.tenantId;
    }
  }

  /**
   * Make an HTTP request with retry logic
   */
  async request<T>(
    method: string,
    path: string,
    options: RequestOptions = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    let lastError: Error | null = null;

    for (let attempt = 0; attempt <= this.retryConfig.maxRetries; attempt++) {
      try {
        const response = await this.makeRequest(method, url, options);
        return await this.handleResponse<T>(response);
      } catch (error) {
        lastError = error as Error;

        if (!this.shouldRetry(error as Error, attempt)) {
          throw error;
        }

        const delay = this.calculateDelay(attempt);
        await this.sleep(delay);
      }
    }

    throw lastError;
  }

  /**
   * Make the actual HTTP request
   */
  private async makeRequest(
    method: string,
    url: string,
    options: RequestOptions
  ): Promise<Response> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const fetchOptions: RequestInit = {
        method,
        headers: { ...this.headers, ...options.headers },
        signal: controller.signal,
      };

      if (options.body) {
        fetchOptions.body = JSON.stringify(options.body);
      }

      if (options.params) {
        const searchParams = new URLSearchParams();
        for (const [key, value] of Object.entries(options.params)) {
          if (value !== undefined && value !== null) {
            searchParams.append(key, String(value));
          }
        }
        const queryString = searchParams.toString();
        if (queryString) {
          url = `${url}?${queryString}`;
        }
      }

      return await fetch(url, fetchOptions);
    } finally {
      clearTimeout(timeoutId);
    }
  }

  /**
   * Handle the HTTP response
   */
  private async handleResponse<T>(response: Response): Promise<T> {
    if (response.ok) {
      if (response.status === 204) {
        return undefined as T;
      }
      return response.json() as Promise<T>;
    }

    // Handle error responses
    let errorData: any;
    try {
      errorData = await response.json();
    } catch {
      errorData = { message: response.statusText };
    }

    const message = errorData.message || `HTTP ${response.status}`;
    const requestId = errorData.requestId;

    switch (response.status) {
      case 401:
        throw new AuthenticationError(message, response.status, errorData, requestId);
      case 429:
        const retryAfter = parseInt(response.headers.get('Retry-After') || '60', 10);
        throw new RateLimitError(message, response.status, errorData, requestId, retryAfter);
      case 400:
      case 422:
        throw new ValidationError(message, response.status, errorData, requestId, errorData.errors);
      case 500:
      case 502:
      case 503:
      case 504:
        throw new ServerError(message, response.status, errorData, requestId);
      default:
        throw new EbotError(message, response.status, errorData, requestId);
    }
  }

  /**
   * Determine if the request should be retried
   */
  private shouldRetry(error: Error, attempt: number): boolean {
    if (attempt >= this.retryConfig.maxRetries) {
      return false;
    }

    if (error instanceof RateLimitError) {
      return true;
    }

    if (error instanceof ServerError) {
      return true;
    }

    if (error instanceof NetworkError) {
      return true;
    }

    if (error.name === 'AbortError') {
      return true; // Timeout, can retry
    }

    return false;
  }

  /**
   * Calculate delay for retry with exponential backoff and jitter
   */
  private calculateDelay(attempt: number): number {
    const exponentialDelay = this.retryConfig.baseDelay * Math.pow(2, attempt);
    const cappedDelay = Math.min(exponentialDelay, this.retryConfig.maxDelay);
    
    // Add jitter (±25%)
    const jitter = cappedDelay * 0.25 * (Math.random() * 2 - 1);
    return Math.floor(cappedDelay + jitter);
  }

  /**
   * Sleep for a specified duration
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Convenience methods

  /**
   * GET request
   */
  async get<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('GET', path, options);
  }

  /**
   * POST request
   */
  async post<T>(path: string, body?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('POST', path, { ...options, body });
  }

  /**
   * PUT request
   */
  async put<T>(path: string, body?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('PUT', path, { ...options, body });
  }

  /**
   * PATCH request
   */
  async patch<T>(path: string, body?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>('PATCH', path, { ...options, body });
  }

  /**
   * DELETE request
   */
  async delete<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('DELETE', path, options);
  }
}

export default EbotClient;
