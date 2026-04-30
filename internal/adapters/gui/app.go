package gui

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

type App struct {
	dashboard     *DashboardBinding
	forms         *FormsBinding
	subscriptions *SubscriptionLookupBinding
	graphs        *GraphsBinding
	insights      *InsightsBinding
	alerts        *AlertsBinding
}

func NewApp(dashboard *DashboardBinding, forms *FormsBinding, subscriptions *SubscriptionLookupBinding, graphs *GraphsBinding, insights *InsightsBinding, alerts *AlertsBinding) *App {
	return &App{dashboard: dashboard, forms: forms, subscriptions: subscriptions, graphs: graphs, insights: insights, alerts: alerts}
}

func (a *App) startup(ctx context.Context) {
	if a == nil {
		return
	}
	if a.dashboard != nil {
		a.dashboard.startup(ctx)
	}
	if a.forms != nil {
		a.forms.startup(ctx)
	}
	if a.subscriptions != nil {
		a.subscriptions.startup(ctx)
	}
	if a.graphs != nil {
		a.graphs.startup(ctx)
	}
	if a.insights != nil {
		a.insights.startup(ctx)
	}
	if a.alerts != nil {
		a.alerts.startup(ctx)
	}
}

func (a *App) options() *options.App {
	bindings := []interface{}{}
	if a != nil && a.dashboard != nil {
		bindings = append(bindings, a.dashboard)
	}
	if a != nil && a.forms != nil {
		bindings = append(bindings, a.forms)
	}
	if a != nil && a.subscriptions != nil {
		bindings = append(bindings, a.subscriptions)
	}
	if a != nil && a.graphs != nil {
		bindings = append(bindings, a.graphs)
	}
	if a != nil && a.insights != nil {
		bindings = append(bindings, a.insights)
	}
	if a != nil && a.alerts != nil {
		bindings = append(bindings, a.alerts)
	}

	return &options.App{
		Title:     "LLM Budget Tracker",
		Width:     1240,
		Height:    820,
		MinWidth:  640,
		MinHeight: 480,
		OnStartup: a.startup,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		Bind: bindings,
	}
}
