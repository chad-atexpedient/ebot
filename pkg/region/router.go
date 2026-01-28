package region

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Router provides region-aware request routing
type Router struct {
	manager RegionManager
	logger  *slog.Logger
}

// NewRouter creates a new region-aware router
func NewRouter(manager RegionManager, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &Router{
		manager: manager,
		logger:  logger.With("component", "region-router"),
	}
}

// RouteRequest determines the best region to handle a request
func (r *Router) RouteRequest(ctx context.Context, req *RoutingRequest) (*RegionRoutingDecision, error) {
	startTime := time.Now()
	
	// Extract user and resource info
	userID := req.UserID
	resourceType := req.ResourceType
	resourceID := req.ResourceID
	
	r.logger.Debug("routing request",
		"userId", userID,
		"resourceType", resourceType,
		"resourceId", resourceID)
	
	// Strategy 1: If resource exists, use its primary region
	if resourceID != "" {
		primaryRegion, err := r.manager.GetPrimaryRegion(ctx, resourceID)
		if err == nil && primaryRegion != "" {
			// Validate user can access this region
			if err := r.manager.ValidateRegionAccess(ctx, userID, primaryRegion); err != nil {
				r.logger.Warn("user cannot access resource's region",
					"userId", userID,
					"resourceId", resourceID,
					"region", primaryRegion,
					"error", err)
				return nil, err
			}
			
			decision := &RegionRoutingDecision{
				TargetRegion:  primaryRegion,
				Reason:        "resource-primary-region",
				PolicyApplied: "",
				Latency:       time.Since(startTime),
				Metadata:      map[string]string{"resourceId": resourceID},
			}
			
			r.logger.Info("routed to resource region",
				"userId", userID,
				"targetRegion", primaryRegion,
				"latency", decision.Latency)
			
			return decision, nil
		}
	}
	
	// Strategy 2: Use user's policy to determine region
	targetRegion, err := r.manager.RouteRequest(ctx, userID, resourceType)
	if err != nil {
		r.logger.Error("routing failed",
			"userId", userID,
			"resourceType", resourceType,
			"error", err)
		return nil, fmt.Errorf("routing failed: %w", err)
	}
	
	// Validate region is accessible
	if err := r.manager.ValidateRegionAccess(ctx, userID, targetRegion); err != nil {
		r.logger.Error("region access validation failed",
			"userId", userID,
			"targetRegion", targetRegion,
			"error", err)
		return nil, err
	}
	
	decision := &RegionRoutingDecision{
		TargetRegion:  targetRegion,
		Reason:        "user-policy",
		PolicyApplied: userID,
		Latency:       time.Since(startTime),
		Metadata:      map[string]string{},
	}
	
	r.logger.Info("routed by policy",
		"userId", userID,
		"targetRegion", targetRegion,
		"latency", decision.Latency)
	
	return decision, nil
}

// ValidateOperation checks if an operation is allowed based on region policies
func (r *Router) ValidateOperation(ctx context.Context, req *OperationRequest) error {
	userID := req.UserID
	operation := req.Operation
	targetRegion := req.TargetRegion
	resourceID := req.ResourceID
	
	r.logger.Debug("validating operation",
		"userId", userID,
		"operation", operation,
		"targetRegion", targetRegion,
		"resourceId", resourceID)
	
	// Check if user can access target region
	if err := r.manager.ValidateRegionAccess(ctx, userID, targetRegion); err != nil {
		r.logger.Warn("operation denied - region access",
			"userId", userID,
			"operation", operation,
			"targetRegion", targetRegion,
			"error", err)
		return err
	}
	
	// If resource exists, check data residency policy
	if resourceID != "" {
		policy, err := r.manager.GetPolicy(ctx, resourceID)
		if err == nil {
			// Check if operation violates policy
			if operation == "export" || operation == "copy" {
				// Check export restrictions
				for _, restriction := range policy.DataExportRestrictions {
					if restriction == targetRegion {
						r.logger.Warn("operation denied - export restriction",
							"userId", userID,
							"operation", operation,
							"targetRegion", targetRegion,
							"restriction", restriction)
						return ErrPolicyViolation
					}
				}
			}
			
			// Check if target region is in allowed regions
			allowed := false
			for _, allowedRegion := range policy.AllowedRegions {
				if allowedRegion == targetRegion {
					allowed = true
					break
				}
			}
			
			if !allowed {
				r.logger.Warn("operation denied - not in allowed regions",
					"userId", userID,
					"operation", operation,
					"targetRegion", targetRegion,
					"allowedRegions", policy.AllowedRegions)
				return ErrPolicyViolation
			}
		}
	}
	
	r.logger.Info("operation validated",
		"userId", userID,
		"operation", operation,
		"targetRegion", targetRegion)
	
	return nil
}

// GetOptimalRegion finds the best region for a new resource
func (r *Router) GetOptimalRegion(ctx context.Context, req *OptimalRegionRequest) (string, error) {
	userID := req.UserID
	workspaceID := req.WorkspaceID
	resourceType := req.ResourceType
	
	r.logger.Debug("finding optimal region",
		"userId", userID,
		"workspaceId", workspaceID,
		"resourceType", resourceType)
	
	// Get user's allowed regions
	allowedRegions, err := r.manager.GetAllowedRegions(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get allowed regions: %w", err)
	}
	
	if len(allowedRegions) == 0 {
		return "", fmt.Errorf("no allowed regions for user %s", userID)
	}
	
	// If constraints specified, filter
	if len(req.RequiredCapabilities) > 0 {
		filteredRegions := make([]string, 0)
		
		regions, _ := r.manager.ListRegions(ctx)
		for _, region := range regions {
			// Check if region is in allowed list
			isAllowed := false
			for _, allowed := range allowedRegions {
				if region.ID == allowed {
					isAllowed = true
					break
				}
			}
			
			if !isAllowed {
				continue
			}
			
			// Check if region has required capabilities
			hasAllCapabilities := true
			for _, requiredCap := range req.RequiredCapabilities {
				found := false
				for _, regionCap := range region.Capabilities {
					if regionCap == requiredCap {
						found = true
						break
					}
				}
				if !found {
					hasAllCapabilities = false
					break
				}
			}
			
			if hasAllCapabilities && region.Status == RegionStatusActive {
				filteredRegions = append(filteredRegions, region.ID)
			}
		}
		
		if len(filteredRegions) == 0 {
			return "", fmt.Errorf("no regions match required capabilities")
		}
		
		allowedRegions = filteredRegions
	}
	
	// Return first allowed region (can be enhanced with load balancing)
	optimalRegion := allowedRegions[0]
	
	r.logger.Info("optimal region selected",
		"userId", userID,
		"resourceType", resourceType,
		"optimalRegion", optimalRegion,
		"candidateCount", len(allowedRegions))
	
	return optimalRegion, nil
}

// RoutingRequest represents a request to route
type RoutingRequest struct {
	UserID       string
	ResourceType string
	ResourceID   string
	WorkspaceID  string
	Metadata     map[string]string
}

// OperationRequest represents a request to validate an operation
type OperationRequest struct {
	UserID       string
	ResourceID   string
	Operation    string // "read", "write", "delete", "export", "copy"
	TargetRegion string
	Metadata     map[string]string
}

// OptimalRegionRequest represents a request to find the optimal region
type OptimalRegionRequest struct {
	UserID               string
	WorkspaceID          string
	ResourceType         string
	RequiredCapabilities []string
	PreferredRegions     []string
	Metadata             map[string]string
}
