package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/chad-atexpedient/ebot/logger"
	"github.com/chad-atexpedient/ebot/pkg/auth/abac"
	"github.com/chad-atexpedient/ebot/pkg/auth/saml"
	"github.com/chad-atexpedient/ebot/pkg/auth/serviceaccount"
)

// AccessControlHandlers provides HTTP handlers for access control features
type AccessControlHandlers struct {
	abacEngine     abac.ABACEngine
	samlProvider   saml.SAMLProvider
	serviceAcctMgr serviceaccount.ServiceAccountManager
	logger         logger.Logger
}

// NewAccessControlHandlers creates new access control handlers
func NewAccessControlHandlers(
	abacEngine abac.ABACEngine,
	samlProvider saml.SAMLProvider,
	serviceAcctMgr serviceaccount.ServiceAccountManager,
	log logger.Logger,
) *AccessControlHandlers {
	return &AccessControlHandlers{
		abacEngine:     abacEngine,
		samlProvider:   samlProvider,
		serviceAcctMgr: serviceAcctMgr,
		logger:         log,
	}
}

// ABAC Policy Handlers

// CreatePolicy creates a new ABAC policy
// POST /api/policies
func (h *AccessControlHandlers) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var policy abac.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if err := h.abacEngine.AddPolicy(&policy); err != nil {
		h.logger.Error("Failed to create policy", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(policy)
}

// GetPolicy retrieves a policy by ID
// GET /api/policies/:id
func (h *AccessControlHandlers) GetPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	
	policy, err := h.abacEngine.GetPolicy(policyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(policy)
}

// ListPolicies lists all policies
// GET /api/policies
func (h *AccessControlHandlers) ListPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.abacEngine.ListPolicies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policies": policies,
		"total":    len(policies),
	})
}

// UpdatePolicy updates an existing policy
// PUT /api/policies/:id
func (h *AccessControlHandlers) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	
	var policy abac.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	policy.ID = policyID
	
	if err := h.abacEngine.UpdatePolicy(&policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(policy)
}

// DeletePolicy deletes a policy
// DELETE /api/policies/:id
func (h *AccessControlHandlers) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	
	if err := h.abacEngine.RemovePolicy(policyID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// EvaluatePolicy evaluates a policy for an authorization request
// POST /api/policies/evaluate
func (h *AccessControlHandlers) EvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	var request abac.AuthorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	decision, err := h.abacEngine.Evaluate(r.Context(), &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(decision)
}

// SAML Handlers

// InitiateSAMLLogin initiates SAML SSO flow
// GET /auth/saml/login
func (h *AccessControlHandlers) InitiateSAMLLogin(w http.ResponseWriter, r *http.Request) {
	if err := h.samlProvider.HandleSSORequest(w, r); err != nil {
		h.logger.Error("Failed to initiate SAML login", "error", err)
		http.Error(w, "Failed to initiate SAML login", http.StatusInternalServerError)
	}
}

// HandleSAMLCallback handles SAML assertion consumer service callback
// POST /auth/saml/acs
func (h *AccessControlHandlers) HandleSAMLCallback(w http.ResponseWriter, r *http.Request) {
	user, err := h.samlProvider.HandleACS(w, r)
	if err != nil {
		h.logger.Error("Failed to process SAML response", "error", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}
	
	// TODO: Create session for user
	h.logger.Info("SAML authentication successful",
		"name_id", user.NameID,
		"email", user.Email)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

// GetSAMLMetadata returns SAML metadata XML
// GET /auth/saml/metadata
func (h *AccessControlHandlers) GetSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	metadata, err := h.samlProvider.GetMetadata()
	if err != nil {
		http.Error(w, "Failed to generate metadata", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/xml")
	w.Write(metadata)
}

// Service Account Handlers

// CreateServiceAccount creates a new service account
// POST /api/service-accounts
func (h *AccessControlHandlers) CreateServiceAccount(w http.ResponseWriter, r *http.Request) {
	var account serviceaccount.ServiceAccount
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	created, err := h.serviceAcctMgr.CreateServiceAccount(r.Context(), &account)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// GetServiceAccount retrieves a service account by ID
// GET /api/service-accounts/:id
func (h *AccessControlHandlers) GetServiceAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	
	account, err := h.serviceAcctMgr.GetServiceAccount(r.Context(), accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(account)
}

// ListServiceAccounts lists service accounts
// GET /api/service-accounts
func (h *AccessControlHandlers) ListServiceAccounts(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	
	accounts, err := h.serviceAcctMgr.ListServiceAccounts(r.Context(), workspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": accounts,
		"total":    len(accounts),
	})
}

// UpdateServiceAccount updates a service account
// PUT /api/service-accounts/:id
func (h *AccessControlHandlers) UpdateServiceAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	
	var account serviceaccount.ServiceAccount
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	account.ID = accountID
	
	if err := h.serviceAcctMgr.UpdateServiceAccount(r.Context(), &account); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(account)
}

// DeleteServiceAccount deletes a service account
// DELETE /api/service-accounts/:id
func (h *AccessControlHandlers) DeleteServiceAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	
	if err := h.serviceAcctMgr.DeleteServiceAccount(r.Context(), accountID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// GenerateAPIKey generates a new API key for a service account
// POST /api/service-accounts/:id/api-keys
func (h *AccessControlHandlers) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	
	var opts serviceaccount.APIKeyOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	apiKey, err := h.serviceAcctMgr.GenerateAPIKey(r.Context(), accountID, &opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         apiKey.ID,
		"key":        apiKey.KeyHash, // Temporarily contains full key
		"key_prefix": apiKey.KeyPrefix,
		"expires_at": apiKey.ExpiresAt,
		"warning":    "Save this key securely. It won't be shown again.",
	})
}

// ListAPIKeys lists API keys for a service account
// GET /api/service-accounts/:id/api-keys
func (h *AccessControlHandlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	
	keys, err := h.serviceAcctMgr.ListAPIKeys(r.Context(), accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Don't return actual keys or hashes
	safeKeys := make([]map[string]interface{}, len(keys))
	for i, key := range keys {
		safeKeys[i] = map[string]interface{}{
			"id":          key.ID,
			"name":        key.Name,
			"key_prefix":  key.KeyPrefix,
			"status":      key.Status,
			"created_at":  key.CreatedAt,
			"expires_at":  key.ExpiresAt,
			"last_used":   key.LastUsedAt,
			"usage_count": key.UsageCount,
		}
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"keys":  safeKeys,
		"total": len(safeKeys),
	})
}

// RevokeAPIKey revokes an API key
// DELETE /api/api-keys/:id
func (h *AccessControlHandlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID := r.PathValue("id")
	
	if err := h.serviceAcctMgr.RevokeAPIKey(r.Context(), keyID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// RotateAPIKey rotates an API key
// POST /api/api-keys/:id/rotate
func (h *AccessControlHandlers) RotateAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID := r.PathValue("id")
	
	var req struct {
		GracePeriodDays int `json:"gracePeriodDays"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	gracePeriod := time.Duration(req.GracePeriodDays) * 24 * time.Hour
	
	newKey, err := h.serviceAcctMgr.RotateAPIKey(r.Context(), keyID, gracePeriod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":              newKey.ID,
		"key":             newKey.KeyHash, // Temporarily contains full key
		"key_prefix":      newKey.KeyPrefix,
		"old_key_id":      keyID,
		"grace_period":    gracePeriod.String(),
		"warning":         "Save this key securely. It won't be shown again.",
	})
}
