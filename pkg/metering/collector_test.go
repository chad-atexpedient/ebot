package metering

import (
	"context"
	"testing"
	"time"
)

func TestNewUsageCollector(t *testing.T) {
	collector := NewUsageCollector()
	if collector == nil {
		t.Fatal("NewUsageCollector returned nil")
	}

	// Clean up
	if c, ok := collector.(*usageCollector); ok {
		c.Close()
	}
}

func TestNewUsageCollectorWithConfig(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         1000,
		RetentionPeriod:   24 * time.Hour,
		CleanupInterval:   time.Minute,
		EnableAutoCleanup: false, // Disable for testing
	}

	collector := NewUsageCollectorWithConfig(config)
	if collector == nil {
		t.Fatal("NewUsageCollectorWithConfig returned nil")
	}

	c := collector.(*usageCollector)
	if c.config.MaxEvents != 1000 {
		t.Errorf("Expected MaxEvents 1000, got %d", c.config.MaxEvents)
	}

	c.Close()
}

func TestRecordLLMUsage(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()
	event := LLMUsageEvent{
		Timestamp:     time.Now(),
		UserID:        "user-1",
		WorkspaceID:   "ws-1",
		ModelProvider: "openai",
		ModelName:     "gpt-4",
		TotalTokens:   1000,
	}

	err := collector.RecordLLMUsage(ctx, event)
	if err != nil {
		t.Fatalf("RecordLLMUsage failed: %v", err)
	}

	stats := collector.GetStats()
	if stats.LLMEventCount != 1 {
		t.Errorf("Expected 1 LLM event, got %d", stats.LLMEventCount)
	}
}

func TestRecordComputeUsage(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()
	event := ComputeUsageEvent{
		Timestamp:    time.Now(),
		UserID:       "user-1",
		ResourceType: "mcp-server",
		CPUSeconds:   10.5,
	}

	err := collector.RecordComputeUsage(ctx, event)
	if err != nil {
		t.Fatalf("RecordComputeUsage failed: %v", err)
	}

	stats := collector.GetStats()
	if stats.ComputeEventCount != 1 {
		t.Errorf("Expected 1 compute event, got %d", stats.ComputeEventCount)
	}
}

func TestRecordStorageUsage(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()
	event := StorageUsageEvent{
		Timestamp:   time.Now(),
		UserID:      "user-1",
		StorageType: "knowledge-files",
		Bytes:       1024 * 1024,
	}

	err := collector.RecordStorageUsage(ctx, event)
	if err != nil {
		t.Fatalf("RecordStorageUsage failed: %v", err)
	}

	stats := collector.GetStats()
	if stats.StorageEventCount != 1 {
		t.Errorf("Expected 1 storage event, got %d", stats.StorageEventCount)
	}
}

func TestCleanup_RemovesOldEvents(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   time.Hour, // 1 hour retention
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()

	// Add an old event (2 hours ago)
	oldEvent := LLMUsageEvent{
		Timestamp:     time.Now().Add(-2 * time.Hour),
		UserID:        "user-old",
		ModelProvider: "openai",
		ModelName:     "gpt-4",
	}
	collector.RecordLLMUsage(ctx, oldEvent)

	// Add a recent event
	recentEvent := LLMUsageEvent{
		Timestamp:     time.Now(),
		UserID:        "user-new",
		ModelProvider: "openai",
		ModelName:     "gpt-4",
	}
	collector.RecordLLMUsage(ctx, recentEvent)

	// Verify both events exist
	stats := collector.GetStats()
	if stats.LLMEventCount != 2 {
		t.Errorf("Expected 2 events before cleanup, got %d", stats.LLMEventCount)
	}

	// Run cleanup
	err := collector.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify old event was removed
	stats = collector.GetStats()
	if stats.LLMEventCount != 1 {
		t.Errorf("Expected 1 event after cleanup, got %d", stats.LLMEventCount)
	}

	if stats.EventsRemoved != 1 {
		t.Errorf("Expected 1 event removed, got %d", stats.EventsRemoved)
	}
}

func TestMaxEvents_TriggersCleanup(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         5,          // Very low limit
		RetentionPeriod:   time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()

	// Add old events up to max
	for i := 0; i < 5; i++ {
		event := LLMUsageEvent{
			Timestamp:     time.Now().Add(-2 * time.Hour), // Old events
			UserID:        "user",
			ModelProvider: "openai",
			ModelName:     "gpt-4",
		}
		collector.RecordLLMUsage(ctx, event)
	}

	// Adding one more should trigger cleanup (removing old events)
	event := LLMUsageEvent{
		Timestamp:     time.Now(),
		UserID:        "user",
		ModelProvider: "openai",
		ModelName:     "gpt-4",
	}
	collector.RecordLLMUsage(ctx, event)

	stats := collector.GetStats()
	// After cleanup, only the new event should remain
	if stats.LLMEventCount != 1 {
		t.Errorf("Expected 1 event after forced cleanup, got %d", stats.LLMEventCount)
	}
}

func TestGetStats(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()

	// Add events
	now := time.Now()
	collector.RecordLLMUsage(ctx, LLMUsageEvent{
		Timestamp: now.Add(-time.Hour),
		UserID:    "user-1",
	})
	collector.RecordLLMUsage(ctx, LLMUsageEvent{
		Timestamp: now,
		UserID:    "user-2",
	})
	collector.RecordComputeUsage(ctx, ComputeUsageEvent{
		Timestamp: now,
		UserID:    "user-1",
	})

	stats := collector.GetStats()

	if stats.LLMEventCount != 2 {
		t.Errorf("Expected 2 LLM events, got %d", stats.LLMEventCount)
	}

	if stats.ComputeEventCount != 1 {
		t.Errorf("Expected 1 compute event, got %d", stats.ComputeEventCount)
	}

	if stats.OldestEvent.IsZero() {
		t.Error("OldestEvent should not be zero")
	}

	if stats.NewestEvent.IsZero() {
		t.Error("NewestEvent should not be zero")
	}
}

func TestGetUsageReport(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()
	now := time.Now()

	// Add events
	collector.RecordLLMUsage(ctx, LLMUsageEvent{
		Timestamp:     now,
		UserID:        "user-1",
		WorkspaceID:   "ws-1",
		ModelProvider: "openai",
		ModelName:     "gpt-4",
		TotalTokens:   1000,
		CostUSD:       0.10,
	})
	collector.RecordLLMUsage(ctx, LLMUsageEvent{
		Timestamp:     now,
		UserID:        "user-1",
		WorkspaceID:   "ws-1",
		ModelProvider: "openai",
		ModelName:     "gpt-4",
		TotalTokens:   500,
		CostUSD:       0.05,
	})

	filters := UsageFilters{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	report, err := collector.GetUsageReport(ctx, filters)
	if err != nil {
		t.Fatalf("GetUsageReport failed: %v", err)
	}

	if report.TotalRequests != 2 {
		t.Errorf("Expected 2 requests, got %d", report.TotalRequests)
	}

	if report.TotalTokens != 1500 {
		t.Errorf("Expected 1500 tokens, got %d", report.TotalTokens)
	}

	if report.TotalCostUSD != 0.15 {
		t.Errorf("Expected $0.15 cost, got $%.2f", report.TotalCostUSD)
	}

	// Check user aggregation
	userUsage, ok := report.ByUser["user-1"]
	if !ok {
		t.Error("Expected user-1 in report")
	} else if userUsage.TotalRequests != 2 {
		t.Errorf("Expected 2 requests for user-1, got %d", userUsage.TotalRequests)
	}
}

func TestGetUsageReport_WithFilters(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()
	now := time.Now()

	// Add events for different users
	collector.RecordLLMUsage(ctx, LLMUsageEvent{
		Timestamp: now,
		UserID:    "user-1",
	})
	collector.RecordLLMUsage(ctx, LLMUsageEvent{
		Timestamp: now,
		UserID:    "user-2",
	})

	// Filter by user-1 only
	filters := UsageFilters{
		UserID: "user-1",
	}

	report, err := collector.GetUsageReport(ctx, filters)
	if err != nil {
		t.Fatalf("GetUsageReport failed: %v", err)
	}

	if report.TotalRequests != 1 {
		t.Errorf("Expected 1 request (filtered), got %d", report.TotalRequests)
	}
}

func TestGetCostReport(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()
	now := time.Now()

	// Add LLM event
	collector.RecordLLMUsage(ctx, LLMUsageEvent{
		Timestamp: now,
		UserID:    "user-1",
		CostUSD:   10.00,
	})

	// Add compute event
	collector.RecordComputeUsage(ctx, ComputeUsageEvent{
		Timestamp: now,
		UserID:    "user-1",
		CostUSD:   5.00,
	})

	// Add storage event
	collector.RecordStorageUsage(ctx, StorageUsageEvent{
		Timestamp: now,
		UserID:    "user-1",
		CostUSD:   2.00,
	})

	filters := UsageFilters{}
	report, err := collector.GetCostReport(ctx, filters)
	if err != nil {
		t.Fatalf("GetCostReport failed: %v", err)
	}

	if report.CostByCategory["LLM"] != 10.00 {
		t.Errorf("Expected LLM cost $10, got $%.2f", report.CostByCategory["LLM"])
	}

	if report.CostByCategory["Compute"] != 5.00 {
		t.Errorf("Expected Compute cost $5, got $%.2f", report.CostByCategory["Compute"])
	}

	if report.CostByCategory["Storage"] != 2.00 {
		t.Errorf("Expected Storage cost $2, got $%.2f", report.CostByCategory["Storage"])
	}
}

func TestClose(t *testing.T) {
	config := CollectorConfig{
		MaxEvents:         100,
		RetentionPeriod:   24 * time.Hour,
		CleanupInterval:   time.Minute,
		EnableAutoCleanup: true,
	}
	collector := NewUsageCollectorWithConfig(config)

	// Close should not panic and should complete
	err := collector.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestDefaultCollectorConfig(t *testing.T) {
	config := DefaultCollectorConfig()

	if config.MaxEvents != DefaultMaxEvents {
		t.Errorf("Expected MaxEvents %d, got %d", DefaultMaxEvents, config.MaxEvents)
	}

	if config.RetentionPeriod != DefaultRetentionPeriod {
		t.Errorf("Expected RetentionPeriod %v, got %v", DefaultRetentionPeriod, config.RetentionPeriod)
	}

	if config.CleanupInterval != DefaultCleanupInterval {
		t.Errorf("Expected CleanupInterval %v, got %v", DefaultCleanupInterval, config.CleanupInterval)
	}

	if !config.EnableAutoCleanup {
		t.Error("Expected EnableAutoCleanup to be true by default")
	}
}

// Benchmark tests

func BenchmarkRecordLLMUsage(b *testing.B) {
	config := CollectorConfig{
		MaxEvents:         1000000,
		RetentionPeriod:   24 * time.Hour,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()
	event := LLMUsageEvent{
		Timestamp:     time.Now(),
		UserID:        "user-1",
		WorkspaceID:   "ws-1",
		ModelProvider: "openai",
		ModelName:     "gpt-4",
		TotalTokens:   1000,
		CostUSD:       0.10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.Timestamp = time.Now()
		_ = collector.RecordLLMUsage(ctx, event)
	}
}

func BenchmarkCleanup(b *testing.B) {
	config := CollectorConfig{
		MaxEvents:         100000,
		RetentionPeriod:   time.Minute,
		EnableAutoCleanup: false,
	}
	collector := NewUsageCollectorWithConfig(config)
	defer collector.(*usageCollector).Close()

	ctx := context.Background()

	// Add many old events
	for i := 0; i < 10000; i++ {
		collector.RecordLLMUsage(ctx, LLMUsageEvent{
			Timestamp: time.Now().Add(-2 * time.Hour),
			UserID:    "user",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.Cleanup(ctx)
	}
}
