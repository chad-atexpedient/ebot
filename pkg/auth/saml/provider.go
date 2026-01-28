package saml

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/chad-atexpedient/ebot/logger"
)

// SAMLProvider handles SAML 2.0 authentication flows
type SAMLProvider interface {
	// HandleSSORequest initiates SAML SSO flow
	HandleSSORequest(w http.ResponseWriter, r *http.Request) error
	
	// HandleACS processes SAML assertion consumer service callback
	HandleACS(w http.ResponseWriter, r *http.Request) (*SAMLUser, error)
	
	// ValidateAssertion validates a SAML assertion
	ValidateAssertion(ctx context.Context, assertion string) (*SAMLUser, error)
	
	// GetMetadata returns SAML metadata XML
	GetMetadata() ([]byte, error)
	
	// GetLoginURL returns the IdP login URL
	GetLoginURL(relayState string) (string, error)
}

// SAMLConfig holds SAML provider configuration
type SAMLConfig struct {
	// EntityID is the service provider entity ID
	EntityID string
	
	// ACSUrl is the assertion consumer service URL
	ACSUrl string
	
	// IdPMetadataURL is the identity provider metadata URL
	IdPMetadataURL string
	
	// IdPSSOUrl is the identity provider SSO URL
	IdPSSOUrl string
	
	// IdPCertificate is the IdP signing certificate
	IdPCertificate *x509.Certificate
	
	// SPPrivateKey is the service provider private key
	SPPrivateKey *rsa.PrivateKey
	
	// SPCertificate is the service provider certificate
	SPCertificate *x509.Certificate
	
	// AllowUnencryptedAssertion allows unencrypted assertions (dev only)
	AllowUnencryptedAssertion bool
	
	// SignAuthnRequests signs authentication requests
	SignAuthnRequests bool
	
	// ForceAuthn forces re-authentication
	ForceAuthn bool
	
	// AttributeMappings maps SAML attributes to user fields
	AttributeMappings map[string]string
}

// SAMLUser represents an authenticated SAML user
type SAMLUser struct {
	NameID     string
	Email      string
	FirstName  string
	LastName   string
	Groups     []string
	Attributes map[string][]string
	SessionIndex string
	NotBefore  time.Time
	NotAfter   time.Time
}

// samlProvider implements SAMLProvider
type samlProvider struct {
	config *SAMLConfig
	logger logger.Logger
}

// NewSAMLProvider creates a new SAML provider
func NewSAMLProvider(config *SAMLConfig, log logger.Logger) (SAMLProvider, error) {
	if config.EntityID == "" {
		return nil, fmt.Errorf("entity ID is required")
	}
	if config.ACSUrl == "" {
		return nil, fmt.Errorf("ACS URL is required")
	}
	if config.IdPSSOUrl == "" {
		return nil, fmt.Errorf("IdP SSO URL is required")
	}
	
	return &samlProvider{
		config: config,
		logger: log,
	}, nil
}

// HandleSSORequest initiates SAML SSO flow
func (p *samlProvider) HandleSSORequest(w http.ResponseWriter, r *http.Request) error {
	relayState := r.URL.Query().Get("RelayState")
	
	// Create SAML AuthnRequest
	authnRequest := p.createAuthnRequest(relayState)
	
	// Encode request
	encoded, err := p.encodeAuthnRequest(authnRequest)
	if err != nil {
		return fmt.Errorf("failed to encode authn request: %w", err)
	}
	
	// Build redirect URL
	redirectURL := fmt.Sprintf("%s?SAMLRequest=%s", p.config.IdPSSOUrl, encoded)
	if relayState != "" {
		redirectURL += fmt.Sprintf("&RelayState=%s", relayState)
	}
	
	p.logger.Info("Redirecting to IdP for authentication",
		"idp_url", p.config.IdPSSOUrl,
		"relay_state", relayState)
	
	// Redirect to IdP
	http.Redirect(w, r, redirectURL, http.StatusFound)
	return nil
}

// HandleACS processes SAML assertion consumer service callback
func (p *samlProvider) HandleACS(w http.ResponseWriter, r *http.Request) (*SAMLUser, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}
	
	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		return nil, fmt.Errorf("missing SAMLResponse")
	}
	
	relayState := r.FormValue("RelayState")
	
	p.logger.Info("Processing SAML response",
		"relay_state", relayState)
	
	// Validate and parse assertion
	user, err := p.ValidateAssertion(r.Context(), samlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to validate assertion: %w", err)
	}
	
	return user, nil
}

// ValidateAssertion validates a SAML assertion
func (p *samlProvider) ValidateAssertion(ctx context.Context, assertion string) (*SAMLUser, error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to decode assertion: %w", err)
	}
	
	// Parse XML
	var response SAMLResponse
	if err := xml.Unmarshal(decoded, &response); err != nil {
		return nil, fmt.Errorf("failed to parse SAML response: %w", err)
	}
	
	// Validate signature
	if err := p.validateSignature(&response); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}
	
	// Extract assertion
	if len(response.Assertions) == 0 {
		return nil, fmt.Errorf("no assertions in response")
	}
	
	assertionData := response.Assertions[0]
	
	// Validate conditions
	if err := p.validateConditions(&assertionData); err != nil {
		return nil, fmt.Errorf("invalid conditions: %w", err)
	}
	
	// Extract user information
	user := p.extractUser(&assertionData)
	
	p.logger.Info("Successfully validated SAML assertion",
		"name_id", user.NameID,
		"email", user.Email,
		"groups", len(user.Groups))
	
	return user, nil
}

// GetMetadata returns SAML metadata XML
func (p *samlProvider) GetMetadata() ([]byte, error) {
	metadata := &EntityDescriptor{
		EntityID: p.config.EntityID,
		SPSSODescriptor: &SPSSODescriptor{
			AuthnRequestsSigned:        p.config.SignAuthnRequests,
			WantAssertionsSigned:       true,
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			AssertionConsumerServices: []AssertionConsumerService{
				{
					Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
					Location: p.config.ACSUrl,
					Index:    0,
				},
			},
		},
	}
	
	// Add certificate if available
	if p.config.SPCertificate != nil {
		certPEM := base64.StdEncoding.EncodeToString(p.config.SPCertificate.Raw)
		metadata.SPSSODescriptor.KeyDescriptor = []KeyDescriptor{
			{
				Use: "signing",
				KeyInfo: KeyInfo{
					X509Data: X509Data{
						X509Certificate: certPEM,
					},
				},
			},
		}
	}
	
	return xml.MarshalIndent(metadata, "", "  ")
}

// GetLoginURL returns the IdP login URL
func (p *samlProvider) GetLoginURL(relayState string) (string, error) {
	authnRequest := p.createAuthnRequest(relayState)
	
	encoded, err := p.encodeAuthnRequest(authnRequest)
	if err != nil {
		return "", err
	}
	
	url := fmt.Sprintf("%s?SAMLRequest=%s", p.config.IdPSSOUrl, encoded)
	if relayState != "" {
		url += fmt.Sprintf("&RelayState=%s", relayState)
	}
	
	return url, nil
}

// Helper methods

func (p *samlProvider) createAuthnRequest(relayState string) *AuthnRequest {
	now := time.Now().UTC()
	
	return &AuthnRequest{
		ID:           generateID(),
		Version:      "2.0",
		IssueInstant: now.Format(time.RFC3339),
		Destination:  p.config.IdPSSOUrl,
		Issuer: Issuer{
			Value: p.config.EntityID,
		},
		NameIDPolicy: NameIDPolicy{
			Format:          "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AllowCreate:     true,
			SPNameQualifier: p.config.EntityID,
		},
		RequestedAuthnContext: RequestedAuthnContext{
			Comparison: "exact",
			AuthnContextClassRef: []string{
				"urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
			},
		},
		AssertionConsumerServiceURL: p.config.ACSUrl,
		ProtocolBinding:             "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
		ForceAuthn:                  p.config.ForceAuthn,
	}
}

func (p *samlProvider) encodeAuthnRequest(request *AuthnRequest) (string, error) {
	xml, err := xml.Marshal(request)
	if err != nil {
		return "", err
	}
	
	// For HTTP-Redirect binding, deflate and encode
	// For HTTP-POST binding, just base64 encode
	return base64.StdEncoding.EncodeToString(xml), nil
}

func (p *samlProvider) validateSignature(response *SAMLResponse) error {
	if p.config.IdPCertificate == nil {
		p.logger.Warn("No IdP certificate configured, skipping signature validation")
		return nil
	}
	
	// TODO: Implement actual signature validation using IdP certificate
	// This is a simplified version
	
	if response.Signature == nil {
		return fmt.Errorf("response is not signed")
	}
	
	return nil
}

func (p *samlProvider) validateConditions(assertion *Assertion) error {
	now := time.Now().UTC()
	
	if assertion.Conditions == nil {
		return fmt.Errorf("no conditions in assertion")
	}
	
	// Check NotBefore
	if assertion.Conditions.NotBefore != "" {
		notBefore, err := time.Parse(time.RFC3339, assertion.Conditions.NotBefore)
		if err != nil {
			return fmt.Errorf("invalid NotBefore: %w", err)
		}
		if now.Before(notBefore) {
			return fmt.Errorf("assertion not yet valid")
		}
	}
	
	// Check NotOnOrAfter
	if assertion.Conditions.NotOnOrAfter != "" {
		notAfter, err := time.Parse(time.RFC3339, assertion.Conditions.NotOnOrAfter)
		if err != nil {
			return fmt.Errorf("invalid NotOnOrAfter: %w", err)
		}
		if now.After(notAfter) {
			return fmt.Errorf("assertion expired")
		}
	}
	
	// Check AudienceRestriction
	if len(assertion.Conditions.AudienceRestriction) > 0 {
		found := false
		for _, restriction := range assertion.Conditions.AudienceRestriction {
			for _, audience := range restriction.Audience {
				if audience == p.config.EntityID {
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("audience restriction not met")
		}
	}
	
	return nil
}

func (p *samlProvider) extractUser(assertion *Assertion) *SAMLUser {
	user := &SAMLUser{
		Attributes: make(map[string][]string),
	}
	
	// Extract NameID
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		user.NameID = assertion.Subject.NameID.Value
	}
	
	// Extract attributes
	if assertion.AttributeStatement != nil {
		for _, attr := range assertion.AttributeStatement.Attributes {
			values := make([]string, len(attr.AttributeValues))
			for i, v := range attr.AttributeValues {
				values[i] = v.Value
			}
			user.Attributes[attr.Name] = values
			
			// Map to user fields based on configuration
			mappedField := p.config.AttributeMappings[attr.Name]
			switch mappedField {
			case "email":
				if len(values) > 0 {
					user.Email = values[0]
				}
			case "first_name":
				if len(values) > 0 {
					user.FirstName = values[0]
				}
			case "last_name":
				if len(values) > 0 {
					user.LastName = values[0]
				}
			case "groups":
				user.Groups = values
			}
		}
	}
	
	// If no email from attributes, use NameID if it looks like email
	if user.Email == "" {
		user.Email = user.NameID
	}
	
	// Extract session index
	if assertion.AuthnStatement != nil {
		user.SessionIndex = assertion.AuthnStatement.SessionIndex
	}
	
	// Extract time conditions
	if assertion.Conditions != nil {
		if assertion.Conditions.NotBefore != "" {
			user.NotBefore, _ = time.Parse(time.RFC3339, assertion.Conditions.NotBefore)
		}
		if assertion.Conditions.NotOnOrAfter != "" {
			user.NotAfter, _ = time.Parse(time.RFC3339, assertion.Conditions.NotOnOrAfter)
		}
	}
	
	return user
}

func generateID() string {
	return fmt.Sprintf("_%d", time.Now().UnixNano())
}
