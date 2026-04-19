2026-04-19T02:28:16+09:00 - Added explicit dependency `github.com/NimbleMarkets/ntcharts v0.5.1` to `go.mod` and recorded its checksums in `go.sum`; verified the project still builds with `go build ./...`.
2026-04-19T02:36:18+09:00 - Added `GraphQueryService` using monthly `UsageFilter` + in-memory accumulators only; token-oriented outputs use stable model normalization (`unknown-model`), deterministic descending ranking, and fold ranks beyond the top 10 into `Other` while daily trends still emit every day in the month.
2026-04-19T02:41:05+09:00 - Added TUI graph-mode scaffolding in `model.go`: `viewGraphs`, a four-tab `graphTab`, `graphLoadedMsg`, graph state fields, and `loadGraphs()` so `g` lazily queries `QueryGraphs` while Esc/Backspace exits and Tab/Shift+Tab/Left/Right/h/l cycle tabs without changing dashboard/form/insight behavior.
2026-04-19T02:46:07+09:00 - Task 3 needs minimal `view.go` integration to be testable: a dedicated `viewGraphs` render branch plus visible tab labels, graph-mode header/help text, and loading/error placeholders are enough for QA without pulling real ntcharts rendering into this step.
- `ntcharts` `barchart` requires `WithNoAutoBarWidth()` and `WithBarWidth(1)` to render horizontal bars correctly when the height is constrained.
- The origin of the `barchart` is calculated based on the label length. When using `WithDataSet`, it's important to pass it before `WithHorizontalBars` so that the origin is calculated correctly, or call `Resize` / `SetHorizontal` after adding data.
- `humanize.Comma` is useful for formatting large numbers with commas.
- Added OpenRouter-style rank and percentage share to the graph labels.
- Updated the truncation logic to use `utf8.RuneCountInString` and `[]rune` to correctly handle multi-byte characters like `…` and ensure the label fits within the allowed width.
- Always clean up root-level scratch files (`test_*.go`) created for quick testing, as they can cause `go test ./...` and `go build ./...` to fail due to duplicate `main` functions.
- `ntcharts` `timeserieslinechart` requires `DrawBrailleAll()` to render all datasets correctly.
- When using `timeserieslinechart`, it's important to provide a data point for every time step for every dataset, even if the value is 0, to ensure the lines are drawn continuously and correctly aligned.
- A custom legend can be built using `lipgloss` to display the colors associated with each dataset in the line chart.
- When rendering charts with potentially many series (like daily trends for all models), it's important to limit the number of series displayed (e.g., top 5) to maintain readability and avoid cluttering the chart and legend.
- Implemented a custom segmented bar chart using lipgloss for the token breakdown tab, avoiding external chart libraries for better control over layout and colors.
- Fixed empty state string to exactly match the plan requirement: 'No token breakdown data for this month.'
- Added a check to return the empty state message if all breakdown rows have zero total tokens, preventing an empty string from being returned.

### Task 7: Graph Screen Integration
- The graph screen integration was largely complete from previous tasks.
- `renderView`, `renderHeader`, and `renderHelp` correctly handle the `viewGraphs` mode.
- Loading and error states are handled globally for the graph screen in `renderGraphs`, ensuring consistency across all tabs.
- Empty states are handled individually within each graph rendering function, providing specific messages for each graph type.
- Viewport sizing is managed by the `viewport` component, which handles scrolling if the content exceeds the terminal height. The width is passed down to the rendering functions to ensure responsive layouts (e.g., truncating labels, scaling bar charts).
- Status messages for loading and refreshing graphs are clear and consistent.

2026-04-19T03:21:30+09:00 - Task 8 QA: `go build ./cmd/tui`, `go test ./...`, and `go vet ./...` all passed and evidence was written under `.sisyphus/evidence/`; Bubble Tea pexpect automation successfully captured dashboard -> graph mode -> tab cycling -> dashboard -> quit, but the raw transcript did not preserve a distinct settled active render for the final `Token Breakdown` tab before exit.
2026-04-19T03:30:00+09:00 - Task 8 QA repair: updated `.sisyphus/evidence/task8_tui_qa.py` to wait for a Token Breakdown-specific legend marker (`C.Read:` / `C.Write:` or the empty-state string), keep the fourth tab visible briefly before `Esc`, and emit `.sisyphus/evidence/task8-tui-graph-flow.snapshots.txt` with focused per-tab excerpts so all four graph tabs are explicitly evidenced.

## Final-Wave Remediation
- Updated `internal/adapters/tui/graph_view.go` to use a consistent per-model color palette across all charts (token usage, cost, and daily trend) by hashing the model name to select a color from a predefined palette.
- Created the consolidated evidence file `.sisyphus/evidence/task-8-build-test-vet.txt` to record the success of `go build`, `go test`, and `go vet`.
