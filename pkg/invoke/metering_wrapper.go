package invoke

import (
	"context"
	"time"

	"github.com/chad-atexpedient/ebot/pkg/metering"
)

// MeteringWrapper wraps LLM invocations to track usage and costs
type MeteringWrapper struct {
	collector metering.Collector
}

// NewMeteringWrapper creates a new metering wrapper
func NewMeteringWrapper(collector metering.Collector) *MeteringWrapper {
	return &MeteringWrapper{
		collector: collector,
	}
}

// WrapInvocation wraps an LLM call and records metrics
func (m *MeteringWrapper) WrapInvocation(
	ctx context.Context,
	userID, workspaceID, threadID string,
	provider, model string,
	invoke func() (tokens TokenUsage, err error),
) error {
	startTime := time.Now()

	// Execute the actual LLM call
	tokens, err := invoke()
	duration := time.Since(startTime)

	// Record the usage event
	event := metering.LLMUsageEvent{
		Timestamp:        startTime,
		UserID:           userID,
		WorkspaceID:      workspaceID,
		ThreadID:         threadID,
		ModelProvider:    provider,
		ModelName:        model,
		PromptTokens:     tokens.PromptTokens,
		CompletionTokens: tokens.CompletionTokens,
		TotalTokens:      tokens.TotalTokens,
		Duration:         duration,
		Success:          err == nil,
	}

	// Calculate cost (this will use pricing tables in the calculator)
	if calcErr := m.collector.RecordLLMUsage(ctx, event); calcErr != nil {
		// Log but don't fail the request if metering fails
		// TODO: Add proper logging
		_ = calcErr
	}

	return err
}

// TokenUsage represents token counts from an LLM call
type TokenUsage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}
