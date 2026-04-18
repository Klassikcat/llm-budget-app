package service

import (
	"cmp"
	"context"
	"slices"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const defaultDashboardRecentSessionLimit = 5

type DashboardQuery struct {
	Period             domain.MonthlyPeriod
	RecentSessionLimit int
}

type DashboardSnapshot struct {
	Period            domain.MonthlyPeriod
	Totals            DashboardTotals
	ProviderSummaries []DashboardProviderSummary
	Budgets           []DashboardBudgetSummary
	RecentSessions    []DashboardRecentSession
	Empty             bool
}

type DashboardTotals struct {
	VariableSpendUSD     float64
	SubscriptionSpendUSD float64
	TotalSpendUSD        float64
}

type DashboardProviderSummary struct {
	Provider             domain.ProviderName
	VariableSpendUSD     float64
	SubscriptionSpendUSD float64
	TotalSpendUSD        float64
	UsageEntryCount      int
	SessionCount         int
}

type DashboardBudgetSummary struct {
	BudgetID                   string
	Name                       string
	Provider                   domain.ProviderName
	ProjectHash                string
	LimitUSD                   float64
	CurrentSpendUSD            float64
	RemainingUSD               float64
	TriggeredThresholdPercents []float64
	BudgetOverrunActive        bool
	Currency                   string
}

type DashboardRecentSession struct {
	SessionID      string
	Provider       domain.ProviderName
	BillingMode    domain.BillingMode
	ProjectName    string
	AgentName      string
	ModelID        string
	StartedAt      time.Time
	EndedAt        time.Time
	TotalCostUSD   float64
	TotalTokens    int64
	DurationSecond int64
}

type DashboardQueryService struct {
	usageRepo           ports.UsageEntryRepository
	sessionRepo         ports.SessionRepository
	budgetRepo          ports.BudgetRepository
	subscriptionService *SubscriptionService
	clock               func() time.Time
}

func NewDashboardQueryService(usageRepo ports.UsageEntryRepository, sessionRepo ports.SessionRepository, budgetRepo ports.BudgetRepository, subscriptionRepo ports.SubscriptionRepository) *DashboardQueryService {
	return &DashboardQueryService{
		usageRepo:           usageRepo,
		sessionRepo:         sessionRepo,
		budgetRepo:          budgetRepo,
		subscriptionService: NewSubscriptionService(subscriptionRepo, usageRepo),
		clock:               func() time.Time { return time.Now().UTC() },
	}
}

func (s *DashboardQueryService) ClockForTest(clock func() time.Time) {
	if s == nil || clock == nil {
		return
	}
	s.clock = clock
}

func (s *DashboardQueryService) QueryDashboard(ctx context.Context, query DashboardQuery) (DashboardSnapshot, error) {
	if s == nil || s.usageRepo == nil {
		return DashboardSnapshot{}, errUsageEntryRepositoryRequired
	}
	if s.sessionRepo == nil {
		return DashboardSnapshot{}, errSessionRepositoryRequired
	}
	if s.budgetRepo == nil {
		return DashboardSnapshot{}, errBudgetRepositoryRequired
	}
	if s.subscriptionService == nil {
		return DashboardSnapshot{}, errSubscriptionRepoRequired
	}

	period := query.Period
	if period.StartAt.IsZero() || period.EndExclusive.IsZero() {
		anchor := time.Now().UTC()
		if s.clock != nil {
			anchor = s.clock().UTC()
		}
		var err error
		period, err = domain.NewMonthlyPeriod(anchor)
		if err != nil {
			return DashboardSnapshot{}, err
		}
	}

	rollup, err := s.subscriptionService.RollupMonthlySpend(ctx, period)
	if err != nil {
		return DashboardSnapshot{}, err
	}

	sessions, err := s.sessionRepo.ListSessions(ctx, ports.SessionFilter{Period: &period})
	if err != nil {
		return DashboardSnapshot{}, err
	}

	budgets, err := s.budgetRepo.ListMonthlyBudgets(ctx, ports.BudgetFilter{Period: &period})
	if err != nil {
		return DashboardSnapshot{}, err
	}

	providerSummaries := buildDashboardProviderSummaries(rollup.UsageEntries, rollup.SubscriptionFees, sessions)
	budgetSummaries, err := buildDashboardBudgetSummaries(budgets, rollup.UsageEntries, rollup.SubscriptionFees)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	recentSessions := buildDashboardRecentSessions(sessions, query.RecentSessionLimit)

	return DashboardSnapshot{
		Period: period,
		Totals: DashboardTotals{
			VariableSpendUSD:     rollup.VariableSpendUSD,
			SubscriptionSpendUSD: rollup.SubscriptionSpendUSD,
			TotalSpendUSD:        rollup.TotalSpendUSD,
		},
		ProviderSummaries: providerSummaries,
		Budgets:           budgetSummaries,
		RecentSessions:    recentSessions,
		Empty:             len(providerSummaries) == 0 && len(budgetSummaries) == 0 && len(recentSessions) == 0 && rollup.TotalSpendUSD == 0,
	}, nil
}

type providerSummaryAccumulator struct {
	provider             domain.ProviderName
	variableSpendUSD     float64
	subscriptionSpendUSD float64
	usageEntryCount      int
	sessionIDs           map[string]struct{}
}

func buildDashboardProviderSummaries(entries []domain.UsageEntry, fees []domain.SubscriptionFee, sessions []domain.SessionSummary) []DashboardProviderSummary {
	accumulators := map[domain.ProviderName]*providerSummaryAccumulator{}

	ensure := func(provider domain.ProviderName) *providerSummaryAccumulator {
		if current, ok := accumulators[provider]; ok {
			return current
		}
		created := &providerSummaryAccumulator{provider: provider, sessionIDs: map[string]struct{}{}}
		accumulators[provider] = created
		return created
	}

	for _, entry := range entries {
		accumulator := ensure(entry.Provider)
		accumulator.variableSpendUSD += entry.CostBreakdown.TotalUSD
		accumulator.usageEntryCount++
		if entry.SessionID != "" {
			accumulator.sessionIDs[entry.SessionID] = struct{}{}
		}
	}

	for _, fee := range fees {
		accumulator := ensure(fee.Provider)
		accumulator.subscriptionSpendUSD += fee.FeeUSD
	}

	for _, session := range sessions {
		accumulator := ensure(session.Provider)
		accumulator.sessionIDs[session.SessionID] = struct{}{}
	}

	providers := make([]domain.ProviderName, 0, len(accumulators))
	for provider := range accumulators {
		providers = append(providers, provider)
	}
	slices.SortFunc(providers, func(a, b domain.ProviderName) int {
		return cmp.Compare(a.String(), b.String())
	})

	results := make([]DashboardProviderSummary, 0, len(providers))
	for _, provider := range providers {
		accumulator := accumulators[provider]
		results = append(results, DashboardProviderSummary{
			Provider:             provider,
			VariableSpendUSD:     accumulator.variableSpendUSD,
			SubscriptionSpendUSD: accumulator.subscriptionSpendUSD,
			TotalSpendUSD:        accumulator.variableSpendUSD + accumulator.subscriptionSpendUSD,
			UsageEntryCount:      accumulator.usageEntryCount,
			SessionCount:         len(accumulator.sessionIDs),
		})
	}

	return results
}

func buildDashboardBudgetSummaries(budgets []domain.MonthlyBudget, entries []domain.UsageEntry, fees []domain.SubscriptionFee) ([]DashboardBudgetSummary, error) {
	results := make([]DashboardBudgetSummary, 0, len(budgets))
	for _, budget := range budgets {
		matchedEntries, matchedFees := filterDashboardBudgetInputs(budget, entries, fees)
		currentSpend := sumUsageSpend(matchedEntries) + sumSubscriptionSpend(matchedFees)
		status, err := budget.EvaluateSpend(currentSpend)
		if err != nil {
			return nil, err
		}

		triggered := make([]float64, 0, len(status.TriggeredThresholds))
		for _, threshold := range status.TriggeredThresholds {
			triggered = append(triggered, threshold.Percent)
		}

		results = append(results, DashboardBudgetSummary{
			BudgetID:                   budget.BudgetID,
			Name:                       budget.Name,
			Provider:                   budget.Provider,
			ProjectHash:                budget.ProjectHash,
			LimitUSD:                   budget.LimitUSD,
			CurrentSpendUSD:            currentSpend,
			RemainingUSD:               status.RemainingUSD,
			TriggeredThresholdPercents: triggered,
			BudgetOverrunActive:        status.IsOverrun,
			Currency:                   budget.Currency,
		})
	}

	return results, nil
}

func filterDashboardBudgetInputs(budget domain.MonthlyBudget, usageEntries []domain.UsageEntry, subscriptionFees []domain.SubscriptionFee) ([]domain.UsageEntry, []domain.SubscriptionFee) {
	filteredEntries := make([]domain.UsageEntry, 0, len(usageEntries))
	for _, entry := range usageEntries {
		if budget.Provider != "" && entry.Provider != budget.Provider {
			continue
		}
		if budget.ProjectHash != "" && entry.Metadata["project_hash"] != budget.ProjectHash {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	filteredFees := make([]domain.SubscriptionFee, 0, len(subscriptionFees))
	for _, fee := range subscriptionFees {
		if budget.Provider != "" && fee.Provider != budget.Provider {
			continue
		}
		if budget.ProjectHash != "" {
			continue
		}
		filteredFees = append(filteredFees, fee)
	}

	return filteredEntries, filteredFees
}

func buildDashboardRecentSessions(sessions []domain.SessionSummary, limit int) []DashboardRecentSession {
	if limit <= 0 {
		limit = defaultDashboardRecentSessionLimit
	}

	sorted := make([]domain.SessionSummary, len(sessions))
	copy(sorted, sessions)
	slices.SortFunc(sorted, func(a, b domain.SessionSummary) int {
		if !a.EndedAt.Equal(b.EndedAt) {
			if a.EndedAt.After(b.EndedAt) {
				return -1
			}
			return 1
		}
		if !a.StartedAt.Equal(b.StartedAt) {
			if a.StartedAt.After(b.StartedAt) {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.SessionID, b.SessionID)
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	results := make([]DashboardRecentSession, 0, len(sorted))
	for _, session := range sorted {
		modelID := ""
		if session.PricingRef != nil {
			modelID = session.PricingRef.ModelID
		}
		results = append(results, DashboardRecentSession{
			SessionID:      session.SessionID,
			Provider:       session.Provider,
			BillingMode:    session.BillingMode,
			ProjectName:    session.ProjectName,
			AgentName:      session.AgentName,
			ModelID:        modelID,
			StartedAt:      session.StartedAt,
			EndedAt:        session.EndedAt,
			TotalCostUSD:   session.CostBreakdown.TotalUSD,
			TotalTokens:    session.Tokens.TotalTokens,
			DurationSecond: int64(session.Duration() / time.Second),
		})
	}

	return results
}
