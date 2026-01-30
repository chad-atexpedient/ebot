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
}

func TestRecordLLMUsage(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	event := LLMUsageEvent{
		Timestamp:        time.Now(),
		UserID:           "user-1",
		WorkspaceID:      "ws-1",
		ThreadID:         "thread-1",
		ModelProvider:    "openai",
		ModelName:        "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		CostUSD:          0.0045, // Pre-calculated
		Latency:          time.Second,
		Success:          true,
	}

	err := collector.RecordLLMUsage(ctx, event)
	if err != nil {
		t.Fatalf("RecordLLMUsage failed: %v", err)
	}
}

func TestRecordLLMUsage_AutoCalculateCost(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	event := LLMUsageEvent{
		Timestamp:        time.Now(),
		UserID:           "user-1",
		WorkspaceID:      "ws-1",
		ModelProvider:    "openai",
		ModelName:        "gpt-4",
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
		CostUSD:          0, // Should be auto-calculated
		Success:          true,
	}

	err := collector.RecordLLMUsage(ctx, event)
	if err != nil {
		t.Fatalf("RecordLLMUsage failed: %v", err)
	}
}

func TestRecordComputeUsage(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	event := ComputeUsageEvent{
		Timestamp:       time.Now(),
		UserID:          "user-1",
		WorkspaceID:     "ws-1",
		ResourceType:    "mcp-server",
		ResourceID:      "mcp-1",
		CPUSeconds:      60.0,
		MemoryGBSeconds: 30.0,
		CostUSD:         0.05,
		Success:         true,
	}

	err := collector.RecordComputeUsage(ctx, event)
	if err != nil {
		t.Fatalf("RecordComputeUsage failed: %v", err)
	}
}

func TestRecordStorageUsage(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	event := StorageUsageEvent{
		Timestamp:   time.Now(),
		UserID:      "user-1",
		WorkspaceID: "ws-1",
		StorageType: "knowledge-files",
		Bytes:       1024 * 1024 * 100, // 100 MB
		CostUSD:     0.01,
	}

	err := collector.RecordStorageUsage(ctx, event)
	if err != nil {
		t.Fatalf("RecordStorageUsage failed: %v", err)
	}
}

func TestGetUsageReport(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// Record some events
	for i := 0; i < 10; i++ {
		event := LLMUsageEvent{
			Timestamp:        now,
			UserID:           "user-1",
			WorkspaceID:      "ws-1",
			ModelProvider:    "openai",
			ModelName:        "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			CostUSD:          0.005,
			Success:          true,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	filters := UsageFilters{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	report, err := collector.GetUsageReport(ctx, filters)
	if err != nil {
		t.Fatalf("GetUsageReport failed: %v", err)
	}

	if report.TotalRequests != 10 {
		t.Errorf("Expected 10 requests, got %d", report.TotalRequests)
	}

	if report.TotalTokens != 1500 {
		t.Errorf("Expected 1500 tokens, got %d", report.TotalTokens)
	}

	if report.TotalCostUSD < 0.05 {
		t.Errorf("Expected cost >= 0.05, got %f", report.TotalCostUSD)
	}
}

func TestGetUsageReport_FilterByUser(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// Record events for different users
	for i := 0; i < 5; i++ {
		event := LLMUsageEvent{
			Timestamp:   now,
			UserID:      "user-1",
			TotalTokens: 100,
			CostUSD:     0.01,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	for i := 0; i < 3; i++ {
		event := LLMUsageEvent{
			Timestamp:   now,
			UserID:      "user-2",
			TotalTokens: 100,
			CostUSD:     0.01,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	// Filter by user-1
	filters := UsageFilters{
		UserID: "user-1",
	}

	report, _ := collector.GetUsageReport(ctx, filters)

	if report.TotalRequests != 5 {
		t.Errorf("Expected 5 requests for user-1, got %d", report.TotalRequests)
	}
}

func TestGetUsageReport_FilterByWorkspace(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// Record events for different workspaces
	for i := 0; i < 4; i++ {
		event := LLMUsageEvent{
			Timestamp:   now,
			WorkspaceID: "ws-1",
			TotalTokens: 100,
			CostUSD:     0.01,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	for i := 0; i < 6; i++ {
		event := LLMUsageEvent{
			Timestamp:   now,
			WorkspaceID: "ws-2",
			TotalTokens: 100,
			CostUSD:     0.01,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	// Filter by ws-2
	filters := UsageFilters{
		WorkspaceID: "ws-2",
	}

	report, _ := collector.GetUsageReport(ctx, filters)

	if report.TotalRequests != 6 {
		t.Errorf("Expected 6 requests for ws-2, got %d", report.TotalRequests)
	}
}

func TestGetUsageReport_FilterByTimeRange(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// Record events at different times
	for i := 0; i < 3; i++ {
		event := LLMUsageEvent{
			Timestamp:   now.Add(-2 * time.Hour), // 2 hours ago
			TotalTokens: 100,
			CostUSD:     0.01,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	for i := 0; i < 5; i++ {
		event := LLMUsageEvent{
			Timestamp:   now, // Now
			TotalTokens: 100,
			CostUSD:     0.01,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	// Filter last hour only
	filters := UsageFilters{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}

	report, _ := collector.GetUsageReport(ctx, filters)

	if report.TotalRequests != 5 {
		t.Errorf("Expected 5 requests in last hour, got %d", report.TotalRequests)
	}
}

func TestGetUsageReport_ByModel(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// GPT-4 usage
	for i := 0; i < 4; i++ {
		event := LLMUsageEvent{
			Timestamp:     now,
			ModelProvider: "openai",
			ModelName:     "gpt-4",
			TotalTokens:   100,
			CostUSD:       0.03,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	// Claude usage
	for i := 0; i < 3; i++ {
		event := LLMUsageEvent{
			Timestamp:     now,
			ModelProvider: "anthropic",
			ModelName:     "claude-3-opus",
			TotalTokens:   100,
			CostUSD:       0.02,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	report, _ := collector.GetUsageReport(ctx, UsageFilters{})

	if len(report.ByModel) != 2 {
		t.Errorf("Expected 2 models, got %d", len(report.ByModel))
	}

	gpt4Usage := report.ByModel["openai/gpt-4"]
	if gpt4Usage.RequestCount != 4 {
		t.Errorf("Expected 4 GPT-4 requests, got %d", gpt4Usage.RequestCount)
	}

	claudeUsage := report.ByModel["anthropic/claude-3-opus"]
	if claudeUsage.RequestCount != 3 {
		t.Errorf("Expected 3 Claude requests, got %d", claudeUsage.RequestCount)
	}
}

func TestGetCostReport(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// Record LLM events
	for i := 0; i < 5; i++ {
		event := LLMUsageEvent{
			Timestamp: now,
			UserID:    "user-1",
			CostUSD:   0.10,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	// Record compute events
	for i := 0; i < 3; i++ {
		event := ComputeUsageEvent{
			Timestamp: now,
			CostUSD:   0.05,
		}
		_ = collector.RecordComputeUsage(ctx, event)
	}

	// Record storage events
	event := StorageUsageEvent{
		Timestamp: now,
		CostUSD:   0.02,
	}
	_ = collector.RecordStorageUsage(ctx, event)

	report, err := collector.GetCostReport(ctx, UsageFilters{})
	if err != nil {
		t.Fatalf("GetCostReport failed: %v", err)
	}

	// Total: 5*0.10 + 3*0.05 + 0.02 = 0.50 + 0.15 + 0.02 = 0.67
	expectedTotal := 0.67
	if report.TotalCostUSD < expectedTotal-0.01 || report.TotalCostUSD > expectedTotal+0.01 {
		t.Errorf("Expected total cost ~%f, got %f", expectedTotal, report.TotalCostUSD)
	}

	// Check category breakdown
	if report.CostByCategory["LLM"] != 0.50 {
		t.Errorf("Expected LLM cost 0.50, got %f", report.CostByCategory["LLM"])
	}
	if report.CostByCategory["Compute"] != 0.15 {
		t.Errorf("Expected Compute cost 0.15, got %f", report.CostByCategory["Compute"])
	}
}

func TestGetCostReport_TopCostDrivers(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// Create significant LLM costs
	for i := 0; i < 100; i++ {
		event := LLMUsageEvent{
			Timestamp: now,
			CostUSD:   1.0, // $100 total
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	// Create smaller compute costs
	for i := 0; i < 10; i++ {
		event := ComputeUsageEvent{
			Timestamp: now,
			CostUSD:   0.50, // $5 total
		}
		_ = collector.RecordComputeUsage(ctx, event)
	}

	report, _ := collector.GetCostReport(ctx, UsageFilters{})

	if len(report.TopCostDrivers) == 0 {
		t.Fatal("Expected top cost drivers")
	}

	// LLM should be top driver
	topDriver := report.TopCostDrivers[0]
	if topDriver.Category != "LLM" {
		t.Errorf("Expected LLM as top driver, got %s", topDriver.Category)
	}
}

func TestGetCostReport_Recommendations(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	now := time.Now()

	// Create high-cost model usage
	for i := 0; i < 5000; i++ {
		event := LLMUsageEvent{
			Timestamp:     now,
			ModelProvider: "openai",
			ModelName:     "gpt-4",
			CostUSD:       0.50, // High average cost
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	report, _ := collector.GetCostReport(ctx, UsageFilters{})

	if len(report.Recommendations) == 0 {
		t.Error("Expected cost recommendations for high-cost usage")
	}
}

func TestConcurrentRecording(t *testing.T) {
	collector := NewUsageCollector()
	ctx := context.Background()

	done := make(chan bool)

	// Concurrent LLM recordings
	go func() {
		for i := 0; i < 100; i++ {
			event := LLMUsageEvent{
				Timestamp: time.Now(),
				UserID:    "user-concurrent",
				CostUSD:   0.01,
			}
			_ = collector.RecordLLMUsage(ctx, event)
		}
		done <- true
	}()

	// Concurrent compute recordings
	go func() {
		for i := 0; i < 100; i++ {
			event := ComputeUsageEvent{
				Timestamp: time.Now(),
				CostUSD:   0.01,
			}
			_ = collector.RecordComputeUsage(ctx, event)
		}
		done <- true
	}()

	// Concurrent report generation
	go func() {
		for i := 0; i < 50; i++ {
			_, _ = collector.GetUsageReport(ctx, UsageFilters{})
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

func BenchmarkRecordLLMUsage(b *testing.B) {
	collector := NewUsageCollector()
	ctx := context.Background()

	event := LLMUsageEvent{
		Timestamp:        time.Now(),
		UserID:           "user-1",
		WorkspaceID:      "ws-1",
		ModelProvider:    "openai",
		ModelName:        "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		CostUSD:          0.005,
		Success:          true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.RecordLLMUsage(ctx, event)
	}
}

func BenchmarkGetUsageReport(b *testing.B) {
	collector := NewUsageCollector()
	ctx := context.Background()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		event := LLMUsageEvent{
			Timestamp:   time.Now(),
			UserID:      "user-1",
			TotalTokens: 100,
			CostUSD:     0.01,
		}
		_ = collector.RecordLLMUsage(ctx, event)
	}

	filters := UsageFilters{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.GetUsageReport(ctx, filters)
	}
}
