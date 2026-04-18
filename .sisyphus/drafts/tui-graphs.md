# Draft: TUI Token Usage Graphs

## Requirements (confirmed)
- TUI 그래프 추가: 한 달 동안 "많이 사용한 모델" 보여주기
- 토큰 사용량 확인: OpenRouter Activity 페이지와 비슷한 형태
- 어느 모델에서 토큰을 많이 사용했는지 확인 가능해야 함
- 모두 TUI 형태의 그래프

## Technical Decisions
- **Framework**: bubbletea + bubbles + lipgloss (기존 스택)
- **Chart library**: TBD (librarian 연구 진행 중)

## Research Findings

### Codebase Structure
- TUI entry: `cmd/tui/main.go` -> `internal/adapters/tui/run.go`
- Model: `internal/adapters/tui/model.go` - bubbletea Model with viewMode navigation
- View: `internal/adapters/tui/view.go` - lipgloss-based rendering
- **NO existing chart/graph components**

### Navigation Flow
- viewDashboard (default) -> Tab/Shift+Tab between sections
- viewManualEntryForm (m key)
- viewSubscriptionForm (s key)
- viewInsightList (i key) -> viewInsightDetail (enter)
- **New graph view needs a new viewMode + keybinding**

### Data Available
- `usage_entries` table: has `model_name`, `input_tokens`, `output_tokens`, `cache_creation_tokens`, `cache_read_tokens`, `cost_usd`, `recorded_at`
- `sessions` table: has `model_name`, `input_tokens`, `output_tokens`, tokens, cost, `started_at`, `ended_at`
- `DashboardSnapshot` already aggregates by provider but NOT by model
- **Gap**: No query exists for "aggregate by model" - need new query

### DB Schema Key Fields for Graphs
- `usage_entries.model_name` - the model identifier
- `usage_entries.input_tokens`, `output_tokens`, `cache_creation_tokens`, `cache_read_tokens` - token counts
- `usage_entries.cost_usd` - total cost
- `usage_entries.recorded_at` - timestamp for time-series
- `sessions.model_name` - model per session
- `sessions.total_cost_usd`, token fields

### Data Gaps
- Need aggregation query: GROUP BY model_name, SUM tokens, SUM cost for a given period
- Need time-series query: daily token usage grouped by model for the month
- Existing `ListUsageEntries` returns raw entries (can aggregate in-memory or add SQL query)

## Decisions Made
- **Graph placement**: 새로운 뷰 모드 (g 키)로 전환
- **Graph types**: 4가지 모두 구현
  1. 모델별 토큰 사용량 Bar Chart
  2. 모델별 비용 Bar Chart
  3. 일별 토큰 사용량 추이
  4. 토큰 타입 비율 표시
- **Chart approach**: ntcharts v1 (bubbletea v1 호환) - charmbracelet 공식 추천 라이브러리
  - Fallback: custom lipgloss 렌더링 (ntcharts 문제 시)
- **Data layer**: 새로운 서비스 쿼리 필요 (모델별 집계, 일별 시계열)

## Librarian Research Summary
- ntcharts: 673 stars, bubbletea 공식 추천, bar/line/sparkline 지원, lipgloss 네이티브
- v1: github.com/NimbleMarkets/ntcharts v0.5.1 (bubbletea v1 호환)
- v2: charm.land 기반 (bubbletea v2 전용 - 현재 프로젝트에 부적합)
- termdash/termui: bubbletea와 비호환 (별도 이벤트 루프)
- asciigraph: 라인 차트만, bubbletea 통합 없음
- Custom lipgloss: 간단한 수평 바 차트에 적합, 축/스케일링은 직접 구현

## Scope Boundaries
- INCLUDE: 4가지 그래프, 새 뷰 모드, 모델별 집계 쿼리, 일별 시계열 쿼리
- EXCLUDE: Interactive drill-down, export, real-time streaming, mouse interaction
