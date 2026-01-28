package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataResidency represents a data residency policy
type DataResidency struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataResidencySpec   `json:"spec,omitempty"`
	Status DataResidencyStatus `json:"status,omitempty"`
}

// DataResidencySpec defines the desired state of DataResidency
type DataResidencySpec struct {
	// WorkspaceID is the workspace this policy applies to
	WorkspaceID string `json:"workspaceID,omitempty"`

	// UserID is the user this policy applies to
	UserID string `json:"userID,omitempty"`

	// PrimaryRegion is the primary region for data storage
	PrimaryRegion string `json:"primaryRegion"`

	// AllowedRegions is the list of regions where data can be stored
	AllowedRegions []string `json:"allowedRegions"`

	// CrossRegionReplication enables data replication across regions
	CrossRegionReplication bool `json:"crossRegionReplication,omitempty"`

	// ReplicationRegions is the list of regions to replicate data to
	ReplicationRegions []string `json:"replicationRegions,omitempty"`

	// DataExportRestrictions lists countries where data export is restricted
	DataExportRestrictions []string `json:"dataExportRestrictions,omitempty"`

	// ComplianceFrameworks lists compliance frameworks this policy enforces
	ComplianceFrameworks []string `json:"complianceFrameworks,omitempty"`

	// RetentionPolicy defines data retention rules
	RetentionPolicy *RetentionPolicy `json:"retentionPolicy,omitempty"`

	// EncryptionRequired enforces encryption at rest and in transit
	EncryptionRequired bool `json:"encryptionRequired,omitempty"`
}

// RetentionPolicy defines how long data should be retained
type RetentionPolicy struct {
	// RetentionPeriodDays is the number of days to retain data
	RetentionPeriodDays int `json:"retentionPeriodDays"`

	// DeleteAfterRetention automatically deletes data after retention period
	DeleteAfterRetention bool `json:"deleteAfterRetention,omitempty"`

	// ArchiveBeforeDelete archives data before deletion
	ArchiveBeforeDelete bool `json:"archiveBeforeDelete,omitempty"`

	// ArchiveRegion is the region to archive data to
	ArchiveRegion string `json:"archiveRegion,omitempty"`
}

// DataResidencyStatus defines the observed state of DataResidency
type DataResidencyStatus struct {
	// Conditions represent the latest available observations of the policy's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation most recently observed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ActiveRegions lists regions currently storing data
	ActiveRegions []string `json:"activeRegions,omitempty"`

	// ReplicationStatus indicates the status of cross-region replication
	ReplicationStatus string `json:"replicationStatus,omitempty"`

	// ViolationCount tracks policy violations
	ViolationCount int64 `json:"violationCount,omitempty"`

	// LastViolation records the last policy violation
	LastViolation *metav1.Time `json:"lastViolation,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataResidencyList contains a list of DataResidency
type DataResidencyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataResidency `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionConfig represents a region configuration
type RegionConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegionConfigSpec   `json:"spec,omitempty"`
	Status RegionConfigStatus `json:"status,omitempty"`
}

// RegionConfigSpec defines the desired state of a Region
type RegionConfigSpec struct {
	// ID is the unique identifier for this region
	ID string `json:"id"`

	// DisplayName is the human-readable name for this region
	DisplayName string `json:"displayName"`

	// Location describes the geographic location
	Location string `json:"location"`

	// Endpoint is the API endpoint for this region
	Endpoint string `json:"endpoint"`

	// Capabilities lists the features available in this region
	Capabilities []string `json:"capabilities,omitempty"`

	// DatabaseEndpoint is the database connection string for this region
	DatabaseEndpoint string `json:"databaseEndpoint,omitempty"`

	// StorageBucket is the storage bucket for this region
	StorageBucket string `json:"storageBucket,omitempty"`

	// EncryptionKeyID is the encryption key used in this region
	EncryptionKeyID string `json:"encryptionKeyID,omitempty"`

	// ComplianceCertifications lists compliance certifications for this region
	ComplianceCertifications []string `json:"complianceCertifications,omitempty"`

	// Metadata contains additional region-specific configuration
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RegionConfigStatus defines the observed state of a Region
type RegionConfigStatus struct {
	// Status is the operational status of the region
	Status string `json:"status"` // "active", "degraded", "maintenance", "offline"

	// Conditions represent the latest available observations of the region's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation most recently observed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Health indicates the overall health of the region
	Health string `json:"health,omitempty"`

	// ResourceCount tracks the number of resources in this region
	ResourceCount int64 `json:"resourceCount,omitempty"`

	// LastHealthCheck records when the region was last checked
	LastHealthCheck *metav1.Time `json:"lastHealthCheck,omitempty"`

	// Version is the ebot version running in this region
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionConfigList contains a list of RegionConfig
type RegionConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegionConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&DataResidency{},
		&DataResidencyList{},
		&RegionConfig{},
		&RegionConfigList{},
	)
}
