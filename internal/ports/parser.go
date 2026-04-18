package ports

import (
	"context"
	"time"

	"llm-budget-tracker/internal/domain"
)

type ParseInput struct {
	SourceID    string
	Path        string
	Content     []byte
	StartOffset int64
	ObservedAt  time.Time
}

type SessionEvent struct {
	EntryID          string
	ExternalID       string
	SessionID        string
	OccurredAt       time.Time
	Source           domain.UsageSourceKind
	Provider         domain.ProviderName
	BillingModeHint  domain.BillingMode
	ProjectName      string
	AgentName        string
	PricingRef       *domain.ModelPricingRef
	Tokens           domain.TokenUsage
	CostBreakdown    domain.CostBreakdown
	PrivacySafeTags  map[string]string
	ObservedToolCall int64
}

type ParseResult struct {
	Events     []SessionEvent
	NextOffset int64
	Warnings   []string
}

type SessionParser interface {
	ParserName() string
	Parse(ctx context.Context, input ParseInput) (ParseResult, error)
}
