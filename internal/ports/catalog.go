package ports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
)

const oneMillionTokens = 1_000_000.0

type ModelPrice struct {
	Provider             domain.ProviderName
	ModelID              string
	LookupKey            string
	InputUSDPer1M        float64
	OutputUSDPer1M       float64
	CacheReadUSDPer1M    float64
	CacheWriteUSDPer1M   float64
	ToolUSDPerInvocation float64
}

func (p ModelPrice) Calculate(tokens domain.TokenUsage, toolInvocations int64) (domain.CostBreakdown, error) {
	if strings.TrimSpace(p.ModelID) == "" {
		return domain.CostBreakdown{}, fmt.Errorf("model price requires model id")
	}

	for field, value := range map[string]float64{
		"input_usd_per_1m":        p.InputUSDPer1M,
		"output_usd_per_1m":       p.OutputUSDPer1M,
		"cache_read_usd_per_1m":   p.CacheReadUSDPer1M,
		"cache_write_usd_per_1m":  p.CacheWriteUSDPer1M,
		"tool_usd_per_invocation": p.ToolUSDPerInvocation,
	} {
		if value < 0 {
			return domain.CostBreakdown{}, fmt.Errorf("model price %s must be non-negative", field)
		}
	}

	if toolInvocations < 0 {
		return domain.CostBreakdown{}, fmt.Errorf("tool invocation count must be non-negative")
	}

	return domain.NewCostBreakdown(
		float64(tokens.InputTokens)/oneMillionTokens*p.InputUSDPer1M,
		float64(tokens.OutputTokens)/oneMillionTokens*p.OutputUSDPer1M,
		float64(tokens.CacheReadTokens)/oneMillionTokens*p.CacheReadUSDPer1M,
		float64(tokens.CacheWriteTokens)/oneMillionTokens*p.CacheWriteUSDPer1M,
		float64(toolInvocations)*p.ToolUSDPerInvocation,
		0,
	)
}

type CatalogSnapshot struct {
	Source   string
	Version  string
	SyncedAt time.Time
	Entries  []ModelPrice
}

type PriceCatalog interface {
	LookupModelPrice(ctx context.Context, ref domain.ModelPricingRef, at time.Time) (ModelPrice, error)
	ListProviderPrices(ctx context.Context, provider domain.ProviderName) ([]ModelPrice, error)
	ReplaceCatalog(ctx context.Context, snapshot CatalogSnapshot) error
}

type CatalogSyncSource interface {
	FetchCatalog(ctx context.Context) (CatalogSnapshot, error)
}
