// Package v1 contains API Schema definitions for the ebot v1 API group
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceQuotaSpec defines the desired state of ResourceQuota
type ResourceQuotaSpec struct {
	// UserID is the user this quota applies to (mutually exclusive with WorkspaceID)
	UserID string `json:"userID,omitempty"`
	
	// WorkspaceID is the workspace this quota applies to (mutually exclusive with UserID)
	WorkspaceID string `json:"workspaceID,omitempty"`
	
	// Tier defines the quota tier (default, power_user, enterprise)
	Tier string `json:"tier,omitempty"`
	
	// Hard limits for various resources
	MaxMCPServers int `json:"maxMCPServers,omitempty"`
	MaxThreads int `json:"maxThreads,omitempty"`
	MaxKnowledgeSets int `json:"maxKnowledgeSets,omitempty"`
	MaxKnowledgeFiles int `json:"maxKnowledgeFiles,omitempty"`
	MaxStorageBytes int64 `json:"maxStorageBytes,omitempty"`
	MaxCPUCores float64 `json:"maxCPUCores,omitempty"`
	MaxMemoryBytes int64 `json:"maxMemoryBytes,omitempty"`
	MaxRequestsPerMinute int `json:"maxRequestsPerMinute,omitempty"`
	MaxLLMTokensPerDay int64 `json:"maxLLMTokensPerDay,omitempty"`
	MaxLLMCostPerMonth float64 `json:"maxLLMCostPerMonth,omitempty"`
}

// ResourceQuotaStatus defines the observed state of ResourceQuota
type ResourceQuotaStatus struct {
	// Current usage of resources
	UsedMCPServers int `json:"usedMCPServers,omitempty"`
	UsedThreads int `json:"usedThreads,omitempty"`
	UsedKnowledgeSets int `json:"usedKnowledgeSets,omitempty"`
	UsedKnowledgeFiles int `json:"usedKnowledgeFiles,omitempty"`
	UsedStorageBytes int64 `json:"usedStorageBytes,omitempty"`
	UsedCPUCores float64 `json:"usedCPUCores,omitempty"`
	UsedMemoryBytes int64 `json:"usedMemoryBytes,omitempty"`
	UsedRequestsThisMinute int `json:"usedRequestsThisMinute,omitempty"`
	UsedLLMTokensToday int64 `json:"usedLLMTokensToday,omitempty"`
	UsedLLMCostThisMonth float64 `json:"usedLLMCostThisMonth,omitempty"`
	
	// Last reset times
	LastMinuteReset metav1.Time `json:"lastMinuteReset,omitempty"`
	LastDayReset metav1.Time `json:"lastDayReset,omitempty"`
	LastMonthReset metav1.Time `json:"lastMonthReset,omitempty"`
	
	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="User",type=string,JSONPath=`.spec.userID`
// +kubebuilder:printcolumn:name="Workspace",type=string,JSONPath=`.spec.workspaceID`
// +kubebuilder:printcolumn:name="Tier",type=string,JSONPath=`.spec.tier`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ResourceQuota is the Schema for the resourcequotas API
type ResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceQuotaSpec   `json:"spec,omitempty"`
	Status ResourceQuotaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceQuotaList contains a list of ResourceQuota
type ResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceQuota{}, &ResourceQuotaList{})
}
