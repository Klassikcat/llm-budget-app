package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

type dashboardLoader interface {
	QueryDashboard(ctx context.Context, query service.DashboardQuery) (service.DashboardSnapshot, error)
}

type graphLoader interface {
	QueryGraphs(ctx context.Context, query service.GraphQuery) (service.GraphSnapshot, error)
}

type manualEntrySaver interface {
	Save(ctx context.Context, cmd service.ManualAPIUsageEntryCommand) (domain.UsageEntry, error)
}

type subscriptionManager interface {
	SaveSubscriptions(ctx context.Context, subscriptions []domain.Subscription) error
	ListSubscriptions(ctx context.Context, filter ports.SubscriptionFilter) ([]domain.Subscription, error)
	DisableSubscription(ctx context.Context, subscriptionID string, disabledAt time.Time) error
	DeleteSubscription(ctx context.Context, subscriptionID string) error
}

type insightLister interface {
	ListInsights(ctx context.Context, period domain.MonthlyPeriod) ([]domain.Insight, error)
}

type alertLister interface {
	ListAlerts(ctx context.Context, filter ports.AlertFilter) ([]domain.AlertEvent, error)
}

type focusSection int

const (
	sectionOverview focusSection = iota
	sectionProviders
	sectionBudgets
	sectionRecentSessions
	sectionCount
)

type viewMode int

const (
	viewDashboard viewMode = iota
	viewManualEntryForm
	viewSubscriptionForm
	viewSubscriptionList
	viewInsightList
	viewInsightDetail
	viewGraphs
)

type graphTab int

const (
	graphTabModelTokenUsage graphTab = iota
	graphTabModelCost
	graphTabDailyTokenTrend
	graphTabModelTokenBreakdown
	graphTabCount
)

type dashboardLoadedMsg struct {
	data service.DashboardSnapshot
	err  error
}

type graphLoadedMsg struct {
	data service.GraphSnapshot
	err  error
}

type insightsLoadedMsg struct {
	insights []domain.Insight
	err      error
}

type alertsLoadedMsg struct {
	alerts []domain.AlertEvent
	err    error
}

type manualEntrySavedMsg struct {
	entry domain.UsageEntry
	err   error
}

type subscriptionSavedMsg struct {
	subscriptionID string
	err            error
}

type subscriptionsLoadedMsg struct {
	items []domain.Subscription
	err   error
}

type subscriptionDeletedMsg struct {
	subscriptionID string
	err            error
}

type modelDependencies struct {
	loader        dashboardLoader
	graphs        graphLoader
	manualEntries manualEntrySaver
	subscriptions subscriptionManager
	insights      insightLister
	alerts        alertLister
}

type formField struct {
	key   string
	label string
	input textinput.Model
}

type manualEntryFormModel struct {
	fields      []formField
	focus       int
	errors      map[string]string
	submitError string
	submitting  bool
}

type subscriptionFormModel struct {
	fields                []formField
	presetOptions         []subscriptionPresetOption
	presetCursor          int
	selectedPresetIndices map[int]bool
	now                   time.Time
	focus                 int
	errors                map[string]string
	submitError           string
	submitting            bool
}

type subscriptionPresetOption struct {
	Label    string
	Key      string
	Manual   bool
	Provider domain.ProviderName
	PlanName string
	FeeUSD   float64
	Renewal  int
}

const defaultSubscriptionRenewalDay = 1

type model struct {
	deps                  modelDependencies
	period                domain.MonthlyPeriod
	width                 int
	height                int
	loading               bool
	err                   error
	data                  service.DashboardSnapshot
	graphData             service.GraphSnapshot
	graphLoading          bool
	graphErr              error
	graphTab              graphTab
	alerts                []domain.AlertEvent
	insights              []domain.Insight
	focus                 focusSection
	mode                  viewMode
	viewport              viewport.Model
	ready                 bool
	recentLimit           int
	manualForm            manualEntryFormModel
	subscriptionForm      subscriptionFormModel
	subscriptionsList     []domain.Subscription
	subscriptionsErr      error
	subscriptionSelection int
	insightSelection      int
	selectedInsightID     string
	statusMessage         string
}

func newModel(deps modelDependencies, period domain.MonthlyPeriod) model {
	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.KeyMap{}

	manualForm := newManualEntryForm()
	manualForm.fields[0].input.SetValue("")

	return model{
		deps:             deps,
		period:           period,
		loading:          true,
		focus:            sectionOverview,
		mode:             viewDashboard,
		graphTab:         graphTabModelTokenUsage,
		viewport:         vp,
		recentLimit:      8,
		manualForm:       manualForm,
		subscriptionForm: newSubscriptionForm(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadAll())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.syncViewport()
		return m, nil
	case dashboardLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.data = msg.data
		}
		m.syncViewport()
		return m, nil
	case graphLoadedMsg:
		m.graphLoading = false
		m.graphErr = msg.err
		if msg.err == nil {
			m.graphData = msg.data
		}
		m.syncViewport()
		return m, nil
	case insightsLoadedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Insight refresh failed: %v", msg.err)
		} else {
			m.insights = msg.insights
			if len(m.insights) == 0 {
				m.insightSelection = 0
				m.selectedInsightID = ""
			} else {
				if m.insightSelection >= len(m.insights) {
					m.insightSelection = len(m.insights) - 1
				}
				if m.selectedInsightID == "" {
					m.selectedInsightID = m.insights[m.insightSelection].InsightID
				}
			}
		}
		m.syncViewport()
		return m, nil
	case alertsLoadedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Alert refresh failed: %v", msg.err)
		} else {
			m.alerts = msg.alerts
		}
		m.syncViewport()
		return m, nil
	case manualEntrySavedMsg:
		m.manualForm.submitting = false
		if msg.err != nil {
			m.manualForm.submitError = "Fix the highlighted fields and try again."
			applyValidationErrorToFields(msg.err, &m.manualForm.errors)
			m.statusMessage = "Manual entry was rejected by shared validation."
			m.syncViewport()
			return m, nil
		}
		m.manualForm = newManualEntryForm()
		m.manualForm.fields[0].input.SetValue("")
		m.mode = viewDashboard
		m.statusMessage = fmt.Sprintf("Saved manual API entry %s.", msg.entry.EntryID)
		m.loading = true
		m.syncViewport()
		return m, m.loadAll()
	case subscriptionSavedMsg:
		m.subscriptionForm.submitting = false
		if msg.err != nil {
			m.subscriptionForm.submitError = "Fix the highlighted fields and try again."
			applyValidationErrorToFields(msg.err, &m.subscriptionForm.errors)
			m.statusMessage = "Subscription form was rejected by shared validation."
			m.syncViewport()
			return m, nil
		}
		m.subscriptionForm = newSubscriptionForm()
		m.mode = viewDashboard
		if msg.subscriptionID == "batch" {
			m.statusMessage = "Saved selected subscription fee records."
		} else {
			m.statusMessage = "Saved subscription fee record."
		}
		m.loading = true
		m.syncViewport()
		return m, m.loadAll()
	case subscriptionsLoadedMsg:
		m.subscriptionsErr = msg.err
		if msg.err == nil {
			m.subscriptionsList = msg.items
			if m.subscriptionSelection >= len(m.subscriptionsList) {
				m.subscriptionSelection = max(0, len(m.subscriptionsList)-1)
			}
		}
		m.syncViewport()
		return m, nil
	case subscriptionDeletedMsg:
		if msg.err != nil {
			m.subscriptionsErr = msg.err
			m.statusMessage = fmt.Sprintf("Failed to delete subscription %s.", msg.subscriptionID)
			m.syncViewport()
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Deleted subscription %s.", msg.subscriptionID)
		m.syncViewport()
		return m, m.loadSubscriptions()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.mode {
		case viewDashboard:
			return m.updateDashboard(msg)
		case viewManualEntryForm:
			return m.updateManualEntryForm(msg)
		case viewSubscriptionForm:
			return m.updateSubscriptionForm(msg)
		case viewSubscriptionList:
			return m.updateSubscriptionList(msg)
		case viewInsightList:
			return m.updateInsightList(msg)
		case viewInsightDetail:
			return m.updateInsightDetail(msg)
		case viewGraphs:
			return m.updateGraphs(msg)
		}
	}

	if m.mode == viewManualEntryForm {
		form, cmd := m.manualForm.update(msg)
		m.manualForm = form
		m.syncViewport()
		return m, cmd
	}
	if m.mode == viewSubscriptionForm {
		form, cmd := m.subscriptionForm.update(msg)
		m.subscriptionForm = form
		m.syncViewport()
		return m, cmd
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "Loading dashboard shell..."
	}
	if usesViewportChrome(m.mode) {
		contentWidth := m.width - 2
		if contentWidth < 20 {
			contentWidth = 20
		}
		chrome := renderViewportChrome(&m, contentWidth)
		if chrome == "" {
			return m.viewport.View()
		}
		return chrome + "\n" + m.viewport.View()
	}
	return m.viewport.View()
}

func (m model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.loading = true
		m.err = nil
		m.statusMessage = "Refreshing dashboard, alerts, and insights..."
		m.syncViewport()
		return m, m.loadAll()
	case "tab", "right", "down", "j":
		m.focus = (m.focus + 1) % sectionCount
		m.syncViewport()
		return m, nil
	case "shift+tab", "left", "up", "k":
		m.focus = (m.focus + sectionCount - 1) % sectionCount
		m.syncViewport()
		return m, nil
	case "m":
		m.mode = viewManualEntryForm
		m.statusMessage = "Manual API entries use the shared validation and pricing catalog."
		m.syncViewport()
		return m, nil
	case "s":
		m.subscriptionForm = newSubscriptionForm()
		m.mode = viewSubscriptionForm
		m.statusMessage = "Subscriptions are upserted through the shared subscription service."
		m.syncViewport()
		return m, nil
	case "l":
		m.mode = viewSubscriptionList
		m.subscriptionsErr = nil
		m.statusMessage = "Loading saved subscriptions..."
		m.syncViewport()
		return m, m.loadSubscriptions()
	case "i":
		m.mode = viewInsightList
		m.syncViewport()
		return m, nil
	case "g":
		m.mode = viewGraphs
		m.graphLoading = true
		m.graphErr = nil
		m.statusMessage = "Loading graph data for this month..."
		m.syncViewport()
		return m, m.loadGraphs()
	}
	return m, nil
}

func (m model) updateSubscriptionList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.mode = viewDashboard
	case "up", "k":
		if m.subscriptionSelection > 0 {
			m.subscriptionSelection--
		}
	case "down", "j":
		if m.subscriptionSelection < len(m.subscriptionsList)-1 {
			m.subscriptionSelection++
		}
	case "r":
		m.subscriptionsErr = nil
		m.statusMessage = "Refreshing saved subscriptions..."
		m.syncViewport()
		return m, m.loadSubscriptions()
	case "d":
		if len(m.subscriptionsList) == 0 || m.deps.subscriptions == nil {
			m.syncViewport()
			return m, nil
		}
		target := m.subscriptionsList[m.subscriptionSelection]
		if isSettingsManagedSubscription(target.SubscriptionID) {
			m.statusMessage = fmt.Sprintf("Disable settings-managed subscription %s from Settings instead.", target.SubscriptionID)
			m.syncViewport()
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Deleting subscription %s...", target.SubscriptionID)
		m.syncViewport()
		return m, m.deleteSubscription(target.SubscriptionID)
	}
	m.syncViewport()
	return m, nil
}

func (m model) updateGraphs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.mode = viewDashboard
		m.syncViewport()
		return m, nil
	case "tab", "right", "l":
		m.graphTab = (m.graphTab + 1) % graphTabCount
	case "shift+tab", "left", "h":
		m.graphTab = (m.graphTab + graphTabCount - 1) % graphTabCount
	case "r":
		m.graphLoading = true
		m.graphErr = nil
		m.statusMessage = "Refreshing graph data..."
		m.syncViewport()
		return m, m.loadGraphs()
	}

	m.syncViewport()
	return m, nil
}

func (m model) updateInsightList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.mode = viewDashboard
	case "r":
		m.loading = true
		m.syncViewport()
		return m, m.loadAll()
	case "up", "k", "h":
		if len(m.insights) > 0 && m.insightSelection > 0 {
			m.insightSelection--
			m.selectedInsightID = m.insights[m.insightSelection].InsightID
		}
	case "down", "j", "l":
		if len(m.insights) > 0 && m.insightSelection < len(m.insights)-1 {
			m.insightSelection++
			m.selectedInsightID = m.insights[m.insightSelection].InsightID
		}
	case "enter":
		if len(m.insights) > 0 {
			m.selectedInsightID = m.insights[m.insightSelection].InsightID
			m.mode = viewInsightDetail
		}
	}
	m.syncViewport()
	return m, nil
}

func (m model) updateInsightDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.mode = viewInsightList
		m.syncViewport()
		return m, nil
	case "down", "j", "l":
		m.viewport.LineDown(1)
		return m, nil
	case "up", "k", "h":
		m.viewport.LineUp(1)
		return m, nil
	case "pgdown":
		m.viewport.ViewDown()
		return m, nil
	case "pgup":
		m.viewport.ViewUp()
		return m, nil
	}
	return m, nil
}

func (m model) updateManualEntryForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = viewDashboard
		m.manualForm.submitError = ""
		m.syncViewport()
		return m, nil
	}
	if msg.String() == "ctrl+s" || (msg.String() == "enter" && m.manualForm.focus == len(m.manualForm.fields)-1) {
		cmd, err := m.manualForm.submitCommand(m.deps.manualEntries)
		if err == nil {
			m.syncViewport()
			return m, cmd
		}
		m.syncViewport()
		return m, nil
	}

	form, cmd := m.manualForm.update(msg)
	m.manualForm = form
	m.syncViewport()
	return m, cmd
}

func (m model) updateSubscriptionForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.subscriptionForm = newSubscriptionForm()
		m.mode = viewDashboard
		m.subscriptionForm.submitError = ""
		m.syncViewport()
		return m, nil
	}
	if msg.String() == "ctrl+s" {
		cmd, err := m.subscriptionForm.submitCommand(m.deps.subscriptions)
		if err == nil {
			m.syncViewport()
			return m, cmd
		}
		m.syncViewport()
		return m, nil
	}

	form, cmd := m.subscriptionForm.update(msg)
	m.subscriptionForm = form
	m.syncViewport()
	return m, cmd
}

func (m model) loadDashboard() tea.Cmd {
	return func() tea.Msg {
		if m.deps.loader == nil {
			return dashboardLoadedMsg{err: errors.New("dashboard loader is not configured")}
		}
		data, err := m.deps.loader.QueryDashboard(context.Background(), service.DashboardQuery{Period: m.period, RecentSessionLimit: m.recentLimit})
		return dashboardLoadedMsg{data: data, err: err}
	}
}

func (m model) loadGraphs() tea.Cmd {
	return func() tea.Msg {
		if m.deps.graphs == nil {
			return graphLoadedMsg{err: errors.New("graph loader is not configured")}
		}
		data, err := m.deps.graphs.QueryGraphs(context.Background(), service.GraphQuery{Period: m.period})
		return graphLoadedMsg{data: data, err: err}
	}
}

func (m model) loadInsights() tea.Cmd {
	return func() tea.Msg {
		if m.deps.insights == nil {
			return insightsLoadedMsg{}
		}
		items, err := m.deps.insights.ListInsights(context.Background(), m.period)
		return insightsLoadedMsg{insights: items, err: err}
	}
}

func (m model) loadAlerts() tea.Cmd {
	return func() tea.Msg {
		if m.deps.alerts == nil {
			return alertsLoadedMsg{}
		}
		items, err := m.deps.alerts.ListAlerts(context.Background(), ports.AlertFilter{Period: &m.period})
		return alertsLoadedMsg{alerts: items, err: err}
	}
}

func (m model) loadSubscriptions() tea.Cmd {
	return func() tea.Msg {
		if m.deps.subscriptions == nil {
			return subscriptionsLoadedMsg{}
		}
		items, err := m.deps.subscriptions.ListSubscriptions(context.Background(), ports.SubscriptionFilter{})
		return subscriptionsLoadedMsg{items: items, err: err}
	}
}

func (m model) deleteSubscription(subscriptionID string) tea.Cmd {
	return func() tea.Msg {
		if m.deps.subscriptions == nil {
			return subscriptionDeletedMsg{subscriptionID: subscriptionID}
		}
		err := m.deps.subscriptions.DeleteSubscription(context.Background(), subscriptionID)
		return subscriptionDeletedMsg{subscriptionID: subscriptionID, err: err}
	}
}

func isSettingsManagedSubscription(subscriptionID string) bool {
	return strings.HasPrefix(strings.TrimSpace(subscriptionID), "settings-")
}

func usesViewportChrome(mode viewMode) bool {
	return mode == viewInsightList || mode == viewInsightDetail
}

func (m model) loadAll() tea.Cmd {
	return tea.Batch(m.loadDashboard(), m.loadInsights(), m.loadAlerts())
}

func (m *model) syncViewport() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	contentWidth := m.width - 2
	if contentWidth < 20 {
		contentWidth = 20
	}
	contentHeight := m.height - 2
	if contentHeight < 6 {
		contentHeight = 6
	}
	if usesViewportChrome(m.mode) {
		chromeLines := strings.Count(renderViewportChrome(m, contentWidth), "\n") + 1
		contentHeight -= chromeLines
		if contentHeight < 3 {
			contentHeight = 3
		}
	}

	previousOffset := m.viewport.YOffset
	m.viewport.Width = contentWidth
	m.viewport.Height = contentHeight
	if usesViewportChrome(m.mode) {
		m.viewport.SetContent(renderViewportBody(m, contentWidth))
	} else {
		m.viewport.SetContent(renderView(m, contentWidth))
	}
	if m.mode == viewInsightDetail {
		maxOffset := max(0, m.viewport.TotalLineCount()-m.viewport.Height)
		if previousOffset > maxOffset {
			previousOffset = maxOffset
		}
		m.viewport.SetYOffset(previousOffset)
		return
	}
	if m.mode == viewInsightList {
		m.viewport.SetYOffset(m.insightListViewportOffset(contentWidth, previousOffset))
		return
	}
	m.viewport.GotoTop()
}

func renderViewportBody(m *model, width int) string {
	switch m.mode {
	case viewInsightList:
		return renderInsightListBody(m, width)
	case viewInsightDetail:
		return renderInsightDetailBody(m, width)
	default:
		return renderView(m, width)
	}
}

func (m model) insightListViewportOffset(contentWidth int, currentOffset int) int {
	if len(m.insights) == 0 {
		return 0
	}

	selectedLine := insightListSelectionLine(m, contentWidth)
	if selectedLine < currentOffset {
		currentOffset = selectedLine
	}
	if selectedLine >= currentOffset+m.viewport.Height {
		currentOffset = selectedLine - m.viewport.Height + 1
	}

	maxOffset := max(0, m.viewport.TotalLineCount()-m.viewport.Height)
	if currentOffset > maxOffset {
		currentOffset = maxOffset
	}
	return max(0, currentOffset)
}

func insightListSelectionLine(m model, width int) int {
	_ = width
	return m.insightSelection
}

func newManualEntryForm() manualEntryFormModel {
	fields := []formField{
		newFormField("entry_id", "Entry ID (optional update key)", "manual-openai-2026-04"),
		newFormField("provider", "Provider", "openai"),
		newFormField("model_id", "Model ID", "gpt-4.1"),
		newFormField("occurred_at", "Occurred At (RFC3339 or 2006-01-02 15:04)", time.Now().UTC().Format(time.RFC3339)),
		newFormField("input_tokens", "Input Tokens", "1500"),
		newFormField("output_tokens", "Output Tokens", "250"),
		newFormField("cached_tokens", "Cached Tokens", "0"),
		newFormField("cache_write_tokens", "Cache Write Tokens", "0"),
		newFormField("project_name", "Project Name", "llm-budget-tracker"),
	}
	for i := 1; i < len(fields); i++ {
		fields[i].input.SetValue(fields[i].input.Placeholder)
	}
	return manualEntryFormModel{fields: fields, errors: map[string]string{}}
}

func newSubscriptionForm() subscriptionFormModel {
	return newSubscriptionFormAt(time.Now().UTC())
}

func newSubscriptionFormAt(now time.Time) subscriptionFormModel {
	defaultStartsAt := defaultSubscriptionStartsAt(now, defaultSubscriptionRenewalDay)
	presetOptions := make([]subscriptionPresetOption, 0, len(service.ListSubscriptionPresets())+1)
	for _, preset := range service.ListSubscriptionPresets() {
		presetOptions = append(presetOptions, subscriptionPresetOption{
			Label:    preset.PlanName,
			Key:      preset.Key,
			Provider: preset.Provider,
			PlanName: preset.PlanName,
			FeeUSD:   preset.DefaultFeeUSD,
			Renewal:  preset.DefaultRenewalDay,
		})
	}
	presetOptions = append(presetOptions, subscriptionPresetOption{Label: "Others (Manual)", Manual: true})
	fields := []formField{
		newFormField("provider", "Provider", ""),
		newFormField("plan_name", "Plan Name", ""),
		newFormField("renewal_day", "Renewal Day", ""),
		newFormField("starts_at", "Starts At (YYYY-MM-DD)", defaultStartsAt.Format("2006-01-02")),
		newFormField("fee_usd", "Fee USD", ""),
		newFormField("active", "Active (true/false)", "true"),
		newFormField("ends_at", "Ends At (YYYY-MM-DD, required when inactive)", ""),
	}
	for i := range fields {
		if fields[i].key == "ends_at" {
			continue
		}
		fields[i].input.SetValue(fields[i].input.Placeholder)
	}
	selectedPreset := map[int]bool{}
	return subscriptionFormModel{fields: fields, presetOptions: presetOptions, selectedPresetIndices: selectedPreset, now: now, errors: map[string]string{}}
}

func defaultSubscriptionStartsAt(now time.Time, renewalDay int) time.Time {
	now = now.UTC()
	lastDayOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if renewalDay < 1 {
		renewalDay = 1
	}
	if renewalDay > lastDayOfMonth {
		renewalDay = lastDayOfMonth
	}

	return time.Date(now.Year(), now.Month(), renewalDay, 0, 0, 0, 0, time.UTC)
}

func newFormField(key, label, placeholder string) formField {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 128
	input.Width = 48
	return formField{key: key, label: label, input: input}
}

func (f manualEntryFormModel) update(msg tea.Msg) (manualEntryFormModel, tea.Cmd) {
	fields, focus, cmd := updateFormFields(f.fields, f.focus, msg)
	f.fields = fields
	f.focus = focus
	f.errors = map[string]string{}
	f.submitError = ""
	return f, cmd
}

func (f subscriptionFormModel) update(msg tea.Msg) (subscriptionFormModel, tea.Cmd) {
	f.errors = map[string]string{}
	f.submitError = ""

	visible := f.visibleFieldIndices()
	maxFocus := len(visible)
	if maxFocus < 0 {
		maxFocus = 0
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab":
			if maxFocus == 0 {
				f.focus = 0
				f.syncFocus()
				return f, nil
			}
			f.focus = (f.focus + 1) % (maxFocus + 1)
			f.syncFocus()
			return f, nil
		case "shift+tab":
			if maxFocus == 0 {
				f.focus = 0
				f.syncFocus()
				return f, nil
			}
			f.focus = (f.focus + maxFocus) % (maxFocus + 1)
			f.syncFocus()
			return f, nil
		case "up", "k":
			if f.focus == 0 {
				f.presetCursor = (f.presetCursor + len(f.presetOptions) - 1) % len(f.presetOptions)
				f.focus = 0
				f.syncFocus()
				return f, nil
			}
			f.focus--
			f.syncFocus()
			return f, nil
		case "down", "j":
			if f.focus == 0 {
				f.presetCursor = (f.presetCursor + 1) % len(f.presetOptions)
				f.focus = 0
				f.syncFocus()
				return f, nil
			}
			f.focus = (f.focus + 1) % (maxFocus + 1)
			f.syncFocus()
			return f, nil
		case "left", "h":
			if f.focus == 0 {
				f.presetCursor = (f.presetCursor + len(f.presetOptions) - 1) % len(f.presetOptions)
				f.focus = 0
				f.syncFocus()
				return f, nil
			}
		case "right", "l":
			if f.focus == 0 {
				f.presetCursor = (f.presetCursor + 1) % len(f.presetOptions)
				f.focus = 0
				f.syncFocus()
				return f, nil
			}
		case "enter":
			if f.focus == 0 {
				f.togglePresetSelection(f.presetCursor)
				if f.manualSelected() && len(f.visibleFieldIndices()) > 0 {
					f.focus = 1
				} else {
					f.focus = 0
				}
				f.syncFocus()
				return f, nil
			}
		}
	}

	f.syncFocus()
	if f.focus == 0 {
		return f, nil
	}
	fieldIndex := visible[f.focus-1]
	updated, cmd := f.fields[fieldIndex].input.Update(msg)
	f.fields[fieldIndex].input = updated
	return f, cmd
}

func (f *subscriptionFormModel) syncFocus() {
	visible := f.visibleFieldIndices()
	for i := range f.fields {
		f.fields[i].input.Blur()
	}
	if f.focus <= 0 {
		return
	}
	fieldIndex := visible[f.focus-1]
	f.fields[fieldIndex].input.Focus()
}

func (f subscriptionFormModel) selectedPreset() subscriptionPresetOption {
	if len(f.presetOptions) == 0 {
		return subscriptionPresetOption{Label: "Others (Manual)", Manual: true}
	}
	for i := range f.presetOptions {
		if f.selectedPresetIndices[i] {
			return f.presetOptions[i]
		}
	}
	return f.presetOptions[0]
}

func (f subscriptionFormModel) selectedPresetCount() int {
	count := 0
	for i, option := range f.presetOptions {
		if option.Manual {
			continue
		}
		if f.selectedPresetIndices[i] {
			count++
		}
	}
	return count
}

func (f subscriptionFormModel) visibleFieldIndices() []int {
	if f.manualSelected() {
		return []int{0, 1, 2, 3, 4, 5, 6}
	}
	return []int{}
}

func (f subscriptionFormModel) manualSelected() bool {
	for i, option := range f.presetOptions {
		if option.Manual && f.selectedPresetIndices[i] {
			return true
		}
	}
	return false
}

func (f *subscriptionFormModel) togglePresetSelection(index int) {
	if index < 0 || index >= len(f.presetOptions) {
		return
	}
	option := f.presetOptions[index]
	if option.Manual {
		for key := range f.selectedPresetIndices {
			delete(f.selectedPresetIndices, key)
		}
		f.selectedPresetIndices[index] = true
		return
	}
	for i, preset := range f.presetOptions {
		if preset.Manual {
			delete(f.selectedPresetIndices, i)
		}
	}
	if f.selectedPresetIndices[index] {
		delete(f.selectedPresetIndices, index)
		return
	}
	f.selectedPresetIndices[index] = true
}

func updateFormFields(fields []formField, focus int, msg tea.Msg) ([]formField, int, tea.Cmd) {
	if len(fields) == 0 {
		return fields, 0, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			focus = (focus + 1) % len(fields)
			setFocusedField(fields, focus)
			return fields, focus, nil
		case "shift+tab", "up":
			focus = (focus + len(fields) - 1) % len(fields)
			setFocusedField(fields, focus)
			return fields, focus, nil
		}
	}

	setFocusedField(fields, focus)
	updated, cmd := fields[focus].input.Update(msg)
	fields[focus].input = updated
	return fields, focus, cmd
}

func setFocusedField(fields []formField, focus int) {
	for i := range fields {
		if i == focus {
			fields[i].input.Focus()
		} else {
			fields[i].input.Blur()
		}
	}
}

func (f *manualEntryFormModel) submitCommand(saver manualEntrySaver) (tea.Cmd, error) {
	f.errors = map[string]string{}
	f.submitError = ""
	if saver == nil {
		f.submitError = "Manual entry service is not configured."
		return nil, errors.New(f.submitError)
	}
	command, ok := f.parseCommand()
	if !ok {
		f.submitError = "Fix the highlighted fields and try again."
		return nil, errors.New(f.submitError)
	}
	f.submitting = true
	return func() tea.Msg {
		entry, err := saver.Save(context.Background(), command)
		return manualEntrySavedMsg{entry: entry, err: err}
	}, nil
}

func (f *subscriptionFormModel) submitCommand(manager subscriptionManager) (tea.Cmd, error) {
	f.errors = map[string]string{}
	f.submitError = ""
	if manager == nil {
		f.submitError = "Subscription service is not configured."
		return nil, errors.New(f.submitError)
	}
	if len(f.selectedPresetIndices) == 0 {
		f.submitError = "Choose at least one subscription option first."
		return nil, errors.New(f.submitError)
	}
	subscriptions, ok := f.parseSubscriptions(manager)
	if !ok {
		f.submitError = "Fix the highlighted fields and try again."
		return nil, errors.New(f.submitError)
	}
	f.submitting = true
	return func() tea.Msg {
		err := manager.SaveSubscriptions(context.Background(), subscriptions)
		subscriptionID := "batch"
		if len(subscriptions) == 1 {
			subscriptionID = subscriptions[0].SubscriptionID
		}
		return subscriptionSavedMsg{subscriptionID: subscriptionID, err: err}
	}, nil
}

func (f *manualEntryFormModel) parseCommand() (service.ManualAPIUsageEntryCommand, bool) {
	values := collectFormValues(f.fields)
	if strings.TrimSpace(values["provider"]) == "" {
		f.errors["provider"] = "Provider is required."
	}
	if strings.TrimSpace(values["model_id"]) == "" {
		f.errors["model_id"] = "Model ID is required."
	}
	occurredAt, okTime := parseFormTime(values["occurred_at"], "occurred_at", f.errors)
	inputTokens, okInput := parseFormInt(values["input_tokens"], "input_tokens", f.errors)
	outputTokens, okOutput := parseFormInt(values["output_tokens"], "output_tokens", f.errors)
	cachedTokens, okCached := parseFormInt(values["cached_tokens"], "cached_tokens", f.errors)
	cacheWriteTokens, okCacheWrite := parseFormInt(values["cache_write_tokens"], "cache_write_tokens", f.errors)
	if len(f.errors) > 0 || !okTime || !okInput || !okOutput || !okCached || !okCacheWrite {
		return service.ManualAPIUsageEntryCommand{}, false
	}

	return service.ManualAPIUsageEntryCommand{
		EntryID:          strings.TrimSpace(values["entry_id"]),
		Provider:         strings.TrimSpace(values["provider"]),
		ModelID:          strings.TrimSpace(values["model_id"]),
		OccurredAt:       occurredAt,
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		CachedTokens:     cachedTokens,
		CacheWriteTokens: cacheWriteTokens,
		ProjectName:      strings.TrimSpace(values["project_name"]),
	}, true
}

func (f *subscriptionFormModel) parseSubscriptions(manager subscriptionManager) ([]domain.Subscription, bool) {
	if f.manualSelected() {
		subscription, ok := f.parseManualSubscription(manager)
		if !ok {
			return nil, false
		}
		return []domain.Subscription{subscription}, true
	}
	return f.buildPresetSubscriptions(manager)
}

func (f *subscriptionFormModel) parseManualSubscription(manager subscriptionManager) (domain.Subscription, bool) {
	values := collectFormValues(f.fields)
	renewalDay, okRenewal := parseOptionalFormInt(values["renewal_day"], "renewal_day", f.errors)
	startsAt, okStartsAt := parseFormDate(values["starts_at"], "starts_at", f.errors)
	feeUSD, okFee := parseOptionalFormFloat(values["fee_usd"], "fee_usd", f.errors)
	active, okActive := parseFormBool(values["active"], "active", f.errors)
	var endsAt *time.Time
	if strings.TrimSpace(values["ends_at"]) != "" {
		parsedEndsAt, okEndsAt := parseFormDate(values["ends_at"], "ends_at", f.errors)
		if !okEndsAt {
			return domain.Subscription{}, false
		}
		endsAt = &parsedEndsAt
	}

	providerRaw := strings.TrimSpace(values["provider"])
	planName := strings.TrimSpace(values["plan_name"])
	if providerRaw == "" {
		f.errors["provider"] = "Provider is required for manual subscriptions."
	}
	if planName == "" {
		f.errors["plan_name"] = "Plan name is required for manual subscriptions."
	}
	if renewalDay == nil {
		f.errors["renewal_day"] = "A whole number is required for manual subscriptions."
	}
	if feeUSD == nil {
		f.errors["fee_usd"] = "A decimal amount is required for manual subscriptions."
	}
	if okActive && !active && endsAt == nil {
		f.errors["ends_at"] = "Inactive subscriptions require an ends_at date."
	}
	if len(f.errors) > 0 || !okRenewal || !okStartsAt || !okFee || !okActive {
		return domain.Subscription{}, false
	}

	provider, err := domain.NewProviderName(providerRaw)
	if err != nil {
		applyValidationErrorToFields(err, &f.errors)
		return domain.Subscription{}, false
	}
	planCode, err := domain.GenerateSubscriptionPlanCode(provider, planName)
	if err != nil {
		applyValidationErrorToFields(err, &f.errors)
		return domain.Subscription{}, false
	}
	subscriptionID, err := domain.GenerateSubscriptionID(provider, planName, startsAt)
	if err != nil {
		applyValidationErrorToFields(err, &f.errors)
		return domain.Subscription{}, false
	}

	createdAt := time.Time{}
	if manager != nil {
		existing, err := manager.ListSubscriptions(context.Background(), ports.SubscriptionFilter{SubscriptionID: subscriptionID})
		if err == nil && len(existing) > 0 {
			createdAt = existing[0].CreatedAt
		}
	}
	updatedAt := time.Time{}

	return domain.Subscription{
		SubscriptionID: subscriptionID,
		Provider:       provider,
		PlanCode:       planCode,
		PlanName:       planName,
		RenewalDay:     int(*renewalDay),
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		FeeUSD:         *feeUSD,
		IsActive:       active,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, true
}

func (f *subscriptionFormModel) buildPresetSubscriptions(manager subscriptionManager) ([]domain.Subscription, bool) {
	subscriptions := make([]domain.Subscription, 0, len(f.selectedPresetIndices))
	for i, preset := range f.presetOptions {
		if preset.Manual || !f.selectedPresetIndices[i] {
			continue
		}
		startsAt := defaultSubscriptionStartsAt(f.now, preset.Renewal)
		provider := preset.Provider
		planCode, err := domain.GenerateSubscriptionPlanCode(provider, preset.PlanName)
		if err != nil {
			applyValidationErrorToFields(err, &f.errors)
			return nil, false
		}
		subscriptionID, err := domain.GenerateSubscriptionID(provider, preset.PlanName, startsAt)
		if err != nil {
			applyValidationErrorToFields(err, &f.errors)
			return nil, false
		}
		createdAt := time.Time{}
		if manager != nil {
			existing, err := manager.ListSubscriptions(context.Background(), ports.SubscriptionFilter{SubscriptionID: subscriptionID})
			if err == nil && len(existing) > 0 {
				createdAt = existing[0].CreatedAt
			}
		}
		subscriptions = append(subscriptions, domain.Subscription{
			SubscriptionID: subscriptionID,
			Provider:       provider,
			PlanCode:       planCode,
			PlanName:       preset.PlanName,
			RenewalDay:     preset.Renewal,
			StartsAt:       startsAt,
			FeeUSD:         preset.FeeUSD,
			IsActive:       true,
			CreatedAt:      createdAt,
		})
	}
	if len(subscriptions) == 0 {
		f.submitError = "Choose at least one preset or Others (Manual)."
		return nil, false
	}
	return subscriptions, true
}

func resolveSubscriptionPresetInput(value string) (service.SubscriptionPreset, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return service.SubscriptionPreset{}, &domain.ValidationError{Code: domain.ValidationCodeRequired, Field: "preset_key", Message: "value is required"}
	}
	if preset, err := service.ResolveSubscriptionPreset(trimmed); err == nil {
		return preset, nil
	}
	for _, preset := range service.ListSubscriptionPresets() {
		if strings.EqualFold(trimmed, preset.PlanName) {
			return preset, nil
		}
		display := preset.Provider.String() + " " + preset.PlanName
		if strings.EqualFold(trimmed, display) {
			return preset, nil
		}
	}
	return service.SubscriptionPreset{}, &domain.ValidationError{Code: domain.ValidationCodeInvalidPreset, Field: "preset_key", Message: "choose one of the visible preset plans"}
}

func collectFormValues(fields []formField) map[string]string {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		values[field.key] = field.input.Value()
	}
	return values
}

func parseFormInt(value, field string, fieldErrors map[string]string) (int64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		fieldErrors[field] = "A numeric value is required."
		return 0, false
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		fieldErrors[field] = "Enter a whole number."
		return 0, false
	}
	return parsed, true
}

func parseFormFloat(value, field string, fieldErrors map[string]string) (float64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		fieldErrors[field] = "A decimal value is required."
		return 0, false
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		fieldErrors[field] = "Enter a decimal amount like 20 or 20.50."
		return 0, false
	}
	return parsed, true
}

func parseOptionalFormInt(value, field string, fieldErrors map[string]string) (*int64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, true
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		fieldErrors[field] = "Enter a whole number."
		return nil, false
	}
	return &parsed, true
}

func parseOptionalFormFloat(value, field string, fieldErrors map[string]string) (*float64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, true
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		fieldErrors[field] = "Enter a decimal amount like 20 or 20.50."
		return nil, false
	}
	return &parsed, true
}

func parseFormBool(value, field string, fieldErrors map[string]string) (bool, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		fieldErrors[field] = "Enter true or false."
		return false, false
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		fieldErrors[field] = "Enter true or false."
		return false, false
	}
	return parsed, true
}

func parseFormTime(value, field string, fieldErrors map[string]string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		fieldErrors[field] = "A timestamp is required."
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04", "2006-01-02"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC(), true
		}
	}
	fieldErrors[field] = "Use RFC3339 or 2006-01-02 15:04."
	return time.Time{}, false
}

func parseFormDate(value, field string, fieldErrors map[string]string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		fieldErrors[field] = "A date is required."
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		fieldErrors[field] = "Use YYYY-MM-DD."
		return time.Time{}, false
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), true
}

func applyValidationErrorToFields(err error, fields *map[string]string) {
	var validationErr *domain.ValidationError
	if !errors.As(err, &validationErr) {
		return
	}
	if *fields == nil {
		*fields = map[string]string{}
	}
	(*fields)[validationErr.Field] = validationErr.Message
}

func currentInsight(m model) (domain.Insight, bool) {
	if len(m.insights) == 0 {
		return domain.Insight{}, false
	}
	if m.selectedInsightID != "" {
		for _, insight := range m.insights {
			if insight.InsightID == m.selectedInsightID {
				return insight, true
			}
		}
	}
	if m.insightSelection < 0 || m.insightSelection >= len(m.insights) {
		return domain.Insight{}, false
	}
	return m.insights[m.insightSelection], true
}
