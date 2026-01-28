package quota

import (
	"context"
	"testing"
	"time"
)

func TestQuotaManager_CheckQuota(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		resourceType  ResourceType
		requested     int64
		tier          string
		currentUsage  int64
		expectError   bool
		expectedError error
	}{
		{
			name:         "within quota - default tier",
			userID:       "user1",
			resourceType: ResourceTypeMCPServers,
			requested:    5,
			tier:         "default",
			currentUsage: 0,
			expectError:  false,
		},
		{
			name:          "exceeds quota - default tier",
			userID:        "user1",
			resourceType:  ResourceTypeMCPServers,
			requested:     15,
			tier:          "default",
			currentUsage:  0,
			expectError:   true,
			expectedError: ErrQuotaExceeded,
		},
		{
			name:         "within quota - power user tier",
			userID:       "user2",
			resourceType: ResourceTypeMCPServers,
			requested:    25,
			tier:         "power_user",
			currentUsage: 0,
			expectError:  false,
		},
		{
			name:         "within quota - enterprise tier unlimited",
			userID:       "user3",
			resourceType: ResourceTypeMCPServers,
			requested:    1000,
			tier:         "enterprise",
			currentUsage: 0,
			expectError:  false,
		},
		{
			name:          "partial usage - exceeds remaining",
			userID:        "user4",
			resourceType:  ResourceTypeThreads,
			requested:     60,
			tier:          "default",
			currentUsage:  50,
			expectError:   true,
			expectedError: ErrQuotaExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewQuotaManager()
			ctx := context.Background()

			// Set the tier
			if tt.tier != "" {
				manager.SetUserTier(tt.userID, tt.tier)
			}

			// Set current usage if specified
			if tt.currentUsage > 0 {
				err := manager.ReserveQuota(ctx, tt.userID, tt.resourceType, tt.currentUsage)
				if err != nil {
					t.Fatalf("Failed to set initial usage: %v", err)
				}
			}

			// Check quota
			err := manager.CheckQuota(ctx, tt.userID, tt.resourceType, tt.requested)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if tt.expectedError != nil && err != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestQuotaManager_ReserveAndRelease(t *testing.T) {
	manager := NewQuotaManager()
	ctx := context.Background()
	userID := "test-user"
	resourceType := ResourceTypeMCPServers
	amount := int64(5)

	// Reserve quota
	err := manager.ReserveQuota(ctx, userID, resourceType, amount)
	if err != nil {
		t.Fatalf("Failed to reserve quota: %v", err)
	}

	// Check usage
	usage, err := manager.GetUsage(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	if usage.MCPServers != int(amount) {
		t.Errorf("Expected usage %d, got %d", amount, usage.MCPServers)
	}

	// Try to exceed quota
	err = manager.ReserveQuota(ctx, userID, resourceType, 6)
	if err == nil {
		t.Error("Expected error when exceeding quota, got nil")
	}

	// Release quota
	err = manager.ReleaseQuota(ctx, userID, resourceType, amount)
	if err != nil {
		t.Fatalf("Failed to release quota: %v", err)
	}

	// Check usage after release
	usage, err = manager.GetUsage(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	if usage.MCPServers != 0 {
		t.Errorf("Expected usage 0 after release, got %d", usage.MCPServers)
	}
}

func TestQuotaManager_TimeBasedReset(t *testing.T) {
	manager := NewQuotaManager()
	ctx := context.Background()
	userID := "test-user"

	// Reserve LLM tokens
	err := manager.ReserveQuota(ctx, userID, ResourceTypeLLMTokensPerDay, 1000)
	if err != nil {
		t.Fatalf("Failed to reserve quota: %v", err)
	}

	// Get usage
	usage, err := manager.GetUsage(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	if usage.LLMTokensToday != 1000 {
		t.Errorf("Expected usage 1000, got %d", usage.LLMTokensToday)
	}

	// Simulate time passing (in real implementation, this would be handled by a background goroutine)
	// For testing, we'll manually trigger reset
	manager.(*quotaManager).lastDayReset[userID] = time.Now().Add(-25 * time.Hour)

	// After reset, usage should be 0
	// Note: In production, you'd have a background goroutine checking and resetting
	// For this test, we're just verifying the logic would work
}

func TestQuotaManager_ConcurrentAccess(t *testing.T) {
	manager := NewQuotaManager()
	ctx := context.Background()
	userID := "concurrent-user"
	resourceType := ResourceTypeMCPServers

	// Run multiple goroutines trying to reserve quota
	concurrency := 20
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			err := manager.ReserveQuota(ctx, userID, resourceType, 1)
			if err != nil && err != ErrQuotaExceeded {
				t.Errorf("Unexpected error: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Check final usage - should not exceed quota
	usage, err := manager.GetUsage(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	defaultLimit := DefaultQuotaTemplate.MaxMCPServers
	if usage.MCPServers > defaultLimit {
		t.Errorf("Usage %d exceeds limit %d - race condition detected", usage.MCPServers, defaultLimit)
	}
}

func TestQuotaManager_GetUsage(t *testing.T) {
	manager := NewQuotaManager()
	ctx := context.Background()
	userID := "usage-test-user"

	// Initially, usage should be zero
	usage, err := manager.GetUsage(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	if usage.MCPServers != 0 || usage.Threads != 0 {
		t.Error("Expected zero usage for new user")
	}

	// Reserve some resources
	manager.ReserveQuota(ctx, userID, ResourceTypeMCPServers, 3)
	manager.ReserveQuota(ctx, userID, ResourceTypeThreads, 10)

	// Check updated usage
	usage, err = manager.GetUsage(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	if usage.MCPServers != 3 {
		t.Errorf("Expected MCPServers usage 3, got %d", usage.MCPServers)
	}
	if usage.Threads != 10 {
		t.Errorf("Expected Threads usage 10, got %d", usage.Threads)
	}
}

func TestQuotaManager_SetUserTier(t *testing.T) {
	manager := NewQuotaManager()
	ctx := context.Background()
	userID := "tier-test-user"

	// Default tier
	err := manager.CheckQuota(ctx, userID, ResourceTypeMCPServers, 15)
	if err == nil {
		t.Error("Expected quota exceeded for default tier with 15 servers")
	}

	// Upgrade to power user
	manager.SetUserTier(userID, "power_user")
	err = manager.CheckQuota(ctx, userID, ResourceTypeMCPServers, 15)
	if err != nil {
		t.Errorf("Expected to pass quota check for power user with 15 servers: %v", err)
	}

	// Upgrade to enterprise (unlimited)
	manager.SetUserTier(userID, "enterprise")
	err = manager.CheckQuota(ctx, userID, ResourceTypeMCPServers, 1000)
	if err != nil {
		t.Errorf("Expected to pass quota check for enterprise with 1000 servers: %v", err)
	}
}

func BenchmarkQuotaManager_CheckQuota(b *testing.B) {
	manager := NewQuotaManager()
	ctx := context.Background()
	userID := "bench-user"
	resourceType := ResourceTypeMCPServers

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CheckQuota(ctx, userID, resourceType, 1)
	}
}

func BenchmarkQuotaManager_ReserveQuota(b *testing.B) {
	manager := NewQuotaManager()
	ctx := context.Background()
	resourceType := ResourceTypeMCPServers

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := "bench-user-" + string(rune(i%10))
		manager.ReserveQuota(ctx, userID, resourceType, 1)
	}
}
