package service

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestGraphQueryServiceQueryGraphsEmptyData(t *testing.T) {
	period := mustGraphMonthlyPeriod(t, 2026, time.April)
	repo := &graphStubUsageRepo{}
	svc := NewGraphQueryService(repo)

	data, err := svc.QueryGraphs(context.Background(), GraphQuery{Period: period})
	if err != nil {
		t.Fatalf("QueryGraphs() error = %v", err)
	}

	if repo.lastFilter.Period == nil {
		t.Fatal("ListUsageEntries() filter period = nil, want monthly period")
	}
	if got := repo.lastFilter.Period.StartAt; !got.Equal(period.StartAt) {
		t.Fatalf("filter period start = %v, want %v", got, period.StartAt)
	}
	if got := repo.lastFilter.Period.EndExclusive; !got.Equal(period.EndExclusive) {
		t.Fatalf("filter period end = %v, want %v", got, period.EndExclusive)
	}
	if len(data.ModelTokenUsages) != 0 || len(data.ModelCosts) != 0 || len(data.DailyTokenTrends) != 0 || len(data.ModelTokenBreakdowns) != 0 {
		t.Fatalf("QueryGraphs() = %+v, want empty snapshot", data)
	}
}

func TestGraphQueryServiceQueryGraphsSortsDescendingAndNormalizesUnknownModel(t *testing.T) {
	period := mustGraphMonthlyPeriod(t, 2026, time.April)
	repo := &graphStubUsageRepo{entries: []domain.UsageEntry{
		mustGraphUsageEntry(t, "entry-1", time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC), "gpt-4.1", 200, 50, 10, 0, 1.5),
		mustGraphUsageEntry(t, "entry-2", time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC), "gpt-5-mini", 100, 20, 0, 0, 3.2),
		mustGraphUsageEntry(t, "entry-3", time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC), "", 150, 25, 5, 0, 2.4),
	}}
	svc := NewGraphQueryService(repo)

	data, err := svc.QueryGraphs(context.Background(), GraphQuery{Period: period})
	if err != nil {
		t.Fatalf("QueryGraphs() error = %v", err)
	}

	if got, want := data.ModelTokenUsages[0].ModelName, "gpt-4.1"; got != want {
		t.Fatalf("ModelTokenUsages[0].ModelName = %q, want %q", got, want)
	}
	if got, want := data.ModelTokenUsages[1].ModelName, graphUnknownModelName; got != want {
		t.Fatalf("ModelTokenUsages[1].ModelName = %q, want %q", got, want)
	}
	if got, want := data.ModelTokenUsages[2].ModelName, "gpt-5-mini"; got != want {
		t.Fatalf("ModelTokenUsages[2].ModelName = %q, want %q", got, want)
	}

	if got, want := data.ModelCosts[0].ModelName, "gpt-5-mini"; got != want {
		t.Fatalf("ModelCosts[0].ModelName = %q, want %q", got, want)
	}
	if got, want := data.ModelCosts[1].ModelName, graphUnknownModelName; got != want {
		t.Fatalf("ModelCosts[1].ModelName = %q, want %q", got, want)
	}
	if got, want := data.ModelCosts[2].ModelName, "gpt-4.1"; got != want {
		t.Fatalf("ModelCosts[2].ModelName = %q, want %q", got, want)
	}
}

func TestGraphQueryServiceQueryGraphsBuildsDailyGroupingAcrossFullMonth(t *testing.T) {
	period := mustGraphMonthlyPeriod(t, 2026, time.April)
	repo := &graphStubUsageRepo{entries: []domain.UsageEntry{
		mustGraphUsageEntry(t, "entry-1", time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC), "gpt-4.1", 80, 20, 0, 0, 0.8),
		mustGraphUsageEntry(t, "entry-2", time.Date(2026, 4, 2, 14, 0, 0, 0, time.UTC), "claude-3-7-sonnet", 40, 10, 0, 0, 0.5),
		mustGraphUsageEntry(t, "entry-3", time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC), "gpt-4.1", 25, 5, 0, 0, 0.3),
	}}
	svc := NewGraphQueryService(repo)

	data, err := svc.QueryGraphs(context.Background(), GraphQuery{Period: period})
	if err != nil {
		t.Fatalf("QueryGraphs() error = %v", err)
	}

	if got, want := len(data.DailyTokenTrends), 30; got != want {
		t.Fatalf("len(DailyTokenTrends) = %d, want %d", got, want)
	}

	day1 := data.DailyTokenTrends[0]
	if !day1.Date.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("DailyTokenTrends[0].Date = %v, want 2026-04-01 UTC", day1.Date)
	}
	if len(day1.ModelBreakdown) != 0 {
		t.Fatalf("DailyTokenTrends[0].ModelBreakdown = %+v, want empty", day1.ModelBreakdown)
	}

	day2 := data.DailyTokenTrends[1]
	if got, want := len(day2.ModelBreakdown), 2; got != want {
		t.Fatalf("len(DailyTokenTrends[1].ModelBreakdown) = %d, want %d", got, want)
	}
	if got, want := day2.ModelBreakdown[0], (ModelDailyTokens{ModelName: "gpt-4.1", TotalTokens: 100}); got != want {
		t.Fatalf("DailyTokenTrends[1].ModelBreakdown[0] = %+v, want %+v", got, want)
	}
	if got, want := day2.ModelBreakdown[1], (ModelDailyTokens{ModelName: "claude-3-7-sonnet", TotalTokens: 50}); got != want {
		t.Fatalf("DailyTokenTrends[1].ModelBreakdown[1] = %+v, want %+v", got, want)
	}

	day3 := data.DailyTokenTrends[2]
	if len(day3.ModelBreakdown) != 0 {
		t.Fatalf("DailyTokenTrends[2].ModelBreakdown = %+v, want empty", day3.ModelBreakdown)
	}

	day4 := data.DailyTokenTrends[3]
	if got, want := day4.ModelBreakdown[0], (ModelDailyTokens{ModelName: "gpt-4.1", TotalTokens: 30}); got != want {
		t.Fatalf("DailyTokenTrends[3].ModelBreakdown[0] = %+v, want %+v", got, want)
	}
}

func TestGraphQueryServiceQueryGraphsBuildsTokenBreakdowns(t *testing.T) {
	period := mustGraphMonthlyPeriod(t, 2026, time.April)
	repo := &graphStubUsageRepo{entries: []domain.UsageEntry{
		mustGraphUsageEntry(t, "entry-1", time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC), "gpt-4.1", 100, 20, 10, 5, 1.0),
		mustGraphUsageEntry(t, "entry-2", time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC), "gpt-4.1", 40, 10, 5, 0, 0.5),
	}}
	svc := NewGraphQueryService(repo)

	data, err := svc.QueryGraphs(context.Background(), GraphQuery{Period: period})
	if err != nil {
		t.Fatalf("QueryGraphs() error = %v", err)
	}

	if got, want := len(data.ModelTokenBreakdowns), 1; got != want {
		t.Fatalf("len(ModelTokenBreakdowns) = %d, want %d", got, want)
	}

	if got, want := data.ModelTokenBreakdowns[0], (ModelTokenBreakdown{
		ModelName:        "gpt-4.1",
		InputTokens:      140,
		OutputTokens:     30,
		CacheReadTokens:  15,
		CacheWriteTokens: 5,
		TotalTokens:      190,
	}); got != want {
		t.Fatalf("ModelTokenBreakdowns[0] = %+v, want %+v", got, want)
	}

	if got, want := data.ModelTokenUsages[0].TotalTokens, int64(190); got != want {
		t.Fatalf("ModelTokenUsages[0].TotalTokens = %d, want %d", got, want)
	}
}

func TestGraphQueryServiceQueryGraphsBucketsModelsBeyondTopTenIntoOther(t *testing.T) {
	period := mustGraphMonthlyPeriod(t, 2026, time.April)
	entries := make([]domain.UsageEntry, 0, 12)
	for index := 0; index < 12; index++ {
		entries = append(entries, mustGraphUsageEntry(
			t,
			graphEntryID(index),
			time.Date(2026, 4, index+1, 12, 0, 0, 0, time.UTC),
			graphModelName(index),
			int64(120-index),
			0,
			0,
			0,
			float64(120-index)/10,
		))
	}
	repo := &graphStubUsageRepo{entries: entries}
	svc := NewGraphQueryService(repo)

	data, err := svc.QueryGraphs(context.Background(), GraphQuery{Period: period})
	if err != nil {
		t.Fatalf("QueryGraphs() error = %v", err)
	}

	if got, want := len(data.ModelTokenUsages), 11; got != want {
		t.Fatalf("len(ModelTokenUsages) = %d, want %d", got, want)
	}
	if got, want := data.ModelTokenUsages[10].ModelName, graphOtherModelName; got != want {
		t.Fatalf("ModelTokenUsages[10].ModelName = %q, want %q", got, want)
	}
	if got, want := data.ModelTokenUsages[10].TotalTokens, int64(110+109); got != want {
		t.Fatalf("ModelTokenUsages[10].TotalTokens = %d, want %d", got, want)
	}

	if got, want := len(data.ModelCosts), 11; got != want {
		t.Fatalf("len(ModelCosts) = %d, want %d", got, want)
	}
	if got, want := data.ModelCosts[10].ModelName, graphOtherModelName; got != want {
		t.Fatalf("ModelCosts[10].ModelName = %q, want %q", got, want)
	}
	if got, want := data.ModelCosts[10].TotalCostUSD, 21.9; got != want {
		t.Fatalf("ModelCosts[10].TotalCostUSD = %v, want %v", got, want)
	}

	if got, want := data.ModelTokenBreakdowns[10].TotalTokens, int64(219); got != want {
		t.Fatalf("ModelTokenBreakdowns[10].TotalTokens = %d, want %d", got, want)
	}

	otherDay := data.DailyTokenTrends[10]
	if got, want := otherDay.ModelBreakdown[0], (ModelDailyTokens{ModelName: graphOtherModelName, TotalTokens: 110}); got != want {
		t.Fatalf("DailyTokenTrends[10].ModelBreakdown[0] = %+v, want %+v", got, want)
	}
	otherDay = data.DailyTokenTrends[11]
	if got, want := otherDay.ModelBreakdown[0], (ModelDailyTokens{ModelName: graphOtherModelName, TotalTokens: 109}); got != want {
		t.Fatalf("DailyTokenTrends[11].ModelBreakdown[0] = %+v, want %+v", got, want)
	}
}

type graphStubUsageRepo struct {
	entries     []domain.UsageEntry
	lastFilter  ports.UsageFilter
	listInvoked bool
}

func (s *graphStubUsageRepo) UpsertUsageEntries(context.Context, []domain.UsageEntry) error {
	return nil
}

func (s *graphStubUsageRepo) ListUsageEntries(_ context.Context, filter ports.UsageFilter) ([]domain.UsageEntry, error) {
	s.lastFilter = filter
	s.listInvoked = true
	return append([]domain.UsageEntry(nil), s.entries...), nil
}

func mustGraphMonthlyPeriod(t *testing.T, year int, month time.Month) domain.MonthlyPeriod {
	t.Helper()

	period, err := domain.NewMonthlyPeriodFromParts(year, month)
	if err != nil {
		t.Fatalf("NewMonthlyPeriodFromParts() error = %v", err)
	}

	return period
}

func mustGraphUsageEntry(t *testing.T, entryID string, occurredAt time.Time, modelName string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int64, totalCostUSD float64) domain.UsageEntry {
	t.Helper()

	tokens, err := domain.NewTokenUsage(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}

	costs, err := domain.NewCostBreakdown(totalCostUSD, 0, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}

	var pricingRef *domain.ModelPricingRef
	if modelName != "" {
		ref, err := domain.NewModelPricingRef(domain.ProviderOpenAI, modelName, modelName)
		if err != nil {
			t.Fatalf("NewModelPricingRef() error = %v", err)
		}
		pricingRef = &ref
	}

	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeBYOK,
		OccurredAt:    occurredAt,
		SessionID:     "session-" + entryID,
		ProjectName:   "graph-tests",
		AgentName:     "tester",
		PricingRef:    pricingRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}

	return entry
}

func graphEntryID(index int) string {
	return "entry-" + string(rune('a'+index))
}

func graphModelName(index int) string {
	return "model-" + string(rune('a'+index))
}
