/**
 * Resource managers for different ebot API endpoints.
 */

import type { EbotClient } from './client';
import type {
  MCPServer,
  Thread,
  Agent,
  Workspace,
  QuotaUsage,
  CostReport,
  Region,
  DataResidency,
  ServiceAccount,
  ABACPolicy,
  ListResponse,
} from './types';

class BaseResource {
  constructor(protected client: EbotClient) {}
}

export class MCPServers extends BaseResource {
  /**
   * List all MCP servers.
   */
  async list(workspaceId?: string): Promise<MCPServer[]> {
    const params = workspaceId ? { workspaceId } : undefined;
    const response = await this.client.get<ListResponse<MCPServer>>('/api/mcp-servers', params);
    return response.items || [];
  }

  /**
   * Get a specific MCP server.
   */
  async get(serverId: string): Promise<MCPServer> {
    return this.client.get<MCPServer>(`/api/mcp-servers/${serverId}`);
  }

  /**
   * Create a new MCP server.
   */
  async create(data: {
    name: string;
    manifest: Record<string, any>;
    workspaceId?: string;
  }): Promise<MCPServer> {
    return this.client.post<MCPServer>('/api/mcp-servers', data);
  }

  /**
   * Update an MCP server.
   */
  async update(serverId: string, data: Partial<MCPServer>): Promise<MCPServer> {
    return this.client.put<MCPServer>(`/api/mcp-servers/${serverId}`, data);
  }

  /**
   * Delete an MCP server.
   */
  async delete(serverId: string): Promise<void> {
    await this.client.delete(`/api/mcp-servers/${serverId}`);
  }
}

export class Threads extends BaseResource {
  /**
   * List all threads.
   */
  async list(projectId?: string): Promise<Thread[]> {
    const params = projectId ? { projectId } : undefined;
    const response = await this.client.get<ListResponse<Thread>>('/api/threads', params);
    return response.items || [];
  }

  /**
   * Get a specific thread.
   */
  async get(threadId: string): Promise<Thread> {
    return this.client.get<Thread>(`/api/threads/${threadId}`);
  }

  /**
   * Create a new thread.
   */
  async create(data: { projectId: string; title?: string }): Promise<Thread> {
    return this.client.post<Thread>('/api/threads', data);
  }

  /**
   * Send a message to a thread.
   */
  async sendMessage(threadId: string, content: string): Promise<any> {
    return this.client.post(`/api/threads/${threadId}/messages`, { content });
  }

  /**
   * Delete a thread.
   */
  async delete(threadId: string): Promise<void> {
    await this.client.delete(`/api/threads/${threadId}`);
  }
}

export class Agents extends BaseResource {
  /**
   * List all agents.
   */
  async list(): Promise<Agent[]> {
    const response = await this.client.get<ListResponse<Agent>>('/api/agents');
    return response.items || [];
  }

  /**
   * Get a specific agent.
   */
  async get(agentId: string): Promise<Agent> {
    return this.client.get<Agent>(`/api/agents/${agentId}`);
  }

  /**
   * Create a new agent.
   */
  async create(data: {
    name: string;
    description?: string;
    tools?: string[];
  }): Promise<Agent> {
    return this.client.post<Agent>('/api/agents', data);
  }
}

export class Workspaces extends BaseResource {
  /**
   * List all workspaces.
   */
  async list(): Promise<Workspace[]> {
    const response = await this.client.get<ListResponse<Workspace>>('/api/workspaces');
    return response.items || [];
  }

  /**
   * Get a specific workspace.
   */
  async get(workspaceId: string): Promise<Workspace> {
    return this.client.get<Workspace>(`/api/workspaces/${workspaceId}`);
  }
}

export class Quotas extends BaseResource {
  /**
   * Get quota usage.
   */
  async getUsage(userId?: string): Promise<QuotaUsage> {
    const params = userId ? { userId } : undefined;
    return this.client.get<QuotaUsage>('/api/quota/usage', params);
  }

  /**
   * Update quota limits (admin only).
   */
  async updateLimits(userId: string, limits: Record<string, number>): Promise<QuotaUsage> {
    return this.client.put<QuotaUsage>('/api/quota/update', { userId, limits });
  }
}

export class Costs extends BaseResource {
  /**
   * Get cost report.
   */
  async getReport(options?: {
    rangeDays?: number;
    userId?: string;
    workspaceId?: string;
  }): Promise<CostReport> {
    const params: Record<string, string> = {
      range: `${options?.rangeDays || 7}d`,
    };
    if (options?.userId) params.userId = options.userId;
    if (options?.workspaceId) params.workspaceId = options.workspaceId;

    return this.client.get<CostReport>('/api/costs/report', params);
  }

  /**
   * Export cost data as CSV.
   */
  async exportCsv(rangeDays: number = 30): Promise<string> {
    return this.client.get<string>('/api/costs/export', {
      range: `${rangeDays}d`,
      format: 'csv',
    });
  }
}

export class Regions extends BaseResource {
  /**
   * List all available regions.
   */
  async list(): Promise<Region[]> {
    const response = await this.client.get<ListResponse<Region>>('/api/regions');
    return response.items || [];
  }

  /**
   * Get region details.
   */
  async get(regionId: string): Promise<Region> {
    return this.client.get<Region>(`/api/regions/${regionId}`);
  }

  /**
   * Set data residency policy.
   */
  async setDataResidency(data: {
    workspaceId: string;
    allowedRegions: string[];
    primaryRegion: string;
  }): Promise<DataResidency> {
    return this.client.put<DataResidency>('/api/regions/data-residency', data);
  }
}

export class AccessControl extends BaseResource {
  /**
   * List all ABAC policies.
   */
  async listPolicies(): Promise<ABACPolicy[]> {
    const response = await this.client.get<ListResponse<ABACPolicy>>('/api/access-control/policies');
    return response.items || [];
  }

  /**
   * Create a new ABAC policy.
   */
  async createPolicy(data: {
    name: string;
    rules: Array<{ condition: string; effect: 'allow' | 'deny' }>;
    priority?: number;
  }): Promise<ABACPolicy> {
    return this.client.post<ABACPolicy>('/api/access-control/policies', data);
  }

  /**
   * Create a service account.
   */
  async createServiceAccount(data: {
    name: string;
    description?: string;
    permissions?: string[];
  }): Promise<ServiceAccount> {
    return this.client.post<ServiceAccount>('/api/access-control/service-accounts', data);
  }
}
