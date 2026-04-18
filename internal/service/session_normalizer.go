package service

import (
	"context"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type SessionNormalizationResult struct {
	UsageEntries []domain.UsageEntry
	Sessions     []domain.SessionSummary
	Warnings     []AttributionWarning
}

type SessionNormalizerService struct {
	usageRepo        ports.UsageEntryRepository
	sessionRepo      ports.SessionRepository
	subscriptionRepo ports.SubscriptionRepository
}

func NewSessionNormalizerService(usageRepo ports.UsageEntryRepository, sessionRepo ports.SessionRepository, subscriptionRepo ports.SubscriptionRepository) *SessionNormalizerService {
	return &SessionNormalizerService{
		usageRepo:        usageRepo,
		sessionRepo:      sessionRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

func (s *SessionNormalizerService) IngestUsageEntries(ctx context.Context, entries []domain.UsageEntry) error {
	if s == nil || s.usageRepo == nil {
		return errUsageEntryRepositoryRequired
	}

	return s.usageRepo.UpsertUsageEntries(ctx, entries)
}

func (s *SessionNormalizerService) IngestSubscriptionFees(ctx context.Context, fees []domain.SubscriptionFee) error {
	if s == nil || s.subscriptionRepo == nil {
		return errSubscriptionRepoRequired
	}

	return s.subscriptionRepo.UpsertSubscriptionFees(ctx, fees)
}

func (s *SessionNormalizerService) IngestSessionEvents(ctx context.Context, events []ports.SessionEvent) error {
	_, err := s.Normalize(ctx, events)
	return err
}

func (s *SessionNormalizerService) Normalize(ctx context.Context, events []ports.SessionEvent) (SessionNormalizationResult, error) {
	if s == nil || s.usageRepo == nil {
		return SessionNormalizationResult{}, errUsageEntryRepositoryRequired
	}

	if s.sessionRepo == nil {
		return SessionNormalizationResult{}, errSessionRepositoryRequired
	}

	if len(events) == 0 {
		return SessionNormalizationResult{}, nil
	}

	bySession := make(map[string]*sessionAggregate, len(events))

	for _, event := range events {
		validated, err := normalizeIncomingEvent(event)
		if err != nil {
			return SessionNormalizationResult{}, err
		}

		agg := bySession[event.SessionID]
		if agg == nil {
			agg = &sessionAggregate{
				sessionID:             event.SessionID,
				source:                validated.Source,
				provider:              validated.Provider,
				startedAt:             validated.OccurredAt,
				endedAt:               validated.OccurredAt,
				projectName:           validated.ProjectName,
				agentName:             validated.AgentName,
				pricingRef:            validated.PricingRef,
				tokens:                validated.Tokens,
				costs:                 validated.CostBreakdown,
				events:                []ports.SessionEvent{validated},
				billingModeCandidates: map[domain.BillingMode]int{},
				projectCandidates:     map[string]int{},
				agentCandidates:       map[string]int{},
				providerCandidates:    map[string]int{},
				modelCandidates:       map[string]int{},
			}
			agg.recordAttributionHints(validated)
			bySession[event.SessionID] = agg
			continue
		}

		if validated.OccurredAt.Before(agg.startedAt) {
			agg.startedAt = validated.OccurredAt
		}
		if validated.OccurredAt.After(agg.endedAt) {
			agg.endedAt = validated.OccurredAt
		}
		if name := strings.TrimSpace(validated.ProjectName); name != "" {
			agg.projectName = name
		}
		if name := strings.TrimSpace(validated.AgentName); name != "" {
			agg.agentName = name
		}
		if validated.PricingRef != nil {
			agg.pricingRef = validated.PricingRef
		}
		agg.provider = validated.Provider
		agg.events = append(agg.events, validated)
		agg.recordAttributionHints(validated)

		agg.tokens, err = mergeTokens(agg.tokens, validated.Tokens)
		if err != nil {
			return SessionNormalizationResult{}, err
		}

		agg.costs, err = mergeCosts(agg.costs, validated.CostBreakdown)
		if err != nil {
			return SessionNormalizationResult{}, err
		}
	}

	usageEntries := make([]domain.UsageEntry, 0, len(events))
	sessions := make([]domain.SessionSummary, 0, len(bySession))
	warnings := make([]AttributionWarning, 0, len(bySession))
	for _, agg := range bySession {
		billingMode, modeWarnings := resolveSessionBillingMode(agg.sessionID, agg.billingModeCandidates)
		warnings = append(warnings, modeWarnings...)

		projectName, projectWarnings := resolveStringAttribution(agg.sessionID, "project_name", agg.projectName, agg.projectCandidates)
		warnings = append(warnings, projectWarnings...)

		agentName, agentWarnings := resolveStringAttribution(agg.sessionID, "agent_name", agg.agentName, agg.agentCandidates)
		warnings = append(warnings, agentWarnings...)

		providerName, providerWarnings := resolveStringAttribution(agg.sessionID, "provider", agg.provider.String(), agg.providerCandidates)
		warnings = append(warnings, providerWarnings...)
		provider, err := domain.NewProviderName(providerName)
		if err != nil {
			return SessionNormalizationResult{}, err
		}
		agg.provider = provider

		if agg.pricingRef != nil {
			_, modelWarnings := resolveStringAttribution(agg.sessionID, "model", agg.pricingRef.ModelID, agg.modelCandidates)
			warnings = append(warnings, modelWarnings...)
		}

		for _, event := range agg.events {
			entry, err := normalizeEvent(event, billingMode)
			if err != nil {
				return SessionNormalizationResult{}, err
			}
			if strings.TrimSpace(entry.ProjectName) == "" {
				entry.ProjectName = projectName
			}
			if strings.TrimSpace(entry.AgentName) == "" {
				entry.AgentName = agentName
			}
			usageEntries = append(usageEntries, entry)
		}

		summary, err := domain.NewSessionSummary(domain.SessionSummary{
			SessionID:     agg.sessionID,
			Source:        agg.source,
			Provider:      provider,
			BillingMode:   billingMode,
			StartedAt:     agg.startedAt,
			EndedAt:       agg.endedAt,
			ProjectName:   projectName,
			AgentName:     agentName,
			PricingRef:    agg.pricingRef,
			Tokens:        agg.tokens,
			CostBreakdown: agg.costs,
		})
		if err != nil {
			return SessionNormalizationResult{}, err
		}

		sessions = append(sessions, summary)
	}

	if err := s.sessionRepo.UpsertSessions(ctx, sessions); err != nil {
		return SessionNormalizationResult{}, err
	}

	if err := s.usageRepo.UpsertUsageEntries(ctx, usageEntries); err != nil {
		return SessionNormalizationResult{}, err
	}

	return SessionNormalizationResult{UsageEntries: usageEntries, Sessions: sessions, Warnings: warnings}, nil
}

type sessionAggregate struct {
	sessionID             string
	source                domain.UsageSourceKind
	provider              domain.ProviderName
	startedAt             time.Time
	endedAt               time.Time
	projectName           string
	agentName             string
	pricingRef            *domain.ModelPricingRef
	tokens                domain.TokenUsage
	costs                 domain.CostBreakdown
	events                []ports.SessionEvent
	billingModeCandidates map[domain.BillingMode]int
	projectCandidates     map[string]int
	agentCandidates       map[string]int
	providerCandidates    map[string]int
	modelCandidates       map[string]int
}

func (a *sessionAggregate) recordAttributionHints(event ports.SessionEvent) {
	mode := canonicalSessionBillingMode(event.BillingModeHint)
	if mode != domain.BillingModeUnknown {
		a.billingModeCandidates[mode]++
	}
	if value := strings.TrimSpace(event.ProjectName); value != "" {
		a.projectCandidates[value]++
	}
	if value := strings.TrimSpace(event.AgentName); value != "" {
		a.agentCandidates[value]++
	}
	if value := strings.TrimSpace(event.Provider.String()); value != "" {
		a.providerCandidates[value]++
	}
	if event.PricingRef != nil {
		if value := strings.TrimSpace(event.PricingRef.ModelID); value != "" {
			a.modelCandidates[value]++
		}
	}
}

func normalizeIncomingEvent(event ports.SessionEvent) (ports.SessionEvent, error) {
	if strings.TrimSpace(event.SessionID) == "" {
		return ports.SessionEvent{}, errSessionIDRequired
	}

	if strings.TrimSpace(event.EntryID) == "" {
		return ports.SessionEvent{}, errEntryIDRequired
	}

	if _, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       strings.TrimSpace(event.EntryID),
		Source:        sourceOrDefault(event.Source),
		Provider:      event.Provider,
		BillingMode:   domain.BillingModeUnknown,
		OccurredAt:    event.OccurredAt,
		SessionID:     strings.TrimSpace(event.SessionID),
		ExternalID:    strings.TrimSpace(event.ExternalID),
		ProjectName:   strings.TrimSpace(event.ProjectName),
		AgentName:     strings.TrimSpace(event.AgentName),
		Metadata:      usageEntryMetadata(event),
		PricingRef:    event.PricingRef,
		Tokens:        event.Tokens,
		CostBreakdown: event.CostBreakdown,
	}); err != nil {
		return ports.SessionEvent{}, err
	}

	event.EntryID = strings.TrimSpace(event.EntryID)
	event.SessionID = strings.TrimSpace(event.SessionID)
	event.ExternalID = strings.TrimSpace(event.ExternalID)
	event.ProjectName = strings.TrimSpace(event.ProjectName)
	event.AgentName = strings.TrimSpace(event.AgentName)
	event.Source = sourceOrDefault(event.Source)
	return event, nil
}

func normalizeEvent(event ports.SessionEvent, sessionMode domain.BillingMode) (domain.UsageEntry, error) {
	mode := canonicalSessionBillingMode(sessionMode)
	if mode == "" {
		mode = domain.BillingModeUnknown
	}

	metadata := usageEntryMetadata(event)

	return domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       strings.TrimSpace(event.EntryID),
		Source:        sourceOrDefault(event.Source),
		Provider:      event.Provider,
		BillingMode:   mode,
		OccurredAt:    event.OccurredAt,
		SessionID:     strings.TrimSpace(event.SessionID),
		ExternalID:    strings.TrimSpace(event.ExternalID),
		ProjectName:   strings.TrimSpace(event.ProjectName),
		AgentName:     strings.TrimSpace(event.AgentName),
		Metadata:      metadata,
		PricingRef:    event.PricingRef,
		Tokens:        event.Tokens,
		CostBreakdown: event.CostBreakdown,
	})
}

func sourceOrDefault(source domain.UsageSourceKind) domain.UsageSourceKind {
	if source == "" {
		return domain.UsageSourceCLISession
	}
	return source
}

func mergeTokens(left, right domain.TokenUsage) (domain.TokenUsage, error) {
	return domain.NewTokenUsage(
		left.InputTokens+right.InputTokens,
		left.OutputTokens+right.OutputTokens,
		left.CacheReadTokens+right.CacheReadTokens,
		left.CacheWriteTokens+right.CacheWriteTokens,
	)
}

func mergeCosts(left, right domain.CostBreakdown) (domain.CostBreakdown, error) {
	return domain.NewCostBreakdown(
		left.InputUSD+right.InputUSD,
		left.OutputUSD+right.OutputUSD,
		left.CacheReadUSD+right.CacheReadUSD,
		left.CacheWriteUSD+right.CacheWriteUSD,
		left.ToolUSD+right.ToolUSD,
		left.FlatUSD+right.FlatUSD,
	)
}

var _ ports.IngestionService = (*SessionNormalizerService)(nil)
