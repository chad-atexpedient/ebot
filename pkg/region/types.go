package region

import (
	"context"
	"time"
)

// Region represents a deployment region for ebot
type Region struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Location    string            `json:"location"` // Geographic location (e.g., "US East", "EU West")
	Endpoint    string            `json:"endpoint"`
	Status      RegionStatus      `json:"status"`
	Capabilities []string         `json:"capabilities"` // ["mcp-hosting", "knowledge-storage", etc.]
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// RegionStatus represents the operational status of a region
type RegionStatus string

const (
	RegionStatusActive      RegionStatus = "active"
	RegionStatusDegraded    RegionStatus = "degraded"
	RegionStatusMaintenance RegionStatus = "maintenance"
	RegionStatusOffline     RegionStatus = "offline"
)

// DataResidencyPolicy defines where data can be stored
type DataResidencyPolicy struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	WorkspaceID            string            `json:"workspaceId,omitempty"`
	UserID                 string            `json:"userId,omitempty"`
	AllowedRegions         []string          `json:"allowedRegions"`         // List of allowed region IDs
	PrimaryRegion          string            `json:"primaryRegion"`          // Primary region for data storage
	CrossRegionReplication bool              `json:"crossRegionReplication"` // Enable replication
	ReplicationRegions     []string          `json:"replicationRegions,omitempty"`
	DataExportRestrictions []string          `json:"dataExportRestrictions,omitempty"` // Countries where data export is restricted
	ComplianceFrameworks   []string          `json:"complianceFrameworks,omitempty"`   // ["GDPR", "HIPAA", "SOC2"]
	Metadata               map[string]string `json:"metadata"`
	CreatedAt              time.Time         `json:"createdAt"`
	UpdatedAt              time.Time         `json:"updatedAt"`
}

// RegionManager provides regional orchestration and data residency
type RegionManager interface {
	// Region Management
	RegisterRegion(ctx context.Context, region *Region) error
	GetRegion(ctx context.Context, regionID string) (*Region, error)
	ListRegions(ctx context.Context) ([]*Region, error)
	UpdateRegionStatus(ctx context.Context, regionID string, status RegionStatus) error
	
	// Data Residency
	CreatePolicy(ctx context.Context, policy *DataResidencyPolicy) error
	GetPolicy(ctx context.Context, resourceID string) (*DataResidencyPolicy, error)
	ValidateRegionAccess(ctx context.Context, userID, regionID string) error
	GetAllowedRegions(ctx context.Context, userID string) ([]string, error)
	
	// Regional Routing
	GetPrimaryRegion(ctx context.Context, resourceID string) (string, error)
	RouteRequest(ctx context.Context, userID, resourceType string) (string, error)
	
	// Replication
	ReplicateToRegion(ctx context.Context, resourceID, sourceRegion, targetRegion string) error
	GetReplicationStatus(ctx context.Context, resourceID string) (*ReplicationStatus, error)
}

// ReplicationStatus tracks cross-region replication
type ReplicationStatus struct {
	ResourceID     string                       `json:"resourceId"`
	PrimaryRegion  string                       `json:"primaryRegion"`
	Replicas       map[string]*ReplicaStatus    `json:"replicas"` // regionID -> status
	LastReplication time.Time                   `json:"lastReplication"`
	Status         ReplicationState             `json:"status"`
}

// ReplicaStatus represents the status of a replica in a specific region
type ReplicaStatus struct {
	RegionID       string           `json:"regionId"`
	Status         ReplicationState `json:"status"`
	LastSync       time.Time        `json:"lastSync"`
	Lag            time.Duration    `json:"lag"` // Replication lag
	Health         string           `json:"health"`
}

// ReplicationState represents the state of replication
type ReplicationState string

const (
	ReplicationStateHealthy  ReplicationState = "healthy"
	ReplicationStateLagging  ReplicationState = "lagging"
	ReplicationStateFailed   ReplicationState = "failed"
	ReplicationStateSyncing  ReplicationState = "syncing"
)

// RegionalResource represents a resource that is region-aware
type RegionalResource struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"` // "workspace", "mcp-server", "knowledge-set", etc.
	PrimaryRegion  string    `json:"primaryRegion"`
	CurrentRegion  string    `json:"currentRegion"`
	AllowedRegions []string  `json:"allowedRegions"`
	PolicyID       string    `json:"policyId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// RegionConfig holds regional configuration
type RegionConfig struct {
	// Current region this instance is running in
	CurrentRegion string `json:"currentRegion"`
	
	// Enable multi-region features
	MultiRegionEnabled bool `json:"multiRegionEnabled"`
	
	// Global control plane endpoint
	ControlPlaneEndpoint string `json:"controlPlaneEndpoint,omitempty"`
	
	// Cross-region replication settings
	ReplicationEnabled  bool          `json:"replicationEnabled"`
	ReplicationInterval time.Duration `json:"replicationInterval"`
	
	// Database per-region or shared
	DatabaseStrategy string `json:"databaseStrategy"` // "per-region" or "shared"
	
	// Storage per-region or shared
	StorageStrategy string `json:"storageStrategy"` // "per-region" or "shared"
}

// RegionRoutingDecision represents a routing decision
type RegionRoutingDecision struct {
	TargetRegion   string            `json:"targetRegion"`
	Reason         string            `json:"reason"`
	PolicyApplied  string            `json:"policyApplied,omitempty"`
	Latency        time.Duration     `json:"latency"`
	Metadata       map[string]string `json:"metadata"`
}

// Common errors
var (
	ErrRegionNotFound        = &RegionError{Code: "REGION_NOT_FOUND", Message: "region not found"}
	ErrRegionUnauthorized    = &RegionError{Code: "REGION_UNAUTHORIZED", Message: "access to region not authorized"}
	ErrPolicyNotFound        = &RegionError{Code: "POLICY_NOT_FOUND", Message: "data residency policy not found"}
	ErrPolicyViolation       = &RegionError{Code: "POLICY_VIOLATION", Message: "operation violates data residency policy"}
	ErrReplicationFailed     = &RegionError{Code: "REPLICATION_FAILED", Message: "cross-region replication failed"}
	ErrRegionOffline         = &RegionError{Code: "REGION_OFFLINE", Message: "target region is offline"}
)

// RegionError represents a region-specific error
type RegionError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *RegionError) Error() string {
	return e.Message
}
