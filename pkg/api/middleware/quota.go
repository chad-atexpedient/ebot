package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chad-atexpedient/ebot/pkg/quota"
)

// QuotaMiddleware enforces resource quotas on API requests
type QuotaMiddleware struct {
	quotaManager quota.Manager
}

// NewQuotaMiddleware creates a new quota enforcement middleware
func NewQuotaMiddleware(qm quota.Manager) *QuotaMiddleware {
	return &QuotaMiddleware{
		quotaManager: qm,
	}
}

// EnforceMCPServerQuota checks if user can create MCP servers
func (m *QuotaMiddleware) EnforceMCPServerQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := getUserIDFromContext(ctx)
		workspaceID := getWorkspaceIDFromContext(ctx)

		// Check quota
		if err := m.quotaManager.CheckQuota(ctx, userID, quota.ResourceTypeMCPServers, 1); err != nil {
			http.Error(w, fmt.Sprintf("Quota exceeded: %v", err), http.StatusTooManyRequests)
			return
		}

		// Try to reserve quota
		if err := m.quotaManager.ReserveQuota(ctx, userID, quota.ResourceTypeMCPServers, 1); err != nil {
			http.Error(w, fmt.Sprintf("Failed to reserve quota: %v", err), http.StatusInternalServerError)
			return
		}

		// Store workspace for potential rollback
		ctx = context.WithValue(ctx, "quotaReserved", true)
		ctx = context.WithValue(ctx, "quotaUserID", userID)
		ctx = context.WithValue(ctx, "quotaWorkspaceID", workspaceID)
		ctx = context.WithValue(ctx, "quotaResourceType", quota.ResourceTypeMCPServers)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EnforceThreadQuota checks if user can create threads
func (m *QuotaMiddleware) EnforceThreadQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := getUserIDFromContext(ctx)

		if err := m.quotaManager.CheckQuota(ctx, userID, quota.ResourceTypeThreads, 1); err != nil {
			http.Error(w, fmt.Sprintf("Quota exceeded: %v", err), http.StatusTooManyRequests)
			return
		}

		if err := m.quotaManager.ReserveQuota(ctx, userID, quota.ResourceTypeThreads, 1); err != nil {
			http.Error(w, fmt.Sprintf("Failed to reserve quota: %v", err), http.StatusInternalServerError)
			return
		}

		ctx = context.WithValue(ctx, "quotaReserved", true)
		ctx = context.WithValue(ctx, "quotaUserID", userID)
		ctx = context.WithValue(ctx, "quotaResourceType", quota.ResourceTypeThreads)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EnforceKnowledgeQuota checks if user can create knowledge sets
func (m *QuotaMiddleware) EnforceKnowledgeQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := getUserIDFromContext(ctx)

		if err := m.quotaManager.CheckQuota(ctx, userID, quota.ResourceTypeKnowledgeSets, 1); err != nil {
			http.Error(w, fmt.Sprintf("Quota exceeded: %v", err), http.StatusTooManyRequests)
			return
		}

		if err := m.quotaManager.ReserveQuota(ctx, userID, quota.ResourceTypeKnowledgeSets, 1); err != nil {
			http.Error(w, fmt.Sprintf("Failed to reserve quota: %v", err), http.StatusInternalServerError)
			return
		}

		ctx = context.WithValue(ctx, "quotaReserved", true)
		ctx = context.WithValue(ctx, "quotaUserID", userID)
		ctx = context.WithValue(ctx, "quotaResourceType", quota.ResourceTypeKnowledgeSets)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EnforceStorageQuota checks if user has enough storage quota
func (m *QuotaMiddleware) EnforceStorageQuota(sizeBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := getUserIDFromContext(ctx)

			if err := m.quotaManager.CheckQuota(ctx, userID, quota.ResourceTypeStorageBytes, sizeBytes); err != nil {
				http.Error(w, fmt.Sprintf("Storage quota exceeded: %v", err), http.StatusTooManyRequests)
				return
			}

			if err := m.quotaManager.ReserveQuota(ctx, userID, quota.ResourceTypeStorageBytes, sizeBytes); err != nil {
				http.Error(w, fmt.Sprintf("Failed to reserve storage quota: %v", err), http.StatusInternalServerError)
				return
			}

			ctx = context.WithValue(ctx, "quotaReserved", true)
			ctx = context.WithValue(ctx, "quotaUserID", userID)
			ctx = context.WithValue(ctx, "quotaResourceType", quota.ResourceTypeStorageBytes)
			ctx = context.WithValue(ctx, "quotaAmount", sizeBytes)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RollbackQuotaOnError releases quota if operation fails
func (m *QuotaMiddleware) RollbackQuotaOnError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Wrap response writer to capture status code
		wrw := &wrappedResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrw, r)

		// If request failed, rollback quota
		if wrw.statusCode >= 400 {
			if reserved, ok := ctx.Value("quotaReserved").(bool); ok && reserved {
				userID := ctx.Value("quotaUserID").(string)
				resourceType := ctx.Value("quotaResourceType").(quota.ResourceType)
				amount := int64(1)
				if amt, ok := ctx.Value("quotaAmount").(int64); ok {
					amount = amt
				}

				_ = m.quotaManager.ReleaseQuota(ctx, userID, resourceType, amount)
			}
		}
	})
}

// GetQuotaUsage returns current quota usage for user
func (m *QuotaMiddleware) GetQuotaUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)

	usage, err := m.quotaManager.GetUsage(ctx, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get quota usage: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, usage)
}

// Helper functions

func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("userID").(string); ok {
		return userID
	}
	return ""
}

func getWorkspaceIDFromContext(ctx context.Context) string {
	if wsID, ok := ctx.Value("workspaceID").(string); ok {
		return wsID
	}
	return ""
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Implement JSON encoding
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
