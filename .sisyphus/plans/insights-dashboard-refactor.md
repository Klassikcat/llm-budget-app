# Insights Tab Refactor — Dashboard as Default Sub-Tab (TUI)

## TL;DR

> **Quick Summary**: Refactor the TUI Insights view (`viewInsightList` + `viewInsightDetail`) into a tabbed container with `[Dashboard]` (new default) + `[Logs]` (existing insight list) sub-tabs. Add a new `WasteSummaryService` that attributes cost per detector, powering 6 dashboard visualizations. Scope strictly 1 month. TDD throughout.
>
> **Deliverables**:
> - New `internal/domain/waste_summary.go` — domain types for waste summary
> - New `internal/service/waste_summary.go` + tests — per-detector cost attribution (no double-counting)
> - Refactored `internal/adapters/tui/model.go` — `insightTab` state machine (Dashboard / Logs sub-tabs)
> - New `internal/adapters/tui/insights_dashboard_view.go` — Dashboard rendering (6 widgets)
> - Refactored `internal/adapters/tui/view.go` — tab chrome renderer (extends existing Graphs pattern)
> - Wired dependencies in `run.go` and TUI model constructor
> - Comprehensive tests: service unit, state-machine, rendering goldens
> - tmux-based QA scenarios with evidence capture
>
> **Estimated Effort**: Medium (8 implementation tasks + 1 wiring + 4-task final verification wave)
> **Parallel Execution**: YES — 3 waves (foundation, implementation, integration) + final review wave
> **Critical Path**: T1 (domain types) → T2 (service impl) → T7 (wire service) → T8 (end-to-end TUI) → FINAL
> **Max Concurrent**: 5 tasks (Wave 2)

---

## Context

### Original Request (user's exact words)
> Target: insights tab
>
> Insights 변경 내용:
> - 기존 insights는 insights 내부 [logs]나 [details]등 적당한 탭으로 이동
> - [dashboard]라는 새로운 탭이 기본
>
> Dashboards 주요 내용:
> - 각 token 낭비 주요 원인을 그래프화 (시각화 방법은 선택)
> - token으로 낭비된 금액을 일별, 주별, 월별로 확인
> - scope: 한달 범위
> - 그 외 필요하다고 네가 판단하는 모든 기능 추가

### Interview Summary

**Key Decisions**:
- **UI Scope**: TUI only (`internal/adapters/tui/`). Wails GUI untouched.
- **Refactor Interpretation (A)**: The Insights view (currently 2 modes: `viewInsightList` + `viewInsightDetail`) becomes a tabbed container with `[Dashboard]` (default) + `[Logs]` sub-tabs. `viewInsightDetail` stays as a modal drill-down reachable only from the Logs tab. Top-level `viewDashboard` and `viewGraphs` modes remain UNTOUCHED.
- **Waste cost computation**: NEW `WasteSummaryService` added under `internal/service/` with deterministic per-detector attribution (no double-counting).
- **Time unit display**: Daily = 30-day line chart via `ntcharts`. Weekly + Monthly = summary cards (no chart).
- **Feature set**: "Essential + Trend" = 6 visualizations (see Dashboard Widgets below).
- **Test strategy**: TDD (RED → GREEN → REFACTOR) with Go `testing` package. tmux-based agent-executed QA.

### Research Findings

1. **Existing Graphs tab pattern is directly reusable**:
   - `graphTab int` enum in `internal/adapters/tui/model.go:69-77`
   - `renderGraphTabs(m.graphTab, width)` call in `internal/adapters/tui/view.go:57`
   - Separate `graph_view.go` file for graph-specific rendering
   - Same pattern SHALL be used for `insightTab`
2. **Existing backend services ready to consume**:
   - `DashboardQueryService.QueryDashboard()` → `DashboardSnapshot` (`internal/service/dashboard_query.go:96`)
   - `GraphQueryService.QueryGraphs()` → `GraphSnapshot` (`internal/service/graph_query.go:84`)
   - `InsightExecutor` + `ports.InsightRepository.ListInsights(period)`
3. **`ntcharts` v0.5.1 already installed** — no new chart deps needed. Usage demonstrable in `graph_view.go`.
4. **`domain.Insight.Payload`** contains `SessionIDs`, `UsageEntryIDs`, `Metrics` — sufficient for joining with `UsageEntry` cost to compute attributed waste.
5. **`model.go` is 1385 lines** — Metis flagged risk of accidental regression. Surgical edits + new helper files preferred over broad restructuring.
6. **Pre-existing lint error** in `go.mod` about `bubblezone` — **NOT IN SCOPE** (explicitly excluded below).

### Metis Review — Incorporated Guardrails

Metis identified the following which this plan explicitly addresses:
- Attribution semantics defined BEFORE coding (§ Waste Attribution Rules below)
- Projection formula locked (§ Projection Formula below)
- Detail-view containment model resolved: detail remains a separate mode reachable from Logs only (§ State Machine below)
- Empty-state and narrow-terminal behavior pinned as acceptance criteria
- "Logs" naming confirmed over "Details" (Details = the single-item drill-down screen, Logs = the list)
- Surgical edits mandated for `model.go` — no broad cleanup

---

## Work Objectives

### Core Objective
Transform the Insights experience from "a flat list of detection events" into "a monthly waste intelligence dashboard with drill-down to raw events", without disturbing Dashboard/Graphs/Subscription/Form modes, and add deterministic per-detector cost attribution to the backend.

### Concrete Deliverables

| # | Artifact | Path |
|---|---|---|
| D1 | Waste summary domain types | `internal/domain/waste_summary.go` |
| D2 | Waste summary service + tests | `internal/service/waste_summary.go`, `internal/service/waste_summary_test.go` |
| D3 | `insightTab` state machine | `internal/adapters/tui/model.go` (surgical edits) |
| D4 | Dashboard sub-tab rendering | `internal/adapters/tui/insights_dashboard_view.go` |
| D5 | Logs sub-tab rendering (extracted) | `internal/adapters/tui/insights_logs_view.go` |
| D6 | Tab chrome renderer (reused pattern) | `internal/adapters/tui/view.go` (edit — add `renderInsightTabs`) |
| D7 | Dependency wiring | `internal/adapters/tui/run.go`, constructor signature changes |
| D8 | State-machine tests | `internal/adapters/tui/model_test.go` (additions) |
| D9 | Rendering golden tests | `internal/adapters/tui/insights_dashboard_view_test.go` |
| D10 | QA evidence | `.sisyphus/evidence/task-{N}-*.txt` and `.sisyphus/evidence/final-qa/*` |

### Definition of Done

- [ ] `go test ./internal/service/... -run WasteSummary -count=1` → PASS with ≥ 12 table-driven cases covering happy path + all edge cases defined below
- [ ] `go test ./internal/adapters/tui/... -run Insight -count=1` → PASS (state machine + rendering)
- [ ] `go test ./... -count=1` → full suite PASS (no regressions)
- [ ] `go build ./cmd/tui && ./tui --help` → PASS (binary builds cleanly)
- [ ] tmux QA: entering Insights from top-level menu lands on Dashboard sub-tab by default
- [ ] tmux QA: Tab key within Insights cycles Dashboard ↔ Logs
- [ ] tmux QA: Enter on a Logs row opens Insight Detail; Esc returns to Logs (not Dashboard)
- [ ] tmux QA: Dashboard renders 6 widgets with labels matching spec (see Dashboard Widgets § below)
- [ ] tmux QA: Empty-data case renders graceful empty state (no crash, no "$NaN")
- [ ] tmux QA: Narrow terminal (80 cols) renders without layout breakage
- [ ] All 6 visualizations present with exact labels specified in § Dashboard Widgets

### Must Have

- **Attribution rule**: Every `UsageEntry.CostBreakdown.Total` dollar attributed to AT MOST ONE detector (primary-owner rule; see § Waste Attribution Rules)
- **Default sub-tab**: Entering Insights lands on Dashboard every time
- **State preservation**: Switching tabs within Insights does NOT reset selected Logs row
- **Detail return**: Esc from Detail returns to Logs (last-known row), NOT Dashboard
- **Month scope**: All aggregations scoped to `domain.MonthlyPeriod` for "current month at render time"
- **Projection formula**: `projected_total = sum_so_far * (days_in_month / elapsed_days)`, guarded against `elapsed_days == 0` (returns `sum_so_far`)
- **Waste % formula**: `waste_percent = total_waste_cost / total_spend_cost * 100.0`, guarded against `total_spend_cost == 0` (returns `0.0`)
- **Terminal-chart width**: Line chart MUST render at widths ≥ 60 columns; narrower widths MUST show a fallback text summary ("Trend requires ≥60 cols")
- **Graphs tab pattern reused**: New code SHALL follow the structure of `graphTab` + `renderGraphTabs` exactly
- **Keybindings inside Insights**:
  - `Tab` / `Shift+Tab` / `h` / `l` / `←` / `→`: cycle sub-tabs (Dashboard ↔ Logs)
  - `↑` / `↓` / `k` / `j`: move selection in Logs tab (no-op in Dashboard)
  - `Enter`: in Logs → open detail; in Dashboard → no-op
  - `Esc`: in detail → back to Logs; in Logs/Dashboard → back to top-level dashboard view
  - `r`: refresh current sub-tab data
  - `i`: (from top-level) re-enter Insights, ALWAYS resets to Dashboard sub-tab

### Must NOT Have (Guardrails — from Metis review)

- ❌ **NO** multi-month comparisons, custom date ranges, or month toggles (scope = current month only)
- ❌ **NO** filtering/sorting/customization controls on dashboard widgets
- ❌ **NO** drill-down from Dashboard cards to Logs (read-only widgets only)
- ❌ **NO** new external dependencies (use `ntcharts` + `lipgloss` only)
- ❌ **NO** broad `model.go` cleanup beyond the surgical edits listed in tasks
- ❌ **NO** renames of existing `viewMode` constants (keep `viewInsightList`, `viewInsightDetail`)
- ❌ **NO** changes to Wails GUI (`internal/adapters/gui/**`)
- ❌ **NO** detector logic changes (`internal/service/detector_set_*.go`)
- ❌ **NO** persistence schema changes (SQLite migrations)
- ❌ **NO** fix of the pre-existing `go.mod bubblezone` lint error (out of scope)
- ❌ **NO** "reporting engine" or "analytics framework" abstractions — `WasteSummaryService` stays narrow (monthly waste only)
- ❌ **NO** premature abstraction: presentation structs stay in TUI, domain structs stay in `internal/domain/`
- ❌ **NO** excessive commenting / JSDoc-style comments on every field
- ❌ **NO** generic variable names (`data`, `result`, `temp`, `item`) — use domain-meaningful names
- ❌ **NO** commented-out code left in place
- ❌ **NO** `as any` equivalents (`interface{}` without justification)

---

## Waste Attribution Rules (LOCKED)

> This section is the single source of truth for `WasteSummaryService` semantics. All service tests derive their expectations from these rules.

### Rule W1: Primary-owner assignment (no double-counting)
Each `UsageEntry.EntryID` in the month is assigned to AT MOST ONE `DetectorCategory` as "primary waste owner", by the following priority:

1. Among all `Insight` records for the current month whose `Payload.UsageEntryIDs` contains the entry's ID:
   - Rank by `(severity DESC, detected_at ASC, insight_id ASC)` (where `severity` ordinal: Critical=3, High=2, Medium=1, Low=0; deterministic via `detected_at` + `insight_id` tie-breakers).
2. The entry's primary owner = the top-ranked insight's `DetectorCategory`.
3. Entries NOT referenced by any insight contribute `$0` to waste but DO contribute to `total_spend_cost` denominator.

### Rule W2: Cost attribution
- `WasteByDetector[category].AttributedCostUSD = sum of UsageEntry.CostBreakdown.TotalUSD for entries owned by this category`
- `WasteSummary.TotalWasteCostUSD = sum over all categories`
- `WasteSummary.TotalSpendCostUSD = sum of UsageEntry.CostBreakdown.TotalUSD for all entries in month (owned or not)`
- `WasteSummary.WastePercent = TotalWasteCostUSD / TotalSpendCostUSD * 100.0` (returns `0.0` if denominator is 0)

### Rule W3: Daily trend
- `WasteTrendDaily[d].WasteCostUSD = sum of attributed waste for entries whose `OccurredAt` falls on local-calendar day `d`
- Output array length = days elapsed in current month (NOT full month pre-filled with zeros)
- Days with no waste still appear with `WasteCostUSD = 0.0` (so chart x-axis is contiguous)

### Rule W4: Weekly / Monthly summary
- `WeeklyWasteCostUSD` = sum of `WasteTrendDaily` entries within the ISO week containing today
- `MonthlyWasteCostUSD` = `TotalWasteCostUSD` (alias for clarity in UI)

### Rule W5: Top-N and category breakdown
- `TopCauses` = `WasteByDetector` sorted DESC by `AttributedCostUSD`, take top 5. If fewer than 5 categories have `> $0`, include all non-zero.
- `CategoryBreakdown` = ALL 8 `DetectorCategory` values with their `AttributedCostUSD` (including zeros, for consistent bar chart). Order = enum declaration order in `domain.DetectorCategory`.

### Rule W6: Projection (month-end extrapolation)
```
days_in_month = calendar days in current month (e.g., 30 for April)
elapsed_days = days elapsed as of "today" (inclusive; today counts as 1 if any minute has passed)
if elapsed_days == 0: projected = TotalWasteCostUSD  (usually 0.0 at month start)
else:                projected = TotalWasteCostUSD * (days_in_month / elapsed_days)
```

### Rule W7: Missing data tolerance
- If an `Insight.Payload.UsageEntryIDs` references an ID that no longer exists in the entry set (e.g., data cleanup): SKIP that reference silently. Attribution proceeds on remaining valid IDs.
- If NO insights exist for month: all `WasteByDetector` = 0; `TotalWasteCostUSD = 0`; `WastePercent = 0`; projection = 0. Dashboard renders empty state.

---

## State Machine (LOCKED)

```
Top-level viewMode: viewDashboard | viewManualEntryForm | viewSubscriptionForm |
                    viewSubscriptionList | viewInsightList | viewInsightDetail | viewGraphs

When viewMode == viewInsightList:
  sub-state: insightTab ∈ { insightTabDashboard, insightTabLogs }
  invariants:
    (I1) Entering from another viewMode ALWAYS sets insightTab = insightTabDashboard
    (I2) Tab/Shift+Tab toggles insightTab; never leaves viewInsightList
    (I3) insightTabDashboard: key Enter is NO-OP; ↑↓ NO-OP; r reloads waste summary
    (I4) insightTabLogs: key Enter → viewMode = viewInsightDetail (preserves insightTab=Logs)
    (I5) In viewInsightDetail: Esc → viewMode = viewInsightList with insightTab = insightTabLogs
    (I6) From top-level, key 'i' → viewMode = viewInsightList, insightTab = insightTabDashboard (reset)
    (I7) Logs selection index is PRESERVED across Dashboard ↔ Logs tab switches
    (I8) Logs selection index is RESET when Insights is exited and re-entered
```

---

## Dashboard Widgets (LOCKED — exact labels)

Rendered top-to-bottom, grouped into 3 rows for 80-col terminal:

| # | Widget | Label (exact) | Source field | Render style |
|---|---|---|---|---|
| W1 | Headline card | `"This Month Waste"` | `WasteSummary.TotalWasteCostUSD` formatted `$12.34` | `titleStyle` + large value |
| W2 | Efficiency card | `"Waste % of Total Spend"` | `WasteSummary.WastePercent` formatted `12.3%` | card |
| W3 | Projection card | `"Projected Month-End Waste"` | Rule W6 output formatted `$45.67` | card with muted "proj." suffix |
| W4 | Weekly card | `"This Week Waste"` | `WeeklyWasteCostUSD` formatted `$0.00` | card |
| W5 | Top-5 bar list | `"Top Waste Causes"` | `TopCauses[0..4]`, text-based `██████ $X.XX  CategoryName` | horizontal text bars (no ntcharts dependency to keep W5 lightweight) |
| W6 | Daily trend chart | `"Daily Waste Trend (30-day)"` | `WasteTrendDaily[]` x=day, y=cost | ntcharts line chart; fallback text if width < 60 cols |

> W5 is intentionally a text-based bar list (ASCII fill chars), not ntcharts, because ntcharts horizontal-bar support is inconsistent across terminal widths and category labels are long. This choice matches the "no new deps" guardrail while keeping the top-cause display readable.

Category breakdown (all 8 detectors) is folded into the Logs tab header line as a one-liner summary (e.g., `"Categories: ContextAvalanche 3 · MissedCaching 1 · ..."`) to avoid a 7th widget. If user later wants a separate widget, escalate.

> **Note**: The category breakdown one-liner uses counts of insights per category (not cost), to avoid duplicating W5 with $ values.

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — all verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES (Go `testing` + table-driven pattern visible in `*_test.go`)
- **Automated tests**: YES (TDD — RED → GREEN → REFACTOR)
- **Framework**: Go standard `testing` + `testify` (if already in go.sum; otherwise stdlib only)
- **Agent QA**: tmux-based (`interactive_bash` via `tmux` sessions) to drive TUI and capture terminal output

### Per-task QA pattern
- **Service tasks**: `go test` + JSON output diff against expected fixture
- **TUI state-machine tasks**: `go test` on model transitions; assert exact `insightTab` and `viewMode` values after key events
- **TUI rendering tasks**: golden-file snapshot tests (`testdata/*.golden`) comparing `lipgloss`-rendered strings
- **Integration tasks**: tmux session runs `./tui --db /tmp/fixture.sqlite3`, sends keystrokes via `tmux send-keys`, captures output via `tmux capture-pane -p`, asserts via `grep`/`diff`
- **Evidence**: saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`

### Fixture strategy
- Seed SQLite fixture (via existing repository test helpers if present, else direct SQL inserts) with:
  - 30 `UsageEntry` records spread across current month
  - 5 `Insight` records spanning 3 detector categories (including overlapping entry IDs to exercise Rule W1)
  - 1 entry with NO insight attribution (to exercise denominator-only case)
  - Empty-database variant for empty-state tests

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — start immediately, 3 parallel tasks):
├── T1: Domain types + service skeleton with failing tests [quick]       → blocks T2
├── T2: WasteSummaryService implementation (GREEN) [deep]                 → blocks T7
├── T3: insightTab state machine + failing transition tests [quick]       → blocks T6

Wave 2 (Implementation — 3 parallel tasks after Wave 1):
├── T4: Dashboard widgets rendering + golden tests [visual-engineering]   → blocks T7
├── T5: Logs sub-tab rendering extraction + golden tests [quick]          → blocks T7
├── T6: Key-binding update + Update() dispatcher [deep]                   → blocks T7

Wave 3 (Integration — 1 task):
├── T7: Wire WasteSummaryService into TUI model + run.go constructor [unspecified-high]

Wave 4 (QA — 1 task):
├── T8: tmux-based end-to-end QA scenarios [unspecified-high]

Wave FINAL (Review — 4 parallel tasks, all must APPROVE, then user okay):
├── F1: Plan compliance audit (oracle)
├── F2: Code quality review (unspecified-high)
├── F3: Real manual QA via tmux (unspecified-high)
└── F4: Scope fidelity check (deep)
→ Present results → Get explicit user okay

Critical Path: T1 → T2 → T7 → T8 → F1-F4 → user okay
Parallel Speedup: ~55% vs sequential
Max Concurrent: 3 (Waves 1 & 2)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|---|---|---|---|
| T1 | — | T2, T4, T5 | 1 |
| T2 | T1 | T7 | 1 |
| T3 | — | T6 | 1 |
| T4 | T1, T2 | T7 | 2 |
| T5 | T1 | T7 | 2 |
| T6 | T3 | T7 | 2 |
| T7 | T2, T4, T5, T6 | T8 | 3 |
| T8 | T7 | F1-F4 | 4 |
| F1-F4 | T8 | user okay | FINAL |

### Agent Dispatch Summary

- **Wave 1**: 3 tasks — T1 → `quick` (scaffolding, failing tests), T2 → `deep` (attribution logic), T3 → `quick`
- **Wave 2**: 3 tasks — T4 → `visual-engineering`, T5 → `quick`, T6 → `deep`
- **Wave 3**: 1 task — T7 → `unspecified-high` (integration wiring)
- **Wave 4**: 1 task — T8 → `unspecified-high` (tmux QA)
- **FINAL**: 4 tasks — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [ ] 1. Domain types + WasteSummaryService skeleton (RED)

  **What to do**:
  - Create `internal/domain/waste_summary.go` with the following exported types (exact signatures):
    - `type WasteSummary struct { Period MonthlyPeriod; TotalWasteCostUSD float64; TotalSpendCostUSD float64; WastePercent float64; WeeklyWasteCostUSD float64; MonthlyWasteCostUSD float64; ProjectedMonthEndWasteUSD float64; ByDetector []WasteByDetector; TopCauses []WasteByDetector; DailyTrend []WasteTrendPoint; GeneratedAt time.Time }`
    - `type WasteByDetector struct { Category DetectorCategory; AttributedCostUSD float64; InsightCount int }`
    - `type WasteTrendPoint struct { Day time.Time; WasteCostUSD float64 }`
  - Create `internal/service/waste_summary.go` with service struct + constructor + unimplemented method (returns zero value, error sentinel for unimplemented):
    - `type WasteSummaryService struct { usageRepo ports.UsageEntryRepository; insightRepo ports.InsightRepository; clock func() time.Time }`
    - `func NewWasteSummaryService(usageRepo ports.UsageEntryRepository, insightRepo ports.InsightRepository) *WasteSummaryService`
    - `func (s *WasteSummaryService) ClockForTest(clock func() time.Time)`
    - `func (s *WasteSummaryService) QueryWasteSummary(ctx context.Context, period domain.MonthlyPeriod) (domain.WasteSummary, error)` — currently returns `WasteSummary{}, nil` (stub; T2 implements)
  - Create `internal/service/waste_summary_test.go` with TABLE-DRIVEN failing tests covering:
    - Empty data (no entries, no insights) → all zeros
    - Single entry + single insight (simple attribution)
    - One entry referenced by TWO insights of different severities (Rule W1 ranking)
    - Entry with no insight (denominator-only)
    - Insight referencing a non-existent entry ID (Rule W7 — skip silently)
    - Projection with `elapsed_days == 0`
    - Projection with `elapsed_days > 0`
    - WastePercent with `TotalSpendCostUSD == 0`
    - DailyTrend contiguous days (including zeros)
    - TopCauses with 3 categories (<5)
    - TopCauses with 8 categories (take top 5)
    - CategoryBreakdown order = DetectorCategory enum order
  - Tests MUST fail initially (service stub returns zero value).
  - Use existing fake repository pattern from `internal/service/dashboard_query_test.go` for repo fakes.

  **Must NOT do**:
  - Implement actual attribution logic (T2's job)
  - Add `interface{}` fields
  - Add new dependencies to go.mod
  - Touch `internal/adapters/`

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Scaffolding only; well-defined types and stub; table-driven test skeleton from clear spec
  - **Skills**: `[]`
    - No specialized skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T3)
  - **Blocks**: T2, T4, T5
  - **Blocked By**: None

  **References**:
  - Pattern: `internal/service/dashboard_query.go:71-95` — service struct + constructor + ClockForTest pattern
  - Pattern: `internal/service/dashboard_query.go:15-57` — query request/response domain types
  - Pattern: `internal/service/dashboard_query_test.go` — table-driven test layout, fake repository setup
  - Pattern: `internal/service/graph_query.go:20-55` — snapshot types with nested slices
  - Existing domain: `internal/domain/insight.go:12-60` — `DetectorCategory`, `InsightSeverity`
  - Existing domain: `internal/domain/usage.go` — `UsageEntry`, `CostBreakdown`, `MonthlyPeriod`
  - Existing ports: check `internal/service/errors.go` for sentinel error pattern

  **Acceptance Criteria**:
  - [ ] File `internal/domain/waste_summary.go` exists with exact types listed above
  - [ ] File `internal/service/waste_summary.go` exists with exact method signatures
  - [ ] File `internal/service/waste_summary_test.go` exists with ≥12 table-driven cases
  - [ ] `go build ./internal/domain/... ./internal/service/...` → exits 0
  - [ ] `go test ./internal/service/... -run WasteSummary -count=1` → FAILS (all cases — stub returns zero)
  - [ ] `go vet ./internal/domain/... ./internal/service/...` → no output
  - [ ] `gofmt -l internal/domain/waste_summary.go internal/service/waste_summary.go internal/service/waste_summary_test.go` → empty output

  **QA Scenarios**:

  ```
  Scenario: Types compile and tests run (RED state confirmed)
    Tool: Bash
    Preconditions: Clean git state, feature branch checked out
    Steps:
      1. Run `go build ./internal/domain/... ./internal/service/...`
      2. Run `go test ./internal/service/... -run WasteSummary -count=1 -v 2>&1 | tee .sisyphus/evidence/task-1-red.txt`
      3. Grep evidence file for "FAIL" count ≥ 12
    Expected Result: Build succeeds (exit 0); test run shows 12+ FAIL entries confirming RED state
    Failure Indicators: Build error; fewer than 12 tests present; any test PASSES (means stub wrong)
    Evidence: .sisyphus/evidence/task-1-red.txt

  Scenario: go vet + gofmt clean
    Tool: Bash
    Preconditions: T1 files present
    Steps:
      1. Run `go vet ./internal/domain/... ./internal/service/... 2>&1 | tee .sisyphus/evidence/task-1-vet.txt`
      2. Run `gofmt -l internal/domain/waste_summary.go internal/service/waste_summary.go internal/service/waste_summary_test.go 2>&1 | tee .sisyphus/evidence/task-1-fmt.txt`
    Expected Result: Both evidence files empty (no issues)
    Failure Indicators: Any output in either file
    Evidence: .sisyphus/evidence/task-1-vet.txt, .sisyphus/evidence/task-1-fmt.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-1-red.txt` — failing test output (proves RED)
  - [ ] `.sisyphus/evidence/task-1-vet.txt` — empty
  - [ ] `.sisyphus/evidence/task-1-fmt.txt` — empty

  **Commit**: YES
  - Message: `feat(domain): add waste summary types and service skeleton`
  - Files: `internal/domain/waste_summary.go`, `internal/service/waste_summary.go`, `internal/service/waste_summary_test.go`
  - Pre-commit: `go build ./internal/domain/... ./internal/service/... && gofmt -l internal/ | grep -q . && exit 1 || exit 0`

- [ ] 2. WasteSummaryService implementation (GREEN)

  **What to do**:
  - Implement `QueryWasteSummary` in `internal/service/waste_summary.go` per Rules W1–W7 from plan's § Waste Attribution Rules.
  - Algorithm skeleton:
    1. Load all `UsageEntry` for `period` from `usageRepo`
    2. Load all `Insight` for `period` from `insightRepo`
    3. Build `entryID → UsageEntry` index
    4. For each insight (sorted by severity DESC, detected_at ASC, insight_id ASC):
       - For each referenced `UsageEntryID` in payload:
         - If entry exists AND not yet owned → assign ownership to insight's `DetectorCategory`
         - If entry missing → skip (Rule W7)
    5. Accumulate `AttributedCostUSD` per category
    6. Compute totals, percent, weekly, projection, trend, top-5
    7. Return fully populated `WasteSummary`
  - Ensure DETERMINISTIC output: all slices sorted explicitly; map iteration avoided in output paths.
  - Tests from T1 MUST all PASS after this task.

  **Must NOT do**:
  - Modify the type signatures introduced in T1
  - Add new test cases (that's a separate T1 extension)
  - Use `sync` primitives (service is stateless, per-call)
  - Introduce presentation formatting (e.g., `$` strings) — stay numeric in service
  - Call any other service (service depends on repositories only)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Attribution algorithm is subtle (ranking, tie-breakers, double-counting prevention). Requires careful reading of Rules W1–W7 and test cases from T1.
  - **Skills**: `[]`
    - No specialized skills needed; Go standard library suffices

  **Parallelization**:
  - **Can Run In Parallel**: NO (blocks T7, blocks Wave 2's T4 data needs)
  - **Parallel Group**: Wave 1 (but depends on T1)
  - **Blocks**: T4, T7
  - **Blocked By**: T1

  **References**:
  - Plan § Waste Attribution Rules W1–W7 — SINGLE source of truth for semantics
  - Pattern: `internal/service/dashboard_query.go:96-158` — `QueryDashboard` structure (load repos → build accumulators → return snapshot)
  - Pattern: `internal/service/dashboard_query.go:159-220` — accumulator struct + `buildDashboardProviderSummaries` (map → sorted slice)
  - Pattern: `internal/service/graph_query.go:122-170` — building snapshot from entries, deterministic sort helpers
  - Existing: `domain.Insight.Payload.UsageEntryIDs` in `internal/domain/insight.go`
  - Existing: `domain.UsageEntry.CostBreakdown.TotalUSD` in `internal/domain/usage.go`
  - Existing: `domain.InsightSeverity` constants — derive ordinal mapping (Critical=3 High=2 Medium=1 Low=0)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/service/... -run WasteSummary -count=1` → ALL PASS (12+ cases)
  - [ ] `go test ./internal/service/... -run WasteSummary -count=1 -race` → PASS
  - [ ] `go vet ./internal/service/...` → no output
  - [ ] `gofmt -l internal/service/waste_summary.go` → empty
  - [ ] No use of `interface{}` / `any` in new code
  - [ ] No calls to services other than repositories (verify: `grep -E "\.Query[A-Z]|\.Execute" internal/service/waste_summary.go | grep -v "usageRepo\|insightRepo" ` → empty)

  **QA Scenarios**:

  ```
  Scenario: All attribution tests pass (GREEN state)
    Tool: Bash
    Preconditions: T1 files committed, implementation added
    Steps:
      1. Run `go test ./internal/service/... -run WasteSummary -count=1 -race -v 2>&1 | tee .sisyphus/evidence/task-2-green.txt`
      2. Grep for "--- PASS" count ≥ 12
      3. Grep for "--- FAIL" count == 0
    Expected Result: All tests pass under race detector; no failures
    Failure Indicators: Any FAIL; race condition reported; fewer than 12 passes
    Evidence: .sisyphus/evidence/task-2-green.txt

  Scenario: Deterministic output (run twice, diff identical)
    Tool: Bash
    Preconditions: Test fixture with same data
    Steps:
      1. Run `go test ./internal/service/... -run WasteSummary_Determinism -count=10 -v 2>&1 | tee .sisyphus/evidence/task-2-determinism.txt`
         (Test case in T1 must invoke QueryWasteSummary multiple times and compare byte-for-byte)
    Expected Result: Test PASSES 10 times consecutively
    Failure Indicators: Flaky output (map iteration order leakage)
    Evidence: .sisyphus/evidence/task-2-determinism.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-2-green.txt` — all tests PASS
  - [ ] `.sisyphus/evidence/task-2-determinism.txt` — determinism test PASS

  **Commit**: YES
  - Message: `feat(service): implement waste summary attribution (no double-count)`
  - Files: `internal/service/waste_summary.go`, `internal/service/waste_summary_test.go` (if extended)
  - Pre-commit: `go test ./internal/service/... -run WasteSummary -count=1 -race`

- [ ] 3. `insightTab` state machine with failing transition tests (RED)

  **What to do**:
  - Edit `internal/adapters/tui/model.go` — ADD ONLY (no removals):
    - New enum near existing `graphTab` block (~line 69):
      ```go
      type insightTab int
      const (
          insightTabDashboard insightTab = iota
          insightTabLogs
          insightTabCount
      )
      ```
    - New field on `model` struct (around line 180): `insightTab insightTab`
    - In constructor `newModel` (~line 207): initialize `insightTab: insightTabDashboard`
    - Add stub in `Update` that handles `insightTab` transitions but DOES NOT yet wire all key events (T6 completes wiring)
  - Edit `internal/adapters/tui/model_test.go` — ADD NEW table-driven test function `TestInsightTabTransitions` covering invariants I1–I8 (see plan § State Machine):
    - I1: Entering `viewInsightList` from `viewDashboard` sets `insightTab = insightTabDashboard`
    - I2: Tab key in `viewInsightList` toggles `insightTab`; viewMode unchanged
    - I3: Enter key in `insightTabDashboard` → no-op (viewMode unchanged)
    - I4: Enter key in `insightTabLogs` → viewMode = `viewInsightDetail`, `insightTab` preserved as Logs
    - I5: Esc from `viewInsightDetail` → viewMode = `viewInsightList`, `insightTab = insightTabLogs`
    - I6: Key 'i' from top-level → `viewInsightList` + `insightTab = insightTabDashboard` (reset)
    - I7: Logs row selection preserved across Dashboard ↔ Logs tab switches
    - I8: Exiting Insights and re-entering resets Logs selection to 0
  - Tests MUST FAIL (because T6 hasn't wired full keymap). RED state confirmed.

  **Must NOT do**:
  - Modify `viewMode` constants or order
  - Implement full key handling (T6's job)
  - Touch `view.go` rendering (T4/T5's jobs)
  - Rename existing `viewInsightList` / `viewInsightDetail`
  - Add keybindings to unrelated views

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Additive scaffolding + test writing. Pattern copied from existing `graphTab`.
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T1, T2's RED phase)
  - **Parallel Group**: Wave 1
  - **Blocks**: T6
  - **Blocked By**: None (independent of T1/T2 since it's TUI state only)

  **References**:
  - Pattern: `internal/adapters/tui/model.go:69-77` — `graphTab` enum with `graphTabCount` sentinel
  - Pattern: `internal/adapters/tui/model.go:177` — `graphTab graphTab` field on model
  - Pattern: `internal/adapters/tui/model.go:207-208` — model constructor initialization
  - Pattern: `internal/adapters/tui/model_test.go` — existing table-driven test layout (inspect for style)
  - Plan § State Machine — invariants I1–I8 are the test spec

  **Acceptance Criteria**:
  - [ ] `internal/adapters/tui/model.go` contains `insightTab` type, `insightTabDashboard`, `insightTabLogs`, `insightTabCount` constants
  - [ ] `model` struct has `insightTab` field
  - [ ] `newModel` initializes `insightTab: insightTabDashboard`
  - [ ] `TestInsightTabTransitions` function exists in `model_test.go` with ≥8 subtests (one per invariant)
  - [ ] `go build ./internal/adapters/tui/...` → exit 0
  - [ ] `go test ./internal/adapters/tui/... -run TestInsightTabTransitions -count=1 -v` → FAIL (some or all sub-tests)
  - [ ] Existing `TestGraph*` tests still PASS (regression guard)

  **QA Scenarios**:

  ```
  Scenario: Build succeeds; new tests FAIL (RED confirmed)
    Tool: Bash
    Preconditions: T3 edits applied
    Steps:
      1. Run `go build ./internal/adapters/tui/...`
      2. Run `go test ./internal/adapters/tui/... -run TestInsightTabTransitions -count=1 -v 2>&1 | tee .sisyphus/evidence/task-3-red.txt`
      3. Run `go test ./internal/adapters/tui/... -run "TestGraph|TestDashboard" -count=1 2>&1 | tee .sisyphus/evidence/task-3-regression.txt`
    Expected Result:
      - Build exit 0
      - task-3-red.txt shows ≥1 FAIL in new test
      - task-3-regression.txt shows PASS for existing tests (no regression)
    Failure Indicators: Build fails; new tests all PASS (not truly RED); existing tests broken
    Evidence: .sisyphus/evidence/task-3-red.txt, .sisyphus/evidence/task-3-regression.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-3-red.txt` — new tests failing
  - [ ] `.sisyphus/evidence/task-3-regression.txt` — existing tests still pass

  **Commit**: YES
  - Message: `feat(tui): add insightTab state machine (RED)`
  - Files: `internal/adapters/tui/model.go`, `internal/adapters/tui/model_test.go`
  - Pre-commit: `go build ./internal/adapters/tui/...`

- [ ] 4. Dashboard sub-tab widgets rendering (with golden tests)

  **What to do**:
  - Create `internal/adapters/tui/insights_dashboard_view.go` containing:
    - `func renderInsightsDashboard(summary domain.WasteSummary, width int) string` — top-level renderer
    - Private helpers for each widget W1–W6 per plan § Dashboard Widgets
    - `renderWasteHeadlineCard(summary, width) string` — W1
    - `renderWastePercentCard(summary, width) string` — W2
    - `renderWasteProjectionCard(summary, width) string` — W3
    - `renderWasteWeeklyCard(summary, width) string` — W4
    - `renderTopCausesBarList(summary, width) string` — W5 (ASCII bar fill char `█`)
    - `renderDailyWasteTrend(summary, width) string` — W6 (uses `ntcharts` linechart; fallback text if width < 60)
    - `renderEmptyDashboard(period domain.MonthlyPeriod, width int) string` — empty state
  - Use `lipgloss` styles already defined in `view.go` (`titleStyle`, `mutedStyle`, etc.) — DO NOT introduce new global styles
  - Create `internal/adapters/tui/insights_dashboard_view_test.go` with golden tests:
    - `TestInsightsDashboard_FullData` — fixture with 4 categories, 20 entries, 30 trend points → compare to `testdata/insights_dashboard_full.golden`
    - `TestInsightsDashboard_Empty` — zero-value summary → compare to `testdata/insights_dashboard_empty.golden`
    - `TestInsightsDashboard_SingleCategory` — 1 detector only → golden
    - `TestInsightsDashboard_NarrowTerminal` — width=70 (< 60 for chart) → golden shows fallback text "Trend requires ≥60 cols"
    - `TestInsightsDashboard_ZeroSpend` — `WastePercent == 0.0`, no divide-by-zero crash → golden
  - Golden file update flow: support `-update` flag via `flag.Bool("update", false, ...)` in test (follow existing pattern if present, else add).
  - EXACT labels per plan § Dashboard Widgets must appear in rendered output.

  **Must NOT do**:
  - Call `WasteSummaryService` from renderer (renderer takes `WasteSummary` directly)
  - Introduce new lipgloss style variables in `view.go` (keep styles local if needed)
  - Use `ntcharts` barchart (W5 is ASCII bars intentionally)
  - Add color/theme values beyond what's already used in `view.go`
  - Include category breakdown as a separate widget (folded into Logs header per plan decision)
  - Access repositories or services

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Terminal UI rendering with lipgloss + ntcharts; pixel-precise (char-precise) output; golden testing discipline
  - **Skills**: `[]`
    - Optional: if there's a `lipgloss` or `tui-render` skill, load it. Otherwise none.

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T5, T6)
  - **Parallel Group**: Wave 2
  - **Blocks**: T7
  - **Blocked By**: T1, T2 (needs `WasteSummary` type and realistic fixture)

  **References**:
  - Pattern: `internal/adapters/tui/graph_view.go:1-367` — ntcharts usage, width-aware rendering, tab-specific rendering
  - Pattern: `internal/adapters/tui/view.go:16` — `titleStyle = lipgloss.NewStyle().Bold(true)`
  - Pattern: `internal/adapters/tui/view.go:265-280` — header + title rendering
  - Pattern: `internal/adapters/tui/view.go:265-360` — existing dashboard-style card rendering (study for "card" visual)
  - External: `NimbleMarkets/ntcharts` linechart API — search `graph_view.go` for existing invocation pattern; do not import new chart types
  - Plan § Dashboard Widgets — EXACT labels table (W1–W6)

  **Acceptance Criteria**:
  - [ ] `internal/adapters/tui/insights_dashboard_view.go` exists
  - [ ] `renderInsightsDashboard` is called with a non-nil `WasteSummary` → returns non-empty string containing labels "This Month Waste", "Waste % of Total Spend", "Projected Month-End Waste", "This Week Waste", "Top Waste Causes", "Daily Waste Trend (30-day)"
  - [ ] 5 golden test cases pass: Full, Empty, SingleCategory, NarrowTerminal, ZeroSpend
  - [ ] `testdata/insights_dashboard_*.golden` files committed
  - [ ] Width < 60 case contains literal "Trend requires ≥60 cols"
  - [ ] `go test ./internal/adapters/tui/... -run InsightsDashboard -count=1 -v` → ALL PASS
  - [ ] `gofmt -l internal/adapters/tui/insights_dashboard_view.go insights_dashboard_view_test.go` → empty

  **QA Scenarios**:

  ```
  Scenario: All golden tests pass
    Tool: Bash
    Preconditions: T1 and T2 merged
    Steps:
      1. Run `go test ./internal/adapters/tui/... -run InsightsDashboard -count=1 -v 2>&1 | tee .sisyphus/evidence/task-4-golden.txt`
      2. Assert: 5 PASS, 0 FAIL
      3. Run `ls internal/adapters/tui/testdata/insights_dashboard_*.golden | wc -l` → 5
    Expected Result: 5 golden tests PASS, 5 .golden files exist
    Failure Indicators: Any test FAIL; missing golden files; unexpected diff output
    Evidence: .sisyphus/evidence/task-4-golden.txt

  Scenario: Narrow-terminal fallback rendered
    Tool: Bash
    Preconditions: Golden files written
    Steps:
      1. Run `grep -l "Trend requires" internal/adapters/tui/testdata/insights_dashboard_narrow*.golden`
    Expected Result: File name echoed (contains the fallback string)
    Failure Indicators: No match
    Evidence: Captured via grep output into .sisyphus/evidence/task-4-narrow.txt

  Scenario: Empty state renders without NaN or crash
    Tool: Bash
    Preconditions: Empty-state golden exists
    Steps:
      1. Run `grep -E "NaN|Inf|\\\$\\$" internal/adapters/tui/testdata/insights_dashboard_empty.golden; echo "exit=$?"`
    Expected Result: `exit=1` (no match; grep found nothing bad)
    Failure Indicators: `exit=0` (found NaN/Inf/$$)
    Evidence: .sisyphus/evidence/task-4-empty.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-4-golden.txt` — 5/5 pass
  - [ ] `.sisyphus/evidence/task-4-narrow.txt` — fallback present
  - [ ] `.sisyphus/evidence/task-4-empty.txt` — no NaN/Inf

  **Commit**: YES
  - Message: `feat(tui): render dashboard sub-tab widgets with golden tests`
  - Files: `internal/adapters/tui/insights_dashboard_view.go`, `insights_dashboard_view_test.go`, `testdata/insights_dashboard_*.golden`
  - Pre-commit: `go test ./internal/adapters/tui/... -run InsightsDashboard -count=1`

- [ ] 5. Logs sub-tab rendering extraction + tab chrome

  **What to do**:
  - Create `internal/adapters/tui/insights_logs_view.go` containing:
    - `func renderInsightsLogs(insights []domain.Insight, selection int, width int) string` — extracted from current `viewInsightList` rendering in `view.go`
    - Include one-liner category-count header (see plan § Dashboard Widgets note — count of insights per category, NOT $)
    - Selection highlight using existing `focusStyle` pattern
  - Edit `internal/adapters/tui/view.go`:
    - Add `func renderInsightTabs(active insightTab, width int) string` — clone of `renderGraphTabs` pattern; labels: `Dashboard` | `Logs`
    - Update the `viewInsightList` case to:
      ```go
      case viewInsightList:
          return strings.Join([]string{
              renderHeader(m, width),
              titleStyle.Render("Insights"),
              renderInsightTabs(m.insightTab, width),
              bodyForInsightTab(m, width),
          }, "\n")
      ```
    - Helper `bodyForInsightTab(m model, width int) string` dispatches to `renderInsightsDashboard` (T4) or `renderInsightsLogs` (this task)
  - Create `internal/adapters/tui/insights_logs_view_test.go` with golden tests:
    - `TestInsightsLogs_WithData` (5 insights, category header, selected row highlighted)
    - `TestInsightsLogs_Empty` ("No insights found for this month." body)
    - `TestInsightsLogs_SelectionOutOfRange` (selection=99 → defensive clamp to last valid; no crash)
    - `TestInsightTabsChrome` — both active states (Dashboard active, Logs active) → golden

  **Must NOT do**:
  - Change `viewInsightDetail` rendering (stays as-is)
  - Touch Update() logic (T6's job)
  - Modify `renderGraphTabs` signature or behavior
  - Move styles out of `view.go` into a new file
  - Remove the existing insight list rendering code before verifying test parity (extract first, delete original after tests pass)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Extraction + chrome duplication from an established pattern; mechanical refactor
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T4, T6)
  - **Parallel Group**: Wave 2
  - **Blocks**: T7
  - **Blocked By**: T1 (needs domain.Insight usage clear from context — actually always available)

  **References**:
  - Existing: `internal/adapters/tui/view.go:45-50` — current insight list rendering to extract
  - Existing: `internal/adapters/tui/view.go:56-60` — `renderGraphTabs` usage (pattern to clone)
  - Search `view.go` for `viewInsightList` and `viewInsightDetail` cases to identify the exact code block to extract
  - Pattern: `internal/adapters/tui/graph_view.go` — tab-aware rendering dispatcher
  - Existing: `domain.Insight` fields (category, severity, detected_at) for displaying list rows

  **Acceptance Criteria**:
  - [ ] `internal/adapters/tui/insights_logs_view.go` exists with `renderInsightsLogs` exported (or unexported — match package style)
  - [ ] `internal/adapters/tui/view.go` has `renderInsightTabs(active insightTab, width int) string`
  - [ ] `viewInsightList` case in `view.go` dispatches based on `m.insightTab`
  - [ ] 4 new golden tests pass: WithData, Empty, SelectionOutOfRange, TabsChrome
  - [ ] `testdata/insights_logs_*.golden` and `testdata/insight_tabs_*.golden` committed
  - [ ] All pre-existing TUI tests still PASS
  - [ ] `go test ./internal/adapters/tui/... -count=1` → all pass (Dashboard golden tests from T4 + new Logs tests)

  **QA Scenarios**:

  ```
  Scenario: Tab chrome renders correctly for both active states
    Tool: Bash
    Preconditions: T3, T4 committed
    Steps:
      1. Run `go test ./internal/adapters/tui/... -run "TestInsightsLogs|TestInsightTabs" -count=1 -v 2>&1 | tee .sisyphus/evidence/task-5-golden.txt`
      2. Assert 4 PASS, 0 FAIL
      3. Run `ls internal/adapters/tui/testdata/insight*.golden | wc -l` — confirm ≥ 4
    Expected Result: 4 tests pass, golden files present
    Failure Indicators: Any FAIL; missing files
    Evidence: .sisyphus/evidence/task-5-golden.txt

  Scenario: Full TUI test suite passes (no regression)
    Tool: Bash
    Preconditions: All above + existing tests
    Steps:
      1. Run `go test ./internal/adapters/tui/... -count=1 2>&1 | tee .sisyphus/evidence/task-5-full.txt`
    Expected Result: All TUI tests PASS
    Failure Indicators: Any FAIL in pre-existing tests
    Evidence: .sisyphus/evidence/task-5-full.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-5-golden.txt` — 4/4 new tests pass
  - [ ] `.sisyphus/evidence/task-5-full.txt` — full TUI suite pass

  **Commit**: YES
  - Message: `refactor(tui): extract logs sub-tab rendering + tab chrome`
  - Files: `internal/adapters/tui/insights_logs_view.go`, `insights_logs_view_test.go`, `view.go`, `testdata/insight*.golden`
  - Pre-commit: `go test ./internal/adapters/tui/... -count=1`

- [ ] 6. Sub-tab key handling in Update() (GREEN for T3 tests)

  **What to do**:
  - Edit `internal/adapters/tui/model.go` — `Update()` function. Locate the case handling `viewInsightList` (~line 420–520). Modify to:
    - On `Tab` / `Shift+Tab` / `h` / `l` / `←` / `→` key: toggle `m.insightTab` via modulo `insightTabCount`. viewMode unchanged.
    - On `↑` / `↓` / `k` / `j`: ONLY when `m.insightTab == insightTabLogs`, adjust `m.insightSelection`. When Dashboard, no-op.
    - On `Enter`: ONLY when `insightTabLogs` AND `insightSelection` is valid, transition to `viewInsightDetail`. Dashboard → no-op.
    - On `Esc`: transition to `viewDashboard` (top-level)
    - On `r`: trigger reload of active tab data (emit appropriate `tea.Cmd` — `loadWasteSummary` for Dashboard, `loadInsights` for Logs)
  - Locate the case handling `viewInsightDetail` (~line 510-530). Modify:
    - On `Esc` / `Backspace`: set viewMode = `viewInsightList`, ENSURE `insightTab = insightTabLogs` (invariant I5)
  - Locate top-level keymap that handles `i` key (~line 418-420). Ensure entering Insights sets `insightTab = insightTabDashboard` (invariant I6 — reset on entry).
  - Update help text in `view.go` (~line 416-420):
    - For insight list mode: `"Tab/Shift+Tab cycle tabs • ↑↓ move • Enter detail • r refresh • Esc back • q quit"`
  - All 8 invariant tests from T3 MUST PASS after this task.

  **Must NOT do**:
  - Change key handling for other viewModes
  - Add NEW keybindings not listed above
  - Introduce a second "back" stack beyond the explicit Esc rules
  - Modify `manualEntryForm` / `subscriptionForm` / `graphs` key handlers
  - Remove or rename existing keybindings in top-level view

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Surgical edits to a 1385-line central state machine. High risk of regression. Requires careful reading of existing switch cases.
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T4, T5 — T6 edits `model.go`'s Update() while T4/T5 edit view.go and new files; no conflict expected BUT reviewer must verify)
  - **Parallel Group**: Wave 2
  - **Blocks**: T7
  - **Blocked By**: T3

  **References**:
  - Existing: `internal/adapters/tui/model.go:420-424` — top-level `i`/`g` key handler (entry point for Insights)
  - Existing: `internal/adapters/tui/model.go:495-525` — existing insight list/detail Update cases
  - Pattern: `internal/adapters/tui/model.go` graphs case — how graphTab transitions are implemented for reference on modulo cycling
  - Existing: `internal/adapters/tui/view.go:416-420` — help text strings to update
  - Plan § State Machine invariants I1–I8
  - T3's test file for exact expected behaviors

  **Acceptance Criteria**:
  - [ ] `go test ./internal/adapters/tui/... -run TestInsightTabTransitions -count=1 -v` → ALL 8+ subtests PASS (GREEN)
  - [ ] `go test ./internal/adapters/tui/... -count=1` → all pass (no regressions)
  - [ ] `go test ./internal/adapters/tui/... -count=1 -race` → PASS
  - [ ] `grep -n "insightTabLogs\|insightTabDashboard" internal/adapters/tui/model.go` shows ≥ 5 usages in Update()
  - [ ] Help text for insight list contains "Tab/Shift+Tab cycle tabs"

  **QA Scenarios**:

  ```
  Scenario: All 8 invariants pass (state machine GREEN)
    Tool: Bash
    Preconditions: T3 merged (RED), T6 edits applied
    Steps:
      1. Run `go test ./internal/adapters/tui/... -run TestInsightTabTransitions -count=1 -v 2>&1 | tee .sisyphus/evidence/task-6-green.txt`
      2. Assert: exactly 8+ PASS, 0 FAIL
    Expected Result: All invariants pass
    Failure Indicators: Any FAIL
    Evidence: .sisyphus/evidence/task-6-green.txt

  Scenario: No regression in other views
    Tool: Bash
    Preconditions: T6 applied
    Steps:
      1. Run `go test ./internal/adapters/tui/... -run "TestGraph|TestDashboard|TestManualEntry|TestSubscription" -count=1 2>&1 | tee .sisyphus/evidence/task-6-regression.txt`
    Expected Result: All pre-existing tests PASS
    Failure Indicators: Any FAIL
    Evidence: .sisyphus/evidence/task-6-regression.txt

  Scenario: Race-safe under -race flag
    Tool: Bash
    Preconditions: T6 applied
    Steps:
      1. Run `go test ./internal/adapters/tui/... -count=1 -race 2>&1 | tee .sisyphus/evidence/task-6-race.txt`
    Expected Result: PASS under race detector
    Failure Indicators: Data race reported
    Evidence: .sisyphus/evidence/task-6-race.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-6-green.txt` — GREEN state
  - [ ] `.sisyphus/evidence/task-6-regression.txt` — no regressions
  - [ ] `.sisyphus/evidence/task-6-race.txt` — race-clean

  **Commit**: YES
  - Message: `feat(tui): wire insights sub-tab key handling (GREEN)`
  - Files: `internal/adapters/tui/model.go`, `internal/adapters/tui/view.go` (help text only)
  - Pre-commit: `go test ./internal/adapters/tui/... -count=1 -race`

- [ ] 7. Integrate WasteSummaryService into TUI model + run.go

  **What to do**:
  - Edit `internal/adapters/tui/model.go`:
    - Add interface `wasteSummaryLoader` (following existing pattern at lines 20-40):
      ```go
      type wasteSummaryLoader interface {
          QueryWasteSummary(ctx context.Context, period domain.MonthlyPeriod) (domain.WasteSummary, error)
      }
      ```
    - Add `wasteSummary wasteSummaryLoader` to `modelDependencies` struct (~line 119)
    - Add `wasteSummaryData domain.WasteSummary` field to `model` struct
    - Add `wasteSummaryLoadedMsg struct { data domain.WasteSummary; err error }` near existing message types
    - Add `loadWasteSummary() tea.Cmd` that invokes the loader
    - Wire the command into:
      - Initial load when entering Insights (invariant I1/I6 entry point)
      - `r` key handler in Dashboard sub-tab (from T6)
    - Add handler for `wasteSummaryLoadedMsg` in Update() (pattern after `dashboardLoadedMsg` at line 228)
  - Edit `internal/adapters/tui/run.go`:
    - Accept a `wasteSummaryLoader` in the constructor signature
    - Pass through to `newModel`
  - Edit `cmd/tui/main.go` (or whichever file wires services — locate via `grep -rn "NewDashboardQueryService\|NewGraphQueryService" cmd/`):
    - Construct `service.NewWasteSummaryService(usageRepo, insightRepo)` alongside existing services
    - Pass into the TUI runner
  - Update `model_test.go` if existing dependency-injection test helpers need updating for the new field (use fake with no-op behavior).

  **Must NOT do**:
  - Change signatures of OTHER loader interfaces
  - Add the service to GUI bindings (TUI only per plan)
  - Modify the service itself (T2 fixed its surface)
  - Block Update() on service call (use `tea.Cmd` async pattern)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Multi-file wiring across constructor, cmd entrypoint, and model. Requires understanding existing dependency-injection layout.
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO (last integration step)
  - **Parallel Group**: Wave 3
  - **Blocks**: T8
  - **Blocked By**: T2, T4, T5, T6

  **References**:
  - Pattern: `internal/adapters/tui/model.go:20-40` — existing loader interfaces
  - Pattern: `internal/adapters/tui/model.go:119-126` — `modelDependencies` struct
  - Pattern: `internal/adapters/tui/model.go:228-244` — `dashboardLoadedMsg` handling in Update
  - Existing: `internal/adapters/tui/run.go` (79 lines) — constructor invocation
  - Existing: `cmd/tui/` — entry point with service construction
  - Plan § Dashboard Widgets and § State Machine — for when to trigger loads

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/tui` → exit 0 (binary builds)
  - [ ] `go build ./...` → exit 0 (full build)
  - [ ] `go test ./... -count=1 -race` → all pass
  - [ ] `./tui --help` (after build) → help text displayed, exits cleanly
  - [ ] Running `grep -rn "NewWasteSummaryService" cmd/ internal/` shows it wired in `cmd/tui/` and used in `internal/adapters/tui/`
  - [ ] `gofmt -l .` → empty

  **QA Scenarios**:

  ```
  Scenario: Binary builds and boots without error
    Tool: Bash
    Preconditions: T2, T4, T5, T6 merged
    Steps:
      1. Run `go build -o /tmp/tui-integrate ./cmd/tui 2>&1 | tee .sisyphus/evidence/task-7-build.txt`
      2. Assert exit code 0
      3. Run `/tmp/tui-integrate --help 2>&1 | tee .sisyphus/evidence/task-7-help.txt`
      4. Assert help text appears
    Expected Result: Build succeeds; help renders
    Failure Indicators: Build errors; crash on --help
    Evidence: .sisyphus/evidence/task-7-build.txt, .sisyphus/evidence/task-7-help.txt

  Scenario: Full test suite passes with race
    Tool: Bash
    Preconditions: Build succeeds
    Steps:
      1. Run `go test ./... -count=1 -race 2>&1 | tee .sisyphus/evidence/task-7-fulltest.txt`
    Expected Result: All packages PASS
    Failure Indicators: Any package FAIL
    Evidence: .sisyphus/evidence/task-7-fulltest.txt

  Scenario: Service is wired (grep verification)
    Tool: Bash
    Steps:
      1. Run `grep -rn "NewWasteSummaryService" cmd/ internal/ 2>&1 | tee .sisyphus/evidence/task-7-wiring.txt`
      2. Assert ≥ 2 matches (one in cmd/tui, one in internal/adapters/tui or similar)
    Expected Result: Wiring present in expected files
    Failure Indicators: No matches or only in service itself
    Evidence: .sisyphus/evidence/task-7-wiring.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-7-build.txt` — build succeeds
  - [ ] `.sisyphus/evidence/task-7-help.txt` — help renders
  - [ ] `.sisyphus/evidence/task-7-fulltest.txt` — all pass
  - [ ] `.sisyphus/evidence/task-7-wiring.txt` — wiring verified

  **Commit**: YES
  - Message: `feat(tui): integrate waste summary service end-to-end`
  - Files: `internal/adapters/tui/model.go`, `internal/adapters/tui/run.go`, `cmd/tui/main.go` (and any test helpers updated)
  - Pre-commit: `go build ./cmd/tui && go test ./... -count=1`

- [ ] 8. tmux end-to-end QA scenarios

  **What to do**:
  - Use `interactive_bash` (tmux) to launch the TUI against a seeded SQLite fixture and verify user-visible behavior.
  - Prepare fixture:
    - Create fresh DB at `/tmp/llmbudget-qa.sqlite3`
    - Seed via short Go helper script OR via existing repository fixture helpers:
      - 30 `UsageEntry` in current month (varying models, costs)
      - 5 `Insight` records across 3 categories, some sharing entries (exercises W1)
      - 1 entry with no attribution
    - Fixture helper committed under `internal/adapters/sqlite/testdata/` or as a dev-only `cmd/qa-seed/main.go` (mark clearly as dev tool)
  - Execute tmux scenarios (capture pane for each step into evidence):
    - S1: Default landing on Dashboard sub-tab after `i` keystroke
    - S2: Tab cycles to Logs; Shift+Tab cycles back to Dashboard
    - S3: Enter on Logs row opens Detail; Esc returns to Logs (insightTab preserved)
    - S4: `r` refreshes active tab
    - S5: Esc from Dashboard/Logs returns to top-level Dashboard
    - S6: Empty fixture (fresh DB, no seed) — Dashboard shows empty state, no crash
    - S7: Narrow terminal (`tmux resize-pane -x 80`) — chart fallback text appears
    - S8: Rapid tab presses (send-keys 20x Tab in quick succession) — no state corruption, selection intact
  - Each scenario saves evidence: `.sisyphus/evidence/task-8-sN-{slug}.txt` containing `tmux capture-pane -p` output
  - Each scenario has an `assert` line: `grep -q "This Month Waste" .sisyphus/evidence/task-8-s1-dashboard-default.txt` style

  **Must NOT do**:
  - Modify production code to make tests pass (this is QA, not implementation)
  - Use Playwright or browser tools (TUI only)
  - Skip scenarios due to time pressure

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Requires tmux orchestration, keystroke injection, output parsing, evidence management
  - **Skills**: `[]`
    - If `interactive-bash` or `tmux-tui-qa` skill exists, load it

  **Parallelization**:
  - **Can Run In Parallel**: NO (last step before review wave)
  - **Parallel Group**: Wave 4
  - **Blocks**: F1, F2, F3, F4 (they consume evidence files)
  - **Blocked By**: T7

  **References**:
  - Pattern: Existing `internal/adapters/tui/evidence_test.go` — may have tmux-style capture helpers
  - Existing: `README.md` "TUI" section — how to launch `./tui --db <path>`
  - Existing: `install.sh` / `Makefile` — for build variants
  - Plan § Dashboard Widgets — EXACT labels to grep for in captures

  **Acceptance Criteria**:
  - [ ] 8 evidence files exist: `.sisyphus/evidence/task-8-s1-*.txt` through `task-8-s8-*.txt`
  - [ ] Each evidence file contains expected landmark string (e.g., S1 contains "This Month Waste"; S3 contains an insight detail identifier)
  - [ ] No "panic" substring in any evidence file (`! grep -rl 'panic' .sisyphus/evidence/task-8-*.txt`)
  - [ ] No "NaN" substring in any evidence file
  - [ ] S6 empty-state evidence contains "No insights" or similar empty text, not a crash
  - [ ] S7 narrow-terminal evidence contains "Trend requires ≥60 cols"

  **QA Scenarios**:

  ```
  Scenario S1: Default Dashboard sub-tab
    Tool: interactive_bash (tmux session 'qa')
    Preconditions: `go build -o /tmp/tui ./cmd/tui` succeeded; seeded DB at /tmp/llmbudget-qa.sqlite3
    Steps:
      1. tmux new-session -d -s qa -x 120 -y 40 "/tmp/tui --db /tmp/llmbudget-qa.sqlite3"
      2. sleep 1
      3. tmux send-keys -t qa 'i' (enter Insights)
      4. sleep 1
      5. tmux capture-pane -t qa -p > .sisyphus/evidence/task-8-s1-dashboard-default.txt
      6. grep -q "This Month Waste" .sisyphus/evidence/task-8-s1-dashboard-default.txt
      7. grep -q "Dashboard" .sisyphus/evidence/task-8-s1-dashboard-default.txt  (tab bar)
      8. ! grep -q "panic\|NaN" .sisyphus/evidence/task-8-s1-dashboard-default.txt
    Expected Result: All grep assertions pass
    Failure Indicators: Missing landmark, panic, NaN
    Evidence: .sisyphus/evidence/task-8-s1-dashboard-default.txt

  Scenario S3: Detail drill-down and return to Logs
    Tool: interactive_bash (tmux session 'qa')
    Preconditions: Already in Insights after S1
    Steps:
      1. tmux send-keys -t qa 'Tab' (switch to Logs)
      2. sleep 0.3
      3. tmux send-keys -t qa 'Enter' (open detail on first row)
      4. sleep 0.5
      5. tmux capture-pane -t qa -p > .sisyphus/evidence/task-8-s3a-detail.txt
      6. grep -q "Insight Detail" .sisyphus/evidence/task-8-s3a-detail.txt
      7. tmux send-keys -t qa 'Escape'
      8. sleep 0.3
      9. tmux capture-pane -t qa -p > .sisyphus/evidence/task-8-s3b-back-to-logs.txt
      10. grep -q "Logs" .sisyphus/evidence/task-8-s3b-back-to-logs.txt
      11. ! grep -q "This Month Waste" .sisyphus/evidence/task-8-s3b-back-to-logs.txt  (NOT Dashboard)
    Expected Result: Detail opens; Esc returns to Logs, NOT Dashboard
    Failure Indicators: Esc returns to Dashboard (invariant I5 violation); no detail view
    Evidence: .sisyphus/evidence/task-8-s3a-detail.txt, task-8-s3b-back-to-logs.txt

  Scenario S6: Empty DB shows empty state, no crash (negative)
    Tool: interactive_bash (tmux session 'qa-empty')
    Preconditions: Fresh empty DB at /tmp/llmbudget-qa-empty.sqlite3
    Steps:
      1. rm -f /tmp/llmbudget-qa-empty.sqlite3
      2. tmux new-session -d -s qa-empty -x 120 -y 40 "/tmp/tui --db /tmp/llmbudget-qa-empty.sqlite3"
      3. sleep 1
      4. tmux send-keys -t qa-empty 'i'
      5. sleep 1
      6. tmux capture-pane -t qa-empty -p > .sisyphus/evidence/task-8-s6-empty.txt
      7. grep -qE "\$0\.00|No insights|No waste" .sisyphus/evidence/task-8-s6-empty.txt
      8. ! grep -q "panic\|NaN\|runtime error" .sisyphus/evidence/task-8-s6-empty.txt
    Expected Result: Empty state text; no crash
    Failure Indicators: Panic; NaN; hang (test times out)
    Evidence: .sisyphus/evidence/task-8-s6-empty.txt

  Scenario S7: Narrow terminal fallback (negative)
    Tool: interactive_bash
    Preconditions: Active session with data
    Steps:
      1. tmux resize-pane -t qa -x 70 -y 40
      2. sleep 0.5
      3. tmux capture-pane -t qa -p > .sisyphus/evidence/task-8-s7-narrow.txt
      4. grep -q "Trend requires" .sisyphus/evidence/task-8-s7-narrow.txt
    Expected Result: Fallback text visible
    Failure Indicators: Broken chart rendering; crash
    Evidence: .sisyphus/evidence/task-8-s7-narrow.txt
  ```

  > Scenarios S2, S4, S5, S8 follow the same shape — documented inline in evidence files.

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-8-s1-dashboard-default.txt`
  - [ ] `.sisyphus/evidence/task-8-s2-tab-cycle.txt`
  - [ ] `.sisyphus/evidence/task-8-s3a-detail.txt`, `task-8-s3b-back-to-logs.txt`
  - [ ] `.sisyphus/evidence/task-8-s4-refresh.txt`
  - [ ] `.sisyphus/evidence/task-8-s5-esc-toplevel.txt`
  - [ ] `.sisyphus/evidence/task-8-s6-empty.txt`
  - [ ] `.sisyphus/evidence/task-8-s7-narrow.txt`
  - [ ] `.sisyphus/evidence/task-8-s8-rapid-tab.txt`

  **Commit**: YES
  - Message: `test(tui): add tmux end-to-end QA scenarios for insights tabs`
  - Files: `.sisyphus/evidence/task-8-*.txt` (evidence only; no code changes unless dev-only seed helper committed)
  - Pre-commit: evidence files present and pass assertions

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read this plan end-to-end. For each "Must Have": verify implementation exists (open file, run command, check output). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found (e.g., `rg "bubblezone" go.mod`, `rg "internal/adapters/gui" git-diff`, `rg "month.*toggle|custom.*range" internal/adapters/tui/`). Verify all evidence files exist in `.sisyphus/evidence/`. Confirm 6 widgets render with EXACT labels from § Dashboard Widgets. Confirm state machine invariants I1–I8 via reading tests.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT with file:line for any failures`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./... 2>&1`, `go test ./... -count=1 -race 2>&1`, `go vet ./...`, `gofmt -l internal/`. Review all changed files (new + edited) for: `interface{}` without justification, `panic()` in production paths, empty `if err != nil { }` branches, `fmt.Println` / `log.Println` debug leftovers, TODO/FIXME comments added in this session, commented-out code blocks, unused imports, generic names (data/result/item/temp). Specifically audit `WasteSummaryService` for: wildcard allocations, unbounded slice appends without pre-size, map iteration without sorted output (determinism hazard for tests).
  Output: `Build [PASS/FAIL] | Tests [N pass / N fail] | Vet [PASS/FAIL] | Gofmt [N files clean] | Files [N clean / N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state (fresh `/tmp/llmbudget-qa.sqlite3`). Seed with the documented fixture (30 entries, 5 insights, etc.). Execute EVERY QA scenario from EVERY task (T1–T8) using tmux. Capture screenshots of terminal via `tmux capture-pane -p > .sisyphus/evidence/final-qa/{scenario}.txt`. Test cross-task integration: enter Insights → Dashboard renders → Tab to Logs → Enter on row → Detail → Esc → back to Logs (not Dashboard) → Tab → Dashboard → Esc → top level. Test edge cases: empty DB (all zeros, no crash), narrow terminal (80 cols, 60 cols), rapid Tab presses (no state corruption).
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | Evidence files [N created] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task (T1–T8): read task's "What to do" spec, read actual `git diff` for the commit(s) associated, verify 1:1 correspondence (everything specified was built; nothing beyond spec was added). Check each "Must NOT Have" guardrail against `git diff`: no changes to `internal/adapters/gui/**`, no changes to `internal/service/detector_set_*.go`, no SQLite migrations, no changes to `go.mod` beyond necessary, no changes to `viewDashboard`/`viewGraphs` top-level logic. Detect cross-task contamination: e.g., T4 (Dashboard rendering) should NOT touch `model.go` `Update()` logic (that's T6). Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN / N issues with file:line] | Unaccounted [CLEAN / N files] | Guardrails violated [NONE / list] | VERDICT`

---

## Commit Strategy

Per-task atomic commits. Message format: `type(scope): desc`

- **T1**: `feat(domain): add waste summary types` — `internal/domain/waste_summary.go`, test file skeleton. Pre-commit: `go build ./internal/domain/...`
- **T2**: `feat(service): implement waste summary attribution` — `internal/service/waste_summary.go`, `_test.go`. Pre-commit: `go test ./internal/service/... -run WasteSummary -count=1`
- **T3**: `feat(tui): add insightTab state machine` — edits to `model.go`, `model_test.go`. Pre-commit: `go test ./internal/adapters/tui/... -run InsightTab -count=1`
- **T4**: `feat(tui): render dashboard sub-tab widgets` — `insights_dashboard_view.go`, `_test.go`, `testdata/*.golden`. Pre-commit: `go test ./internal/adapters/tui/... -run InsightsDashboard -count=1`
- **T5**: `refactor(tui): extract logs sub-tab rendering` — `insights_logs_view.go`, `_test.go`, `view.go` edits. Pre-commit: `go test ./internal/adapters/tui/... -run InsightsLogs -count=1`
- **T6**: `feat(tui): wire insights sub-tab key handling` — `model.go` edits (Update dispatch), `model_test.go` edits. Pre-commit: `go test ./internal/adapters/tui/... -count=1`
- **T7**: `feat(tui): integrate waste summary service` — `model.go` constructor, `run.go`. Pre-commit: `go build ./cmd/tui && go test ./... -count=1`
- **T8**: `test(tui): tmux end-to-end QA scenarios` — `.sisyphus/evidence/task-8-*.txt`, no code changes. Pre-commit: all tmux scenarios captured

---

## Success Criteria

### Verification Commands

```bash
# 1. Full build
go build ./...  # Expected: no output (success)

# 2. Full test suite, race, count=1
go test ./... -race -count=1  # Expected: PASS across all packages

# 3. Focused service tests
go test ./internal/service/... -run WasteSummary -count=1 -v  # Expected: 12+ table cases PASS

# 4. Focused TUI tests
go test ./internal/adapters/tui/... -run "Insight|insightTab|Dashboard" -count=1 -v  # Expected: PASS

# 5. No forbidden file changes
git diff --name-only origin/HEAD..HEAD | grep -E "internal/adapters/gui/|internal/service/detector_set_" && echo "VIOLATION" || echo "clean"

# 6. Binary smoke test
go build -o /tmp/tui-smoke ./cmd/tui && /tmp/tui-smoke --help  # Expected: help text displayed

# 7. Gofmt clean
gofmt -l internal/  # Expected: empty output

# 8. Vet clean
go vet ./...  # Expected: no output
```

### Final Checklist

- [ ] All 6 dashboard widgets present with EXACT labels from § Dashboard Widgets
- [ ] State machine invariants I1–I8 verified by tests
- [ ] Attribution rules W1–W7 verified by `WasteSummaryService` tests
- [ ] All "Must Have" items present
- [ ] All "Must NOT Have" guardrails respected (F4 confirms)
- [ ] `go test ./... -race -count=1` passes
- [ ] tmux QA evidence files exist for all 8 scenarios
- [ ] F1, F2, F3, F4 all APPROVE
- [ ] User has given explicit "okay" after reviewing final verification output
