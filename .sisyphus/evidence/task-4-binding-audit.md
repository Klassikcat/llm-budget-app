# Task 4 Wails Binding Audit

Date: 2026-04-30

## Registered Wails bindings

`internal/adapters/gui/app.go` registers these binding objects in `options.App.Bind` when they are non-nil:

1. `*gui.DashboardBinding`
2. `*gui.FormsBinding`
3. `*gui.SubscriptionLookupBinding`
4. `*gui.GraphsBinding`
5. `*gui.InsightsBinding`
6. `*gui.AlertsBinding`

No generated `internal/adapters/gui/frontend/wailsjs/` source is committed. The frontend wrapper layer therefore uses an injectable client and a dynamic default loader for Wails generated JS modules.

## Existing and added binding methods

| Binding | Method | Parameters | Return type | Wrapper | TUI parity status |
| --- | --- | --- | --- | --- | --- |
| `DashboardBinding` | `LoadDashboard` | `month string` (`YYYY-MM`, empty resolves current month) | `(DashboardResponse, error)` | `loadDashboard(month?: string): Promise<DashboardResponse>` | AVAILABLE. TUI dashboard calls `QueryDashboard`; GUI exposes the same dashboard snapshot category. Existing method signature preserved. |
| `FormsBinding` | `ListSubscriptionPresets` | none | `SubscriptionPresetsResponse` | `listSubscriptionPresets(): Promise<SubscriptionPresetsResponse>` | AVAILABLE for TUI subscription preset selector parity. |
| `FormsBinding` | `LoadSettings` | none | `SettingsFormResponse` | `loadSettings(): Promise<SettingsFormResponse>` | AVAILABLE as GUI-only settings support; no TUI equivalent required. |
| `FormsBinding` | `SaveSettings` | `SettingsFormInput` | `SettingsFormResponse` | `saveSettings(input): Promise<SettingsFormResponse>` | AVAILABLE as GUI-only settings support; no TUI equivalent required. |
| `FormsBinding` | `SaveProviderSecret` | `ProviderSecretInput` | `MutationResponse` | `saveProviderSecret(input): Promise<MutationResponse>` | AVAILABLE as GUI-only secret support; no TUI equivalent required. |
| `FormsBinding` | `DeleteProviderSecret` | `ProviderSecretDeleteInput` | `MutationResponse` | `deleteProviderSecret(input): Promise<MutationResponse>` | AVAILABLE as GUI-only secret support; no TUI equivalent required. |
| `FormsBinding` | `SaveSubscription` | `SubscriptionFormInput` | `SubscriptionMutationResponse` | `saveSubscription(input): Promise<SubscriptionMutationResponse>` | AVAILABLE for TUI subscription save parity. |
| `FormsBinding` | `SaveManualEntry` | `ManualEntryFormInput` | `ManualEntryMutationResponse` | `saveManualEntry(input): Promise<ManualEntryMutationResponse>` | AVAILABLE for TUI manual API entry save parity. |
| `FormsBinding` | `SaveBudget` | `BudgetFormInput` | `BudgetMutationResponse` | `saveBudget(input): Promise<BudgetMutationResponse>` | AVAILABLE for GUI budget upsert; TUI displays budgets but does not expose a budget edit form. |
| `FormsBinding` | `DispatchAlertNotification` | `AlertNotificationInput` | `NotificationDispatchResponse` | `dispatchAlertNotification(input): Promise<NotificationDispatchResponse>` | AVAILABLE as GUI desktop notification support; TUI lists alerts but does not dispatch desktop notifications from the model. |
| `SubscriptionLookupBinding` | `LoadSubscriptions` | none | `(SubscriptionListResponse, error)` | `loadSubscriptions(): Promise<SubscriptionListResponse>` | AVAILABLE for TUI subscription list load parity. |
| `SubscriptionLookupBinding` | `DeleteSubscription` | `subscriptionID string` | `(MutationResponse, error)` | `deleteSubscription(subscriptionId): Promise<MutationResponse>` | ADDED. Thin adapter over existing `SubscriptionService.DeleteSubscription`; no business logic changed. |
| `GraphsBinding` | `LoadGraphs` | `month string` (`YYYY-MM`, empty resolves current month) | `(GraphResponse, error)` | `loadGraphs(month?: string): Promise<GraphResponse>` | ADDED. Thin adapter over `service.GraphQueryService.QueryGraphs`, constructed from `app.Graph.Store` in `run.go`; exposes chart data for Task 8. |
| `InsightsBinding` | `LoadWasteSummary` | `month string` (`YYYY-MM`, empty resolves current month) | `(WasteSummaryResponse, error)` | `loadWasteSummary(month?: string): Promise<WasteSummaryResponse>` | ADDED. Thin adapter over `service.WasteSummaryService.QueryWasteSummary`, constructed from existing `app.Graph.Store`. |
| `InsightsBinding` | `LoadInsights` | `month string` (`YYYY-MM`, empty resolves current month) | `(InsightListResponse, error)` | `loadInsights(month?: string): Promise<InsightListResponse>` | ADDED. Thin adapter over existing `ports.InsightRepository.ListInsights` implemented by `app.Graph.Store`. |
| `AlertsBinding` | `LoadAlerts` | `month string` (`YYYY-MM`, empty resolves current month) | `(AlertListResponse, error)` | `loadAlerts(month?: string): Promise<AlertListResponse>` | ADDED. Thin adapter over existing `ports.AlertRepository.ListAlerts` implemented by `app.Graph.Store`. |

## TUI parity resolution

| TUI capability | Source | Existing service/repository | GUI binding status | Rationale |
| --- | --- | --- | --- | --- |
| Graph data load (`QueryGraphs`) | `internal/adapters/tui/model.go`, `graph_view.go` | `service.GraphQueryService` over `ports.UsageEntryRepository` | ADDED via `GraphsBinding.LoadGraphs` | `run.go` constructs `service.NewGraphQueryService(graph.Store)` from existing graph fields. |
| Waste summary load (`QueryWasteSummary`) | `internal/adapters/tui/model.go`, `insights_dashboard_view.go` | `service.WasteSummaryService` over usage and insight repositories | ADDED via `InsightsBinding.LoadWasteSummary` | `run.go` constructs `service.NewWasteSummaryService(graph.Store, graph.Store)` from existing graph fields. |
| Insight list/detail load (`ListInsights`) | `internal/adapters/tui/model.go`, `insights_logs_view.go`, `view.go` | `ports.InsightRepository` implemented by SQLite store | ADDED via `InsightsBinding.LoadInsights` | Direct thin repository read using `graph.Store`; DTO includes privacy-safe payload metadata only. |
| Alert list load (`ListAlerts`) | `internal/adapters/tui/model.go`, `view.go` | `ports.AlertRepository` implemented by SQLite store | ADDED via `AlertsBinding.LoadAlerts` | Direct thin repository read using `graph.Store`. |
| Subscription deletion (`DeleteSubscription`) | `internal/adapters/tui/model.go` | `service.SubscriptionService.DeleteSubscription` | ADDED via `SubscriptionLookupBinding.DeleteSubscription` | Existing constructor remains compatible via variadic manager dependency; `run.go` supplies `graph.SubscriptionService`. |

## Client-layer implementation notes

- `src/lib/bindings/client.ts` defines `WailsBindingClient`, `setBindingClient`, `resetBindingClient`, and the default Wails generated-module loader.
- `src/lib/bindings/index.ts` is now a pure re-export barrel to avoid wrapper-to-barrel import cycles.
- `src/lib/bindings/dashboard.ts`, `forms.ts`, `subscriptions.ts`, `graphs.ts`, `insights.ts`, and `alerts.ts` expose small typed wrappers over registered methods.
- Form and delete wrappers preserve typed return values and convert failed mutation payloads into `BindingMutationError` while retaining the original `MutationResponse`.
- Vitest tests inject a mock `WailsBindingClient` and do not require generated Wails JS files.

## Go binding changes

Go changes are limited to thin adapters and Wails registration:

- Added `internal/adapters/gui/graphs_binding.go` and tests.
- Added `internal/adapters/gui/insights_binding.go` and tests.
- Added `internal/adapters/gui/alerts_binding.go` and tests.
- Extended `internal/adapters/gui/subscriptions_binding.go` with `DeleteSubscription` without changing existing `LoadSubscriptions` signature.
- Updated `internal/adapters/gui/app.go` and `run.go` to register new bindings.
- No backend domain/service business rules were changed.
