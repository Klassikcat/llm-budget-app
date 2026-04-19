2026-04-19T02:28:16+09:00 - Warning: `go mod tidy` removes `github.com/NimbleMarkets/ntcharts` until later graph code imports it, so this task keeps the dependency pinned explicitly in `go.mod` and `go.sum` for downstream work.
2026-04-19T02:36:18+09:00 - Gotcha: empty-month graph queries return fully empty slices, but once any usage exists the daily trend output must include every date in the month and only leave `ModelBreakdown` empty on zero-usage days.
2026-04-19T02:41:05+09:00 - Gotcha: the startup `app.Graph` does not currently expose a `GraphQueryService` sibling yet, so `internal/adapters/tui/run.go` instantiates `service.NewGraphQueryService(graph.Store)` directly to keep this scaffolding task scoped to the requested TUI files.
2026-04-19T02:46:07+09:00 - Gotcha: `view.go` defaulted unknown modes back to the dashboard, so state-machine-only graph work was invisible until a minimal graph renderer and graph-specific header/help integration were added.
- `ntcharts` `barchart` with `AutoBarWidth` can result in a bar width of 0 if the height is too small for the number of bars and gaps, causing nothing to be drawn. We solved this by calculating the exact height needed (`len(data)*2 - 1`) and disabling `AutoBarWidth`.
- `ntcharts` `timeserieslinechart` doesn't have a built-in legend, so we have to build one manually using `lipgloss` and append it to the chart view.
- 2026-04-19T02:55:00+09:00 - Gotcha: The daily trend chart initially rendered all models, which made it unreadable. Fixed by limiting the rendered series to the top 5 models based on total tokens using the already sorted `ModelTokenUsages` data.
- No major blockers. Used largest remainder method to distribute blocks in the segmented bar chart to avoid rounding errors.

### Task 7: Graph Screen Integration
- No major issues encountered. The integration was straightforward as the foundation was well-laid in previous tasks.
