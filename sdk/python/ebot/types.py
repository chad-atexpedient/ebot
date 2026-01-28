"""
Type definitions for the ebot Python SDK.
"""

from typing import Optional, List, Dict, Any
from dataclasses import dataclass
from datetime import datetime
from enum import Enum


class ServerStatus(str, Enum):
    """MCP server status."""
    PENDING = "pending"
    RUNNING = "running"
    STOPPED = "stopped"
    ERROR = "error"


class ResourceType(str, Enum):
    """Quota resource types."""
    MCP_SERVERS = "mcp_servers"
    THREADS = "threads"
    KNOWLEDGE_SETS = "knowledge_sets"
    KNOWLEDGE_FILES = "knowledge_files"
    STORAGE_BYTES = "storage_bytes"
    CPU_CORES = "cpu_cores"
    MEMORY_BYTES = "memory_bytes"
    LLM_TOKENS = "llm_tokens"
    LLM_COST_USD = "llm_cost_usd"


@dataclass
class MCPServer:
    """MCP Server resource."""
    id: str
    name: str
    status: ServerStatus
    manifest: Dict[str, Any]
    workspace_id: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    
    def __post_init__(self):
        if isinstance(self.status, str):
            self.status = ServerStatus(self.status)
        if isinstance(self.created_at, str):
            self.created_at = datetime.fromisoformat(self.created_at.replace("Z", "+00:00"))
        if isinstance(self.updated_at, str):
            self.updated_at = datetime.fromisoformat(self.updated_at.replace("Z", "+00:00"))


@dataclass
class Thread:
    """Conversation thread."""
    id: str
    project_id: str
    title: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    message_count: int = 0
    
    def __post_init__(self):
        if isinstance(self.created_at, str):
            self.created_at = datetime.fromisoformat(self.created_at.replace("Z", "+00:00"))
        if isinstance(self.updated_at, str):
            self.updated_at = datetime.fromisoformat(self.updated_at.replace("Z", "+00:00"))


@dataclass
class Agent:
    """AI Agent."""
    id: str
    name: str
    description: Optional[str] = None
    tools: List[str] = None
    created_at: Optional[datetime] = None
    
    def __post_init__(self):
        if self.tools is None:
            self.tools = []
        if isinstance(self.created_at, str):
            self.created_at = datetime.fromisoformat(self.created_at.replace("Z", "+00:00"))


@dataclass
class Workspace:
    """Workspace."""
    id: str
    name: str
    description: Optional[str] = None
    created_at: Optional[datetime] = None
    
    def __post_init__(self):
        if isinstance(self.created_at, str):
            self.created_at = datetime.fromisoformat(self.created_at.replace("Z", "+00:00"))


@dataclass
class QuotaLimit:
    """Quota limit for a single resource."""
    resource_type: str
    limit: int
    used: int
    remaining: int
    reset_at: Optional[datetime] = None
    
    @property
    def percentage_used(self) -> float:
        """Calculate percentage of quota used."""
        if self.limit == 0:
            return 0.0
        return (self.used / self.limit) * 100
    
    @property
    def is_exceeded(self) -> bool:
        """Check if quota is exceeded."""
        return self.used >= self.limit
    
    def __post_init__(self):
        if isinstance(self.reset_at, str):
            self.reset_at = datetime.fromisoformat(self.reset_at.replace("Z", "+00:00"))


@dataclass
class QuotaUsage:
    """Overall quota usage."""
    user_id: str
    workspace_id: Optional[str]
    limits: List[QuotaLimit]
    tier: str = "default"
    
    def get_limit(self, resource_type: str) -> Optional[QuotaLimit]:
        """Get quota limit for a specific resource type."""
        for limit in self.limits:
            if limit.resource_type == resource_type:
                return limit
        return None


@dataclass
class CostEntry:
    """Single cost entry."""
    date: str
    model: str
    tokens: int
    cost_usd: float
    user_id: Optional[str] = None
    workspace_id: Optional[str] = None


@dataclass
class CostReport:
    """Cost report."""
    start_date: str
    end_date: str
    total_cost_usd: float
    total_tokens: int
    entries: List[CostEntry]
    by_model: Dict[str, float]
    by_user: Dict[str, float]
    by_workspace: Dict[str, float]


@dataclass
class Region:
    """Geographic region."""
    id: str
    name: str
    location: str
    status: str
    capabilities: List[str]
    endpoint: str
    
    @property
    def is_active(self) -> bool:
        """Check if region is active."""
        return self.status == "active"


@dataclass
class DataResidency:
    """Data residency policy."""
    workspace_id: str
    allowed_regions: List[str]
    primary_region: str
    cross_region_replication: bool = False
    
    def is_region_allowed(self, region_id: str) -> bool:
        """Check if a region is allowed."""
        return region_id in self.allowed_regions


@dataclass
class ServiceAccount:
    """Service account."""
    id: str
    name: str
    description: Optional[str]
    created_at: datetime
    last_used_at: Optional[datetime] = None
    permissions: List[str] = None
    
    def __post_init__(self):
        if self.permissions is None:
            self.permissions = []
        if isinstance(self.created_at, str):
            self.created_at = datetime.fromisoformat(self.created_at.replace("Z", "+00:00"))
        if isinstance(self.last_used_at, str):
            self.last_used_at = datetime.fromisoformat(self.last_used_at.replace("Z", "+00:00"))


@dataclass
class APIKey:
    """API key for service account."""
    id: str
    key: str
    service_account_id: str
    created_at: datetime
    expires_at: Optional[datetime] = None
    last_used_at: Optional[datetime] = None
    
    @property
    def is_expired(self) -> bool:
        """Check if key is expired."""
        if self.expires_at is None:
            return False
        return datetime.now() > self.expires_at
    
    def __post_init__(self):
        if isinstance(self.created_at, str):
            self.created_at = datetime.fromisoformat(self.created_at.replace("Z", "+00:00"))
        if isinstance(self.expires_at, str):
            self.expires_at = datetime.fromisoformat(self.expires_at.replace("Z", "+00:00"))
        if isinstance(self.last_used_at, str):
            self.last_used_at = datetime.fromisoformat(self.last_used_at.replace("Z", "+00:00"))
