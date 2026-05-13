package gui

import (
	"context"

	"github.com/wailsapp/wails/v2"

	"llm-budget-tracker/internal/app"
	"llm-budget-tracker/internal/service"
)

func Run() error {
	notifier := NewWailsAlertNotifier()
	graph, err := app.Start(context.Background(), app.Options{Notifier: notifier})
	if err != nil {
		return err
	}
	defer graph.Close()

	app := NewApp(
		NewDashboardBinding(graph.DashboardQueryService),
		NewFormsBinding(graph.SettingsService, graph.SubscriptionService, graph.ManualEntryService, graph.MonthlyBudgetService, notifier, graph.Paths.DatabaseFile),
		NewSubscriptionLookupBinding(graph.SubscriptionQueryService, graph.SubscriptionService),
		NewGraphsBinding(service.NewGraphQueryService(graph.Store)),
		NewInsightsBinding(service.NewWasteSummaryService(graph.Store, graph.Store), graph.Store),
		NewAlertsBinding(graph.Store),
	)

	return wails.Run(app.options())
}
