// sdk/typescript/src/types/index.ts
/**
 * Type definitions for the ebot TypeScript SDK.
 */

// ============================================================================
// Common Types
// ============================================================================

export interface ListOptions {
  limit?: number;
  offset?: number;
  cursor?: string;
  sort?: string;
  order?: 'asc' | 'desc';
  filter?: Record<string, string>;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
  has_more: boolean;
  next_cursor?: string;
}

export interface LogEntry {
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  metadata?: Record<string, unknown>;
}

export interface LogOptions {
  since?: Date;
  until?: Date;
  limit?: number;
  follow?: boolean;
  level?: 'debug' | 'info' | 'warn' | 'error';
}

// ============================================================================
// MCP Server Types
// ============================================================================

export interface MCPServer {
  id: string;
  name: string;
  description?: string;
  status: MCPServerStatus;
  manifest: MCPManifest;
  workspace_id: string;
  region: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  stopped_at?: string;
  health?: MCPServerHealth;
  resource_usage?: ResourceUsage;
}

export type MCPServerStatus = 
  | 'pending'
  | 'starting'
  | 'running'
  | 'stopping'
  | 'stopped'
  | 'error'
  | 'updating';

export interface MCPManifest {
  version: string;
  name?: string;
  description?: string;
  tools?: MCPTool[];
  resources?: MCPResource[];
  prompts?: MCPPrompt[];
}

export interface MCPTool {
  name: string;
  description: string;
  input_schema: JSONSchema;
}

export interface MCPResource {
  uri: string;
  name: string;
  description?: string;
  mime_type?: string;
}

export interface MCPPrompt {
  name: string;
  description?: string;
  arguments?: MCPPromptArgument[];
}

export interface MCPPromptArgument {
  name: string;
  description?: string;
  required?: boolean;
}

export interface MCPServerHealth {
  healthy: boolean;
  latency_ms: number;
  last_check: string;
  consecutive_failures: number;
}

export interface ResourceUsage {
  cpu_percent: number;
  memory_mb: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
}

export interface CreateMCPServerInput {
  name: string;
  description?: string;
  manifest: MCPManifest;
  workspace_id?: string;
  region?: string;
  environment?: Record<string, string>;
  auto_start?: boolean;
}

export interface UpdateMCPServerInput {
  name?: string;
  description?: string;
  manifest?: Partial<MCPManifest>;
  environment?: Record<string, string>;
}

// ============================================================================
// Thread Types
// ============================================================================

export interface Thread {
  id: string;
  title?: string;
  agent_id?: string;
  workspace_id: string;
  user_id: string;
  status: ThreadStatus;
  message_count: number;
  token_count: number;
  created_at: string;
  updated_at: string;
  last_message_at?: string;
  metadata?: Record<string, string>;
  archived: boolean;
}

export type ThreadStatus = 'active' | 'processing' | 'archived' | 'deleted';

export interface CreateThreadInput {
  title?: string;
  agent_id?: string;
  workspace_id?: string;
  metadata?: Record<string, string>;
  initial_message?: string;
}

export interface Message {
  id: string;
  thread_id: string;
  role: MessageRole;
  content: string;
  attachments?: Attachment[];
  tool_calls?: ToolCall[];
  tool_results?: ToolResult[];
  token_count: number;
  created_at: string;
  metadata?: Record<string, string>;
}

export type MessageRole = 'user' | 'assistant' | 'system' | 'tool';

export interface MessageChunk {
  type: 'content' | 'tool_call' | 'done' | 'error';
  content?: string;
  tool_call?: ToolCall;
  error?: string;
  message_id?: string;
}

export interface Attachment {
  type: 'file' | 'image' | 'url';
  url: string;
  name?: string;
  mime_type?: string;
  size_bytes?: number;
}

export interface ToolCall {
  id: string;
  name: string;
  arguments: Record<string, unknown>;
}

export interface ToolResult {
  tool_call_id: string;
  output: unknown;
  error?: string;
}

// ============================================================================
// Agent Types
// ============================================================================

export interface Agent {
  id: string;
  name: string;
  description?: string;
  system_prompt: string;
  model: string;
  temperature: number;
  max_tokens?: number;
  tools?: string[];
  mcp_servers?: string[];
  knowledge_bases?: string[];
  workspace_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAgentInput {
  name: string;
  description?: string;
  system_prompt: string;
  model?: string;
  temperature?: number;
  max_tokens?: number;
  tools?: string[];
  mcp_servers?: string[];
  knowledge_bases?: string[];
  workspace_id?: string;
}

export interface UpdateAgentInput {
  name?: string;
  description?: string;
  system_prompt?: string;
  model?: string;
  temperature?: number;
  max_tokens?: number;
  tools?: string[];
  mcp_servers?: string[];
  knowledge_bases?: string[];
}

// ============================================================================
// Workspace Types
// ============================================================================

export interface Workspace {
  id: string;
  name: string;
  description?: string;
  slug: string;
  owner_id: string;
  plan: WorkspacePlan;
  settings: WorkspaceSettings;
  created_at: string;
  updated_at: string;
  member_count: number;
}

export type WorkspacePlan = 'free' | 'pro' | 'enterprise';

export interface WorkspaceSettings {
  default_model?: string;
  allowed_models?: string[];
  require_2fa?: boolean;
  allowed_domains?: string[];
  ip_whitelist?: string[];
}

export interface CreateWorkspaceInput {
  name: string;
  description?: string;
  slug?: string;
  settings?: Partial<WorkspaceSettings>;
}

// ============================================================================
// Quota Types
// ============================================================================

export interface Quota {
  id: string;
  name: string;
  description?: string;
  resource_type: QuotaResourceType;
  limit: number;
  current_usage: number;
  reset_period?: QuotaResetPeriod;
  reset_at?: string;
  workspace_id?: string;
  user_id?: string;
}

export type QuotaResourceType =
  | 'tokens'
  | 'requests'
  | 'storage_bytes'
  | 'mcp_servers'
  | 'threads'
  | 'users';

export type QuotaResetPeriod = 'hourly' | 'daily' | 'weekly' | 'monthly' | 'never';

// ============================================================================
// Cost Types
// ============================================================================

export interface CostReport {
  period: {
    start: string;
    end: string;
  };
  total_cost_usd: number;
  cost_by_category: Record<string, number>;
  cost_by_model: Record<string, number>;
  cost_by_user: Record<string, number>;
  daily_breakdown: Array<{
    date: string;
    cost_usd: number;
  }>;
  top_cost_drivers: CostDriver[];
  recommendations: CostRecommendation[];
}

export interface CostDriver {
  category: string;
  description: string;
  cost_usd: number;
  percentage: number;
}

export interface CostRecommendation {
  priority: 'high' | 'medium' | 'low';
  title: string;
  description: string;
  estimated_savings: number;
}

// ============================================================================
// Region Types
// ============================================================================

export interface Region {
  id: string;
  name: string;
  code: string;
  provider: CloudProvider;
  location: string;
  available: boolean;
  latency_ms?: number;
  capabilities: string[];
}

export type CloudProvider = 'aws' | 'gcp' | 'azure' | 'expedient';

// ============================================================================
// Access Control Types
// ============================================================================

export interface Policy {
  id: string;
  name: string;
  description?: string;
  effect: 'allow' | 'deny';
  principals: string[];
  actions: string[];
  resources: string[];
  conditions?: PolicyCondition[];
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface PolicyCondition {
  type: string;
  key: string;
  operator: string;
  value: string | string[];
}

// ============================================================================
// Utility Types
// ============================================================================

export interface JSONSchema {
  type: string;
  properties?: Record<string, JSONSchema>;
  required?: string[];
  items?: JSONSchema;
  description?: string;
  enum?: string[];
  default?: unknown;
  [key: string]: unknown;
}
