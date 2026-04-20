package domain

import "time"

type WasteSummary struct {
	Period                    MonthlyPeriod
	TotalWasteCostUSD         float64
	TotalSpendCostUSD         float64
	WastePercent              float64
	WeeklyWasteCostUSD        float64
	MonthlyWasteCostUSD       float64
	ProjectedMonthEndWasteUSD float64
	ByDetector                []WasteByDetector
	TopCauses                 []WasteByDetector
	DailyTrend                []WasteTrendPoint
	GeneratedAt               time.Time
}

type WasteByDetector struct {
	Category          DetectorCategory
	AttributedCostUSD float64
	InsightCount      int
}

type WasteTrendPoint struct {
	Day          time.Time
	WasteCostUSD float64
}
