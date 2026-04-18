package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type ManualAPIUsageEntryCommand struct {
	EntryID          string
	Provider         string
	ModelID          string
	OccurredAt       time.Time
	InputTokens      int64
	OutputTokens     int64
	CachedTokens     int64
	CacheWriteTokens int64
	ProjectName      string
	Metadata         map[string]string
}

type ManualAPIUsageEntryService struct {
	catalog   ports.PriceCatalog
	usageRepo ports.UsageEntryRepository
}

func NewManualAPIUsageEntryService(catalog ports.PriceCatalog, usageRepo ports.UsageEntryRepository) *ManualAPIUsageEntryService {
	return &ManualAPIUsageEntryService{catalog: catalog, usageRepo: usageRepo}
}

func (s *ManualAPIUsageEntryService) Save(ctx context.Context, cmd ManualAPIUsageEntryCommand) (domain.UsageEntry, error) {
	if s == nil || s.catalog == nil {
		return domain.UsageEntry{}, errPriceCatalogRequired
	}

	if s.usageRepo == nil {
		return domain.UsageEntry{}, errUsageEntryRepositoryRequired
	}

	provider, err := domain.NewProviderName(cmd.Provider)
	if err != nil {
		return domain.UsageEntry{}, err
	}
	if provider != domain.ProviderOpenAI && provider != domain.ProviderAnthropic {
		return domain.UsageEntry{}, &domain.ValidationError{
			Code:    domain.ValidationCodeUnsupportedProvider,
			Field:   "provider",
			Message: "manual API entries support only openai and anthropic",
		}
	}

	modelID := strings.TrimSpace(cmd.ModelID)
	if modelID == "" {
		return domain.UsageEntry{}, &domain.ValidationError{
			Code:    domain.ValidationCodeRequired,
			Field:   "model_id",
			Message: "value is required",
		}
	}

	tokens, err := domain.NewTokenUsage(cmd.InputTokens, cmd.OutputTokens, cmd.CachedTokens, cmd.CacheWriteTokens)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	pricingRef, err := domain.NewModelPricingRef(provider, modelID, fmt.Sprintf("%s/%s", provider, modelID))
	if err != nil {
		return domain.UsageEntry{}, err
	}

	price, err := s.catalog.LookupModelPrice(ctx, pricingRef, cmd.OccurredAt)
	if err != nil {
		return domain.UsageEntry{}, &domain.ValidationError{
			Code:    domain.ValidationCodeUnknownModel,
			Field:   "model_id",
			Message: fmt.Sprintf("model %q is not available for provider %q in the price catalog", modelID, provider),
		}
	}

	costBreakdown, err := price.Calculate(tokens, 0)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	entryID := strings.TrimSpace(cmd.EntryID)
	if entryID == "" {
		entryID = manualAPIUsageEntryID(provider, modelID, cmd.OccurredAt, tokens, strings.TrimSpace(cmd.ProjectName))
	}

	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		Source:        domain.UsageSourceManualAPI,
		Provider:      provider,
		BillingMode:   domain.BillingModeDirectAPI,
		OccurredAt:    cmd.OccurredAt,
		ProjectName:   strings.TrimSpace(cmd.ProjectName),
		Metadata:      cmd.Metadata,
		PricingRef:    &pricingRef,
		Tokens:        tokens,
		CostBreakdown: costBreakdown,
	})
	if err != nil {
		return domain.UsageEntry{}, err
	}

	if err := s.usageRepo.UpsertUsageEntries(ctx, []domain.UsageEntry{entry}); err != nil {
		return domain.UsageEntry{}, err
	}

	return entry, nil
}

func manualAPIUsageEntryID(provider domain.ProviderName, modelID string, occurredAt time.Time, tokens domain.TokenUsage, projectName string) string {
	return fmt.Sprintf(
		"manual:%s:%s:%s:%d:%d:%d:%d:%s",
		provider,
		strings.TrimSpace(modelID),
		occurredAt.UTC().Format(time.RFC3339Nano),
		tokens.InputTokens,
		tokens.OutputTokens,
		tokens.CacheReadTokens,
		tokens.CacheWriteTokens,
		strings.TrimSpace(projectName),
	)
}
