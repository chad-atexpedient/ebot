package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chad-atexpedient/ebot/pkg/region"
)

// RegionsHandler provides HTTP handlers for multi-region management
type RegionsHandler struct {
	manager region.RegionManager
	router  *region.Router
}

// NewRegionsHandler creates a new regions handler
func NewRegionsHandler(manager region.RegionManager, router *region.Router) *RegionsHandler {
	return &RegionsHandler{
		manager: manager,
		router:  router,
	}
}

// ListRegions returns all available regions
// GET /api/regions
func (h *RegionsHandler) ListRegions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	regions, err := h.manager.ListRegions(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"regions": regions,
		"count":   len(regions),
	})
}

// GetRegion returns details for a specific region
// GET /api/regions/{regionId}
func (h *RegionsHandler) GetRegion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	regionID := r.URL.Query().Get("regionId")

	if regionID == "" {
		http.Error(w, "regionId is required", http.StatusBadRequest)
		return
	}

	region, err := h.manager.GetRegion(ctx, regionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(region)
}

// GetUserRegions returns regions accessible to the current user
// GET /api/regions/user
func (h *RegionsHandler) GetUserRegions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.URL.Query().Get("userId")

	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	allowedRegions, err := h.manager.GetAllowedRegions(ctx, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get full region details
	regions := make([]*region.Region, 0)
	for _, regionID := range allowedRegions {
		reg, err := h.manager.GetRegion(ctx, regionID)
		if err == nil {
			regions = append(regions, reg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"regions": regions,
		"count":   len(regions),
	})
}

// CreateDataResidencyPolicy creates a new data residency policy
// POST /api/data-residency/policies
func (h *RegionsHandler) CreateDataResidencyPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var policy region.DataResidencyPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.manager.CreatePolicy(ctx, &policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteStatus(http.StatusCreated)
	json.NewEncoder(w).Encode(policy)
}

// GetDataResidencyPolicy retrieves a data residency policy
// GET /api/data-residency/policies/{resourceId}
func (h *RegionsHandler) GetDataResidencyPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resourceID := r.URL.Query().Get("resourceId")

	if resourceID == "" {
		http.Error(w, "resourceId is required", http.StatusBadRequest)
		return
	}

	policy, err := h.manager.GetPolicy(ctx, resourceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policy)
}

// RouteRequest determines which region should handle a request
// POST /api/regions/route
func (h *RegionsHandler) RouteRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req region.RoutingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	decision, err := h.router.RouteRequest(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decision)
}

// ValidateOperation checks if an operation is allowed
// POST /api/regions/validate
func (h *RegionsHandler) ValidateOperation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req region.OperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.router.ValidateOperation(ctx, &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": false,
			"reason":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": true,
	})
}

// GetOptimalRegion finds the best region for a new resource
// POST /api/regions/optimal
func (h *RegionsHandler) GetOptimalRegion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req region.OptimalRegionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	optimalRegion, err := h.router.GetOptimalRegion(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"region": optimalRegion,
	})
}

// GetReplicationStatus returns replication status for a resource
// GET /api/regions/replication/{resourceId}
func (h *RegionsHandler) GetReplicationStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resourceID := r.URL.Query().Get("resourceId")

	if resourceID == "" {
		http.Error(w, "resourceId is required", http.StatusBadRequest)
		return
	}

	status, err := h.manager.GetReplicationStatus(ctx, resourceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// InitiateReplication starts replication to a target region
// POST /api/regions/replication
func (h *RegionsHandler) InitiateReplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ResourceID   string `json:"resourceId"`
		SourceRegion string `json:"sourceRegion"`
		TargetRegion string `json:"targetRegion"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResourceID == "" || req.SourceRegion == "" || req.TargetRegion == "" {
		http.Error(w, "resourceId, sourceRegion, and targetRegion are required", http.StatusBadRequest)
		return
	}

	err := h.manager.ReplicateToRegion(ctx, req.ResourceID, req.SourceRegion, req.TargetRegion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "initiated",
		"message": "Replication initiated successfully",
	})
}
