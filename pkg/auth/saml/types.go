package saml

import "encoding/xml"

// SAML XML structures

// AuthnRequest represents a SAML authentication request
type AuthnRequest struct {
	XMLName                     xml.Name               `xml:"urn:oasis:names:tc:SAML:2.0:protocol AuthnRequest"`
	ID                          string                 `xml:"ID,attr"`
	Version                     string                 `xml:"Version,attr"`
	IssueInstant                string                 `xml:"IssueInstant,attr"`
	Destination                 string                 `xml:"Destination,attr"`
	AssertionConsumerServiceURL string                 `xml:"AssertionConsumerServiceURL,attr"`
	ProtocolBinding             string                 `xml:"ProtocolBinding,attr"`
	ForceAuthn                  bool                   `xml:"ForceAuthn,attr,omitempty"`
	Issuer                      Issuer                 `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	NameIDPolicy                NameIDPolicy           `xml:"urn:oasis:names:tc:SAML:2.0:protocol NameIDPolicy"`
	RequestedAuthnContext       RequestedAuthnContext  `xml:"urn:oasis:names:tc:SAML:2.0:protocol RequestedAuthnContext,omitempty"`
	Signature                   *Signature             `xml:"Signature,omitempty"`
}

// SAMLResponse represents a SAML response
type SAMLResponse struct {
	XMLName      xml.Name     `xml:"urn:oasis:names:tc:SAML:2.0:protocol Response"`
	ID           string       `xml:"ID,attr"`
	InResponseTo string       `xml:"InResponseTo,attr"`
	Version      string       `xml:"Version,attr"`
	IssueInstant string       `xml:"IssueInstant,attr"`
	Destination  string       `xml:"Destination,attr"`
	Issuer       Issuer       `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Status       Status       `xml:"urn:oasis:names:tc:SAML:2.0:protocol Status"`
	Assertions   []Assertion  `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	Signature    *Signature   `xml:"Signature,omitempty"`
}

// Assertion represents a SAML assertion
type Assertion struct {
	XMLName            xml.Name            `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	ID                 string              `xml:"ID,attr"`
	Version            string              `xml:"Version,attr"`
	IssueInstant       string              `xml:"IssueInstant,attr"`
	Issuer             Issuer              `xml:"Issuer"`
	Subject            *Subject            `xml:"Subject,omitempty"`
	Conditions         *Conditions         `xml:"Conditions,omitempty"`
	AuthnStatement     *AuthnStatement     `xml:"AuthnStatement,omitempty"`
	AttributeStatement *AttributeStatement `xml:"AttributeStatement,omitempty"`
	Signature          *Signature          `xml:"Signature,omitempty"`
}

// Issuer represents the entity that issued the assertion
type Issuer struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Value   string   `xml:",chardata"`
	Format  string   `xml:"Format,attr,omitempty"`
}

// Subject represents the subject of the assertion
type Subject struct {
	XMLName             xml.Name             `xml:"urn:oasis:names:tc:SAML:2.0:assertion Subject"`
	NameID              *NameID              `xml:"NameID,omitempty"`
	SubjectConfirmation *SubjectConfirmation `xml:"SubjectConfirmation,omitempty"`
}

// NameID represents the name identifier
type NameID struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`
	Format  string   `xml:"Format,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// NameIDPolicy represents the name ID policy
type NameIDPolicy struct {
	XMLName         xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol NameIDPolicy"`
	Format          string   `xml:"Format,attr,omitempty"`
	AllowCreate     bool     `xml:"AllowCreate,attr,omitempty"`
	SPNameQualifier string   `xml:"SPNameQualifier,attr,omitempty"`
}

// SubjectConfirmation represents subject confirmation
type SubjectConfirmation struct {
	XMLName                 xml.Name                 `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmation"`
	Method                  string                   `xml:"Method,attr"`
	SubjectConfirmationData *SubjectConfirmationData `xml:"SubjectConfirmationData,omitempty"`
}

// SubjectConfirmationData represents subject confirmation data
type SubjectConfirmationData struct {
	XMLName      xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmationData"`
	InResponseTo string   `xml:"InResponseTo,attr,omitempty"`
	NotOnOrAfter string   `xml:"NotOnOrAfter,attr,omitempty"`
	Recipient    string   `xml:"Recipient,attr,omitempty"`
}

// Conditions represents assertion conditions
type Conditions struct {
	XMLName              xml.Name              `xml:"urn:oasis:names:tc:SAML:2.0:assertion Conditions"`
	NotBefore            string                `xml:"NotBefore,attr,omitempty"`
	NotOnOrAfter         string                `xml:"NotOnOrAfter,attr,omitempty"`
	AudienceRestriction  []AudienceRestriction `xml:"AudienceRestriction,omitempty"`
}

// AudienceRestriction represents audience restriction
type AudienceRestriction struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AudienceRestriction"`
	Audience []string `xml:"Audience"`
}

// AuthnStatement represents authentication statement
type AuthnStatement struct {
	XMLName             xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnStatement"`
	AuthnInstant        string   `xml:"AuthnInstant,attr"`
	SessionIndex        string   `xml:"SessionIndex,attr,omitempty"`
	SessionNotOnOrAfter string   `xml:"SessionNotOnOrAfter,attr,omitempty"`
	AuthnContext        AuthnContext
}

// AuthnContext represents authentication context
type AuthnContext struct {
	XMLName              xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnContext"`
	AuthnContextClassRef string   `xml:"AuthnContextClassRef"`
}

// RequestedAuthnContext represents requested authentication context
type RequestedAuthnContext struct {
	XMLName              xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol RequestedAuthnContext"`
	Comparison           string   `xml:"Comparison,attr,omitempty"`
	AuthnContextClassRef []string `xml:"urn:oasis:names:tc:SAML:2.0:assertion AuthnContextClassRef"`
}

// AttributeStatement represents attribute statement
type AttributeStatement struct {
	XMLName    xml.Name    `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeStatement"`
	Attributes []Attribute `xml:"Attribute"`
}

// Attribute represents a SAML attribute
type Attribute struct {
	XMLName         xml.Name         `xml:"urn:oasis:names:tc:SAML:2.0:assertion Attribute"`
	Name            string           `xml:"Name,attr"`
	NameFormat      string           `xml:"NameFormat,attr,omitempty"`
	FriendlyName    string           `xml:"FriendlyName,attr,omitempty"`
	AttributeValues []AttributeValue `xml:"AttributeValue"`
}

// AttributeValue represents an attribute value
type AttributeValue struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeValue"`
	Type    string   `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// Status represents response status
type Status struct {
	XMLName    xml.Name    `xml:"urn:oasis:names:tc:SAML:2.0:protocol Status"`
	StatusCode StatusCode  `xml:"StatusCode"`
	StatusMessage string   `xml:"StatusMessage,omitempty"`
}

// StatusCode represents status code
type StatusCode struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol StatusCode"`
	Value   string   `xml:"Value,attr"`
}

// Signature represents XML signature
type Signature struct {
	XMLName        xml.Name       `xml:"http://www.w3.org/2000/09/xmldsig# Signature"`
	SignedInfo     SignedInfo     `xml:"SignedInfo"`
	SignatureValue SignatureValue `xml:"SignatureValue"`
	KeyInfo        *KeyInfo       `xml:"KeyInfo,omitempty"`
}

// SignedInfo represents signed info
type SignedInfo struct {
	XMLName                xml.Name               `xml:"SignedInfo"`
	CanonicalizationMethod CanonicalizationMethod `xml:"CanonicalizationMethod"`
	SignatureMethod        SignatureMethod        `xml:"SignatureMethod"`
	Reference              Reference              `xml:"Reference"`
}

// CanonicalizationMethod represents canonicalization method
type CanonicalizationMethod struct {
	XMLName   xml.Name `xml:"CanonicalizationMethod"`
	Algorithm string   `xml:"Algorithm,attr"`
}

// SignatureMethod represents signature method
type SignatureMethod struct {
	XMLName   xml.Name `xml:"SignatureMethod"`
	Algorithm string   `xml:"Algorithm,attr"`
}

// Reference represents signature reference
type Reference struct {
	XMLName      xml.Name      `xml:"Reference"`
	URI          string        `xml:"URI,attr"`
	Transforms   Transforms    `xml:"Transforms"`
	DigestMethod DigestMethod  `xml:"DigestMethod"`
	DigestValue  string        `xml:"DigestValue"`
}

// Transforms represents signature transforms
type Transforms struct {
	XMLName   xml.Name    `xml:"Transforms"`
	Transform []Transform `xml:"Transform"`
}

// Transform represents a single transform
type Transform struct {
	XMLName   xml.Name `xml:"Transform"`
	Algorithm string   `xml:"Algorithm,attr"`
}

// DigestMethod represents digest method
type DigestMethod struct {
	XMLName   xml.Name `xml:"DigestMethod"`
	Algorithm string   `xml:"Algorithm,attr"`
}

// SignatureValue represents signature value
type SignatureValue struct {
	XMLName xml.Name `xml:"SignatureValue"`
	Value   string   `xml:",chardata"`
}

// KeyInfo represents key information
type KeyInfo struct {
	XMLName  xml.Name `xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
	X509Data X509Data `xml:"X509Data"`
}

// X509Data represents X.509 data
type X509Data struct {
	XMLName         xml.Name `xml:"X509Data"`
	X509Certificate string   `xml:"X509Certificate"`
}

// Metadata structures

// EntityDescriptor represents SAML metadata
type EntityDescriptor struct {
	XMLName         xml.Name         `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	EntityID        string           `xml:"entityID,attr"`
	SPSSODescriptor *SPSSODescriptor `xml:"SPSSODescriptor,omitempty"`
	IDPSSODescriptor *IDPSSODescriptor `xml:"IDPSSODescriptor,omitempty"`
}

// SPSSODescriptor represents service provider SSO descriptor
type SPSSODescriptor struct {
	XMLName                    xml.Name                    `xml:"urn:oasis:names:tc:SAML:2.0:metadata SPSSODescriptor"`
	AuthnRequestsSigned        bool                        `xml:"AuthnRequestsSigned,attr"`
	WantAssertionsSigned       bool                        `xml:"WantAssertionsSigned,attr"`
	ProtocolSupportEnumeration string                      `xml:"protocolSupportEnumeration,attr"`
	KeyDescriptor              []KeyDescriptor             `xml:"KeyDescriptor,omitempty"`
	AssertionConsumerServices  []AssertionConsumerService  `xml:"AssertionConsumerService"`
	SingleLogoutServices       []SingleLogoutService       `xml:"SingleLogoutService,omitempty"`
}

// IDPSSODescriptor represents identity provider SSO descriptor
type IDPSSODescriptor struct {
	XMLName                    xml.Name                   `xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
	ProtocolSupportEnumeration string                     `xml:"protocolSupportEnumeration,attr"`
	KeyDescriptor              []KeyDescriptor            `xml:"KeyDescriptor,omitempty"`
	SingleSignOnServices       []SingleSignOnService      `xml:"SingleSignOnService"`
	SingleLogoutServices       []SingleLogoutService      `xml:"SingleLogoutService,omitempty"`
}

// KeyDescriptor represents key descriptor
type KeyDescriptor struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata KeyDescriptor"`
	Use     string   `xml:"use,attr,omitempty"`
	KeyInfo KeyInfo  `xml:"KeyInfo"`
}

// AssertionConsumerService represents assertion consumer service
type AssertionConsumerService struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata AssertionConsumerService"`
	Binding  string   `xml:"Binding,attr"`
	Location string   `xml:"Location,attr"`
	Index    int      `xml:"index,attr"`
}

// SingleSignOnService represents single sign-on service
type SingleSignOnService struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata SingleSignOnService"`
	Binding  string   `xml:"Binding,attr"`
	Location string   `xml:"Location,attr"`
}

// SingleLogoutService represents single logout service
type SingleLogoutService struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata SingleLogoutService"`
	Binding  string   `xml:"Binding,attr"`
	Location string   `xml:"Location,attr"`
}
