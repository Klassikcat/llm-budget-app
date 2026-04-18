package gui

import (
	"context"

	"github.com/wailsapp/wails/v2"

	"llm-budget-tracker/internal/app"
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
		NewFormsBinding(graph.SettingsService, graph.SubscriptionService, graph.ManualEntryService, graph.MonthlyBudgetService, notifier),
	)

	return wails.Run(app.options())
}
