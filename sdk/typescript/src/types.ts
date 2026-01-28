/**
 * Type definitions for the ebot TypeScript SDK.
 */

export enum ServerStatus {
  PENDING = 'pending',
  RUNNING = 'running',
  STOPPED = 'stopped',
  ERROR = 'error',
}

export enum ResourceType {
  MCP_SERVERS = 'mcp_servers',
  THREADS = 'threads',
  KNOWLEDGE_SETS = 'knowledge_sets',
  KNOWLEDGE_FILES = 'knowledge_files',
  STORAGE_BYTES = 'storage_bytes',
  CPU_CORES = 'cpu_cores',
  MEMORY_BYTES = 'memory_bytes',
  LLM_TOKENS = 'llm_tokens',
  LLM_COST_USD = 'llm_cost_usd',
}

export interface MCPServer {
  id: string;
  name: string;
  status: ServerStatus;
  manifest: Record<string, any>;
  workspaceId?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface Thread {
  id: string;
  projectId: string;
  title?: string;
  createdAt?: string;
  updatedAt?: string;
  messageCount?: number;
}

export interface Agent {
  id: string;
  name: string;
  description?: string;
  tools?: string[];
  createdAt?: string;
}

export interface Workspace {
  id: string;
  name: string;
  description?: string;
  createdAt?: string;
}

export interface QuotaLimit {
  resourceType: string;
  limit: number;
  used: number;
  remaining: number;
  resetAt?: string;
}

export interface QuotaUsage {
  userId: string;
  workspaceId?: string;
  limits: QuotaLimit[];
  tier: string;
}

export interface CostEntry {
  date: string;
  model: string;
  tokens: number;
  costUsd: number;
  userId?: string;
  workspaceId?: string;
}

export interface CostReport {
  startDate: string;
  endDate: string;
  totalCostUsd: number;
  totalTokens: number;
  entries: CostEntry[];
  byModel: Record<string, number>;
  byUser: Record<string, number>;
  byWorkspace: Record<string, number>;
}

export interface Region {
  id: string;
  name: string;
  location: string;
  status: string;
  capabilities: string[];
  endpoint: string;
}

export interface DataResidency {
  workspaceId: string;
  allowedRegions: string[];
  primaryRegion: string;
  crossRegionReplication: boolean;
}

export interface ServiceAccount {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  lastUsedAt?: string;
  permissions: string[];
}

export interface APIKey {
  id: string;
  key: string;
  serviceAccountId: string;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
}

export interface ABACPolicy {
  id: string;
  name: string;
  rules: ABACRule[];
  priority: number;
  enabled: boolean;
}

export interface ABACRule {
  condition: string;
  effect: 'allow' | 'deny';
}

export interface ListResponse<T> {
  items: T[];
  total?: number;
  page?: number;
  pageSize?: number;
}
