"""
Resource managers for different ebot API endpoints.
"""

from typing import List, Optional, Dict, Any
from .types import MCPServer, Thread, Agent, Workspace, QuotaUsage, CostReport, Region, DataResidency


class BaseResource:
    """Base class for resource managers."""
    
    def __init__(self, client):
        self.client = client


class MCPServers(BaseResource):
    """Manage MCP servers."""
    
    async def list(self, workspace_id: Optional[str] = None) -> List[MCPServer]:
        """
        List all MCP servers.
        
        Args:
            workspace_id: Filter by workspace ID
            
        Returns:
            List of MCP servers
        """
        params = {}
        if workspace_id:
            params["workspaceId"] = workspace_id
        
        data = await self.client.request("GET", "/api/mcp-servers", params=params)
        return [MCPServer(**item) for item in data.get("items", [])]
    
    async def get(self, server_id: str) -> MCPServer:
        """Get a specific MCP server."""
        data = await self.client.request("GET", f"/api/mcp-servers/{server_id}")
        return MCPServer(**data)
    
    async def create(
        self,
        name: str,
        manifest: Dict[str, Any],
        workspace_id: Optional[str] = None,
    ) -> MCPServer:
        """
        Create a new MCP server.
        
        Args:
            name: Server name
            manifest: MCP server manifest
            workspace_id: Optional workspace ID
            
        Returns:
            Created MCP server
        """
        data = await self.client.request(
            "POST",
            "/api/mcp-servers",
            json={"name": name, "manifest": manifest, "workspaceId": workspace_id}
        )
        return MCPServer(**data)
    
    async def update(self, server_id: str, **kwargs) -> MCPServer:
        """Update an MCP server."""
        data = await self.client.request(
            "PUT",
            f"/api/mcp-servers/{server_id}",
            json=kwargs
        )
        return MCPServer(**data)
    
    async def delete(self, server_id: str) -> None:
        """Delete an MCP server."""
        await self.client.request("DELETE", f"/api/mcp-servers/{server_id}")


class Threads(BaseResource):
    """Manage conversation threads."""
    
    async def list(self, project_id: Optional[str] = None) -> List[Thread]:
        """List all threads."""
        params = {}
        if project_id:
            params["projectId"] = project_id
        
        data = await self.client.request("GET", "/api/threads", params=params)
        return [Thread(**item) for item in data.get("items", [])]
    
    async def get(self, thread_id: str) -> Thread:
        """Get a specific thread."""
        data = await self.client.request("GET", f"/api/threads/{thread_id}")
        return Thread(**data)
    
    async def create(
        self,
        project_id: str,
        title: Optional[str] = None,
    ) -> Thread:
        """Create a new thread."""
        data = await self.client.request(
            "POST",
            "/api/threads",
            json={"projectId": project_id, "title": title}
        )
        return Thread(**data)
    
    async def send_message(
        self,
        thread_id: str,
        content: str,
    ) -> Dict[str, Any]:
        """Send a message to a thread."""
        return await self.client.request(
            "POST",
            f"/api/threads/{thread_id}/messages",
            json={"content": content}
        )
    
    async def delete(self, thread_id: str) -> None:
        """Delete a thread."""
        await self.client.request("DELETE", f"/api/threads/{thread_id}")


class Agents(BaseResource):
    """Manage agents."""
    
    async def list(self) -> List[Agent]:
        """List all agents."""
        data = await self.client.request("GET", "/api/agents")
        return [Agent(**item) for item in data.get("items", [])]
    
    async def get(self, agent_id: str) -> Agent:
        """Get a specific agent."""
        data = await self.client.request("GET", f"/api/agents/{agent_id}")
        return Agent(**data)
    
    async def create(
        self,
        name: str,
        description: Optional[str] = None,
        tools: Optional[List[str]] = None,
    ) -> Agent:
        """Create a new agent."""
        data = await self.client.request(
            "POST",
            "/api/agents",
            json={"name": name, "description": description, "tools": tools or []}
        )
        return Agent(**data)


class Workspaces(BaseResource):
    """Manage workspaces."""
    
    async def list(self) -> List[Workspace]:
        """List all workspaces."""
        data = await self.client.request("GET", "/api/workspaces")
        return [Workspace(**item) for item in data.get("items", [])]
    
    async def get(self, workspace_id: str) -> Workspace:
        """Get a specific workspace."""
        data = await self.client.request("GET", f"/api/workspaces/{workspace_id}")
        return Workspace(**data)


class Quotas(BaseResource):
    """Manage resource quotas."""
    
    async def get_usage(self, user_id: Optional[str] = None) -> QuotaUsage:
        """
        Get quota usage for the authenticated user or specified user.
        
        Args:
            user_id: Optional user ID (admin only)
            
        Returns:
            Quota usage information
        """
        params = {}
        if user_id:
            params["userId"] = user_id
        
        data = await self.client.request("GET", "/api/quota/usage", params=params)
        return QuotaUsage(**data)
    
    async def update_limits(
        self,
        user_id: str,
        limits: Dict[str, int],
    ) -> QuotaUsage:
        """Update quota limits for a user (admin only)."""
        data = await self.client.request(
            "PUT",
            "/api/quota/update",
            json={"userId": user_id, "limits": limits}
        )
        return QuotaUsage(**data)


class Costs(BaseResource):
    """Manage cost tracking and reporting."""
    
    async def get_report(
        self,
        range_days: int = 7,
        user_id: Optional[str] = None,
        workspace_id: Optional[str] = None,
    ) -> CostReport:
        """
        Get cost report.
        
        Args:
            range_days: Number of days to include (default: 7)
            user_id: Filter by user
            workspace_id: Filter by workspace
            
        Returns:
            Cost report
        """
        params = {"range": f"{range_days}d"}
        if user_id:
            params["userId"] = user_id
        if workspace_id:
            params["workspaceId"] = workspace_id
        
        data = await self.client.request("GET", "/api/costs/report", params=params)
        return CostReport(**data)
    
    async def export_csv(self, range_days: int = 30) -> str:
        """Export cost data as CSV."""
        params = {"range": f"{range_days}d", "format": "csv"}
        return await self.client.request("GET", "/api/costs/export", params=params)


class Regions(BaseResource):
    """Manage multi-region deployments."""
    
    async def list(self) -> List[Region]:
        """List all available regions."""
        data = await self.client.request("GET", "/api/regions")
        return [Region(**item) for item in data.get("items", [])]
    
    async def get(self, region_id: str) -> Region:
        """Get region details."""
        data = await self.client.request("GET", f"/api/regions/{region_id}")
        return Region(**data)
    
    async def set_data_residency(
        self,
        workspace_id: str,
        allowed_regions: List[str],
        primary_region: str,
    ) -> DataResidency:
        """Set data residency policy for a workspace."""
        data = await self.client.request(
            "PUT",
            "/api/regions/data-residency",
            json={
                "workspaceId": workspace_id,
                "allowedRegions": allowed_regions,
                "primaryRegion": primary_region,
            }
        )
        return DataResidency(**data)


class AccessControl(BaseResource):
    """Manage access control policies."""
    
    async def list_policies(self) -> List[Dict[str, Any]]:
        """List all ABAC policies."""
        data = await self.client.request("GET", "/api/access-control/policies")
        return data.get("items", [])
    
    async def create_policy(
        self,
        name: str,
        rules: List[Dict[str, Any]],
        priority: int = 100,
    ) -> Dict[str, Any]:
        """Create a new ABAC policy."""
        return await self.client.request(
            "POST",
            "/api/access-control/policies",
            json={"name": name, "rules": rules, "priority": priority}
        )
    
    async def create_service_account(
        self,
        name: str,
        description: Optional[str] = None,
        permissions: Optional[List[str]] = None,
    ) -> Dict[str, Any]:
        """Create a service account."""
        return await self.client.request(
            "POST",
            "/api/access-control/service-accounts",
            json={
                "name": name,
                "description": description,
                "permissions": permissions or []
            }
        )
