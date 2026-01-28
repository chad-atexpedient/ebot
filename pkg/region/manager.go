package region

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// manager implements RegionManager
type manager struct {
	mu                sync.RWMutex
	regions           map[string]*Region                // regionID -> Region
	policies          map[string]*DataResidencyPolicy   // resourceID -> Policy
	userPolicies      map[string]*DataResidencyPolicy   // userID -> Policy
	workspacePolicies map[string]*DataResidencyPolicy   // workspaceID -> Policy
	resources         map[string]*RegionalResource      // resourceID -> Resource
	config            *RegionConfig
	logger            *slog.Logger
}

// NewManager creates a new RegionManager
func NewManager(config *RegionConfig, logger *slog.Logger) RegionManager {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &manager{
		regions:           make(map[string]*Region),
		policies:          make(map[string]*DataResidencyPolicy),
		userPolicies:      make(map[string]*DataResidencyPolicy),
		workspacePolicies: make(map[string]*DataResidencyPolicy),
		resources:         make(map[string]*RegionalResource),
		config:            config,
		logger:            logger.With("component", "region-manager"),
	}
}

// RegisterRegion registers a new region
func (m *manager) RegisterRegion(ctx context.Context, region *Region) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if region.ID == "" {
		return fmt.Errorf("region ID cannot be empty")
	}
	
	now := time.Now()
	if region.CreatedAt.IsZero() {
		region.CreatedAt = now
	}
	region.UpdatedAt = now
	
	m.regions[region.ID] = region
	
	m.logger.Info("region registered",
		"regionId", region.ID,
		"name", region.Name,
		"location", region.Location,
		"status", region.Status)
	
	return nil
}

// GetRegion retrieves a region by ID
func (m *manager) GetRegion(ctx context.Context, regionID string) (*Region, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	region, exists := m.regions[regionID]
	if !exists {
		return nil, ErrRegionNotFound
	}
	
	return region, nil
}

// ListRegions returns all registered regions
func (m *manager) ListRegions(ctx context.Context) ([]*Region, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	regions := make([]*Region, 0, len(m.regions))
	for _, region := range m.regions {
		regions = append(regions, region)
	}
	
	return regions, nil
}

// UpdateRegionStatus updates a region's status
func (m *manager) UpdateRegionStatus(ctx context.Context, regionID string, status RegionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	region, exists := m.regions[regionID]
	if !exists {
		return ErrRegionNotFound
	}
	
	oldStatus := region.Status
	region.Status = status
	region.UpdatedAt = time.Now()
	
	m.logger.Info("region status updated",
		"regionId", regionID,
		"oldStatus", oldStatus,
		"newStatus", status)
	
	return nil
}

// CreatePolicy creates a new data residency policy
func (m *manager) CreatePolicy(ctx context.Context, policy *DataResidencyPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if policy.ID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}
	
	// Validate primary region exists
	if _, exists := m.regions[policy.PrimaryRegion]; !exists {
		return fmt.Errorf("primary region %s not found", policy.PrimaryRegion)
	}
	
	// Validate all allowed regions exist
	for _, regionID := range policy.AllowedRegions {
		if _, exists := m.regions[regionID]; !exists {
			return fmt.Errorf("allowed region %s not found", regionID)
		}
	}
	
	now := time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	policy.UpdatedAt = now
	
	// Store in appropriate index
	if policy.UserID != "" {
		m.userPolicies[policy.UserID] = policy
	}
	if policy.WorkspaceID != "" {
		m.workspacePolicies[policy.WorkspaceID] = policy
	}
	
	m.policies[policy.ID] = policy
	
	m.logger.Info("data residency policy created",
		"policyId", policy.ID,
		"name", policy.Name,
		"primaryRegion", policy.PrimaryRegion,
		"allowedRegions", policy.AllowedRegions,
		"userId", policy.UserID,
		"workspaceId", policy.WorkspaceID)
	
	return nil
}

// GetPolicy retrieves a policy by resource ID
func (m *manager) GetPolicy(ctx context.Context, resourceID string) (*DataResidencyPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	policy, exists := m.policies[resourceID]
	if !exists {
		return nil, ErrPolicyNotFound
	}
	
	return policy, nil
}

// ValidateRegionAccess checks if a user can access a region
func (m *manager) ValidateRegionAccess(ctx context.Context, userID, regionID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if region exists
	region, exists := m.regions[regionID]
	if !exists {
		return ErrRegionNotFound
	}
	
	// Check if region is available
	if region.Status == RegionStatusOffline {
		return ErrRegionOffline
	}
	
	// Check user's policy if exists
	if policy, exists := m.userPolicies[userID]; exists {
		allowed := false
		for _, allowedRegion := range policy.AllowedRegions {
			if allowedRegion == regionID {
				allowed = true
				break
			}
		}
		
		if !allowed {
			m.logger.Warn("region access denied by policy",
				"userId", userID,
				"regionId", regionID,
				"policyId", policy.ID)
			return ErrRegionUnauthorized
		}
	}
	
	// If no policy, allow access (default behavior)
	return nil
}

// GetAllowedRegions returns all regions a user can access
func (m *manager) GetAllowedRegions(ctx context.Context, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if user has a specific policy
	if policy, exists := m.userPolicies[userID]; exists {
		return policy.AllowedRegions, nil
	}
	
	// Default: all active regions
	allowedRegions := make([]string, 0)
	for regionID, region := range m.regions {
		if region.Status == RegionStatusActive {
			allowedRegions = append(allowedRegions, regionID)
		}
	}
	
	return allowedRegions, nil
}

// GetPrimaryRegion returns the primary region for a resource
func (m *manager) GetPrimaryRegion(ctx context.Context, resourceID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if resource has explicit region assignment
	if resource, exists := m.resources[resourceID]; exists {
		return resource.PrimaryRegion, nil
	}
	
	// Check if there's a policy
	if policy, exists := m.policies[resourceID]; exists {
		return policy.PrimaryRegion, nil
	}
	
	// Default to current region
	if m.config.CurrentRegion != "" {
		return m.config.CurrentRegion, nil
	}
	
	return "", fmt.Errorf("no region found for resource %s", resourceID)
}

// RouteRequest determines which region should handle a request
func (m *manager) RouteRequest(ctx context.Context, userID, resourceType string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check user policy first
	if policy, exists := m.userPolicies[userID]; exists {
		// Use primary region from policy
		region := policy.PrimaryRegion
		
		// Validate region is active
		if r, exists := m.regions[region]; exists && r.Status == RegionStatusActive {
			m.logger.Debug("routing decision",
				"userId", userID,
				"resourceType", resourceType,
				"targetRegion", region,
				"reason", "user-policy")
			return region, nil
		}
		
		// Primary region unavailable, try allowed regions
		for _, allowedRegion := range policy.AllowedRegions {
			if r, exists := m.regions[allowedRegion]; exists && r.Status == RegionStatusActive {
				m.logger.Debug("routing decision",
					"userId", userID,
					"resourceType", resourceType,
					"targetRegion", allowedRegion,
					"reason", "fallback-to-allowed-region")
				return allowedRegion, nil
			}
		}
		
		return "", fmt.Errorf("no active regions available for user %s", userID)
	}
	
	// No policy - route to current region
	if m.config.CurrentRegion != "" {
		if r, exists := m.regions[m.config.CurrentRegion]; exists && r.Status == RegionStatusActive {
			return m.config.CurrentRegion, nil
		}
	}
	
	// Fallback: route to any active region
	for regionID, region := range m.regions {
		if region.Status == RegionStatusActive {
			m.logger.Debug("routing decision",
				"userId", userID,
				"resourceType", resourceType,
				"targetRegion", regionID,
				"reason", "fallback-any-active")
			return regionID, nil
		}
	}
	
	return "", fmt.Errorf("no active regions available")
}

// ReplicateToRegion initiates replication of a resource to another region
func (m *manager) ReplicateToRegion(ctx context.Context, resourceID, sourceRegion, targetRegion string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Validate source region
	source, exists := m.regions[sourceRegion]
	if !exists {
		return fmt.Errorf("source region %s not found", sourceRegion)
	}
	if source.Status != RegionStatusActive {
		return fmt.Errorf("source region %s is not active", sourceRegion)
	}
	
	// Validate target region
	target, exists := m.regions[targetRegion]
	if !exists {
		return fmt.Errorf("target region %s not found", targetRegion)
	}
	if target.Status != RegionStatusActive {
		return fmt.Errorf("target region %s is not active", targetRegion)
	}
	
	// Check policy allows replication
	if resource, exists := m.resources[resourceID]; exists {
		if policy, exists := m.policies[resource.PolicyID]; exists {
			if !policy.CrossRegionReplication {
				return fmt.Errorf("cross-region replication not enabled for resource %s", resourceID)
			}
			
			// Validate target is in allowed replication regions
			allowed := false
			for _, region := range policy.ReplicationRegions {
				if region == targetRegion {
					allowed = true
					break
				}
			}
			
			if !allowed {
				return fmt.Errorf("replication to region %s not allowed by policy", targetRegion)
			}
		}
	}
	
	m.logger.Info("initiating cross-region replication",
		"resourceId", resourceID,
		"sourceRegion", sourceRegion,
		"targetRegion", targetRegion)
	
	// TODO: Implement actual replication logic
	// This would involve:
	// 1. Copying data from source to target
	// 2. Updating replication status
	// 3. Maintaining consistency
	
	return nil
}

// GetReplicationStatus returns the replication status for a resource
func (m *manager) GetReplicationStatus(ctx context.Context, resourceID string) (*ReplicationStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	resource, exists := m.resources[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}
	
	// Build replication status
	status := &ReplicationStatus{
		ResourceID:      resourceID,
		PrimaryRegion:   resource.PrimaryRegion,
		Replicas:        make(map[string]*ReplicaStatus),
		LastReplication: time.Now(), // TODO: Get actual last replication time
		Status:          ReplicationStateHealthy,
	}
	
	// Check policy for replication regions
	if policy, exists := m.policies[resource.PolicyID]; exists {
		for _, regionID := range policy.ReplicationRegions {
			if region, exists := m.regions[regionID]; exists {
				replicaStatus := &ReplicaStatus{
					RegionID: regionID,
					Status:   ReplicationStateHealthy,
					LastSync: time.Now(), // TODO: Get actual sync time
					Lag:      0,          // TODO: Calculate actual lag
					Health:   string(region.Status),
				}
				
				status.Replicas[regionID] = replicaStatus
			}
		}
	}
	
	return status, nil
}

// RegisterResource registers a resource with regional information
func (m *manager) RegisterResource(ctx context.Context, resource *RegionalResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if resource.ID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	
	// Validate primary region exists
	if _, exists := m.regions[resource.PrimaryRegion]; !exists {
		return fmt.Errorf("primary region %s not found", resource.PrimaryRegion)
	}
	
	now := time.Now()
	if resource.CreatedAt.IsZero() {
		resource.CreatedAt = now
	}
	resource.UpdatedAt = now
	
	m.resources[resource.ID] = resource
	
	m.logger.Info("regional resource registered",
		"resourceId", resource.ID,
		"type", resource.Type,
		"primaryRegion", resource.PrimaryRegion)
	
	return nil
}
