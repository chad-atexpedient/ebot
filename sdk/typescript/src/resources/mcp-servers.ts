// sdk/typescript/src/resources/mcp-servers.ts
/**
 * MCP Servers resource for managing Model Context Protocol servers.
 */

import { EbotClient } from '../client';
import {
  MCPServer,
  CreateMCPServerInput,
  UpdateMCPServerInput,
  ListOptions,
  PaginatedResponse,
  LogEntry,
  LogOptions,
} from '../types';

export class MCPServers {
  constructor(private client: EbotClient) {}

  /**
   * List all MCP servers.
   * 
   * @param options - Pagination and filter options
   * @returns Paginated list of MCP servers
   * 
   * @example
   * ```typescript
   * const servers = await client.mcpServers.list({ limit: 10 });
   * for (const server of servers.items) {
   *   console.log(server.name, server.status);
   * }
   * ```
   */
  async list(options?: ListOptions): Promise<PaginatedResponse<MCPServer>> {
    return this.client.get<PaginatedResponse<MCPServer>>('/api/mcp-servers', {
      params: options,
    });
  }

  /**
   * Get a specific MCP server by ID.
   * 
   * @param id - The MCP server ID
   * @returns The MCP server details
   * 
   * @example
   * ```typescript
   * const server = await client.mcpServers.get('mcp_123');
   * console.log(server.name, server.status);
   * ```
   */
  async get(id: string): Promise<MCPServer> {
    return this.client.get<MCPServer>(`/api/mcp-servers/${id}`);
  }

  /**
   * Create a new MCP server.
   * 
   * @param data - The MCP server configuration
   * @returns The created MCP server
   * 
   * @example
   * ```typescript
   * const server = await client.mcpServers.create({
   *   name: 'my-server',
   *   manifest: {
   *     version: '1.0',
   *     tools: [{ name: 'search', description: 'Search the web' }],
   *   },
   * });
   * ```
   */
  async create(data: CreateMCPServerInput): Promise<MCPServer> {
    return this.client.post<MCPServer>('/api/mcp-servers', { json: data });
  }

  /**
   * Update an existing MCP server.
   * 
   * @param id - The MCP server ID
   * @param data - The fields to update
   * @returns The updated MCP server
   */
  async update(id: string, data: UpdateMCPServerInput): Promise<MCPServer> {
    return this.client.patch<MCPServer>(`/api/mcp-servers/${id}`, { json: data });
  }

  /**
   * Delete an MCP server.
   * 
   * @param id - The MCP server ID
   */
  async delete(id: string): Promise<void> {
    await this.client.delete(`/api/mcp-servers/${id}`);
  }

  /**
   * Start an MCP server.
   * 
   * @param id - The MCP server ID
   * @returns The updated MCP server with running status
   */
  async start(id: string): Promise<MCPServer> {
    return this.client.post<MCPServer>(`/api/mcp-servers/${id}/start`);
  }

  /**
   * Stop an MCP server.
   * 
   * @param id - The MCP server ID
   * @returns The updated MCP server with stopped status
   */
  async stop(id: string): Promise<MCPServer> {
    return this.client.post<MCPServer>(`/api/mcp-servers/${id}/stop`);
  }

  /**
   * Restart an MCP server.
   * 
   * @param id - The MCP server ID
   * @returns The updated MCP server
   */
  async restart(id: string): Promise<MCPServer> {
    return this.client.post<MCPServer>(`/api/mcp-servers/${id}/restart`);
  }

  /**
   * Get MCP server logs.
   * 
   * @param id - The MCP server ID
   * @param options - Log filtering options
   * @returns Async iterator of log entries
   * 
   * @example
   * ```typescript
   * for await (const log of client.mcpServers.logs('mcp_123', { follow: true })) {
   *   console.log(log.timestamp, log.message);
   * }
   * ```
   */
  async *logs(id: string, options?: LogOptions): AsyncIterable<LogEntry> {
    const params = {
      since: options?.since?.toISOString(),
      until: options?.until?.toISOString(),
      limit: options?.limit,
      follow: options?.follow,
    };

    if (options?.follow) {
      // Use SSE for streaming logs
      const stream = this.client.streamSSE(`/api/mcp-servers/${id}/logs`, { params });
      for await (const event of stream) {
        yield event as LogEntry;
      }
    } else {
      // Use regular request for historical logs
      const response = await this.client.get<{ logs: LogEntry[] }>(
        `/api/mcp-servers/${id}/logs`,
        { params }
      );
      for (const log of response.logs) {
        yield log;
      }
    }
  }

  /**
   * Get MCP server health status.
   * 
   * @param id - The MCP server ID
   * @returns Health check result
   */
  async health(id: string): Promise<{
    healthy: boolean;
    latency_ms: number;
    last_check: string;
    details: Record<string, unknown>;
  }> {
    return this.client.get(`/api/mcp-servers/${id}/health`);
  }

  /**
   * Get MCP server metrics.
   * 
   * @param id - The MCP server ID
   * @param period - Time period (1h, 24h, 7d, 30d)
   * @returns Server metrics
   */
  async metrics(id: string, period: '1h' | '24h' | '7d' | '30d' = '24h'): Promise<{
    requests_total: number;
    requests_per_minute: number;
    avg_latency_ms: number;
    p95_latency_ms: number;
    error_rate: number;
    uptime_percentage: number;
  }> {
    return this.client.get(`/api/mcp-servers/${id}/metrics`, {
      params: { period },
    });
  }

  /**
   * Invoke a tool on an MCP server.
   * 
   * @param id - The MCP server ID
   * @param toolName - Name of the tool to invoke
   * @param input - Tool input parameters
   * @returns Tool execution result
   */
  async invokeTool(
    id: string,
    toolName: string,
    input: Record<string, unknown>
  ): Promise<{
    output: unknown;
    execution_time_ms: number;
  }> {
    return this.client.post(`/api/mcp-servers/${id}/tools/${toolName}/invoke`, {
      json: { input },
    });
  }

  /**
   * List available tools on an MCP server.
   * 
   * @param id - The MCP server ID
   * @returns List of available tools
   */
  async listTools(id: string): Promise<{
    tools: Array<{
      name: string;
      description: string;
      input_schema: Record<string, unknown>;
    }>;
  }> {
    return this.client.get(`/api/mcp-servers/${id}/tools`);
  }
}
