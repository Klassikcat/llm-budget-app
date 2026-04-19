package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	focusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
)

func renderView(m *model, width int) string {
	switch m.mode {
	case viewManualEntryForm:
		return renderManualEntryForm(m, width)
	case viewSubscriptionForm:
		return renderSubscriptionForm(m, width)
	case viewSubscriptionList:
		return renderSubscriptionList(m, width)
	case viewInsightList:
		return strings.Join([]string{renderViewportChrome(m, width), renderInsightListBody(m, width)}, "\n")
	case viewInsightDetail:
		return strings.Join([]string{renderViewportChrome(m, width), renderInsightDetailBody(m, width)}, "\n")
	case viewGraphs:
		return renderGraphs(m, width)
	default:
		return renderDashboard(m, width)
	}
}

func renderViewportChrome(m *model, width int) string {
	switch m.mode {
	case viewInsightList:
		return strings.Join([]string{renderHeader(m, width), titleStyle.Render("Insights")}, "\n")
	case viewInsightDetail:
		return strings.Join([]string{renderHeader(m, width), titleStyle.Render("Insight Detail")}, "\n")
	default:
		return ""
	}
}

func renderGraphs(m *model, width int) string {
	sections := []string{
		renderHeader(m, width),
		titleStyle.Render("Model Usage Graphs"),
		renderGraphTabs(m.graphTab, width),
	}

	if m.graphLoading {
		sections = append(sections,
			truncateLine("Loading graph data...", width),
			helpStyle.Render(truncateLine("The TUI is querying the shared graph aggregation service for this month.", width)),
		)
		return strings.Join(sections, "\n\n")
	}

	if m.graphErr != nil {
		sections = append(sections,
			errorStyle.Render("Graph data failed to load"),
			truncateLine(m.graphErr.Error(), width),
			helpStyle.Render(truncateLine("Press r to retry or Esc to return to the dashboard.", width)),
		)
		return strings.Join(sections, "\n\n")
	}

	sections = append(sections, renderGraphView(m, width))
	return strings.Join(sections, "\n\n")
}

func renderDashboard(m *model, width int) string {
	sections := []string{renderHeader(m, width)}

	if m.loading {
		sections = append(sections, "Loading dashboard data...", helpStyle.Render("The TUI is querying shared dashboard, insight, and alert services."))
		return strings.Join(sections, "\n\n")
	}

	if m.err != nil {
		sections = append(sections,
			errorStyle.Render("Dashboard failed to load"),
			truncateLine(m.err.Error(), width),
			helpStyle.Render("Press r to retry or q to quit."),
		)
		return strings.Join(sections, "\n\n")
	}

	if m.data.Empty {
		sections = append(sections,
			renderSectionTitle(sectionOverview, m.focus, "Overview"),
			truncateLine("No spend, budgets, or sessions are available for this month yet.", width),
			truncateLine("Add subscription fees, manual API entries, or CLI session imports to populate the dashboard.", width),
			helpStyle.Render("Navigation stays active: Tab/Shift+Tab changes sections, m and s open forms, l opens subscriptions, i opens insights, g opens graphs."),
		)
		return strings.Join(sections, "\n\n")
	}

	sections = append(sections,
		renderOverviewSection(m.data, m.focus, width),
		renderProvidersSection(m.data.ProviderSummaries, m.focus, width),
		renderBudgetsSection(m.data.Budgets, m.focus, width),
		renderRecentSessionsSection(m.data.RecentSessions, m.focus, width),
	)

	return strings.Join(sections, "\n\n")
}

func renderManualEntryForm(m *model, width int) string {
	sections := []string{
		renderHeader(m, width),
		renderFormFields("Manual API Entry", m.manualForm.fields, m.manualForm.focus, m.manualForm.errors, width),
	}
	if m.manualForm.submitError != "" {
		sections = append(sections, errorStyle.Render(truncateLine(m.manualForm.submitError, width)))
	}
	sections = append(sections,
		mutedStyle.Render(truncateLine("This form never stores free-form notes or prompt text.", width)),
		mutedStyle.Render(truncateLine("Submit with Ctrl+S or Enter on the last field. Esc returns to the dashboard.", width)),
	)
	return strings.Join(sections, "\n\n")
}

func renderSubscriptionForm(m *model, width int) string {
	sections := []string{
		renderHeader(m, width),
		titleStyle.Render("Subscription Fee Upsert"),
		renderSubscriptionPresetSelector(m.subscriptionForm, width),
	}
	if m.subscriptionForm.manualSelected() {
		sections = append(sections, renderFormFieldsSubset(m.subscriptionForm.fields, m.subscriptionForm.visibleFieldIndices(), m.subscriptionForm.focus, m.subscriptionForm.errors, width))
		sections = append(sections, mutedStyle.Render(truncateLine("Others (Manual) lets you enter a custom provider, plan, fee, and renewal day.", width)))
	} else {
		sections = append(sections, mutedStyle.Render(truncateLine(fmt.Sprintf("Selected presets: %d", m.subscriptionForm.selectedPresetCount()), width)))
		sections = append(sections, mutedStyle.Render(truncateLine("Preset mode saves selected subscriptions with their default fee, renewal day, active status, and default start date.", width)))
		sections = append(sections, mutedStyle.Render(truncateLine("Choose Others (Manual) if you need a custom provider, plan, fee, status, or end date.", width)))
	}
	if m.subscriptionForm.submitError != "" {
		sections = append(sections, errorStyle.Render(truncateLine(m.subscriptionForm.submitError, width)))
	}
	sections = append(sections, mutedStyle.Render(truncateLine("Move with ↑↓/←→, press Enter to toggle the highlighted option, then use Ctrl+S to save selected presets or the manual form.", width)))
	if m.subscriptionForm.manualSelected() {
		sections = append(sections, mutedStyle.Render(truncateLine("Inactive subscriptions require an ends_at date so monthly rollups stop cleanly.", width)))
	}
	return strings.Join(sections, "\n\n")
}

func renderSubscriptionList(m *model, width int) string {
	sections := []string{renderHeader(m, width), titleStyle.Render("Subscriptions")}
	if m.subscriptionsErr != nil {
		sections = append(sections,
			errorStyle.Render(truncateLine("Subscriptions failed to load", width)),
			truncateLine(m.subscriptionsErr.Error(), width),
			helpStyle.Render(truncateLine("Press r to retry or Esc to return.", width)),
		)
		return strings.Join(sections, "\n\n")
	}
	if len(m.subscriptionsList) == 0 {
		sections = append(sections,
			truncateLine("No saved subscriptions yet.", width),
			mutedStyle.Render(truncateLine("Use s to add a preset or manual subscription fee record.", width)),
		)
		return strings.Join(sections, "\n\n")
	}
	for i, subscription := range m.subscriptionsList {
		line := fmt.Sprintf("%s  %-10s  %-20s  %7s  renewal %2d  active %t", subscription.StartsAt.Format("2006-01-02"), subscription.Provider.String(), subscription.PlanName, formatUSD(subscription.FeeUSD), subscription.RenewalDay, subscription.IsActive)
		if i == m.subscriptionSelection {
			sections = append(sections, focusStyle.Render(truncateLine("> "+line, width)))
			continue
		}
		sections = append(sections, truncateLine("  "+line, width))
	}
	sections = append(sections, mutedStyle.Render(truncateLine("Use ↑↓ to choose a subscription, d to delete it, r to refresh, or Esc to return.", width)))
	return strings.Join(sections, "\n")
}

func renderInsightListBody(m *model, width int) string {
	lines := []string{}
	if len(m.insights) == 0 {
		lines = append(lines,
			truncateLine("No detector findings are stored for this month yet.", width),
			helpStyle.Render(truncateLine("Press Esc to return or r to reload.", width)),
		)
		return strings.Join(lines, "\n\n")
	}
	for i, insight := range m.insights {
		line := fmt.Sprintf("%s  %-28s  %s  %s", insight.DetectedAt.Local().Format(time.DateTime), string(insight.Category), string(insight.Severity), insight.InsightID)
		if i == m.insightSelection {
			lines = append(lines, focusStyle.Render(truncateLine("> "+line, width)))
		} else {
			lines = append(lines, truncateLine("  "+line, width))
		}
	}
	lines = append(lines, mutedStyle.Render(truncateLine("Enter opens privacy-safe detail metadata. Prompt or response text is never shown here.", width)))
	return strings.Join(lines, "\n")
}

func renderInsightDetailBody(m *model, width int) string {
	insight, ok := currentInsight(*m)
	lines := []string{}
	if !ok {
		lines = append(lines, truncateLine("No insight is selected.", width))
		return strings.Join(lines, "\n\n")
	}
	lines = append(lines,
		truncateLine("Insight ID: "+insight.InsightID, width),
		truncateLine("Category:   "+string(insight.Category), width),
		truncateLine("Severity:   "+string(insight.Severity), width),
		truncateLine("Detected:   "+insight.DetectedAt.Local().Format(time.DateTime), width),
		mutedStyle.Render(truncateLine("Privacy-safe metadata only. No prompt text, response text, or notes are stored.", width)),
	)
	lines = append(lines, renderInsightPayload(insight.Payload, width)...)
	return strings.Join(lines, "\n")
}

func renderInsightPayload(payload domain.InsightPayload, width int) []string {
	lines := []string{}
	if len(payload.SessionIDs) > 0 {
		lines = append(lines, titleStyle.Render("Session IDs"))
		for _, value := range payload.SessionIDs {
			lines = append(lines, truncateLine("- "+value, width))
		}
	}
	if len(payload.UsageEntryIDs) > 0 {
		lines = append(lines, titleStyle.Render("Usage Entry IDs"))
		for _, value := range payload.UsageEntryIDs {
			lines = append(lines, truncateLine("- "+value, width))
		}
	}
	if len(payload.Hashes) > 0 {
		lines = append(lines, titleStyle.Render("Hashes"))
		for _, value := range payload.Hashes {
			lines = append(lines, truncateLine(fmt.Sprintf("- %s: %s", value.Kind, value.Value), width))
		}
	}
	if len(payload.Counts) > 0 {
		lines = append(lines, titleStyle.Render("Counts"))
		for _, value := range payload.Counts {
			lines = append(lines, truncateLine(fmt.Sprintf("- %s: %d", value.Key, value.Value), width))
		}
	}
	if len(payload.Metrics) > 0 {
		lines = append(lines, titleStyle.Render("Metrics"))
		for _, value := range payload.Metrics {
			lines = append(lines, truncateLine(fmt.Sprintf("- %s: %.4f %s", value.Key, value.Value, value.Unit), width))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, truncateLine("No payload metrics were stored for this detector finding.", width))
	}
	return lines
}

func renderHeader(m *model, width int) string {
	periodLabel := m.period.StartAt.Format("January 2006")
	title := titleStyle.Render("LLM Budget Tracker")
	modeLabel := map[viewMode]string{
		viewDashboard:        "Dashboard",
		viewManualEntryForm:  "Manual API Entry Form",
		viewSubscriptionForm: "Subscription Fee Form",
		viewSubscriptionList: "Subscriptions",
		viewInsightList:      "Insight List",
		viewInsightDetail:    "Insight Detail",
		viewGraphs:           "Graphs",
	}[m.mode]
	status := truncateLine(fmt.Sprintf("%s • %s", modeLabel, periodLabel), width)
	banner := renderAlertBanner(m.alerts, width)
	help := helpStyle.Render(truncateLine(renderHelp(m.mode), width))
	lines := []string{title, status, banner}
	if strings.TrimSpace(m.statusMessage) != "" {
		lines = append(lines, mutedStyle.Render(truncateLine(m.statusMessage, width)))
	}
	lines = append(lines, help)
	return strings.Join(lines, "\n")
}

func renderOverviewSection(data service.DashboardSnapshot, focus focusSection, width int) string {
	lines := []string{
		renderSectionTitle(sectionOverview, focus, "Monthly Totals"),
		truncateLine(fmt.Sprintf("Monthly total:      %s", formatUSD(data.Totals.TotalSpendUSD)), width),
		truncateLine(fmt.Sprintf("Usage spend:        %s", formatUSD(data.Totals.VariableSpendUSD)), width),
		truncateLine(fmt.Sprintf("Subscription fees:  %s", formatUSD(data.Totals.SubscriptionSpendUSD)), width),
		truncateLine(fmt.Sprintf("Recent sessions:    %d", len(data.RecentSessions)), width),
	}
	return strings.Join(lines, "\n")
}

func renderProvidersSection(summaries []service.DashboardProviderSummary, focus focusSection, width int) string {
	lines := []string{renderSectionTitle(sectionProviders, focus, "Provider Summary")}
	if len(summaries) == 0 {
		lines = append(lines, truncateLine("No provider activity for this month.", width))
		return strings.Join(lines, "\n")
	}
	for _, summary := range summaries {
		line := fmt.Sprintf("%-12s total %8s  usage %8s  subs %8s  sessions %2d  entries %2d", summary.Provider.String(), formatUSD(summary.TotalSpendUSD), formatUSD(summary.VariableSpendUSD), formatUSD(summary.SubscriptionSpendUSD), summary.SessionCount, summary.UsageEntryCount)
		lines = append(lines, truncateLine(line, width))
	}
	return strings.Join(lines, "\n")
}

func renderBudgetsSection(summaries []service.DashboardBudgetSummary, focus focusSection, width int) string {
	lines := []string{renderSectionTitle(sectionBudgets, focus, "Budgets")}
	if len(summaries) == 0 {
		lines = append(lines, truncateLine("No monthly budgets configured for this period.", width))
		return strings.Join(lines, "\n")
	}
	for _, summary := range summaries {
		name := summary.Name
		if strings.TrimSpace(name) == "" {
			name = summary.BudgetID
		}
		status := "ok"
		if summary.BudgetOverrunActive {
			status = "forecast"
		}
		provider := summary.Provider.String()
		if provider == "" {
			provider = "all"
		}
		line := fmt.Sprintf("%s [%s] spend %s / %s  remaining %s  status %s", name, provider, formatUSD(summary.CurrentSpendUSD), formatUSD(summary.LimitUSD), formatUSD(summary.RemainingUSD), status)
		lines = append(lines, truncateLine(line, width))
	}
	return strings.Join(lines, "\n")
}

func renderRecentSessionsSection(sessions []service.DashboardRecentSession, focus focusSection, width int) string {
	lines := []string{renderSectionTitle(sectionRecentSessions, focus, "Recent Sessions")}
	if len(sessions) == 0 {
		lines = append(lines, truncateLine("No sessions recorded for this month.", width))
		return strings.Join(lines, "\n")
	}
	for _, session := range sessions {
		agent := strings.TrimSpace(session.AgentName)
		if agent == "" {
			agent = "unknown-agent"
		}
		project := strings.TrimSpace(session.ProjectName)
		if project == "" {
			project = "unknown-project"
		}
		model := session.ModelID
		if model == "" {
			model = "n/a"
		}
		line := fmt.Sprintf("%s  %s/%s  %s  %s  %d tokens", session.EndedAt.Local().Format(time.DateTime), session.Provider.String(), agent, formatUSD(session.TotalCostUSD), model, session.TotalTokens)
		lines = append(lines, truncateLine(line, width))
		lines = append(lines, mutedStyle.Render(truncateLine("  project: "+project+"  billing: "+string(session.BillingMode), width)))
	}
	return strings.Join(lines, "\n")
}

func renderFormFields(title string, fields []formField, focus int, fieldErrors map[string]string, width int) string {
	lines := []string{titleStyle.Render(title)}
	for i, field := range fields {
		label := titleStyle.Render(field.label)
		if i == focus {
			label = focusStyle.Render(field.label)
		}
		lines = append(lines, truncateLine(label, width), truncateLine("  "+field.input.View(), width))
		if err := strings.TrimSpace(fieldErrors[field.key]); err != "" {
			lines = append(lines, errorStyle.Render(truncateLine("  "+err, width)))
		}
	}
	return strings.Join(lines, "\n")
}

func renderAlertBanner(alerts []domain.AlertEvent, width int) string {
	if len(alerts) == 0 {
		return mutedStyle.Render(truncateLine("Alert status: no stored alerts for this month.", width))
	}
	latest := alerts[len(alerts)-1]
	message := fmt.Sprintf("Alert status: %s • %s", strings.ToUpper(string(latest.Severity)), describeAlert(latest))
	if latest.Severity == domain.AlertSeverityCritical {
		return errorStyle.Render(truncateLine(message, width))
	}
	return focusStyle.Render(truncateLine(message, width))
}

func describeAlert(alert domain.AlertEvent) string {
	switch alert.Kind {
	case domain.AlertKindBudgetThreshold:
		return fmt.Sprintf("budget %s crossed %.0f%% (%.2f / %.2f USD)", alert.BudgetID, alert.ThresholdPercent*100, alert.CurrentSpendUSD, alert.LimitUSD)
	case domain.AlertKindBudgetOverrun:
		return fmt.Sprintf("budget %s exceeded limit (%.2f / %.2f USD)", alert.BudgetID, alert.CurrentSpendUSD, alert.LimitUSD)
	case domain.AlertKindForecastOverrun:
		return fmt.Sprintf("forecast %s projects overrun", alert.ForecastID)
	case domain.AlertKindInsightDetected:
		return fmt.Sprintf("insight %s detected for %s", alert.InsightID, alert.DetectorCategory)
	default:
		return string(alert.Kind)
	}
}

func renderHelp(mode viewMode) string {
	switch mode {
	case viewManualEntryForm:
		return "Tab/Shift+Tab move fields • Ctrl+S saves • Esc returns • q quits"
	case viewSubscriptionForm:
		return "Tab/Shift+Tab move fields • ↑↓/←→ choose preset • Enter toggles • Ctrl+S saves • Esc returns • q quits"
	case viewSubscriptionList:
		return "↑↓ choose subscription • d deletes • r refreshes • Esc returns • q quits"
	case viewInsightList:
		return "↑↓ or h/j/k/l pick insight • Enter opens detail • Esc returns • r refresh • q quits"
	case viewInsightDetail:
		return "↑↓ or h/j/k/l scroll • PgUp/PgDn page • Esc returns • q quits"
	case viewGraphs:
		return "Tab/Shift+Tab or ←→/h/l cycle graph tabs • r refresh • Esc returns • q quits"
	default:
		return "Tab/Shift+Tab or ↑↓ move focus • m manual form • s subscription form • l subscriptions • i insights • g graphs • r refresh • q quit"
	}
}

func renderSubscriptionPresetSelector(form subscriptionFormModel, width int) string {
	lines := []string{titleStyle.Render("Choose Subscription")}
	for i, option := range form.presetOptions {
		cursor := "  "
		if i == form.presetCursor {
			cursor = "> "
		}
		selected := "[ ]"
		if form.selectedPresetIndices[i] {
			selected = "[v]"
		}
		line := option.Label
		if option.Manual {
			line = option.Label
		} else {
			line = fmt.Sprintf("%s — %s / renewal %d / %s", option.Label, formatUSD(option.FeeUSD), option.Renewal, option.Provider.String())
		}
		rendered := truncateLine(cursor+selected+" "+line, width)
		if form.focus == 0 && i == form.presetCursor {
			lines = append(lines, focusStyle.Render(rendered))
		} else {
			lines = append(lines, rendered)
		}
	}
	return strings.Join(lines, "\n")
}

func renderFormFieldsSubset(fields []formField, indices []int, focus int, fieldErrors map[string]string, width int) string {
	lines := []string{}
	for pos, idx := range indices {
		field := fields[idx]
		label := titleStyle.Render(field.label)
		if focus == pos+1 {
			label = focusStyle.Render(field.label)
		}
		lines = append(lines, truncateLine(label, width), truncateLine("  "+field.input.View(), width))
		if err := strings.TrimSpace(fieldErrors[field.key]); err != "" {
			lines = append(lines, errorStyle.Render(truncateLine("  "+err, width)))
		}
	}
	return strings.Join(lines, "\n")
}

func renderGraphTabs(active graphTab, width int) string {
	tabs := []graphTab{
		graphTabModelTokenUsage,
		graphTabModelCost,
		graphTabDailyTokenTrend,
		graphTabModelTokenBreakdown,
	}

	parts := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		label := graphTabLabel(tab)
		if tab == active {
			parts = append(parts, focusStyle.Render("["+label+"]"))
			continue
		}
		parts = append(parts, mutedStyle.Render(label))
	}

	return truncateLine(strings.Join(parts, "  •  "), width)
}

func renderGraphPlaceholder(m *model, width int) string {
	lines := []string{truncateLine("Active tab: "+graphTabLabel(m.graphTab), width)}

	switch m.graphTab {
	case graphTabModelTokenUsage:
		lines = append(lines,
			truncateLine(fmt.Sprintf("Models with token totals: %d", len(m.graphData.ModelTokenUsages)), width),
			mutedStyle.Render(truncateLine("Bar chart rendering is intentionally deferred to later graph tasks.", width)),
		)
	case graphTabModelCost:
		lines = append(lines,
			truncateLine(fmt.Sprintf("Models with cost totals: %d", len(m.graphData.ModelCosts)), width),
			mutedStyle.Render(truncateLine("Cost chart rendering is intentionally deferred to later graph tasks.", width)),
		)
	case graphTabDailyTokenTrend:
		lines = append(lines,
			truncateLine(fmt.Sprintf("Daily trend points: %d", len(m.graphData.DailyTokenTrends)), width),
			mutedStyle.Render(truncateLine("Trend chart rendering is intentionally deferred to later graph tasks.", width)),
		)
	case graphTabModelTokenBreakdown:
		lines = append(lines,
			truncateLine(fmt.Sprintf("Model token breakdown rows: %d", len(m.graphData.ModelTokenBreakdowns)), width),
			mutedStyle.Render(truncateLine("Breakdown visualization is intentionally deferred to later graph tasks.", width)),
		)
	}

	if len(m.graphData.ModelTokenUsages) == 0 && len(m.graphData.ModelCosts) == 0 && len(m.graphData.DailyTokenTrends) == 0 && len(m.graphData.ModelTokenBreakdowns) == 0 {
		lines = append(lines, mutedStyle.Render(truncateLine("No graph data is available for this month yet.", width)))
	}

	return strings.Join(lines, "\n")
}

func graphTabLabel(tab graphTab) string {
	switch tab {
	case graphTabModelTokenUsage:
		return "Model Token Usage"
	case graphTabModelCost:
		return "Model Cost"
	case graphTabDailyTokenTrend:
		return "Daily Token Trend"
	case graphTabModelTokenBreakdown:
		return "Token Breakdown"
	default:
		return "Unknown Graph"
	}
}

func renderSectionTitle(section, focused focusSection, label string) string {
	if section == focused {
		return focusStyle.Render("[" + label + "]")
	}
	return titleStyle.Render(label)
}

func formatUSD(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func truncateLine(value string, width int) string {
	if width <= 0 {
		return value
	}
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(value)
	return string(runes[:width-1]) + "…"
}
