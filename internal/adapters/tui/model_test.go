package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestModelRendersDashboardSectionsAndNavigation(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	loader := staticLoader{data: service.DashboardSnapshot{
		Period: period,
		Totals: service.DashboardTotals{
			TotalSpendUSD:        123.45,
			VariableSpendUSD:     23.45,
			SubscriptionSpendUSD: 100,
		},
		ProviderSummaries: []service.DashboardProviderSummary{{
			Provider:             domain.ProviderOpenAI,
			TotalSpendUSD:        23.45,
			VariableSpendUSD:     23.45,
			SubscriptionSpendUSD: 0,
			SessionCount:         1,
			UsageEntryCount:      2,
		}},
		Budgets: []service.DashboardBudgetSummary{{
			BudgetID:        "budget-1",
			Name:            "Core Budget",
			Provider:        domain.ProviderOpenAI,
			LimitUSD:        50,
			CurrentSpendUSD: 23.45,
			RemainingUSD:    26.55,
		}},
		RecentSessions: []service.DashboardRecentSession{{
			SessionID:    "session-1",
			Provider:     domain.ProviderOpenAI,
			AgentName:    "codex",
			ProjectName:  "alpha",
			EndedAt:      time.Date(2026, 4, 17, 12, 45, 0, 0, time.UTC),
			TotalCostUSD: 23.45,
			TotalTokens:  1500,
			BillingMode:  domain.BillingModeBYOK,
			ModelID:      "gpt-5-mini",
		}},
	}}
	m := newModel(modelDependencies{loader: loader}, period)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(dashboardLoadedMsg{data: loader.data})
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Monthly Totals", "Provider Summary", "Budgets", "Recent Sessions", "Monthly total:", "g graphs"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if !strings.Contains(m.View(), "[Provider Summary]") {
		t.Fatalf("View() did not move focus to provider section\n%s", m.View())
	}
}

func TestModelRendersExplicitEmptyState(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 18})
	m = updated.(model)
	updated, _ = m.Update(dashboardLoadedMsg{data: service.DashboardSnapshot{Period: period, Empty: true}})
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "No spend, budgets, or sessions are available") {
		t.Fatalf("View() missing empty state message\n%s", view)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if !strings.Contains(m.View(), "Navigation stays active") {
		t.Fatalf("View() lost empty state after navigation\n%s", m.View())
	}
}

func TestModelShowsAlertBannerAndInsightDrillDown(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	hash, err := domain.NewInsightHash("target_hash", "sha256:abc123")
	if err != nil {
		t.Fatalf("NewInsightHash() error = %v", err)
	}
	metric, err := domain.NewInsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, 1.25)
	if err != nil {
		t.Fatalf("NewInsightMetric() error = %v", err)
	}
	payload, err := domain.NewInsightPayload([]string{"session-1"}, []string{"usage-1"}, []domain.InsightHash{hash}, nil, []domain.InsightMetric{metric})
	if err != nil {
		t.Fatalf("NewInsightPayload() error = %v", err)
	}
	insight, err := domain.NewInsight(domain.Insight{
		InsightID:  "insight-1",
		Category:   domain.DetectorToolSchemaBloat,
		Severity:   domain.InsightSeverityHigh,
		DetectedAt: period.StartAt.Add(24 * time.Hour),
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("NewInsight() error = %v", err)
	}
	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          "alert-1",
		Kind:             domain.AlertKindInsightDetected,
		Severity:         domain.AlertSeverityWarning,
		TriggeredAt:      period.StartAt.Add(25 * time.Hour),
		Period:           period,
		InsightID:        insight.InsightID,
		DetectorCategory: insight.Category,
	})
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}

	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	m.alerts = []domain.AlertEvent{alert}
	m.insights = []domain.Insight{insight}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Alert status:", "Insight Detail", "sha256:abc123", "estimated_waste_usd"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}
}

func TestInsightDetailScrollKeysMoveViewport(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	hashes := make([]domain.InsightHash, 0, 24)
	for i := range 24 {
		hash, err := domain.NewInsightHash("target_hash", "sha256:scroll-"+time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC).Add(time.Duration(i)*time.Minute).Format("150405"))
		if err != nil {
			t.Fatalf("NewInsightHash() error = %v", err)
		}
		hashes = append(hashes, hash)
	}
	payload, err := domain.NewInsightPayload([]string{"session-1"}, []string{"usage-1"}, hashes, nil, nil)
	if err != nil {
		t.Fatalf("NewInsightPayload() error = %v", err)
	}
	insight, err := domain.NewInsight(domain.Insight{
		InsightID:  "insight-scroll",
		Category:   domain.DetectorToolSchemaBloat,
		Severity:   domain.InsightSeverityHigh,
		DetectedAt: period.StartAt.Add(24 * time.Hour),
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("NewInsight() error = %v", err)
	}

	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	m.insights = []domain.Insight{insight}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 12})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if got := m.viewport.YOffset; got != 0 {
		t.Fatalf("initial viewport.YOffset = %d, want 0", got)
	}
	if !strings.Contains(m.View(), "Insight Detail") {
		t.Fatalf("View() missing initial insight detail header\n%s", m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if got := m.viewport.YOffset; got < 1 {
		t.Fatalf("viewport.YOffset after KeyDown = %d, want >= 1", got)
	}

	previousOffset := m.viewport.YOffset
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(model)
	if got := m.viewport.YOffset; got <= previousOffset {
		t.Fatalf("viewport.YOffset after l = %d, want > %d", got, previousOffset)
	}
	if !strings.Contains(m.View(), "h/j/k/l scroll") {
		t.Fatalf("View() missing detail scroll help\n%s", m.View())
	}
	for i := 0; i < 12; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
	}
	if !strings.Contains(m.View(), "sha256:scroll") {
		t.Fatalf("View() missing scrolled hash content\n%s", m.View())
	}
}

func TestInsightDetailPreservesScrollOffsetAcrossViewportSync(t *testing.T) {
	m := newScrolledInsightDetailModel(t)

	for range 6 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
	}
	before := m.viewport.YOffset
	if before == 0 {
		t.Fatal("expected non-zero viewport offset before sync")
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 12})
	m = updated.(model)
	if got := m.viewport.YOffset; got != before {
		t.Fatalf("viewport.YOffset after WindowSizeMsg = %d, want %d", got, before)
	}

	updated, _ = m.Update(insightsLoadedMsg{insights: m.insights})
	m = updated.(model)
	if got := m.viewport.YOffset; got != before {
		t.Fatalf("viewport.YOffset after insightsLoadedMsg = %d, want %d", got, before)
	}
}

func TestInsightDetailSupportsPageAndReverseScrollKeys(t *testing.T) {
	m := newScrolledInsightDetailModel(t)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(model)
	pageOffset := m.viewport.YOffset
	if pageOffset == 0 {
		t.Fatal("expected pgdown to move viewport")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(model)
	if got := m.viewport.YOffset; got >= pageOffset {
		t.Fatalf("viewport.YOffset after h = %d, want < %d", got, pageOffset)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(model)
	if got := m.viewport.YOffset; got != 0 {
		t.Fatalf("viewport.YOffset after pgup = %d, want 0", got)
	}
}

func TestInsightListKeepsSelectionVisibleWhenScrollingDown(t *testing.T) {
	m := newInsightListModel(t, 20)

	for range 12 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
	}

	if got := m.insightSelection; got != 12 {
		t.Fatalf("insightSelection = %d, want 12", got)
	}
	if got := m.viewport.YOffset; got == 0 {
		t.Fatalf("viewport.YOffset = %d, want > 0 once selection moves off-screen", got)
	}
	selectedInsight := m.insights[m.insightSelection]
	if !strings.Contains(m.View(), selectedInsight.InsightID) {
		t.Fatalf("View() missing selected insight %q\n%s", selectedInsight.InsightID, m.View())
	}
	for _, needle := range []string{"Insight List", "Alert status:", "Tab/Shift+Tab cycle tabs • ↑↓ move • Enter detail • r refresh • Esc back • q quit", "Insights"} {
		if !strings.Contains(m.View(), needle) {
			t.Fatalf("View() missing fixed insight chrome %q\n%s", needle, m.View())
		}
	}
	for _, needle := range []string{"[Logs]", "Dashboard"} {
		if !strings.Contains(m.View(), needle) {
			t.Fatalf("View() missing insight tab chrome %q\n%s", needle, m.View())
		}
	}
}

func TestInsightListVimKeysScrollSelectionAndViewport(t *testing.T) {
	m := newInsightListModel(t, 18)
	if !strings.Contains(m.View(), "Tab/Shift+Tab cycle tabs • ↑↓ move • Enter detail • r refresh • Esc back • q quit") {
		t.Fatalf("View() missing insight list vim help before scrolling\n%s", m.View())
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(model)
	if got := m.insightSelection; got != 1 {
		t.Fatalf("insightSelection after j = %d, want 1", got)
	}

	for range 10 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(model)
	}
	if got := m.viewport.YOffset; got == 0 {
		t.Fatalf("viewport.YOffset after repeated j = %d, want > 0", got)
	}

	previousSelection := m.insightSelection
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(model)
	if got := m.insightSelection; got != previousSelection-1 {
		t.Fatalf("insightSelection after k = %d, want %d", got, previousSelection-1)
	}
	selectedInsight := m.insights[m.insightSelection]
	if !strings.Contains(m.View(), selectedInsight.InsightID) {
		t.Fatalf("View() missing selected insight %q after k\n%s", selectedInsight.InsightID, m.View())
	}
}

func TestInsightTabTransitions(t *testing.T) {
	t.Run("I1 entering insight list defaults to Dashboard", func(t *testing.T) {
		m := newInsightTabModel(t, 3)
		if m.mode != viewInsightList {
			t.Fatalf("mode = %v, want %v", m.mode, viewInsightList)
		}
		if m.insightTab != insightTabDashboard {
			t.Fatalf("insightTab = %v, want %v", m.insightTab, insightTabDashboard)
		}
	})

	t.Run("I2 tab toggles sub-tab without changing viewMode", func(t *testing.T) {
		m := newInsightTabModel(t, 3)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(model)
		if m.mode != viewInsightList {
			t.Fatalf("mode after tab = %v, want %v", m.mode, viewInsightList)
		}
		if m.insightTab != insightTabLogs {
			t.Fatalf("insightTab after tab = %v, want %v", m.insightTab, insightTabLogs)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
		m = updated.(model)
		if m.insightTab != insightTabDashboard {
			t.Fatalf("insightTab after shift+tab = %v, want %v", m.insightTab, insightTabDashboard)
		}
	})

	t.Run("I3 Enter on Dashboard is no-op", func(t *testing.T) {
		m := newInsightTabModel(t, 3)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(model)
		if m.mode != viewInsightList {
			t.Fatalf("mode after enter on dashboard = %v, want %v", m.mode, viewInsightList)
		}
		if m.insightTab != insightTabDashboard {
			t.Fatalf("insightTab after enter on dashboard = %v, want %v", m.insightTab, insightTabDashboard)
		}
	})

	t.Run("I4 Enter on Logs opens detail preserving Logs tab", func(t *testing.T) {
		m := newInsightLogsTabModel(t, 3)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(model)
		if m.mode != viewInsightDetail {
			t.Fatalf("mode after enter on logs = %v, want %v", m.mode, viewInsightDetail)
		}
		if m.insightTab != insightTabLogs {
			t.Fatalf("insightTab after enter on logs = %v, want %v", m.insightTab, insightTabLogs)
		}
		if got := m.selectedInsightID; got != m.insights[0].InsightID {
			t.Fatalf("selectedInsightID = %q, want %q", got, m.insights[0].InsightID)
		}
	})

	t.Run("I5 Esc from detail returns to insight list with Logs selected", func(t *testing.T) {
		m := newInsightLogsTabModel(t, 3)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(model)
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(model)
		if m.mode != viewInsightList {
			t.Fatalf("mode after esc from detail = %v, want %v", m.mode, viewInsightList)
		}
		if m.insightTab != insightTabLogs {
			t.Fatalf("insightTab after esc from detail = %v, want %v", m.insightTab, insightTabLogs)
		}
	})

	t.Run("I6 top-level i re-entry resets to Dashboard", func(t *testing.T) {
		m := newInsightLogsTabModel(t, 3)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(model)
		if m.mode != viewDashboard {
			t.Fatalf("mode after leaving insights = %v, want %v", m.mode, viewDashboard)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		m = updated.(model)
		if m.mode != viewInsightList {
			t.Fatalf("mode after re-entering insights = %v, want %v", m.mode, viewInsightList)
		}
		if m.insightTab != insightTabDashboard {
			t.Fatalf("insightTab after re-entering insights = %v, want %v", m.insightTab, insightTabDashboard)
		}
	})

	t.Run("I7 row selection preserved across Dashboard and Logs switches", func(t *testing.T) {
		m := newInsightLogsTabModel(t, 5)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
		if got := m.insightSelection; got != 2 {
			t.Fatalf("insightSelection before tab switch = %d, want 2", got)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
		m = updated.(model)
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(model)
		if got := m.insightSelection; got != 2 {
			t.Fatalf("insightSelection after tab switch = %d, want 2", got)
		}
	})

	t.Run("I8 exiting and re-entering resets selection to 0", func(t *testing.T) {
		m := newInsightLogsTabModel(t, 5)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(model)
		if got := m.insightSelection; got != 2 {
			t.Fatalf("insightSelection before exit = %d, want 2", got)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(model)
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		m = updated.(model)
		if got := m.insightSelection; got != 0 {
			t.Fatalf("insightSelection after re-entering insights = %d, want 0", got)
		}
	})
}

func TestInsightDashboardLoadTriggersWasteSummary(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	wasteSummary := domain.WasteSummary{Period: period, TotalWasteCostUSD: 12.34}
	loader := &captureWasteSummaryLoader{data: wasteSummary}
	m := newModel(modelDependencies{
		loader:       staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}},
		wasteSummary: loader,
	}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m = updated.(model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updated.(model)
	if m.mode != viewInsightList {
		t.Fatalf("mode after i = %v, want %v", m.mode, viewInsightList)
	}
	if m.insightTab != insightTabDashboard {
		t.Fatalf("insightTab after i = %v, want %v", m.insightTab, insightTabDashboard)
	}
	if cmd == nil {
		t.Fatal("expected waste summary load command when entering insights")
	}

	msg := cmd()
	loaded, ok := msg.(wasteSummaryLoadedMsg)
	if !ok {
		t.Fatalf("cmd() message type = %T, want wasteSummaryLoadedMsg", msg)
	}
	if got := len(loader.periods); got != 1 {
		t.Fatalf("QueryWasteSummary() calls = %d, want 1", got)
	}
	if got := loader.periods[0]; got != period {
		t.Fatalf("QueryWasteSummary() period = %#v, want %#v", got, period)
	}

	updated, _ = m.Update(loaded)
	m = updated.(model)
	if got := m.wasteSummaryData.TotalWasteCostUSD; got != wasteSummary.TotalWasteCostUSD {
		t.Fatalf("wasteSummaryData.TotalWasteCostUSD = %v, want %v", got, wasteSummary.TotalWasteCostUSD)
	}
}

func TestInsightDashboardRefreshLoadsWasteSummary(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	loader := &captureWasteSummaryLoader{data: domain.WasteSummary{Period: period, TotalWasteCostUSD: 9.87}}
	m := newModel(modelDependencies{
		loader:       staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}},
		wasteSummary: loader,
	}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updated.(model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected refresh command on insight dashboard")
	}

	if got := len(loader.periods); got != 0 {
		t.Fatalf("QueryWasteSummary() calls before running refresh cmd = %d, want 0", got)
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("refresh cmd() message type = %T, want tea.BatchMsg", msg)
	}
	sawWasteSummaryMsg := false
	for _, nested := range batch {
		if nested == nil {
			continue
		}
		if _, ok := nested().(wasteSummaryLoadedMsg); ok {
			sawWasteSummaryMsg = true
		}
	}
	if !sawWasteSummaryMsg {
		t.Fatal("refresh batch did not include wasteSummaryLoadedMsg")
	}
	if got := len(loader.periods); got != 1 {
		t.Fatalf("QueryWasteSummary() calls after running refresh cmd = %d, want 1", got)
	}
}

func newScrolledInsightDetailModel(t *testing.T) model {
	t.Helper()

	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	hashes := make([]domain.InsightHash, 0, 24)
	for i := range 24 {
		hash, err := domain.NewInsightHash("target_hash", "sha256:scroll-"+time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC).Add(time.Duration(i)*time.Minute).Format("150405"))
		if err != nil {
			t.Fatalf("NewInsightHash() error = %v", err)
		}
		hashes = append(hashes, hash)
	}
	payload, err := domain.NewInsightPayload([]string{"session-1"}, []string{"usage-1"}, hashes, nil, nil)
	if err != nil {
		t.Fatalf("NewInsightPayload() error = %v", err)
	}
	insight, err := domain.NewInsight(domain.Insight{
		InsightID:  "insight-scroll",
		Category:   domain.DetectorToolSchemaBloat,
		Severity:   domain.InsightSeverityHigh,
		DetectedAt: period.StartAt.Add(24 * time.Hour),
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("NewInsight() error = %v", err)
	}

	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	m.insights = []domain.Insight{insight}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 12})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return updated.(model)
}

func newInsightListModel(t *testing.T, insightCount int) model {
	t.Helper()

	m := newInsightTabModel(t, insightCount)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	return updated.(model)
}

func newInsightTabModel(t *testing.T, insightCount int) model {
	t.Helper()

	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	insights := make([]domain.Insight, 0, insightCount)
	for i := range insightCount {
		payload, err := domain.NewInsightPayload([]string{fmt.Sprintf("session-%d", i)}, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("NewInsightPayload() error = %v", err)
		}
		insight, err := domain.NewInsight(domain.Insight{
			InsightID:  fmt.Sprintf("insight-%02d", i),
			Category:   domain.DetectorToolSchemaBloat,
			Severity:   domain.InsightSeverityHigh,
			DetectedAt: period.StartAt.Add(time.Duration(i) * time.Hour),
			Period:     period,
			Payload:    payload,
		})
		if err != nil {
			t.Fatalf("NewInsight() error = %v", err)
		}
		insights = append(insights, insight)
	}

	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	m.insights = insights
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	return updated.(model)
}

func newInsightLogsTabModel(t *testing.T, insightCount int) model {
	t.Helper()

	m := newInsightTabModel(t, insightCount)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	return updated.(model)
}

func TestManualEntryValidationPreservesAlertBanner(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          "alert-threshold",
		Kind:             domain.AlertKindBudgetThreshold,
		Severity:         domain.AlertSeverityCritical,
		TriggeredAt:      period.StartAt.Add(3 * time.Hour),
		Period:           period,
		BudgetID:         "budget-1",
		CurrentSpendUSD:  92,
		LimitUSD:         100,
		ThresholdPercent: 0.9,
	})
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, manualEntries: rejectingManualSaver{}}, period)
	m.alerts = []domain.AlertEvent{alert}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(model)
	setFieldValue(&m.manualForm.fields, "provider", "openrouter")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Alert status:", "budget budget-1 crossed 90%", "manual API entries support only openai and anthropic", "Fix the highlighted fields"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}
	if m.mode != viewManualEntryForm {
		t.Fatalf("mode = %v, want manual form after validation failure", m.mode)
	}
}

func TestSubscriptionFormSubmitsThroughSharedService(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	manager := &captureSubscriptionManager{}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: manager}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)

	if len(manager.saved) != 2 {
		t.Fatalf("saved subscriptions = %d, want 2", len(manager.saved))
	}
	if got := manager.saved[0].SubscriptionID; got != "openai-chatgpt-plus-2026-04-01" {
		t.Fatalf("SubscriptionID[0] = %q, want generated openai-chatgpt-plus-2026-04-01", got)
	}
	if got := manager.saved[1].SubscriptionID; got != "openai-chatgpt-pro-5x-2026-04-01" {
		t.Fatalf("SubscriptionID[1] = %q, want generated openai-chatgpt-pro-5x-2026-04-01", got)
	}
	if got := manager.saved[0].PlanCode; got != "openai-chatgpt-plus" {
		t.Fatalf("PlanCode[0] = %q, want generated openai-chatgpt-plus", got)
	}
	if got := manager.saved[1].PlanCode; got != "openai-chatgpt-pro-5x" {
		t.Fatalf("PlanCode[1] = %q, want generated openai-chatgpt-pro-5x", got)
	}
	if !manager.saved[0].CreatedAt.IsZero() || !manager.saved[0].UpdatedAt.IsZero() || !manager.saved[1].CreatedAt.IsZero() || !manager.saved[1].UpdatedAt.IsZero() {
		t.Fatalf("new subscription timestamps = %+v %+v, want zero audit timestamps before shared service save", manager.saved[0], manager.saved[1])
	}
	if m.mode != viewDashboard {
		t.Fatalf("mode = %v, want dashboard after successful save", m.mode)
	}
}

func TestSubscriptionFormViewShowsVisiblePresetChoices(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Choose Subscription", "> [ ] ChatGPT Plus — $20.00 / renewal 1 / openai", "[ ] Claude Max 20x — $200.00 / renewal 1 / claude", "[ ] Others (Manual)"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}
}

func TestDashboardCanOpenSubscriptionList(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	manager := &captureSubscriptionManager{saved: []domain.Subscription{mustTUISubscription(t, domain.ProviderOpenAI, "ChatGPT Plus", 20, time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC))}}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: manager}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected load subscriptions command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)
	if m.mode != viewSubscriptionList {
		t.Fatalf("mode = %v, want subscription list", m.mode)
	}
	view := m.View()
	for _, needle := range []string{"Subscriptions", "ChatGPT Plus", "openai", "d to delete"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}
}

func TestSubscriptionListCanDeleteSelectedSubscription(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	target := mustTUISubscription(t, domain.ProviderOpenAI, "ChatGPT Plus", 20, time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC))
	manager := &captureSubscriptionManager{saved: []domain.Subscription{target}}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: manager}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(model)
	updated, _ = m.Update(cmd())
	m = updated.(model)
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected delete subscription command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)
	if len(manager.disabled) != 1 || manager.disabled[0] != target.SubscriptionID {
		t.Fatalf("deleted subscriptions = %#v, want %q", manager.disabled, target.SubscriptionID)
	}
	if !strings.Contains(m.View(), "> ") {
		t.Fatalf("View() missing visible selection marker\n%s", m.View())
	}
}

func TestSubscriptionListDoesNotDeleteSettingsManagedSubscription(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	target := mustSubscription(t, domain.Subscription{
		SubscriptionID: "settings-openai-subscription",
		Provider:       domain.ProviderOpenAI,
		PlanCode:       "chatgpt-plus",
		PlanName:       "ChatGPT Plus",
		RenewalDay:     5,
		StartsAt:       time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
		FeeUSD:         20,
		IsActive:       true,
		CreatedAt:      time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
	})
	manager := &captureSubscriptionManager{saved: []domain.Subscription{target}}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: manager}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(model)
	updated, _ = m.Update(cmd())
	m = updated.(model)
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(model)
	if cmd != nil {
		t.Fatal("expected no delete command for settings-managed subscription")
	}
	if len(manager.disabled) != 0 {
		t.Fatalf("deleted subscriptions = %#v, want none", manager.disabled)
	}
	if !strings.Contains(m.statusMessage, "Disable settings-managed subscription") {
		t.Fatalf("statusMessage = %q, want settings guidance", m.statusMessage)
	}
}

func TestSubscriptionFormStartsWithNoPresetSelected(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	view := m.View()
	if !strings.Contains(view, "> [ ] ChatGPT Plus") {
		t.Fatalf("View() missing empty initial selection\n%s", view)
	}
}

func TestPresetCursorDoesNotChangeSelectedPresetUntilEnter(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "> [ ] ChatGPT Pro 5x") || strings.Contains(view, "[v] ChatGPT Plus") {
		t.Fatalf("View() should show moved cursor with no selection before Enter\n%s", view)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	view = m.View()
	if !strings.Contains(view, "[v] ChatGPT Pro 5x") || strings.Contains(view, "[v] ChatGPT Plus") {
		t.Fatalf("View() should select only hovered preset after Enter\n%s", view)
	}
}

func TestEmptyDashboardHelpMentionsSubscriptionLookup(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(dashboardLoadedMsg{data: service.DashboardSnapshot{Period: period, Empty: true}})
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"l opens", "subscriptions", "g opens graphs"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q in empty-state help\n%s", needle, view)
		}
	}
}

func TestSubscriptionFormResetsOnReopen(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	selectSubscriptionPreset(&m.subscriptionForm, "Others (Manual)")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "> [ ] ChatGPT Plus") {
		t.Fatalf("View() missing reset initial selector state\n%s", view)
	}
	if strings.Contains(view, "Provider") || strings.Contains(view, "Plan Name") {
		t.Fatalf("View() should not retain manual fields after reopen\n%s", view)
	}
}

func TestSubscriptionFormDefaultsToCurrentBillingDate(t *testing.T) {
	fixedNow := time.Date(2026, time.April, 19, 14, 30, 0, 0, time.UTC)
	form := newSubscriptionFormAt(fixedNow)
	values := collectFormValues(form.fields)

	if got, want := values["renewal_day"], ""; got != want {
		t.Fatalf("renewal_day default = %q, want %q", got, want)
	}
	if got, want := values["fee_usd"], ""; got != want {
		t.Fatalf("fee_usd default = %q, want %q", got, want)
	}

	expectedStartsAt := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	if got := values["starts_at"]; got != expectedStartsAt {
		t.Fatalf("starts_at default = %q, want %q", got, expectedStartsAt)
	}
}

func TestSubscriptionFormRejectsRFC3339DateTime(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "subscription-inactive.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: newTestSubscriptionManager(store)}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected submit command for preset save without manual date entry")
	}
}

func TestSubscriptionFormUsesPresetSpecificDefaults(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	manager := &captureSubscriptionManager{}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: manager}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	selectSubscriptionPreset(&m.subscriptionForm, "Claude Max 20x")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)

	if len(manager.saved) != 1 {
		t.Fatalf("saved subscriptions = %d, want 1", len(manager.saved))
	}
	if got := manager.saved[0].Provider; got != domain.ProviderClaude {
		t.Fatalf("Provider = %q, want claude", got)
	}
	if got := manager.saved[0].FeeUSD; got != 200 {
		t.Fatalf("FeeUSD = %v, want 200", got)
	}
	if got := manager.saved[0].RenewalDay; got != 1 {
		t.Fatalf("RenewalDay = %d, want 1", got)
	}
}

func TestConfirmedPresetFormShowsStartsAtField(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	view := m.View()
	if strings.Contains(view, "Starts At (YYYY-MM-DD)") {
		t.Fatalf("View() should not show starts_at in preset batch mode\n%s", view)
	}
	if strings.Contains(view, "Fee USD") || strings.Contains(view, "Active (true/false)") || strings.Contains(view, "Ends At") {
		t.Fatalf("View() shows manual-only fields in preset mode\n%s", view)
	}
}

func TestOthersManualShowsManualFields(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	selectSubscriptionPreset(&m.subscriptionForm, "Others (Manual)")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Provider", "Plan Name", "Renewal Day", "Fee USD", "Active (true/false)", "Ends At (YYYY-MM-DD, required when inactive)"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing manual field %q\n%s", needle, view)
		}
	}
}

func TestSubscriptionFormPreservesEndsAtWhenActive(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	manager := &captureSubscriptionManager{}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: manager}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	selectSubscriptionPreset(&m.subscriptionForm, "Others (Manual)")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	setFieldValue(&m.subscriptionForm.fields, "provider", "custom-llm")
	setFieldValue(&m.subscriptionForm.fields, "plan_name", "Custom Plan")
	setFieldValue(&m.subscriptionForm.fields, "renewal_day", "5")
	setFieldValue(&m.subscriptionForm.fields, "fee_usd", "20")
	setFieldValue(&m.subscriptionForm.fields, "starts_at", "2026-04-05")
	setFieldValue(&m.subscriptionForm.fields, "ends_at", "2026-05-01")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)

	if len(manager.saved) != 1 {
		t.Fatalf("saved subscriptions = %d, want 1", len(manager.saved))
	}
	if manager.saved[0].EndsAt == nil || manager.saved[0].EndsAt.Format("2006-01-02") != "2026-05-01" {
		t.Fatalf("EndsAt = %v, want preserved 2026-05-01", manager.saved[0].EndsAt)
	}
}

func TestSubscriptionFormRequiresEndsAtWhenInactive(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: &captureSubscriptionManager{}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	selectSubscriptionPreset(&m.subscriptionForm, "Others (Manual)")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	setFieldValue(&m.subscriptionForm.fields, "starts_at", "2026-04-05")
	setFieldValue(&m.subscriptionForm.fields, "active", "false")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd != nil {
		t.Fatal("expected no submit command when inactive subscription is missing ends_at")
	}
	if got := m.subscriptionForm.errors["ends_at"]; got != "Inactive subscriptions require an ends_at date." {
		t.Fatalf("ends_at error = %q, want Inactive subscriptions require an ends_at date.", got)
	}
}

func TestSubscriptionFormDefaultsContributeToCurrentMonthDashboardTotal(t *testing.T) {
	fixedNow := time.Date(2026, time.April, 19, 14, 30, 0, 0, time.UTC)
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "subscription-defaults.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	period, err := domain.NewMonthlyPeriod(fixedNow)
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	form := newSubscriptionFormAt(fixedNow)
	selectSubscriptionPreset(&form, "ChatGPT Plus")
	confirmSubscriptionPreset(&form)
	manager := newTestSubscriptionManager(store)
	subscriptions, ok := form.parseSubscriptions(manager)
	if !ok {
		t.Fatalf("parseSubscriptions() errors = %#v", form.errors)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("len(parseSubscriptions()) = %d, want 1", len(subscriptions))
	}

	if err := manager.SaveSubscriptions(context.Background(), subscriptions); err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	query := service.NewDashboardQueryService(store, store, store, store)
	snapshot, err := query.QueryDashboard(context.Background(), service.DashboardQuery{Period: period, RecentSessionLimit: 8})
	if err != nil {
		t.Fatalf("QueryDashboard() error = %v", err)
	}

	if got, want := snapshot.Totals.SubscriptionSpendUSD, 20.0; got != want {
		t.Fatalf("SubscriptionSpendUSD = %v, want %v", got, want)
	}
	if got, want := snapshot.Totals.TotalSpendUSD, 20.0; got != want {
		t.Fatalf("TotalSpendUSD = %v, want %v", got, want)
	}
}

type staticLoader struct {
	data service.DashboardSnapshot
	err  error
}

func (s staticLoader) QueryDashboard(context.Context, service.DashboardQuery) (service.DashboardSnapshot, error) {
	return s.data, s.err
}

type captureWasteSummaryLoader struct {
	data    domain.WasteSummary
	err     error
	periods []domain.MonthlyPeriod
}

func (c *captureWasteSummaryLoader) QueryWasteSummary(_ context.Context, period domain.MonthlyPeriod) (domain.WasteSummary, error) {
	c.periods = append(c.periods, period)
	return c.data, c.err
}

type rejectingManualSaver struct{}

func (rejectingManualSaver) Save(context.Context, service.ManualAPIUsageEntryCommand) (domain.UsageEntry, error) {
	return domain.UsageEntry{}, &domain.ValidationError{Code: domain.ValidationCodeUnsupportedProvider, Field: "provider", Message: "manual API entries support only openai and anthropic"}
}

type captureSubscriptionManager struct {
	saved    []domain.Subscription
	disabled []string
}

type testSubscriptionManager struct {
	service *service.SubscriptionService
}

func (c *captureSubscriptionManager) SaveSubscriptions(_ context.Context, subscriptions []domain.Subscription) error {
	c.saved = append(c.saved, subscriptions...)
	return nil
}

func (c *captureSubscriptionManager) ListSubscriptions(context.Context, ports.SubscriptionFilter) ([]domain.Subscription, error) {
	return c.saved, nil
}

func (c *captureSubscriptionManager) DeleteSubscription(_ context.Context, subscriptionID string) error {
	c.disabled = append(c.disabled, subscriptionID)
	return nil
}

func (c *captureSubscriptionManager) DisableSubscription(_ context.Context, subscriptionID string, _ time.Time) error {
	return c.DeleteSubscription(context.Background(), subscriptionID)
}

func newTestSubscriptionManager(store *sqlite.Store) *testSubscriptionManager {
	return &testSubscriptionManager{service: service.NewSubscriptionService(store, store)}
}

func (m *testSubscriptionManager) SaveSubscriptions(ctx context.Context, subscriptions []domain.Subscription) error {
	return m.service.SaveSubscriptions(ctx, subscriptions)
}

func (m *testSubscriptionManager) ListSubscriptions(ctx context.Context, filter ports.SubscriptionFilter) ([]domain.Subscription, error) {
	return m.service.ListSubscriptions(ctx, filter)
}

func (m *testSubscriptionManager) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	return m.service.DeleteSubscription(ctx, subscriptionID)
}

func (m *testSubscriptionManager) DisableSubscription(ctx context.Context, subscriptionID string, _ time.Time) error {
	return m.service.DisableSubscription(ctx, subscriptionID, time.Time{})
}

func mustTUISubscription(t *testing.T, provider domain.ProviderName, planName string, fee float64, startsAt time.Time) domain.Subscription {
	t.Helper()
	planCode, err := domain.GenerateSubscriptionPlanCode(provider, planName)
	if err != nil {
		t.Fatalf("GenerateSubscriptionPlanCode() error = %v", err)
	}
	subscriptionID, err := domain.GenerateSubscriptionID(provider, planName, startsAt)
	if err != nil {
		t.Fatalf("GenerateSubscriptionID() error = %v", err)
	}
	subscription, err := domain.NewSubscription(domain.Subscription{
		SubscriptionID: subscriptionID,
		Provider:       provider,
		PlanCode:       planCode,
		PlanName:       planName,
		RenewalDay:     1,
		StartsAt:       startsAt,
		FeeUSD:         fee,
		IsActive:       true,
		CreatedAt:      startsAt,
		UpdatedAt:      startsAt,
	})
	if err != nil {
		t.Fatalf("NewSubscription() error = %v", err)
	}
	return subscription
}

func selectSubscriptionPreset(form *subscriptionFormModel, label string) {
	for i, option := range form.presetOptions {
		if option.Label == label {
			form.presetCursor = i
			form.focus = 0
			return
		}
	}
}

func confirmSubscriptionPreset(form *subscriptionFormModel) {
	form.togglePresetSelection(form.presetCursor)
}
