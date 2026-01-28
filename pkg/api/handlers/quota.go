package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chad-atexpedient/ebot/pkg/quota"
)

// QuotaHandler handles quota-related API requests
type QuotaHandler struct {
	quotaManager quota.Manager
}

// NewQuotaHandler creates a new quota handler
func NewQuotaHandler(qm quota.Manager) *QuotaHandler {
	return &QuotaHandler{
		quotaManager: qm,
	}
}

// GetUsage returns quota usage for the authenticated user
func (h *QuotaHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromRequest(r)

	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	usage, err := h.quotaManager.GetUsage(ctx, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// GetUserUsage returns quota usage for a specific user (admin only)
func (h *QuotaHandler) GetUserUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Check if requester is admin
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	targetUserID := r.URL.Query().Get("userId")
	if targetUserID == "" {
		http.Error(w, "userId parameter required", http.StatusBadRequest)
		return
	}

	usage, err := h.quotaManager.GetUsage(ctx, targetUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// ListAllUsage returns quota usage for all users (admin only)
func (h *QuotaHandler) ListAllUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// TODO: Implement listing all users and their quotas
	// This would require adding a ListAllUsage method to the quota manager

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
	})
}

// UpdateQuota updates quota limits for a user (admin only)
func (h *QuotaHandler) UpdateQuota(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		UserID       string `json:"userId"`
		ResourceType string `json:"resourceType"`
		NewLimit     int64  `json:"newLimit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement quota update functionality
	// This would require adding an UpdateQuota method to the quota manager

	usage, err := h.quotaManager.GetUsage(ctx, req.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// Helper functions

func getUserIDFromRequest(r *http.Request) string {
	// TODO: Extract from JWT token or session
	return r.Header.Get("X-User-ID")
}

func isAdmin(r *http.Request) bool {
	// TODO: Check if user has admin role
	role := r.Header.Get("X-User-Role")
	return role == "admin"
}
