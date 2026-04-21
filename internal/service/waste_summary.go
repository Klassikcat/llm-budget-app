package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

var errWasteSummaryUsageRepoRequired = errors.New("waste summary service requires usage entry repository")
var errWasteSummaryInsightRepoRequired = errors.New("waste summary service requires insight repository")

var wasteSummaryDetectorOrder = []domain.DetectorCategory{
	domain.DetectorContextAvalanche,
	domain.DetectorRepeatedFileReads,
	domain.DetectorRetryAmplification,
	domain.DetectorOverQualifiedModel,
	domain.DetectorToolSchemaBloat,
	domain.DetectorPlanningTax,
	domain.DetectorZombieLoops,
	domain.DetectorMissedPromptCaching,
}

var wasteSummaryDetectorIndex = func() map[domain.DetectorCategory]int {
	index := make(map[domain.DetectorCategory]int, len(wasteSummaryDetectorOrder))
	for i, category := range wasteSummaryDetectorOrder {
		index[category] = i
	}
	return index
}()

type wasteOwnedInsight struct {
	insightID  string
	category   domain.DetectorCategory
	severity   domain.InsightSeverity
	detectedAt time.Time
}

type WasteSummaryService struct {
	usageRepo   ports.UsageEntryRepository
	insightRepo ports.InsightRepository
	clock       func() time.Time
}

func NewWasteSummaryService(usageRepo ports.UsageEntryRepository, insightRepo ports.InsightRepository) *WasteSummaryService {
	return &WasteSummaryService{
		usageRepo:   usageRepo,
		insightRepo: insightRepo,
		clock:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *WasteSummaryService) ClockForTest(clock func() time.Time) {
	if s == nil || clock == nil {
		return
	}

	s.clock = clock
}

func (s *WasteSummaryService) QueryWasteSummary(ctx context.Context, period domain.MonthlyPeriod) (domain.WasteSummary, error) {
	if s == nil || s.usageRepo == nil {
		return domain.WasteSummary{}, errWasteSummaryUsageRepoRequired
	}
	if s.insightRepo == nil {
		return domain.WasteSummary{}, errWasteSummaryInsightRepoRequired
	}

	generatedAt := time.Now().UTC()
	if s.clock != nil {
		generatedAt = s.clock().UTC()
	}

	entries, err := s.usageRepo.ListUsageEntries(ctx, ports.UsageFilter{Period: &period})
	if err != nil {
		return domain.WasteSummary{}, err
	}

	insights, err := s.insightRepo.ListInsights(ctx, period)
	if err != nil {
		return domain.WasteSummary{}, err
	}

	insightCounts := make(map[domain.DetectorCategory]int, len(wasteSummaryDetectorOrder))
	for _, insight := range insights {
		insightCounts[insight.Category]++
	}

	entryByID := make(map[string]domain.UsageEntry, len(entries))
	totalSpendCostUSD := 0.0
	for _, entry := range entries {
		entryByID[entry.EntryID] = entry
		totalSpendCostUSD += entry.CostBreakdown.TotalUSD
	}

	ownedByEntryID := make(map[string]wasteOwnedInsight, len(entries))
	for _, insight := range insights {
		for _, entryID := range insight.Payload.UsageEntryIDs {
			if _, ok := entryByID[entryID]; !ok {
				continue
			}

			candidate := wasteOwnedInsight{
				insightID:  insight.InsightID,
				category:   insight.Category,
				severity:   insight.Severity,
				detectedAt: insight.DetectedAt.UTC(),
			}

			current, ok := ownedByEntryID[entryID]
			if !ok || wasteOwnedInsightLess(candidate, current) {
				ownedByEntryID[entryID] = candidate
			}
		}
	}

	attributedByDetector := make(map[domain.DetectorCategory]float64, len(wasteSummaryDetectorOrder))
	dailyWaste := make(map[time.Time]float64, len(entries))
	totalWasteCostUSD := 0.0
	for entryID, ownedInsight := range ownedByEntryID {
		entry, ok := entryByID[entryID]
		if !ok {
			continue
		}

		cost := entry.CostBreakdown.TotalUSD
		attributedByDetector[ownedInsight.category] += cost
		totalWasteCostUSD += cost

		day := time.Date(entry.OccurredAt.UTC().Year(), entry.OccurredAt.UTC().Month(), entry.OccurredAt.UTC().Day(), 0, 0, 0, 0, time.UTC)
		dailyWaste[day] += cost
	}

	byDetector := make([]domain.WasteByDetector, 0, len(wasteSummaryDetectorOrder))
	for _, category := range wasteSummaryDetectorOrder {
		byDetector = append(byDetector, domain.WasteByDetector{
			Category:          category,
			AttributedCostUSD: attributedByDetector[category],
			InsightCount:      insightCounts[category],
		})
	}

	topCauses := make([]domain.WasteByDetector, 0, len(wasteSummaryDetectorOrder))
	for _, detector := range byDetector {
		if detector.AttributedCostUSD <= 0 {
			continue
		}
		topCauses = append(topCauses, detector)
	}
	sort.Slice(topCauses, func(i, j int) bool {
		if topCauses[i].AttributedCostUSD != topCauses[j].AttributedCostUSD {
			return topCauses[i].AttributedCostUSD > topCauses[j].AttributedCostUSD
		}
		return wasteSummaryDetectorIndex[topCauses[i].Category] < wasteSummaryDetectorIndex[topCauses[j].Category]
	})
	if len(topCauses) > 5 {
		topCauses = topCauses[:5]
	}

	dailyTrend := buildWasteSummaryDailyTrend(period, generatedAt, dailyWaste)
	weeklyWasteCostUSD := sumWasteSummaryCurrentWeek(dailyTrend, generatedAt)
	projectedMonthEndWasteUSD := projectWasteSummaryMonthEnd(totalWasteCostUSD, period, generatedAt)

	wastePercent := 0.0
	if totalSpendCostUSD > 0 {
		wastePercent = totalWasteCostUSD / totalSpendCostUSD * 100.0
	}

	return domain.WasteSummary{
		Period:                    period,
		TotalWasteCostUSD:         totalWasteCostUSD,
		TotalSpendCostUSD:         totalSpendCostUSD,
		WastePercent:              wastePercent,
		WeeklyWasteCostUSD:        weeklyWasteCostUSD,
		MonthlyWasteCostUSD:       totalWasteCostUSD,
		ProjectedMonthEndWasteUSD: projectedMonthEndWasteUSD,
		ByDetector:                byDetector,
		TopCauses:                 topCauses,
		DailyTrend:                dailyTrend,
		GeneratedAt:               generatedAt,
	}, nil
}

func wasteOwnedInsightLess(left, right wasteOwnedInsight) bool {
	leftRank := wasteSummarySeverityRank(left.severity)
	rightRank := wasteSummarySeverityRank(right.severity)
	if leftRank != rightRank {
		return leftRank > rightRank
	}
	if !left.detectedAt.Equal(right.detectedAt) {
		return left.detectedAt.Before(right.detectedAt)
	}
	return left.insightID < right.insightID
}

func wasteSummarySeverityRank(severity domain.InsightSeverity) int {
	switch severity {
	case domain.InsightSeverity("critical"):
		return 3
	case domain.InsightSeverityHigh:
		return 2
	case domain.InsightSeverityMedium:
		return 1
	default:
		return 0
	}
}

func buildWasteSummaryDailyTrend(period domain.MonthlyPeriod, generatedAt time.Time, dailyWaste map[time.Time]float64) []domain.WasteTrendPoint {
	elapsedDays := wasteSummaryElapsedDays(period, generatedAt)
	trend := make([]domain.WasteTrendPoint, 0, elapsedDays)
	for dayOffset := 0; dayOffset < elapsedDays; dayOffset++ {
		day := period.StartAt.AddDate(0, 0, dayOffset)
		trend = append(trend, domain.WasteTrendPoint{
			Day:          day,
			WasteCostUSD: dailyWaste[day],
		})
	}
	return trend
}

func sumWasteSummaryCurrentWeek(dailyTrend []domain.WasteTrendPoint, generatedAt time.Time) float64 {
	_, currentWeek := generatedAt.UTC().ISOWeek()
	currentYear := generatedAt.UTC().Year()
	total := 0.0
	for _, point := range dailyTrend {
		year, week := point.Day.UTC().ISOWeek()
		if year == currentYear && week == currentWeek {
			total += point.WasteCostUSD
		}
	}
	return total
}

func projectWasteSummaryMonthEnd(totalWasteCostUSD float64, period domain.MonthlyPeriod, generatedAt time.Time) float64 {
	elapsedDays := wasteSummaryElapsedDays(period, generatedAt)
	if elapsedDays == 0 {
		return totalWasteCostUSD
	}
	return totalWasteCostUSD * float64(wasteSummaryDaysInMonth(period)) / float64(elapsedDays)
}

func wasteSummaryElapsedDays(period domain.MonthlyPeriod, generatedAt time.Time) int {
	if generatedAt.Before(period.StartAt) {
		return 0
	}
	if !generatedAt.Before(period.EndExclusive) {
		return wasteSummaryDaysInMonth(period)
	}
	return generatedAt.UTC().Day()
}

func wasteSummaryDaysInMonth(period domain.MonthlyPeriod) int {
	return period.EndExclusive.Add(-time.Nanosecond).UTC().Day()
}
