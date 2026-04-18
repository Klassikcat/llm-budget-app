package service

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	graphTopModelLimit    = 10
	graphOtherModelName   = "Other"
	graphUnknownModelName = "unknown-model"
)

type GraphQuery struct {
	Period domain.MonthlyPeriod
}

type GraphSnapshot struct {
	ModelTokenUsages     []ModelTokenUsage
	ModelCosts           []ModelCost
	DailyTokenTrends     []DailyTokenTrend
	ModelTokenBreakdowns []ModelTokenBreakdown
}

type ModelTokenUsage struct {
	ModelName        string
	TotalTokens      int64
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
}

type ModelCost struct {
	ModelName    string
	TotalCostUSD float64
}

type DailyTokenTrend struct {
	Date           time.Time
	ModelBreakdown []ModelDailyTokens
}

type ModelDailyTokens struct {
	ModelName   string
	TotalTokens int64
}

type ModelTokenBreakdown struct {
	ModelName        string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	TotalTokens      int64
}

type GraphQueryService struct {
	usageRepo ports.UsageEntryRepository
	clock     func() time.Time
}

func NewGraphQueryService(usageRepo ports.UsageEntryRepository) *GraphQueryService {
	return &GraphQueryService{
		usageRepo: usageRepo,
		clock:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *GraphQueryService) ClockForTest(clock func() time.Time) {
	if s == nil || clock == nil {
		return
	}

	s.clock = clock
}

func (s *GraphQueryService) QueryGraphs(ctx context.Context, query GraphQuery) (GraphSnapshot, error) {
	if s == nil || s.usageRepo == nil {
		return GraphSnapshot{}, errUsageEntryRepositoryRequired
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
			return GraphSnapshot{}, err
		}
	}

	entries, err := s.usageRepo.ListUsageEntries(ctx, ports.UsageFilter{Period: &period})
	if err != nil {
		return GraphSnapshot{}, err
	}

	return buildGraphSnapshot(period, entries), nil
}

type graphModelAccumulator struct {
	modelName        string
	totalTokens      int64
	inputTokens      int64
	outputTokens     int64
	cacheReadTokens  int64
	cacheWriteTokens int64
	totalCostUSD     float64
	dailyTokens      map[time.Time]int64
}

func buildGraphSnapshot(period domain.MonthlyPeriod, entries []domain.UsageEntry) GraphSnapshot {
	if len(entries) == 0 {
		return GraphSnapshot{}
	}

	accumulators := buildGraphAccumulators(entries)
	tokenModels := buildTopTokenGraphModels(accumulators)
	costModels := buildTopCostGraphModels(accumulators)

	return GraphSnapshot{
		ModelTokenUsages:     buildModelTokenUsages(tokenModels),
		ModelCosts:           buildModelCosts(costModels),
		DailyTokenTrends:     buildDailyTokenTrends(period, tokenModels),
		ModelTokenBreakdowns: buildModelTokenBreakdowns(tokenModels),
	}
}

func buildGraphAccumulators(entries []domain.UsageEntry) map[string]*graphModelAccumulator {
	accumulators := make(map[string]*graphModelAccumulator, len(entries))

	ensure := func(modelName string) *graphModelAccumulator {
		if current, ok := accumulators[modelName]; ok {
			return current
		}

		created := &graphModelAccumulator{
			modelName:   modelName,
			dailyTokens: map[time.Time]int64{},
		}
		accumulators[modelName] = created
		return created
	}

	for _, entry := range entries {
		modelName := normalizeGraphModelName(entry)
		accumulator := ensure(modelName)
		accumulator.inputTokens += entry.Tokens.InputTokens
		accumulator.outputTokens += entry.Tokens.OutputTokens
		accumulator.cacheReadTokens += entry.Tokens.CacheReadTokens
		accumulator.cacheWriteTokens += entry.Tokens.CacheWriteTokens
		accumulator.totalTokens += entry.Tokens.TotalTokens
		accumulator.totalCostUSD += entry.CostBreakdown.TotalUSD

		day := time.Date(entry.OccurredAt.UTC().Year(), entry.OccurredAt.UTC().Month(), entry.OccurredAt.UTC().Day(), 0, 0, 0, 0, time.UTC)
		accumulator.dailyTokens[day] += entry.Tokens.TotalTokens
	}

	return accumulators
}

func normalizeGraphModelName(entry domain.UsageEntry) string {
	if entry.PricingRef == nil {
		return graphUnknownModelName
	}

	modelName := strings.TrimSpace(entry.PricingRef.ModelID)
	if modelName == "" {
		return graphUnknownModelName
	}

	return modelName
}

func buildTopTokenGraphModels(accumulators map[string]*graphModelAccumulator) []graphModelAccumulator {
	models := sortedGraphModels(accumulators, func(left, right *graphModelAccumulator) int {
		if left.totalTokens != right.totalTokens {
			return cmp.Compare(right.totalTokens, left.totalTokens)
		}
		return cmp.Compare(left.modelName, right.modelName)
	})

	return limitGraphModels(models)
}

func buildTopCostGraphModels(accumulators map[string]*graphModelAccumulator) []graphModelAccumulator {
	models := sortedGraphModels(accumulators, func(left, right *graphModelAccumulator) int {
		if left.totalCostUSD != right.totalCostUSD {
			if left.totalCostUSD > right.totalCostUSD {
				return -1
			}
			return 1
		}
		return cmp.Compare(left.modelName, right.modelName)
	})

	return limitGraphModels(models)
}

func sortedGraphModels(accumulators map[string]*graphModelAccumulator, compare func(left, right *graphModelAccumulator) int) []graphModelAccumulator {
	models := make([]graphModelAccumulator, 0, len(accumulators))
	for _, accumulator := range accumulators {
		models = append(models, cloneGraphAccumulator(accumulator))
	}

	slices.SortFunc(models, func(left, right graphModelAccumulator) int {
		return compare(&left, &right)
	})

	return models
}

func cloneGraphAccumulator(accumulator *graphModelAccumulator) graphModelAccumulator {
	cloned := graphModelAccumulator{
		modelName:        accumulator.modelName,
		totalTokens:      accumulator.totalTokens,
		inputTokens:      accumulator.inputTokens,
		outputTokens:     accumulator.outputTokens,
		cacheReadTokens:  accumulator.cacheReadTokens,
		cacheWriteTokens: accumulator.cacheWriteTokens,
		totalCostUSD:     accumulator.totalCostUSD,
		dailyTokens:      make(map[time.Time]int64, len(accumulator.dailyTokens)),
	}

	for date, totalTokens := range accumulator.dailyTokens {
		cloned.dailyTokens[date] = totalTokens
	}

	return cloned
}

func limitGraphModels(models []graphModelAccumulator) []graphModelAccumulator {
	if len(models) <= graphTopModelLimit {
		return models
	}

	limited := make([]graphModelAccumulator, 0, graphTopModelLimit+1)
	limited = append(limited, models[:graphTopModelLimit]...)

	other := graphModelAccumulator{
		modelName:   graphOtherModelName,
		dailyTokens: map[time.Time]int64{},
	}
	for _, model := range models[graphTopModelLimit:] {
		other.inputTokens += model.inputTokens
		other.outputTokens += model.outputTokens
		other.cacheReadTokens += model.cacheReadTokens
		other.cacheWriteTokens += model.cacheWriteTokens
		other.totalTokens += model.totalTokens
		other.totalCostUSD += model.totalCostUSD
		for date, totalTokens := range model.dailyTokens {
			other.dailyTokens[date] += totalTokens
		}
	}

	return append(limited, other)
}

func buildModelTokenUsages(models []graphModelAccumulator) []ModelTokenUsage {
	results := make([]ModelTokenUsage, 0, len(models))
	for _, model := range models {
		results = append(results, ModelTokenUsage{
			ModelName:        model.modelName,
			TotalTokens:      model.totalTokens,
			InputTokens:      model.inputTokens,
			OutputTokens:     model.outputTokens,
			CacheReadTokens:  model.cacheReadTokens,
			CacheWriteTokens: model.cacheWriteTokens,
		})
	}

	return results
}

func buildModelCosts(models []graphModelAccumulator) []ModelCost {
	results := make([]ModelCost, 0, len(models))
	for _, model := range models {
		results = append(results, ModelCost{
			ModelName:    model.modelName,
			TotalCostUSD: model.totalCostUSD,
		})
	}

	return results
}

func buildDailyTokenTrends(period domain.MonthlyPeriod, models []graphModelAccumulator) []DailyTokenTrend {
	if period.StartAt.IsZero() || period.EndExclusive.IsZero() {
		return nil
	}

	trends := make([]DailyTokenTrend, 0, int(period.EndExclusive.Sub(period.StartAt).Hours()/24))
	for date := period.StartAt; date.Before(period.EndExclusive); date = date.AddDate(0, 0, 1) {
		breakdown := make([]ModelDailyTokens, 0, len(models))
		for _, model := range models {
			totalTokens := model.dailyTokens[date]
			if totalTokens == 0 {
				continue
			}

			breakdown = append(breakdown, ModelDailyTokens{
				ModelName:   model.modelName,
				TotalTokens: totalTokens,
			})
		}

		trends = append(trends, DailyTokenTrend{
			Date:           date,
			ModelBreakdown: breakdown,
		})
	}

	return trends
}

func buildModelTokenBreakdowns(models []graphModelAccumulator) []ModelTokenBreakdown {
	results := make([]ModelTokenBreakdown, 0, len(models))
	for _, model := range models {
		results = append(results, ModelTokenBreakdown{
			ModelName:        model.modelName,
			InputTokens:      model.inputTokens,
			OutputTokens:     model.outputTokens,
			CacheReadTokens:  model.cacheReadTokens,
			CacheWriteTokens: model.cacheWriteTokens,
			TotalTokens:      model.totalTokens,
		})
	}

	return results
}
