package region

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestRegionManager_RegisterRegion(t *testing.T) {
	manager := NewManager(&RegionConfig{}, slog.Default())
	ctx := context.Background()

	region := &Region{
		ID:           "us-east-1",
		Name:         "us-east-1",
		DisplayName:  "US East",
		Location:     "Virginia",
		Status:       RegionStatusActive,
		Capabilities: []string{"mcp-hosting"},
	}

	err := manager.RegisterRegion(ctx, region)
	if err != nil {
		t.Fatalf("Failed to register region: %v", err)
	}

	// Verify region was registered
	retrieved, err := manager.GetRegion(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("Failed to get region: %v", err)
	}

	if retrieved.ID != region.ID {
		t.Errorf("Expected region ID %s, got %s", region.ID, retrieved.ID)
	}
}

func TestRegionManager_CreatePolicy(t *testing.T) {
	manager := NewManager(&RegionConfig{}, slog.Default())
	ctx := context.Background()

	// Register regions first
	regions := []*Region{
		{
			ID:          "us-east-1",
			Name:        "us-east-1",
			DisplayName: "US East",
			Status:      RegionStatusActive,
		},
		{
			ID:          "eu-west-1",
			Name:        "eu-west-1",
			DisplayName: "EU West",
			Status:      RegionStatusActive,
		},
	}

	for _, r := range regions {
		if err := manager.RegisterRegion(ctx, r); err != nil {
			t.Fatalf("Failed to register region: %v", err)
		}
	}

	// Create policy
	policy := &DataResidencyPolicy{
		ID:                     "test-policy",
		Name:                   "Test Policy",
		UserID:                 "user123",
		PrimaryRegion:          "us-east-1",
		AllowedRegions:         []string{"us-east-1", "eu-west-1"},
		CrossRegionReplication: true,
		ReplicationRegions:     []string{"eu-west-1"},
	}

	err := manager.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Verify policy was created
	retrieved, err := manager.GetPolicy(ctx, "test-policy")
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if retrieved.ID != policy.ID {
		t.Errorf("Expected policy ID %s, got %s", policy.ID, retrieved.ID)
	}
}

func TestRegionManager_ValidateRegionAccess(t *testing.T) {
	manager := NewManager(&RegionConfig{}, slog.Default())
	ctx := context.Background()

	// Register regions
	regions := []*Region{
		{ID: "us-east-1", Name: "us-east-1", Status: RegionStatusActive},
		{ID: "eu-west-1", Name: "eu-west-1", Status: RegionStatusActive},
	}

	for _, r := range regions {
		manager.RegisterRegion(ctx, r)
	}

	// Create policy restricting user to EU only
	policy := &DataResidencyPolicy{
		ID:             "eu-only",
		Name:           "EU Only",
		UserID:         "user123",
		PrimaryRegion:  "eu-west-1",
		AllowedRegions: []string{"eu-west-1"},
	}

	manager.CreatePolicy(ctx, policy)

	// Test access to allowed region
	err := manager.ValidateRegionAccess(ctx, "user123", "eu-west-1")
	if err != nil {
		t.Errorf("Expected access to eu-west-1, got error: %v", err)
	}

	// Test access to disallowed region
	err = manager.ValidateRegionAccess(ctx, "user123", "us-east-1")
	if err == nil {
		t.Error("Expected access denial to us-east-1, but got nil error")
	}
}

func TestRegionManager_RouteRequest(t *testing.T) {
	manager := NewManager(&RegionConfig{CurrentRegion: "us-east-1"}, slog.Default())
	ctx := context.Background()

	// Register regions
	regions := []*Region{
		{ID: "us-east-1", Name: "us-east-1", Status: RegionStatusActive},
		{ID: "eu-west-1", Name: "eu-west-1", Status: RegionStatusActive},
	}

	for _, r := range regions {
		manager.RegisterRegion(ctx, r)
	}

	// Create policy for user
	policy := &DataResidencyPolicy{
		ID:             "user-policy",
		UserID:         "user123",
		PrimaryRegion:  "eu-west-1",
		AllowedRegions: []string{"eu-west-1"},
	}

	manager.CreatePolicy(ctx, policy)

	// Route request for user with policy
	region, err := manager.RouteRequest(ctx, "user123", "mcp-server")
	if err != nil {
		t.Fatalf("Failed to route request: %v", err)
	}

	if region != "eu-west-1" {
		t.Errorf("Expected routing to eu-west-1, got %s", region)
	}

	// Route request for user without policy (should use current region)
	region, err = manager.RouteRequest(ctx, "user456", "mcp-server")
	if err != nil {
		t.Fatalf("Failed to route request: %v", err)
	}

	if region != "us-east-1" {
		t.Errorf("Expected routing to us-east-1, got %s", region)
	}
}

func TestRegionManager_GetAllowedRegions(t *testing.T) {
	manager := NewManager(&RegionConfig{}, slog.Default())
	ctx := context.Background()

	// Register regions
	regions := []*Region{
		{ID: "us-east-1", Name: "us-east-1", Status: RegionStatusActive},
		{ID: "eu-west-1", Name: "eu-west-1", Status: RegionStatusActive},
		{ID: "ap-south-1", Name: "ap-south-1", Status: RegionStatusOffline},
	}

	for _, r := range regions {
		manager.RegisterRegion(ctx, r)
	}

	// User with policy
	policy := &DataResidencyPolicy{
		ID:             "user-policy",
		UserID:         "user123",
		PrimaryRegion:  "eu-west-1",
		AllowedRegions: []string{"eu-west-1"},
	}

	manager.CreatePolicy(ctx, policy)

	allowed, err := manager.GetAllowedRegions(ctx, "user123")
	if err != nil {
		t.Fatalf("Failed to get allowed regions: %v", err)
	}

	if len(allowed) != 1 || allowed[0] != "eu-west-1" {
		t.Errorf("Expected [eu-west-1], got %v", allowed)
	}

	// User without policy (should get all active regions)
	allowed, err = manager.GetAllowedRegions(ctx, "user456")
	if err != nil {
		t.Fatalf("Failed to get allowed regions: %v", err)
	}

	if len(allowed) != 2 { // Only active regions
		t.Errorf("Expected 2 active regions, got %d", len(allowed))
	}
}

func TestRegionManager_UpdateRegionStatus(t *testing.T) {
	manager := NewManager(&RegionConfig{}, slog.Default())
	ctx := context.Background()

	region := &Region{
		ID:     "us-east-1",
		Name:   "us-east-1",
		Status: RegionStatusActive,
	}

	manager.RegisterRegion(ctx, region)

	// Update status
	err := manager.UpdateRegionStatus(ctx, "us-east-1", RegionStatusMaintenance)
	if err != nil {
		t.Fatalf("Failed to update region status: %v", err)
	}

	// Verify status updated
	retrieved, _ := manager.GetRegion(ctx, "us-east-1")
	if retrieved.Status != RegionStatusMaintenance {
		t.Errorf("Expected status %s, got %s", RegionStatusMaintenance, retrieved.Status)
	}
}

func TestRegionManager_ReplicationStatus(t *testing.T) {
	manager := NewManager(&RegionConfig{}, slog.Default())
	ctx := context.Background()

	// Register regions
	regions := []*Region{
		{ID: "us-east-1", Name: "us-east-1", Status: RegionStatusActive},
		{ID: "eu-west-1", Name: "eu-west-1", Status: RegionStatusActive},
	}

	for _, r := range regions {
		manager.RegisterRegion(ctx, r)
	}

	// Create policy with replication
	policy := &DataResidencyPolicy{
		ID:                     "repl-policy",
		Name:                   "Replication Policy",
		PrimaryRegion:          "us-east-1",
		AllowedRegions:         []string{"us-east-1", "eu-west-1"},
		CrossRegionReplication: true,
		ReplicationRegions:     []string{"eu-west-1"},
	}

	manager.CreatePolicy(ctx, policy)

	// Register resource
	resource := &RegionalResource{
		ID:            "resource-123",
		Type:          "mcp-server",
		PrimaryRegion: "us-east-1",
		PolicyID:      "repl-policy",
		CreatedAt:     time.Now(),
	}

	mgr := manager.(*manager)
	mgr.resources[resource.ID] = resource

	// Get replication status
	status, err := manager.GetReplicationStatus(ctx, "resource-123")
	if err != nil {
		t.Fatalf("Failed to get replication status: %v", err)
	}

	if status.PrimaryRegion != "us-east-1" {
		t.Errorf("Expected primary region us-east-1, got %s", status.PrimaryRegion)
	}

	if len(status.Replicas) != 1 {
		t.Errorf("Expected 1 replica, got %d", len(status.Replicas))
	}
}

func BenchmarkRegionManager_RouteRequest(b *testing.B) {
	manager := NewManager(&RegionConfig{CurrentRegion: "us-east-1"}, slog.Default())
	ctx := context.Background()

	// Setup
	regions := []*Region{
		{ID: "us-east-1", Name: "us-east-1", Status: RegionStatusActive},
		{ID: "eu-west-1", Name: "eu-west-1", Status: RegionStatusActive},
	}

	for _, r := range regions {
		manager.RegisterRegion(ctx, r)
	}

	policy := &DataResidencyPolicy{
		ID:             "bench-policy",
		UserID:         "user123",
		PrimaryRegion:  "eu-west-1",
		AllowedRegions: []string{"eu-west-1"},
	}

	manager.CreatePolicy(ctx, policy)

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.RouteRequest(ctx, "user123", "mcp-server")
	}
}

func BenchmarkRegionManager_ValidateRegionAccess(b *testing.B) {
	manager := NewManager(&RegionConfig{}, slog.Default())
	ctx := context.Background()

	// Setup
	region := &Region{
		ID:     "us-east-1",
		Name:   "us-east-1",
		Status: RegionStatusActive,
	}

	manager.RegisterRegion(ctx, region)

	policy := &DataResidencyPolicy{
		ID:             "bench-policy",
		UserID:         "user123",
		PrimaryRegion:  "us-east-1",
		AllowedRegions: []string{"us-east-1"},
	}

	manager.CreatePolicy(ctx, policy)

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ValidateRegionAccess(ctx, "user123", "us-east-1")
	}
}
