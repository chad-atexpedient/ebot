package metering

import (
	"fmt"
)

// CostCalculator calculates costs for various resources
type CostCalculator struct {
	modelPricing   map[string]ModelPricing
	computeRates   ComputeRates
	storageRates   StorageRates
}

// ModelPricing defines pricing for an LLM model
type ModelPricing struct {
	Provider               string
	Model                  string
	PromptPricePer1KTokens float64
	CompletionPricePer1KTokens float64
	Currency               string
}

// ComputeRates defines pricing for compute resources
type ComputeRates struct {
	CPUPerHour      float64 // Cost per CPU hour
	MemoryGBPerHour float64 // Cost per GB-hour of memory
	Currency        string
}

// StorageRates defines pricing for storage
type StorageRates struct {
	StorageGBPerMonth float64 // Cost per GB-month
	Currency          string
}

// NewCostCalculator creates a new cost calculator with default pricing
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{
		modelPricing: defaultModelPricing(),
		computeRates: ComputeRates{
			CPUPerHour:      0.05,  // $0.05 per CPU hour
			MemoryGBPerHour: 0.01,  // $0.01 per GB-hour
			Currency:        "USD",
		},
		storageRates: StorageRates{
			StorageGBPerMonth: 0.10, // $0.10 per GB-month
			Currency:          "USD",
		},
	}
}

// defaultModelPricing returns pricing for major LLM providers
func defaultModelPricing() map[string]ModelPricing {
	return map[string]ModelPricing{
		// OpenAI Pricing (as of 2024)
		"openai/gpt-4": {
			Provider:                   "openai",
			Model:                      "gpt-4",
			PromptPricePer1KTokens:     0.03,
			CompletionPricePer1KTokens: 0.06,
			Currency:                   "USD",
		},
		"openai/gpt-4-turbo": {
			Provider:                   "openai",
			Model:                      "gpt-4-turbo",
			PromptPricePer1KTokens:     0.01,
			CompletionPricePer1KTokens: 0.03,
			Currency:                   "USD",
		},
		"openai/gpt-3.5-turbo": {
			Provider:                   "openai",
			Model:                      "gpt-3.5-turbo",
			PromptPricePer1KTokens:     0.0015,
			CompletionPricePer1KTokens: 0.002,
			Currency:                   "USD",
		},
		"openai/gpt-4o": {
			Provider:                   "openai",
			Model:                      "gpt-4o",
			PromptPricePer1KTokens:     0.005,
			CompletionPricePer1KTokens: 0.015,
			Currency:                   "USD",
		},
		
		// Anthropic Pricing
		"anthropic/claude-3-opus": {
			Provider:                   "anthropic",
			Model:                      "claude-3-opus",
			PromptPricePer1KTokens:     0.015,
			CompletionPricePer1KTokens: 0.075,
			Currency:                   "USD",
		},
		"anthropic/claude-3-sonnet": {
			Provider:                   "anthropic",
			Model:                      "claude-3-sonnet",
			PromptPricePer1KTokens:     0.003,
			CompletionPricePer1KTokens: 0.015,
			Currency:                   "USD",
		},
		"anthropic/claude-3-haiku": {
			Provider:                   "anthropic",
			Model:                      "claude-3-haiku",
			PromptPricePer1KTokens:     0.00025,
			CompletionPricePer1KTokens: 0.00125,
			Currency:                   "USD",
		},
		"anthropic/claude-3.5-sonnet": {
			Provider:                   "anthropic",
			Model:                      "claude-3.5-sonnet",
			PromptPricePer1KTokens:     0.003,
			CompletionPricePer1KTokens: 0.015,
			Currency:                   "USD",
		},
		
		// Google Pricing
		"google/gemini-pro": {
			Provider:                   "google",
			Model:                      "gemini-pro",
			PromptPricePer1KTokens:     0.00025,
			CompletionPricePer1KTokens: 0.0005,
			Currency:                   "USD",
		},
		"google/gemini-pro-vision": {
			Provider:                   "google",
			Model:                      "gemini-pro-vision",
			PromptPricePer1KTokens:     0.00025,
			CompletionPricePer1KTokens: 0.0005,
			Currency:                   "USD",
		},
		
		// Azure OpenAI (similar pricing to OpenAI)
		"azure/gpt-4": {
			Provider:                   "azure",
			Model:                      "gpt-4",
			PromptPricePer1KTokens:     0.03,
			CompletionPricePer1KTokens: 0.06,
			Currency:                   "USD",
		},
		"azure/gpt-35-turbo": {
			Provider:                   "azure",
			Model:                      "gpt-35-turbo",
			PromptPricePer1KTokens:     0.0015,
			CompletionPricePer1KTokens: 0.002,
			Currency:                   "USD",
		},
	}
}

// CalculateLLMCost calculates the cost of an LLM API call
func (c *CostCalculator) CalculateLLMCost(provider, model string, promptTokens, completionTokens int64) (float64, error) {
	key := fmt.Sprintf("%s/%s", provider, model)
	pricing, exists := c.modelPricing[key]
	
	if !exists {
		// Return default pricing if model not found
		pricing = ModelPricing{
			Provider:                   provider,
			Model:                      model,
			PromptPricePer1KTokens:     0.01,  // Default fallback
			CompletionPricePer1KTokens: 0.03,  // Default fallback
			Currency:                   "USD",
		}
	}
	
	promptCost := (float64(promptTokens) / 1000.0) * pricing.PromptPricePer1KTokens
	completionCost := (float64(completionTokens) / 1000.0) * pricing.CompletionPricePer1KTokens
	
	return promptCost + completionCost, nil
}

// CalculateComputeCost calculates the cost of compute resources
func (c *CostCalculator) CalculateComputeCost(cpuSeconds, memoryGBSeconds float64) float64 {
	cpuHours := cpuSeconds / 3600.0
	memoryGBHours := memoryGBSeconds / 3600.0
	
	cpuCost := cpuHours * c.computeRates.CPUPerHour
	memoryCost := memoryGBHours * c.computeRates.MemoryGBPerHour
	
	return cpuCost + memoryCost
}

// CalculateStorageCost calculates the cost of storage
func (c *CostCalculator) CalculateStorageCost(bytes int64) float64 {
	gb := float64(bytes) / (1024.0 * 1024.0 * 1024.0)
	
	// Assuming monthly cost, prorate to daily
	dailyRate := c.storageRates.StorageGBPerMonth / 30.0
	
	return gb * dailyRate
}

// UpdateModelPricing updates pricing for a specific model
func (c *CostCalculator) UpdateModelPricing(pricing ModelPricing) {
	key := fmt.Sprintf("%s/%s", pricing.Provider, pricing.Model)
	c.modelPricing[key] = pricing
}

// UpdateComputeRates updates compute pricing
func (c *CostCalculator) UpdateComputeRates(rates ComputeRates) {
	c.computeRates = rates
}

// UpdateStorageRates updates storage pricing
func (c *CostCalculator) UpdateStorageRates(rates StorageRates) {
	c.storageRates = rates
}

// GetModelPricing returns pricing for a specific model
func (c *CostCalculator) GetModelPricing(provider, model string) (ModelPricing, bool) {
	key := fmt.Sprintf("%s/%s", provider, model)
	pricing, exists := c.modelPricing[key]
	return pricing, exists
}

// ListModelPricing returns all model pricing
func (c *CostCalculator) ListModelPricing() []ModelPricing {
	pricing := make([]ModelPricing, 0, len(c.modelPricing))
	for _, p := range c.modelPricing {
		pricing = append(pricing, p)
	}
	return pricing
}

// EstimateMonthlyLLMCost estimates monthly cost based on usage patterns
func (c *CostCalculator) EstimateMonthlyLLMCost(provider, model string, avgRequestsPerDay int64, avgTokensPerRequest int64) (float64, error) {
	// Assume 50% prompt, 50% completion tokens
	promptTokens := avgTokensPerRequest / 2
	completionTokens := avgTokensPerRequest / 2
	
	costPerRequest, err := c.CalculateLLMCost(provider, model, promptTokens, completionTokens)
	if err != nil {
		return 0, err
	}
	
	dailyCost := float64(avgRequestsPerDay) * costPerRequest
	monthlyCost := dailyCost * 30
	
	return monthlyCost, nil
}

// EstimateMonthlyComputeCost estimates monthly compute cost
func (c *CostCalculator) EstimateMonthlyComputeCost(cpuCores float64, memoryGB float64, hoursPerDay float64) float64 {
	dailyCPUCost := cpuCores * hoursPerDay * c.computeRates.CPUPerHour
	dailyMemoryCost := memoryGB * hoursPerDay * c.computeRates.MemoryGBPerHour
	
	monthlyCost := (dailyCPUCost + dailyMemoryCost) * 30
	
	return monthlyCost
}

// EstimateMonthlyStorageCost estimates monthly storage cost
func (c *CostCalculator) EstimateMonthlyStorageCost(totalGB float64) float64 {
	return totalGB * c.storageRates.StorageGBPerMonth
}

// CostBreakdown provides detailed cost breakdown
type CostBreakdown struct {
	LLMCost     float64
	ComputeCost float64
	StorageCost float64
	TotalCost   float64
	Currency    string
}

// CalculateTotalCost calculates total cost across all resource types
func (c *CostCalculator) CalculateTotalCost(
	llmEvents []LLMUsageEvent,
	computeEvents []ComputeUsageEvent,
	storageEvents []StorageUsageEvent,
) (*CostBreakdown, error) {
	breakdown := &CostBreakdown{
		Currency: "USD",
	}
	
	// Calculate LLM costs
	for _, event := range llmEvents {
		breakdown.LLMCost += event.CostUSD
	}
	
	// Calculate compute costs
	for _, event := range computeEvents {
		breakdown.ComputeCost += event.CostUSD
	}
	
	// Calculate storage costs
	for _, event := range storageEvents {
		breakdown.StorageCost += event.CostUSD
	}
	
	breakdown.TotalCost = breakdown.LLMCost + breakdown.ComputeCost + breakdown.StorageCost
	
	return breakdown, nil
}

// PricingTier defines tiered pricing for enterprise customers
type PricingTier struct {
	Name        string
	MinSpend    float64
	MaxSpend    float64
	DiscountPct float64 // Discount percentage (e.g., 10 for 10%)
}

// EnterprisePricing applies tiered pricing
type EnterprisePricing struct {
	Tiers []PricingTier
}

// DefaultEnterprisePricing returns default enterprise pricing tiers
func DefaultEnterprisePricing() *EnterprisePricing {
	return &EnterprisePricing{
		Tiers: []PricingTier{
			{Name: "Standard", MinSpend: 0, MaxSpend: 1000, DiscountPct: 0},
			{Name: "Professional", MinSpend: 1000, MaxSpend: 5000, DiscountPct: 10},
			{Name: "Enterprise", MinSpend: 5000, MaxSpend: 20000, DiscountPct: 15},
			{Name: "Enterprise Plus", MinSpend: 20000, MaxSpend: 999999999, DiscountPct: 20},
		},
	}
}

// ApplyDiscount applies enterprise discount to a cost
func (e *EnterprisePricing) ApplyDiscount(cost float64) float64 {
	for _, tier := range e.Tiers {
		if cost >= tier.MinSpend && cost < tier.MaxSpend {
			discount := cost * (tier.DiscountPct / 100.0)
			return cost - discount
		}
	}
	return cost
}

// GetTierForSpend returns the pricing tier for a given spend amount
func (e *EnterprisePricing) GetTierForSpend(spend float64) string {
	for _, tier := range e.Tiers {
		if spend >= tier.MinSpend && spend < tier.MaxSpend {
			return tier.Name
		}
	}
	return "Standard"
}
