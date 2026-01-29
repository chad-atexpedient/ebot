package saml

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chad-atexpedient/ebot/logger"
)

// SAMLHandlers provides HTTP handlers for SAML authentication
type SAMLHandlers struct {
	provider        SAMLProvider
	sessionManager  SessionManager
	logger          logger.Logger
	defaultRedirect string
}

// SessionManager manages user sessions
type SessionManager interface {
	CreateSession(user *SAMLUser) (string, error)
	GetSession(sessionID string) (*Session, error)
	DestroySession(sessionID string) error
}

// Session represents a user session
type Session struct {
	ID           string
	UserID       string
	Email        string
	Groups       []string
	SAMLSession  string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// NewSAMLHandlers creates new SAML HTTP handlers
func NewSAMLHandlers(provider SAMLProvider, sessionMgr SessionManager, log logger.Logger) *SAMLHandlers {
	return &SAMLHandlers{
		provider:        provider,
		sessionManager:  sessionMgr,
		logger:          log,
		defaultRedirect: "/",
	}
}

// RegisterRoutes registers SAML routes with an HTTP mux
func (h *SAMLHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/saml/metadata", h.HandleMetadata)
	mux.HandleFunc("/auth/saml/login", h.HandleLogin)
	mux.HandleFunc("/auth/saml/acs", h.HandleACS)
	mux.HandleFunc("/auth/saml/logout", h.HandleLogout)
	mux.HandleFunc("/auth/saml/slo", h.HandleSLO)
}

// HandleMetadata returns SAML SP metadata
// GET /auth/saml/metadata
func (h *SAMLHandlers) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metadata, err := h.provider.GetMetadata()
	if err != nil {
		h.logger.Error("Failed to generate SAML metadata", "error", err)
		http.Error(w, "Failed to generate metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(metadata)
}

// HandleLogin initiates SAML SSO login
// GET /auth/saml/login
func (h *SAMLHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get relay state (where to redirect after login)
	relayState := r.URL.Query().Get("redirect")
	if relayState == "" {
		relayState = h.defaultRedirect
	}

	// Initiate SSO
	if err := h.provider.HandleSSORequest(w, r); err != nil {
		h.logger.Error("Failed to initiate SAML SSO", "error", err)
		http.Error(w, "Failed to initiate login", http.StatusInternalServerError)
		return
	}
}

// HandleACS handles the SAML Assertion Consumer Service callback
// POST /auth/saml/acs
func (h *SAMLHandlers) HandleACS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Process SAML response
	user, err := h.provider.HandleACS(w, r)
	if err != nil {
		h.logger.Error("SAML ACS failed", "error", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Create session
	sessionID, err := h.sessionManager.CreateSession(user)
	if err != nil {
		h.logger.Error("Failed to create session", "error", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	})

	h.logger.Info("SAML authentication successful",
		"email", user.Email,
		"groups", len(user.Groups))

	// Get relay state for redirect
	relayState := r.FormValue("RelayState")
	if relayState == "" {
		relayState = h.defaultRedirect
	}

	// Redirect to intended destination
	http.Redirect(w, r, relayState, http.StatusFound)
}

// HandleLogout handles SP-initiated logout
// POST /auth/saml/logout
func (h *SAMLHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session
	cookie, err := r.Cookie("session_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Destroy session
	if err := h.sessionManager.DestroySession(cookie.Value); err != nil {
		h.logger.Error("Failed to destroy session", "error", err)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})

	h.logger.Info("User logged out")

	// Redirect to home or IdP logout
	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleSLO handles IdP-initiated Single Logout
// GET/POST /auth/saml/slo
func (h *SAMLHandlers) HandleSLO(w http.ResponseWriter, r *http.Request) {
	// Parse SAML logout request/response
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	samlRequest := r.FormValue("SAMLRequest")
	samlResponse := r.FormValue("SAMLResponse")

	if samlRequest != "" {
		// IdP-initiated logout request
		h.handleSLORequest(w, r, samlRequest)
	} else if samlResponse != "" {
		// Logout response from IdP
		h.handleSLOResponse(w, r, samlResponse)
	} else {
		http.Error(w, "Invalid SLO request", http.StatusBadRequest)
	}
}

func (h *SAMLHandlers) handleSLORequest(w http.ResponseWriter, r *http.Request, request string) {
	h.logger.Info("Processing IdP-initiated logout")

	// TODO: Parse logout request and extract session info
	// For now, just destroy the current session

	cookie, err := r.Cookie("session_id")
	if err == nil {
		h.sessionManager.DestroySession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})

	// TODO: Send logout response to IdP
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *SAMLHandlers) handleSLOResponse(w http.ResponseWriter, r *http.Request, response string) {
	h.logger.Info("Processing SLO response from IdP")

	// Logout complete
	http.Redirect(w, r, "/", http.StatusFound)
}

// InMemorySessionManager is a simple in-memory session manager for development
type InMemorySessionManager struct {
	sessions map[string]*Session
}

// NewInMemorySessionManager creates a new in-memory session manager
func NewInMemorySessionManager() *InMemorySessionManager {
	return &InMemorySessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session
func (m *InMemorySessionManager) CreateSession(user *SAMLUser) (string, error) {
	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())

	session := &Session{
		ID:          sessionID,
		UserID:      user.NameID,
		Email:       user.Email,
		Groups:      user.Groups,
		SAMLSession: user.SessionIndex,
		CreatedAt:   time.Now(),
		ExpiresAt:   user.NotAfter,
	}

	m.sessions[sessionID] = session
	return sessionID, nil
}

// GetSession retrieves a session
func (m *InMemorySessionManager) GetSession(sessionID string) (*Session, error) {
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(m.sessions, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// DestroySession removes a session
func (m *InMemorySessionManager) DestroySession(sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

// SAMLConfigHandler handles SAML configuration API
type SAMLConfigHandler struct {
	configs map[string]*SAMLConfig
	logger  logger.Logger
}

// NewSAMLConfigHandler creates a new config handler
func NewSAMLConfigHandler(log logger.Logger) *SAMLConfigHandler {
	return &SAMLConfigHandler{
		configs: make(map[string]*SAMLConfig),
		logger:  log,
	}
}

// HandleListConfigs lists all SAML configurations
// GET /api/admin/saml/configs
func (h *SAMLConfigHandler) HandleListConfigs(w http.ResponseWriter, r *http.Request) {
	configs := make([]*SAMLConfigInfo, 0, len(h.configs))
	for id, cfg := range h.configs {
		configs = append(configs, &SAMLConfigInfo{
			ID:           id,
			EntityID:     cfg.EntityID,
			ACSUrl:       cfg.ACSUrl,
			IdPSSOUrl:    cfg.IdPSSOUrl,
			SignRequests: cfg.SignAuthnRequests,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// SAMLConfigInfo is a safe view of SAML config (without keys)
type SAMLConfigInfo struct {
	ID           string `json:"id"`
	EntityID     string `json:"entityId"`
	ACSUrl       string `json:"acsUrl"`
	IdPSSOUrl    string `json:"idpSsoUrl"`
	SignRequests bool   `json:"signRequests"`
}

// IdPPresets contains pre-configured settings for popular IdPs
var IdPPresets = map[string]IdPPreset{
	"azure": {
		Name:              "Azure AD (Entra ID)",
		MetadataURLFormat: "https://login.microsoftonline.com/%s/federationmetadata/2007-06/federationmetadata.xml",
		AttributeMappings: map[string]string{
			"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "email",
			"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname":    "first_name",
			"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname":      "last_name",
			"http://schemas.microsoft.com/ws/2008/06/identity/claims/groups":     "groups",
		},
	},
	"okta": {
		Name:              "Okta",
		MetadataURLFormat: "https://%s.okta.com/app/%s/sso/saml/metadata",
		AttributeMappings: map[string]string{
			"email":     "email",
			"firstName": "first_name",
			"lastName":  "last_name",
			"groups":    "groups",
		},
	},
	"google": {
		Name:              "Google Workspace",
		MetadataURLFormat: "https://accounts.google.com/gsi/saml/metadata/%s",
		AttributeMappings: map[string]string{
			"email":     "email",
			"firstName": "first_name",
			"lastName":  "last_name",
		},
	},
}

// IdPPreset contains pre-configured IdP settings
type IdPPreset struct {
	Name              string
	MetadataURLFormat string
	AttributeMappings map[string]string
}
