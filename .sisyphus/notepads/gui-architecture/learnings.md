- 2026-04-30: GUI frontend is now a SvelteKit static SPA under internal/adapters/gui/frontend; Wails still embeds frontend/dist via app.go, so adapter-static must keep pages/assets output at dist.
- 2026-04-30: Browser QA showed Vite dev server requested /favicon.ico by default; placing favicon.ico in SvelteKit static/ makes it available in dev and copied to dist during build.

## Task 2: Design System Tokens
- Created a comprehensive design system using TailwindCSS and CSS variables.
- Defined over 30 CSS variables for colors, spacing, typography, border radius, and shadows.
- Implemented dark and light theme tokens inspired by Grafana/Datadog and the existing TUI colors.
- Configured `tailwind.config.ts` to expose these custom tokens.
- Added `postcss.config.js` to process TailwindCSS.
- Imported the global `tailwind.css` in `src/routes/+page.svelte` while keeping the minimal scaffold intact.
- Wrote focused tests for `tokens.css` and `tailwind.config.ts` to ensure all required tokens are defined and correctly mapped.

### Tailwind CSS Configuration in SvelteKit (ESM)
- **Issue**: Tailwind utility classes were not being generated and applied in the browser, even though the raw CSS with tokens was loading.
- **Root Cause**: The `postcss.config.js` was not being picked up correctly by Vite in an ESM context, and the global CSS was imported in `+page.svelte` instead of a root `+layout.svelte`.
- **Solution**: 
  1. Explicitly wired `tailwindcss` and `autoprefixer` in `vite.config.ts` under `css.postcss.plugins`.
  2. Renamed `postcss.config.js` to `postcss.config.cjs` and used `module.exports` to ensure compatibility.
  3. Created `src/routes/+layout.svelte` and moved the global `tailwind.css` import there, ensuring it applies globally across the app.

- 2026-04-30: GUI frontend domain types live in `internal/adapters/gui/frontend/src/lib/types`; raw domain mirrors use snake_case JSON-style fields from Go validation/domain names, while Wails form and notification DTOs keep existing camelCase binding tags from `internal/adapters/gui/forms_types.go`.

- 2026-04-30: GUI theme state now lives in `internal/adapters/gui/frontend/src/lib/stores/theme.ts`; it persists `dark`/`light` with `llm-budget-tracker-theme`, applies exactly one `html.dark`/`html.light` class, falls back from valid localStorage to OS preference to dark default, and remains safe when browser globals are unavailable.

## Layout Shell Implementation
- Created a responsive layout shell with a collapsible sidebar and a header.
- Used Lucide Svelte icons for navigation items.
- Integrated the existing theme store for dark/light mode toggling.
- Ensured the layout is accessible with proper ARIA labels.
- Added Playwright tests to verify navigation, sidebar collapse/expand, and header functionality.
- Used `page.locator('aside[aria-label="Sidebar Navigation"]').locator('text=Dashboard')` to specifically target the sidebar text in tests, avoiding conflicts with the header title.

## Svelte 5 Testing Library Setup
- When testing Svelte 5 components with Vitest, `@testing-library/svelte` requires the `svelteTesting()` Vite plugin from `@testing-library/svelte/vite`.
- Without this plugin, Vitest may attempt to compile components for SSR, resulting in `lifecycle_function_unavailable` errors (`mount(...) is not available on the server`).
- Additionally, to use `jest-dom` matchers (like `toBeInTheDocument`) without TypeScript errors, include `/// <reference types="@testing-library/jest-dom" />` at the top of the test files.

## Test Cleanup
- Removed unused `page` and `get` imports from `Sidebar.test.ts` to resolve LSP warnings and keep the test file clean.

## Task 7 Test Infrastructure
- 2026-04-30: Playwright E2E can run against the SvelteKit Vite server only by using `webServer.command` with `npm run dev -- --port 5173`; no Wails desktop process is required.
- 2026-04-30: Vitest v8 coverage must use `@vitest/coverage-v8` matching the installed Vitest major/minor version; this project resolved Vitest to 3.2.4, so coverage uses `@vitest/coverage-v8@3.2.4`.
- 2026-04-30: Keep Playwright passing-run output non-committable by using the list reporter only and ignoring frontend `playwright-report/`, `test-results/`, `coverage/`, plus root `.playwright-mcp/` QA artifacts.
- 2026-04-30: Task 4 binding wrappers live in `src/lib/bindings`; use `setBindingClient` to inject Vitest mocks, and keep the default Wails client as a dynamic loader because generated `frontend/wailsjs/` is not committed.
- 2026-04-30: Avoid binding wrapper import cycles by keeping client state in `src/lib/bindings/client.ts` and using `index.ts` only as a pure re-export barrel.
- 2026-04-30: Graph, waste summary, insight, alert, and subscription delete GUI bindings can be added safely as thin adapters from existing `app.Graph.Store` and `SubscriptionService`; no domain/service rule changes are needed.

## Task 8: ECharts Components
- 2026-04-30: ECharts can be cleanly integrated into Svelte 5 using a Svelte action (`use:chartAction`). This avoids the need for heavy wrapper libraries and provides direct access to the ECharts instance for resizing and theme updates.
- 2026-04-30: To handle theme changes dynamically without page reloads, the chart action subscribes to the `theme` store and re-initializes the ECharts instance when the theme changes.
- 2026-04-30: `ResizeObserver` is used within the chart action to automatically call `chart.resize()` when the container size changes, ensuring responsive charts.
- 2026-04-30: When testing ECharts components with Vitest and jsdom, `ResizeObserver` and `echarts` must be mocked because jsdom does not fully support the required DOM APIs (like Canvas).

## Task 9: Data Table Component
- Implemented a generic `DataTable` component in Svelte 5 using existing design tokens (`bg-card`, `border-panel-border`, `text-text`, etc.).
- Used `<script module lang="ts">` to export the `Column` interface, as Svelte 5 does not allow exporting interfaces from the main `<script>` block.
- Replaced `createEventDispatcher` with callback props (`onSort`, `onRowClick`) to align with Svelte 5 best practices and avoid `$on` usage in tests.
- Created specialized cell components (`StatusBadge`, `CurrencyCell`, `TokenCell`, `DateCell`) that handle formatting without business logic.
- Used `Intl.NumberFormat` and `Intl.DateTimeFormat` for robust and localized formatting in cell components.

## Task 10: Form Components
- Implemented form primitives (`TextInput`, `NumberInput`, `SelectInput`, `Toggle`, `DatePicker`, `Form`, `FormField`) using Svelte 5 runes (`$props`, `$bindable`).
- Used callback props (`onchange`, `oninput`, `onblur`, `onfocus`, `onsubmit`) instead of `createEventDispatcher` to align with Svelte 5 idioms.
- In Svelte 5, component instances do not expose their props as properties by default. When testing `bind:value` with `@testing-library/svelte`, it's better to assert on the underlying DOM element's value (e.g., `input.value`) rather than `component.value`.
- For `SelectInput`, when `value` is empty and `required` is false, an empty hidden option is rendered to allow a blank initial state. In tests, `getAllByRole('option', { hidden: true })` is needed to select this hidden option.
- For `DatePicker`, `type="date"` inputs might not have a specific ARIA role in jsdom, so querying by ID or tag name is more reliable in tests.

### Task 11: Card/Panel Components
- Implemented `Panel`, `StatCard`, `SparklineCard`, and `AlertCard` components using Svelte 5 `$props` and `Snippet` patterns.
- Used existing design tokens and Tailwind classes for styling, ensuring consistency with the Grafana/Datadog-style theme.
- Avoided business logic and external chart dependencies in `SparklineCard` by using a simple SVG path generation based on the provided data array.
- Ensured all components are fully tested using Vitest and `@testing-library/svelte`, asserting real render behavior.

### Task 12: Svelte Stores
- 2026-04-30: Budget, usage, subscription, and waste Svelte stores live in `src/lib/stores` and adapt Task 4 binding wrapper DTOs directly into explicit `{ data, loading, error }` state.
- 2026-04-30: Store `refresh()` methods reuse `load()` semantics and remember the active month where relevant; mutation-triggered refresh is implemented only around existing wrappers (`saveBudget`, `saveManualEntry`, `saveSubscription`, `deleteSubscription`) with no timers or polling.
- 2026-04-30: Store tests inject mocks through `setBindingClient`, then verify success, pending loading state, refresh, retained-data errors, and mutation refresh behavior.

## Task 13: Notification Service & Components
- Implemented an in-memory notification store using Svelte 5 `writable` and `get` to manage notification state and prevent duplicate alerts.
- Used a `Set` to track `sentAlertKeys` (e.g., `${budgetId}-${thresholdPercent}`) to ensure the same threshold for the same budget is only emitted once per app session.
- Created `NotificationToast` and `NotificationCenter` components following the dense utility UI and design token conventions (e.g., `bg-panel`, `border-panel-border`, `text-status-warning`).
- Kept the notification service decoupled from the DB and polling, relying on explicit calls to `checkBudgetThresholds` with dashboard data.

## Dashboard Implementation
- Used Svelte 5 `$props` and runes for the dashboard page.
- Created a dedicated `dashboardStore` to load `loadDashboard`, `loadGraphs`, and `loadWasteSummary` concurrently.
- Used `dailyTokenTrends` from `GraphResponse` for the daily trend line chart since there is no `dailyCostTrends` available.
- Used `providerSummaries` from `DashboardResponse` for the provider cost bar chart.
- Handled empty states gracefully by checking `dashboard.empty` and displaying a helpful message.
- Used existing UI components (`Panel`, `StatCard`, `BarChart`, `LineChart`, `DataTable`, `CurrencyCell`, `DateCell`, `TokenCell`, `StatusBadge`) to maintain consistency.
- Used Playwright to mock the Wails IPC calls and capture screenshots for both empty and populated states.

### Task 15: Usage Tracking Screen
- Svelte 5 `bind:value` on `type="number"` inputs with `min={0}` can cause validation issues in tests if negative values are simulated via `fireEvent.input`. Removing `min={0}` allows Zod to handle the validation and show the correct error message.
- `DashboardRecentSession` from the dashboard binding can be reused for the usage history table, even if it only provides `totalTokens` instead of separate input/output tokens.
- `notificationStore.addNotification` is a simple way to show toast notifications without needing a global toast container component, as long as `NotificationCenter` is rendered somewhere (or if we just want to add it to the store for now).

### Task 16: Subscription Management
- **ID Generation**: The frontend needs to generate the `subscriptionId` for deletion using the same logic as the backend (`provider-planSlug-YYYY-MM-DD`) because `SubscriptionState` does not include an ID field.
- **Form Components**: `NumberInput` expects `min` and `max` as numbers, not strings. `Toggle` does not accept a `label` prop, so the label must be rendered alongside it.
- **StatusBadge**: The `StatusBadge` component expects the `status` prop to be the text to display, and it determines the color based on the text (case-insensitive).

## Task 17: Budget Management Screen
- **Date**: 2026-04-30
- **Pattern**: Used the existing `budget` store which conveniently loads the `DashboardResponse` containing both budget data and recent sessions/provider summaries. This allowed us to implement the budget form and monitoring panels (including charts) using a single store.
- **Testing**: When testing components that use `ResizeObserver` (like ECharts wrappers), it's necessary to mock `global.ResizeObserver` in the test setup since it's not available in the jsdom environment.
- **E2E Testing**: Wails bindings are loaded as JS files, so intercepting them in Playwright requires mocking the JS response body rather than just checking `request.postData()`. We used a global window variable to simulate state changes between `SaveBudget` and `LoadDashboard` calls.

## Task 17: Budget Management Screen (Verification Fix)
- **Date**: 2026-04-30
- **Pattern**: Replaced inline `style="width: ..."` on the progress bar with an SVG `<rect>` using `width` attribute and `fill-current` with Tailwind text color classes (`text-status-success`, etc.). This avoids inline styles while allowing dynamic width updates.
- **Validation**: Updated `criticalThresholdPercent` max to 100 to allow the example value of 100.
- **Defaults**: Initialized form threshold fields to stable defaults (80 and 100) instead of inferring from `triggeredThresholdPercents`, which represents triggered state rather than configuration.
- **Clean Code**: Removed all HTML and obvious line comments from `+page.svelte`, `page.test.ts`, and `budgets.spec.ts` to adhere to the self-explanatory code rule.

## Task 18: Insights Dashboard
- **Date**: 2026-04-30
- **Pattern**: Used `DataTable` with custom `componentProps` to render `StatusBadge` and `DateCell` components inside columns. Since `DataTable` is written in Svelte 4 and expects `Component<Record<string, unknown>>`, we had to cast Svelte 5 components using `as unknown as Component<Record<string, unknown>>` to satisfy TypeScript.
- **Pattern**: Implemented a simple modal overlay using Tailwind classes (`fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm`) to show insight details when a row is clicked.
- **Testing**: When mocking Wails bindings for Playwright tests, it's important to mock the entire module in a single `page.route` call (e.g., `**/wailsjs/go/gui/InsightsBinding*`) rather than separate routes for each method, because Wails imports the whole module at once.

## Task 18: Insights Dashboard (Verification Fix)
- **Date**: 2026-04-30
- **Pattern**: Fixed `StatusBadge` severity mapping by passing `High (Danger)`, `Medium (Warning)`, and `Low (Success)` to the `status` prop. This ensures the badge gets the correct styling (danger/warning/success) while preserving readable display text.
- **Pattern**: Updated `StatCard` labels to exactly match the plan requirements (`Waste Headline`, `Waste %`, `Projected Waste`, `Weekly Waste`).
- **Pattern**: Removed all HTML comments and obvious explanatory line comments from the route and test files to adhere to quality standards.

## Task 18: Insights Dashboard (Headline Fix)
- **Date**: 2026-04-30
- **Pattern**: Fixed `Waste Headline` StatCard to display the largest waste cause (e.g., `Context Avalanche`) instead of the total waste cost. Derived `wasteHeadline` from `summary.topCauses[0].category` using the existing `formatCategoryName` helper, with a fallback to `No Waste Detected`.

## Task 19: Graphs Screen
- **Time Range Selector**: The backend `LoadGraphs` binding only accepts a `month` string (e.g., "2026-04") and returns data for that specific month. To implement the "7 days", "30 days", and "All" time range selector, we filter the `dailyTokenTrends` array on the frontend. Since the backend only returns data for the current month, "30 days" and "All" effectively show the same data (the whole month), while "7 days" shows the last 7 days of the month.
- **Svelte 5 Mocking**: When mocking Svelte 5 components in Vitest, we cannot use classes with `constructor` and `$$prop_def`. Instead, we must use functions that return an object with `update` and `destroy` methods, and append the mock DOM elements to the `node` argument (which is the anchor node).

## Task 20: Settings Screen
- Implemented settings screen using existing `Panel` and `Toggle` components.
- Used `window.matchMedia` mock in Vitest to prevent errors when testing theme store.
- Used `page.addInitScript` in Playwright to clear localStorage and set initial theme state to ensure deterministic test behavior.
- Displayed read-only DB path and static version info as requested, avoiding unnecessary backend bindings.
- 2026-04-30: Task 21 full integration keeps routes/stores on the injectable Wails binding wrapper layer; settings now loads/saves through FormsBinding, mutation paths refresh via stores without polling, and Playwright route mocks persist cross-screen state in localStorage for deterministic Wails module tests.

## Task 22: Playwright E2E Suite
- 2026-04-30: Task 22 E2E coverage lives in `internal/adapters/gui/frontend/tests/task-22-e2e.spec.ts` and uses dynamic Wails JS route mocks for Dashboard, Forms, SubscriptionLookup, Insights, Alerts, and Graphs bindings.
- 2026-04-30: Settings notification toggles can be clicked deterministically in Playwright with `locator('input#notification-toggle').evaluate((element: HTMLInputElement) => element.click())` because the notification center overlay may intercept pointer clicks.
- 2026-04-30: Avoid fixed Playwright sleeps by asserting loaded UI state before screenshots; the frontend suite now has no `waitForTimeout` usage in `tests/*.spec.ts`.

## Task 23: Linux/macOS Wails Build Verification
- 2026-04-30: Wails v2 build generation expects a Go entrypoint at the project root; adding a root `main.go` that delegates to `internal/adapters/gui.Run()` lets `wails build -platform linux/amd64` generate bindings and package the existing GUI without moving frontend config.
- 2026-04-30: Linux Wails builds on this host require WebKitGTK 4.1 tags (`-tags webkit2_41`), while `wails.json` must keep `frontend:dir` at `internal/adapters/gui/frontend` and `assetdir` at `dist`.
- 2026-04-30: SQLite WAL behavior is verified with two real repository stores against one database file: an open write transaction does not block a concurrent read of the committed snapshot.

## Final QA Repair Learnings
- 2026-04-30: Wails runtime integration should use dynamic imports plus `EventsOn`/`llmbudget:desktop-notification` dispatching so browser tests and Wails desktop both stay safe without committing generated `wailsjs` files.
- 2026-04-30: GUI state DTOs should expose backend identity/path fields directly (`subscriptionId`, `databasePath`) instead of asking the frontend to reconstruct backend identifiers or hardcode local paths.
- 2026-04-30: Global overlays like `NotificationCenter` need `pointer-events-none` on empty wrappers and `pointer-events-auto` only on interactive children to avoid intercepting route-level Playwright clicks.
- 2026-04-30: When dashboard invalidation is triggered by mutation stores, Playwright specs for those routes must also mock secondary dashboard dependencies such as `GraphsBinding` and `InsightsBinding` modules.
