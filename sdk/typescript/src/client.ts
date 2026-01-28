/**
 * ebot TypeScript SDK
 * 
 * Official TypeScript/JavaScript client for the ebot platform.
 * Works in Node.js, browsers, and edge runtimes.
 */

import {
  MCPServers,
  Threads,
  Agents,
  Workspaces,
  Quotas,
  Costs,
  Regions,
  AccessControl,
} from './resources';
import { EbotError, AuthenticationError, QuotaExceededError, ResourceNotFoundError } from './errors';

export interface EbotClientConfig {
  /** Your ebot API key */
  apiKey: string;
  /** Base URL for the ebot API */
  baseUrl?: string;
  /** Request timeout in milliseconds */
  timeout?: number;
  /** Custom fetch implementation */
  fetch?: typeof fetch;
}

export class EbotClient {
  private apiKey: string;
  private baseUrl: string;
  private timeout: number;
  private fetchImpl: typeof fetch;

  // Resource managers
  public readonly mcpServers: MCPServers;
  public readonly threads: Threads;
  public readonly agents: Agents;
  public readonly workspaces: Workspaces;
  public readonly quotas: Quotas;
  public readonly costs: Costs;
  public readonly regions: Regions;
  public readonly accessControl: AccessControl;

  constructor(config: EbotClientConfig) {
    this.apiKey = config.apiKey;
    this.baseUrl = (config.baseUrl || 'https://api.expedient.cloud').replace(/\/$/, '');
    this.timeout = config.timeout || 30000;
    this.fetchImpl = config.fetch || globalThis.fetch;

    if (!this.apiKey) {
      throw new Error('API key is required');
    }

    // Initialize resource managers
    this.mcpServers = new MCPServers(this);
    this.threads = new Threads(this);
    this.agents = new Agents(this);
    this.workspaces = new Workspaces(this);
    this.quotas = new Quotas(this);
    this.costs = new Costs(this);
    this.regions = new Regions(this);
    this.accessControl = new AccessControl(this);
  }

  /**
   * Make an HTTP request to the ebot API.
   */
  async request<T = any>(
    method: string,
    path: string,
    options: {
      params?: Record<string, string>;
      body?: any;
      headers?: Record<string, string>;
    } = {}
  ): Promise<T> {
    const url = new URL(path, this.baseUrl);
    
    // Add query parameters
    if (options.params) {
      Object.entries(options.params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
      });
    }

    // Build headers
    const headers: Record<string, string> = {
      'Authorization': `Bearer ${this.apiKey}`,
      'Content-Type': 'application/json',
      'User-Agent': 'ebot-typescript-sdk/0.1.0',
      ...options.headers,
    };

    // Build request options
    const requestOptions: RequestInit = {
      method,
      headers,
      signal: AbortSignal.timeout(this.timeout),
    };

    // Add body for POST/PUT/PATCH
    if (options.body && ['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
      requestOptions.body = JSON.stringify(options.body);
    }

    try {
      const response = await this.fetchImpl(url.toString(), requestOptions);

      // Handle error responses
      if (response.status === 401) {
        throw new AuthenticationError('Invalid API key');
      } else if (response.status === 429) {
        throw new QuotaExceededError('Quota exceeded');
      } else if (response.status === 404) {
        throw new ResourceNotFoundError(`Resource not found: ${path}`);
      } else if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new EbotError(
          `API error: ${response.status}`,
          response.status,
          errorData
        );
      }

      // Parse and return JSON
      if (response.headers.get('content-type')?.includes('application/json')) {
        return await response.json();
      }

      return null as T;
    } catch (error) {
      if (error instanceof EbotError) {
        throw error;
      }
      throw new EbotError(`Request failed: ${(error as Error).message}`);
    }
  }

  /**
   * GET request
   */
  async get<T = any>(path: string, params?: Record<string, string>): Promise<T> {
    return this.request<T>('GET', path, { params });
  }

  /**
   * POST request
   */
  async post<T = any>(path: string, body?: any): Promise<T> {
    return this.request<T>('POST', path, { body });
  }

  /**
   * PUT request
   */
  async put<T = any>(path: string, body?: any): Promise<T> {
    return this.request<T>('PUT', path, { body });
  }

  /**
   * DELETE request
   */
  async delete<T = any>(path: string): Promise<T> {
    return this.request<T>('DELETE', path);
  }
}
