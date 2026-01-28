package security

import (
	"net/http"
	"strings"
)

// SecurityHeadersConfig configures security headers
type SecurityHeadersConfig struct {
	// Content Security Policy
	CSP CSPConfig

	// Strict-Transport-Security
	EnableHSTS       bool
	HSTSMaxAge       int  // seconds
	HSTSIncludeSubdomains bool
	HSTSPreload      bool

	// X-Frame-Options
	FrameOptions string // DENY, SAMEORIGIN, or ALLOW-FROM uri

	// X-Content-Type-Options
	EnableNoSniff bool

	// X-XSS-Protection
	EnableXSSProtection bool

	// Referrer-Policy
	ReferrerPolicy string // no-referrer, no-referrer-when-downgrade, etc.

	// Permissions-Policy (formerly Feature-Policy)
	PermissionsPolicy map[string][]string // feature -> allowed origins

	// Cross-Origin policies
	CrossOriginOpenerPolicy   string // same-origin, same-origin-allow-popups, unsafe-none
	CrossOriginResourcePolicy string // same-origin, same-site, cross-origin
	CrossOriginEmbedderPolicy string // require-corp, unsafe-none
}

// CSPConfig configures Content Security Policy
type CSPConfig struct {
	Enable bool

	// Directives
	DefaultSrc       []string
	ScriptSrc        []string
	StyleSrc         []string
	ImgSrc           []string
	FontSrc          []string
	ConnectSrc       []string
	FrameSrc         []string
	ObjectSrc        []string
	MediaSrc         []string
	WorkerSrc        []string
	ChildSrc         []string
	FrameAncestors   []string
	BaseURI          []string
	FormAction       []string
	UpgradeInsecureRequests bool
	BlockAllMixedContent    bool
	ReportURI        string
	ReportTo         string
}

// DefaultSecurityHeaders returns secure default configuration
func DefaultSecurityHeaders() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		CSP: CSPConfig{
			Enable:       true,
			DefaultSrc:   []string{"'self'"},
			ScriptSrc:    []string{"'self'", "'unsafe-inline'", "'unsafe-eval'"}, // Adjust based on needs
			StyleSrc:     []string{"'self'", "'unsafe-inline'"},
			ImgSrc:       []string{"'self'", "data:", "https:"},
			FontSrc:      []string{"'self'", "data:"},
			ConnectSrc:   []string{"'self'"},
			FrameSrc:     []string{"'self'"},
			ObjectSrc:    []string{"'none'"},
			MediaSrc:     []string{"'self'"},
			WorkerSrc:    []string{"'self'"},
			FrameAncestors: []string{"'self'"},
			BaseURI:      []string{"'self'"},
			FormAction:   []string{"'self'"},
			UpgradeInsecureRequests: true,
			BlockAllMixedContent:    true,
		},
		EnableHSTS:              true,
		HSTSMaxAge:              31536000, // 1 year
		HSTSIncludeSubdomains:   true,
		HSTSPreload:             true,
		FrameOptions:            "DENY",
		EnableNoSniff:           true,
		EnableXSSProtection:     true,
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy: map[string][]string{
			"camera":         {},
			"microphone":     {},
			"geolocation":    {},
			"payment":        {},
			"usb":            {},
			"magnetometer":   {},
			"gyroscope":      {},
			"accelerometer":  {},
		},
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		CrossOriginEmbedderPolicy: "require-corp",
	}
}

// SecurityHeadersMiddleware returns HTTP middleware that sets security headers
func SecurityHeadersMiddleware(config *SecurityHeadersConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultSecurityHeaders()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Content Security Policy
			if config.CSP.Enable {
				csp := buildCSP(&config.CSP)
				w.Header().Set("Content-Security-Policy", csp)
			}

			// Strict-Transport-Security
			if config.EnableHSTS {
				hsts := buildHSTS(config)
				w.Header().Set("Strict-Transport-Security", hsts)
			}

			// X-Frame-Options
			if config.FrameOptions != "" {
				w.Header().Set("X-Frame-Options", config.FrameOptions)
			}

			// X-Content-Type-Options
			if config.EnableNoSniff {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// X-XSS-Protection
			if config.EnableXSSProtection {
				w.Header().Set("X-XSS-Protection", "1; mode=block")
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy
			if len(config.PermissionsPolicy) > 0 {
				pp := buildPermissionsPolicy(config.PermissionsPolicy)
				w.Header().Set("Permissions-Policy", pp)
			}

			// Cross-Origin policies
			if config.CrossOriginOpenerPolicy != "" {
				w.Header().Set("Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			}
			if config.CrossOriginResourcePolicy != "" {
				w.Header().Set("Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			}
			if config.CrossOriginEmbedderPolicy != "" {
				w.Header().Set("Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
			}

			// Remove server identification
			w.Header().Set("Server", "")
			w.Header().Del("X-Powered-By")

			next.ServeHTTP(w, r)
		})
	}
}

// buildCSP builds Content-Security-Policy header value
func buildCSP(config *CSPConfig) string {
	var directives []string

	if len(config.DefaultSrc) > 0 {
		directives = append(directives, "default-src "+strings.Join(config.DefaultSrc, " "))
	}
	if len(config.ScriptSrc) > 0 {
		directives = append(directives, "script-src "+strings.Join(config.ScriptSrc, " "))
	}
	if len(config.StyleSrc) > 0 {
		directives = append(directives, "style-src "+strings.Join(config.StyleSrc, " "))
	}
	if len(config.ImgSrc) > 0 {
		directives = append(directives, "img-src "+strings.Join(config.ImgSrc, " "))
	}
	if len(config.FontSrc) > 0 {
		directives = append(directives, "font-src "+strings.Join(config.FontSrc, " "))
	}
	if len(config.ConnectSrc) > 0 {
		directives = append(directives, "connect-src "+strings.Join(config.ConnectSrc, " "))
	}
	if len(config.FrameSrc) > 0 {
		directives = append(directives, "frame-src "+strings.Join(config.FrameSrc, " "))
	}
	if len(config.ObjectSrc) > 0 {
		directives = append(directives, "object-src "+strings.Join(config.ObjectSrc, " "))
	}
	if len(config.MediaSrc) > 0 {
		directives = append(directives, "media-src "+strings.Join(config.MediaSrc, " "))
	}
	if len(config.WorkerSrc) > 0 {
		directives = append(directives, "worker-src "+strings.Join(config.WorkerSrc, " "))
	}
	if len(config.ChildSrc) > 0 {
		directives = append(directives, "child-src "+strings.Join(config.ChildSrc, " "))
	}
	if len(config.FrameAncestors) > 0 {
		directives = append(directives, "frame-ancestors "+strings.Join(config.FrameAncestors, " "))
	}
	if len(config.BaseURI) > 0 {
		directives = append(directives, "base-uri "+strings.Join(config.BaseURI, " "))
	}
	if len(config.FormAction) > 0 {
		directives = append(directives, "form-action "+strings.Join(config.FormAction, " "))
	}
	if config.UpgradeInsecureRequests {
		directives = append(directives, "upgrade-insecure-requests")
	}
	if config.BlockAllMixedContent {
		directives = append(directives, "block-all-mixed-content")
	}
	if config.ReportURI != "" {
		directives = append(directives, "report-uri "+config.ReportURI)
	}
	if config.ReportTo != "" {
		directives = append(directives, "report-to "+config.ReportTo)
	}

	return strings.Join(directives, "; ")
}

// buildHSTS builds Strict-Transport-Security header value
func buildHSTS(config *SecurityHeadersConfig) string {
	hsts := "max-age=" + string(rune(config.HSTSMaxAge))
	
	if config.HSTSIncludeSubdomains {
		hsts += "; includeSubDomains"
	}
	if config.HSTSPreload {
		hsts += "; preload"
	}

	return hsts
}

// buildPermissionsPolicy builds Permissions-Policy header value
func buildPermissionsPolicy(policy map[string][]string) string {
	var policies []string

	for feature, origins := range policy {
		if len(origins) == 0 {
			policies = append(policies, feature+"=()")
		} else {
			policies = append(policies, feature+"=("+strings.Join(origins, " ")+")")
		}
	}

	return strings.Join(policies, ", ")
}

// ProductionSecurityHeaders returns strict production configuration
func ProductionSecurityHeaders() *SecurityHeadersConfig {
	config := DefaultSecurityHeaders()
	
	// Stricter CSP for production
	config.CSP.ScriptSrc = []string{"'self'"} // Remove unsafe-inline, unsafe-eval
	config.CSP.StyleSrc = []string{"'self'"}
	
	return config
}

// DevelopmentSecurityHeaders returns relaxed configuration for development
func DevelopmentSecurityHeaders() *SecurityHeadersConfig {
	config := DefaultSecurityHeaders()
	
	// Relaxed for dev
	config.EnableHSTS = false
	config.CSP.ScriptSrc = []string{"'self'", "'unsafe-inline'", "'unsafe-eval'"}
	config.CSP.StyleSrc = []string{"'self'", "'unsafe-inline'"}
	config.CrossOriginOpenerPolicy = "unsafe-none"
	config.CrossOriginEmbedderPolicy = "unsafe-none"
	
	return config
}
