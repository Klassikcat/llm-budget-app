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
}

func NewApp(dashboard *DashboardBinding, forms *FormsBinding, subscriptions *SubscriptionLookupBinding) *App {
	return &App{dashboard: dashboard, forms: forms, subscriptions: subscriptions}
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

	return &options.App{
		Title:     "LLM Budget Tracker",
		Width:     1240,
		Height:    820,
		MinWidth:  960,
		MinHeight: 640,
		OnStartup: a.startup,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		Bind: bindings,
	}
}
