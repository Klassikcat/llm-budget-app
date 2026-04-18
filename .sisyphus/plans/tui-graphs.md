# TUI Token Usage & Model Activity Graphs

## TL;DR

> **Quick Summary**: LLM Budget Tracker TUI에 4가지 그래프 뷰를 추가한다. 'g' 키를 누르면 별도의 그래프 화면으로 전환되어 이번 달 모델별 토큰 사용량, 비용, 일별 추이, 토큰 타입 비율을 시각화한다.
> 
> **Deliverables**:
> - 모델별 토큰 사용량 수평 바 차트
> - 모델별 비용 수평 바 차트
> - 일별 토큰 사용량 추이 라인 차트 (모델별 색상 구분)
> - 모델별 토큰 타입 비율 표시 (input/output/cache_read/cache_write)
> - 새로운 그래프 전용 viewMode와 서비스 레이어 집계 쿼리
> 
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Task 1 → Task 2 → Tasks 3-6 (parallel) → Task 7 → Task 8

---

## Context

### Original Request
TUI 그래프를 추가하라. 한 달 동안 "많이 사용한 모델"을 보여주는 그래프, 토큰 사용량을 확인할 수 있는 OpenRouter Activity 페이지와 유사한 형태의 그래프, 어느 모델에서 많은 토큰을 썼는지 확인할 수 있어야 한다. 모두 TUI 형태의 그래프여야 한다.

### Interview Summary
**Key Discussions**:
- 그래프 배치: 새로운 뷰 모드 (`g` 키)로 대시보드와 별도 전체 화면
- 4가지 그래프 타입 모두 구현 확정: 모델별 토큰 Bar, 모델별 비용 Bar, 일별 추이, 토큰 타입 비율
- 차트 라이브러리: ntcharts v1 (charmbracelet 공식 추천, bubbletea v1 호환)

**Research Findings**:
- ntcharts v0.5.1: BarChart(horizontal/vertical), LineChart, TimeSeriesLineChart 지원, lipgloss 네이티브 스타일링
- 기존 코드에 모델별 집계 쿼리 없음 - `ListUsageEntries`는 raw 엔트리만 반환
- DB `usage_entries` 테이블에 `model_name`, `input_tokens`, `output_tokens`, `cache_creation_tokens`, `cache_read_tokens`, `cost_usd`, `recorded_at` 필드 존재
- 현재 5개 viewMode 존재: Dashboard, ManualEntryForm, SubscriptionForm, InsightList, InsightDetail

### Metis Review
**Identified Gaps** (addressed):
- `viewGraph`는 단일 뷰가 아닌 4개 서브 뷰를 갖는 구조 필요 → Tab으로 서브 그래프 전환하는 `graphTab` 상태 추가
- 데이터가 없는 경우(빈 월)의 빈 상태 처리 필요 → 각 차트에 empty state 렌더링 포함
- ntcharts 의존성 추가 시 go.mod 업데이트 필요 → Task 1에서 처리
- 그래프 뷰의 header/help text 업데이트 필요 → 기존 renderHelp/renderHeader 패턴 따름

---

## Work Objectives

### Core Objective
이번 달 LLM 모델 사용량을 시각적으로 파악할 수 있는 4가지 TUI 그래프를 추가한다.

### Concrete Deliverables
- `internal/adapters/tui/graph_view.go` - 그래프 렌더링 전용 파일
- `internal/adapters/tui/model.go` - viewGraphs 모드, graphTab 상태, 'g' 키 바인딩 추가
- `internal/adapters/tui/view.go` - renderHelp에 그래프 도움말 추가
- `internal/service/graph_query.go` - 모델별/일별 집계 쿼리 서비스
- ntcharts v1 의존성 추가 (go.mod/go.sum)

### Definition of Done
- [ ] `go build ./cmd/tui` 성공
- [ ] `go test ./...` 모든 테스트 통과
- [ ] TUI에서 'g' 키를 누르면 그래프 뷰로 전환
- [ ] Tab으로 4가지 그래프 사이 전환 가능
- [ ] Esc로 대시보드로 복귀
- [ ] 데이터가 없을 때 빈 상태 메시지 표시

### Must Have
- 모델별 토큰 사용량 수평 바 차트 (상위 10개 모델)
- 모델별 비용 수평 바 차트 (상위 10개 모델)
- 일별 토큰 사용량 추이 라인 차트
- 모델별 토큰 타입 비율 (input/output/cache_read/cache_write)
- 'g' 키로 그래프 뷰 진입, Esc로 복귀
- Tab으로 서브 그래프 전환
- 빈 데이터 상태 처리

### Must NOT Have (Guardrails)
- 마우스 인터랙션 추가하지 않는다 (bubblezone 의존성 불필요)
- 기존 대시보드 뷰를 변경하지 않는다 (새 viewMode만 추가)
- 새로운 DB 마이그레이션을 추가하지 않는다 (기존 테이블의 데이터만 집계)
- 외부 API 호출을 추가하지 않는다
- 그래프 데이터를 캐시하지 않는다 (매번 쿼리)
- 프롬프트 텍스트나 응답 내용을 표시하지 않는다 (프라이버시)

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (go test ./... 동작 확인)
- **Automated tests**: Tests-after (서비스 레이어 집계 로직에 대해)
- **Framework**: `go test` (표준 Go 테스트)

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **TUI**: Use `interactive_bash` (tmux) - 빌드 후 TUI 실행, 키 입력, 화면 캡처
- **Service/Logic**: Use Bash (`go test`) - 테스트 실행, 출력 확인
- **Build**: Use Bash (`go build`) - 컴파일 확인

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - sequential dependency):
├── Task 1: ntcharts 의존성 추가 + go.mod 업데이트 [quick]
├── Task 2: 그래프 집계 서비스 + 테스트 (depends: 1) [deep]

Wave 2 (Graph Views - MAX PARALLEL, depends: Task 2):
├── Task 3: 그래프 뷰 모드 + 네비게이션 스캐폴딩 [unspecified-high]
├── Task 4: 모델별 토큰 바 차트 + 모델별 비용 바 차트 (depends: 2, 3) [unspecified-high]
├── Task 5: 일별 토큰 추이 라인 차트 (depends: 2, 3) [unspecified-high]
├── Task 6: 토큰 타입 비율 표시 (depends: 2, 3) [unspecified-high]

Wave 3 (Integration):
├── Task 7: 통합 + 빈 상태 처리 + 도움말 업데이트 (depends: 3-6) [unspecified-high]
├── Task 8: 빌드 검증 + TUI 실행 QA (depends: 7) [quick]

Wave FINAL (4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Task 1 → Task 2 → Task 3 → Task 4 → Task 7 → Task 8 → F1-F4 → user okay
Parallel Speedup: ~50% faster than sequential
Max Concurrent: 4 (Wave 2)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | - | 2-8 | 1 |
| 2 | 1 | 3-8 | 1 |
| 3 | 2 | 4, 5, 6, 7 | 2 |
| 4 | 2, 3 | 7 | 2 |
| 5 | 2, 3 | 7 | 2 |
| 6 | 2, 3 | 7 | 2 |
| 7 | 3, 4, 5, 6 | 8 | 3 |
| 8 | 7 | F1-F4 | 3 |

### Agent Dispatch Summary

- **Wave 1**: 2 tasks - T1 → `quick`, T2 → `deep`
- **Wave 2**: 4 tasks - T3 → `unspecified-high`, T4 → `unspecified-high`, T5 → `unspecified-high`, T6 → `unspecified-high`
- **Wave 3**: 2 tasks - T7 → `unspecified-high`, T8 → `quick`
- **FINAL**: 4 tasks - F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. ntcharts v1 의존성 추가

  **What to do**:
  - `go get github.com/NimbleMarkets/ntcharts@v0.5.1` 실행하여 ntcharts v1 의존성 추가
  - `go mod tidy` 실행하여 go.mod/go.sum 정리
  - `go build ./...` 실행하여 빌드 확인

  **Must NOT do**:
  - ntcharts v2 (charm.land 기반)를 추가하지 않는다. 현재 프로젝트는 bubbletea v1을 사용한다.
  - bubblezone 의존성을 추가하지 않는다 (마우스 불필요)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1 (sequential start)
  - **Blocks**: Tasks 2-8
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `go.mod:1-72` - 현재 의존성 목록. bubbletea v1.2.4, bubbles v0.20.0, lipgloss v1.0.0 사용 중. ntcharts v1은 이들과 호환된다.

  **External References**:
  - ntcharts v1 릴리즈: `https://github.com/NimbleMarkets/ntcharts/releases/tag/v0.5.1`

  **Acceptance Criteria**:
  - [ ] `go.mod`에 `github.com/NimbleMarkets/ntcharts` 의존성이 추가됨
  - [ ] `go build ./...` 성공 (exit code 0)

  **QA Scenarios**:

  ```
  Scenario: ntcharts 의존성 추가 후 빌드 성공
    Tool: Bash
    Preconditions: 프로젝트 루트 디렉토리
    Steps:
      1. `go get github.com/NimbleMarkets/ntcharts@v0.5.1` 실행
      2. `go mod tidy` 실행
      3. `go build ./...` 실행
      4. exit code 확인 (0이어야 함)
      5. `grep 'NimbleMarkets/ntcharts' go.mod` 실행하여 의존성 확인
    Expected Result: 빌드 성공, go.mod에 ntcharts 엔트리 존재
    Failure Indicators: go get 실패, go build 에러, go.mod에 ntcharts 없음
    Evidence: .sisyphus/evidence/task-1-ntcharts-dependency.txt
  ```

  **Commit**: YES
  - Message: `chore(deps): add ntcharts v1 for TUI graph rendering`
  - Files: `go.mod`, `go.sum`
  - Pre-commit: `go build ./...`

- [x] 2. 그래프 집계 서비스 (GraphQueryService) + 테스트

  **What to do**:
  - `internal/service/graph_query.go` 생성: 4가지 집계 결과를 반환하는 `GraphQueryService`
  - `GraphQuery` 입력 구조체: `Period domain.MonthlyPeriod`
  - `GraphSnapshot` 출력 구조체:
    - `ModelTokenUsages []ModelTokenUsage` (모델별 토큰 합계, 내림차순 정렬)
    - `ModelCosts []ModelCost` (모델별 비용 합계, 내림차순 정렬)
    - `DailyTokenTrends []DailyTokenTrend` (일별 토큰 합계, 모델별 분해)
    - `ModelTokenBreakdowns []ModelTokenBreakdown` (모델별 input/output/cache_read/cache_write 비율)
  - `ModelTokenUsage`: `ModelName string`, `TotalTokens int64`, `InputTokens int64`, `OutputTokens int64`, `CacheReadTokens int64`, `CacheWriteTokens int64`
  - `ModelCost`: `ModelName string`, `TotalCostUSD float64`
  - `DailyTokenTrend`: `Date time.Time`, `ModelBreakdown []ModelDailyTokens`
  - `ModelDailyTokens`: `ModelName string`, `TotalTokens int64`
  - `ModelTokenBreakdown`: `ModelName string`, `InputTokens int64`, `OutputTokens int64`, `CacheReadTokens int64`, `CacheWriteTokens int64`, `TotalTokens int64`
  - 서비스는 `ports.UsageEntryRepository`를 의존성으로 받아 `ListUsageEntries`를 호출한 후 인메모리에서 집계한다
  - 상위 10개 모델까지만 반환, 나머지는 "Other"로 합산
  - `internal/service/graph_query_test.go` 생성: 집계 로직 테스트
    - 빈 데이터 → 빈 결과
    - 여러 모델 → 내림차순 정렬 확인
    - 일별 그룹핑 확인
    - 토큰 타입 비율 계산 확인
    - 10개 초과 모델 → "Other" 합산 확인

  **Must NOT do**:
  - 새로운 SQL 쿼리를 SQLite 어댑터에 추가하지 않는다 (기존 ListUsageEntries 사용)
  - DB 마이그레이션을 추가하지 않는다

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (Task 1 완료 후)
  - **Parallel Group**: Wave 1 (after Task 1)
  - **Blocks**: Tasks 3-8
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - `internal/service/dashboard_query.go:20-69` - DashboardQuery/DashboardSnapshot 패턴. 동일한 서비스 패턴을 따른다: 입력 쿼리 구조체 + 출력 스냅샷 구조체 + 생성자 함수.
  - `internal/service/dashboard_query.go:79-87` - NewDashboardQueryService 생성자 패턴. usageRepo, sessionRepo 등을 주입받는 방식 참고.
  - `internal/service/dashboard_query.go:167-220` - buildDashboardProviderSummaries의 accumulator 패턴. map으로 키별 집계하고 정렬하는 패턴을 재사용.

  **API/Type References**:
  - `internal/domain/usage.go:37-43` - TokenUsage 구조체 (InputTokens, OutputTokens, CacheReadTokens, CacheWriteTokens, TotalTokens)
  - `internal/domain/usage.go:74-82` - CostBreakdown 구조체 (TotalUSD 등)
  - `internal/domain/usage.go:117-131` - UsageEntry 구조체 (PricingRef.ModelID로 모델명 추출)
  - `internal/ports/repository.go:10-16` - UsageFilter 구조체 (Period로 월별 필터링)
  - `internal/ports/repository.go:54-57` - UsageEntryRepository 인터페이스

  **Test References**:
  - `internal/service/dashboard_query_test.go` - 대시보드 쿼리 테스트 패턴 참고 (mock repo, 테스트 데이터 구성, 결과 검증)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/service/...` 모든 테스트 통과
  - [ ] graph_query_test.go에 최소 5개 테스트 케이스 (빈 데이터, 정렬, 일별 그룹핑, 토큰 비율, Other 합산)
  - [ ] GraphSnapshot의 각 필드가 올바르게 채워짐

  **QA Scenarios**:

  ```
  Scenario: 그래프 집계 서비스 테스트 통과
    Tool: Bash
    Preconditions: Task 1 완료 (ntcharts 의존성 추가됨)
    Steps:
      1. `go test -v ./internal/service/ -run TestGraph` 실행
      2. 모든 테스트가 PASS인지 확인
      3. 테스트 출력에서 5개 이상의 테스트 함수 확인
    Expected Result: `ok  llm-budget-tracker/internal/service` 출력, 0 failures
    Failure Indicators: FAIL 출력, 컴파일 에러
    Evidence: .sisyphus/evidence/task-2-graph-query-tests.txt

  Scenario: 빈 데이터 시 빈 결과 반환
    Tool: Bash
    Preconditions: 테스트 코드에 빈 데이터 케이스 포함
    Steps:
      1. `go test -v ./internal/service/ -run TestGraph.*Empty` 실행
      2. 빈 슬라이스가 반환되는지 테스트 확인
    Expected Result: PASS
    Failure Indicators: nil pointer, panic, FAIL
    Evidence: .sisyphus/evidence/task-2-graph-empty-data.txt
  ```

  **Commit**: YES
  - Message: `feat(service): add graph aggregation query for model/daily token stats`
  - Files: `internal/service/graph_query.go`, `internal/service/graph_query_test.go`
  - Pre-commit: `go test ./internal/service/...`

---

- [x] 3. 그래프 뷰 모드 + 네비게이션 스캐폴딩

  **What to do**:
  - `internal/adapters/tui/model.go`에 `viewGraphs` viewMode 추가
  - `graphTab` enum/state 추가: `graphTabModelTokens`, `graphTabModelCosts`, `graphTabDailyTrend`, `graphTabTokenBreakdown`
  - `model` struct에 그래프 관련 상태 추가:
    - `graphData service.GraphSnapshot`
    - `graphLoading bool`
    - `graphErr error`
    - `graphTab graphTab`
  - `graphLoadedMsg` 메시지 타입 추가
  - `graphLoader` 인터페이스 추가: `QueryGraphs(ctx context.Context, query service.GraphQuery) (service.GraphSnapshot, error)`
  - `modelDependencies`에 `graphs graphLoader` 추가
  - `newModel` 초기값에 graphTab default 설정
  - `Update`에 `graphLoadedMsg` 처리 추가
  - `updateDashboard`에 `g` 키 처리 추가: graph view로 진입하면서 graph load 수행
  - `updateGraphs` 함수 신설:
    - `esc`, `backspace` → dashboard 복귀
    - `tab`, `right`, `l` → 다음 graphTab
    - `shift+tab`, `left`, `h` → 이전 graphTab
    - `r` → graph data reload
  - `loadGraphs()` tea.Cmd 추가
  - `loadAll()`에 graphs load 포함 여부를 결정: dashboard 초기 로드 시 함께 로드하거나, g 진입 시 lazy load 중 하나를 선택하되 계획상 **g 진입 시 lazy load**로 고정
  - `internal/adapters/tui/run.go`에 `graphs: graph.GraphQueryService` 주입 추가

  **Must NOT do**:
  - 기존 Dashboard의 탭 순서(sectionOverview/providers/budgets/recentSessions)를 변경하지 않는다
  - `m`, `s`, `i`, `r`, `q` 기존 키 동작을 깨뜨리지 않는다

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (foundation for Tasks 4-6)
  - **Blocks**: Tasks 4, 5, 6, 7
  - **Blocked By**: Task 2

  **References**:

  **Pattern References**:
  - `internal/adapters/tui/model.go:51-59` - 기존 `viewMode` enum 구조. 새 `viewGraphs`를 여기에 추가한다.
  - `internal/adapters/tui/model.go:116-136` - `model` 상태 구조체. 새 graph state 필드를 같은 스타일로 추가한다.
  - `internal/adapters/tui/model.go:281-313` - `updateDashboard` 키 처리 패턴. `g` 키 분기 추가 위치.
  - `internal/adapters/tui/model.go:315-349` - 별도 뷰별 update 함수 패턴 (`updateInsightList`, `updateInsightDetail`). `updateGraphs`도 이 패턴을 따른다.
  - `internal/adapters/tui/model.go:398-430` - `loadDashboard`, `loadInsights`, `loadAlerts`, `loadAll`의 tea.Cmd 패턴.
  - `internal/adapters/tui/run.go:47-53` - 의존성 주입 위치. GraphQueryService도 여기서 전달한다.

  **Acceptance Criteria**:
  - [ ] `g` 키 분기 추가됨
  - [ ] `viewGraphs`와 `graphTab` 상태 정의됨
  - [ ] `Esc`로 dashboard로 되돌아감
  - [ ] `Tab`/`Shift+Tab`으로 4개 그래프 탭 전환 가능

  **QA Scenarios**:

  ```
  Scenario: 그래프 뷰 진입 및 복귀
    Tool: interactive_bash
    Preconditions: Task 2 완료, TUI 빌드 가능 상태
    Steps:
      1. `go run ./cmd/tui` 실행
      2. 대시보드 화면이 표시되면 `g` 입력
      3. 헤더에 Graphs 또는 Graph View 모드명이 표시되는지 확인
      4. `esc` 입력
      5. 헤더가 Dashboard로 돌아오는지 확인
    Expected Result: g로 그래프 뷰 진입, esc로 dashboard 복귀
    Failure Indicators: 키 입력 무반응, 잘못된 화면 전환, panic
    Evidence: .sisyphus/evidence/task-3-graph-mode-navigation.txt

  Scenario: 그래프 탭 순환 전환
    Tool: interactive_bash
    Preconditions: 그래프 뷰 진입 상태
    Steps:
      1. `tab`을 4번 입력
      2. 각 입력마다 활성 탭 라벨이 바뀌는지 확인
      3. `shift+tab` 1회 입력
      4. 이전 탭으로 되돌아가는지 확인
    Expected Result: 4개 탭 순환, 역방향 전환 동작
    Failure Indicators: 탭 고정, 인덱스 범위 에러, 잘못된 탭 표시
    Evidence: .sisyphus/evidence/task-3-graph-tab-cycle.txt
  ```

  **Commit**: NO

- [x] 4. 모델별 토큰/비용 수평 바 차트 구현

  **What to do**:
  - `internal/adapters/tui/graph_view.go` 생성
  - ntcharts `barchart` 패키지를 사용해 2개 렌더러 구현:
    - `renderModelTokenBarChart(snapshot service.GraphSnapshot, width int) string`
    - `renderModelCostBarChart(snapshot service.GraphSnapshot, width int) string`
  - 상위 10개 모델 데이터를 수평 바 차트로 렌더링
  - 각 모델은 일관된 색상 팔레트를 사용 (예: lipgloss.Color 63, 69, 75, 81, ...)
  - 라벨 포맷:
    - 토큰 차트: `model-name   123,456 tokens`
    - 비용 차트: `model-name   $12.34`
  - OpenRouter Activity 스타일처럼 랭킹/점유율 정보 보조 텍스트 표시
  - 빈 데이터 시: `No model token activity for this month.` / `No model cost activity for this month.`
  - 너무 긴 모델명은 truncate 처리

  **Must NOT do**:
  - 세로 차트를 사용하지 않는다 (TUI width 활용이 떨어짐)
  - 10개를 초과하는 개별 모델 바를 모두 노출하지 않는다 (`Other`로 합산)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6)
  - **Blocks**: Task 7
  - **Blocked By**: Tasks 2, 3

  **References**:

  **Pattern References**:
  - `internal/adapters/tui/view.go:214-225` - providers 섹션의 리스트 렌더링 패턴. 모델명 + 수치 조합 출력 스타일 참고.
  - `internal/adapters/tui/view.go:344-356` - `truncateLine` 유틸. 긴 모델명 처리에 재사용.

  **External References**:
  - Context7 ntcharts README example - `barchart.New(11, 10)`, `PushAll([]barchart.BarData{...})`, `Draw()`, `View()` 패턴
  - OpenRouter Activity 페이지 - 모델별 토큰/비용 랭킹 리스트 UX 참고

  **Acceptance Criteria**:
  - [ ] 토큰 탭에서 수평 바 차트 렌더링
  - [ ] 비용 탭에서 수평 바 차트 렌더링
  - [ ] 모델별 값이 내림차순으로 정렬되어 표시
  - [ ] 빈 데이터 상태 메시지 표시

  **QA Scenarios**:

  ```
  Scenario: 모델별 토큰 바 차트 렌더링
    Tool: interactive_bash
    Preconditions: 그래프 뷰 구현 완료, 샘플 usage 데이터가 DB에 존재
    Steps:
      1. `go run ./cmd/tui` 실행 후 `g` 입력
      2. 첫 번째 탭(모델별 토큰)이 활성인지 확인
      3. 화면에 최소 1개 이상의 `█` 또는 차트 바 문자 표시 확인
      4. 모델명과 `tokens` 텍스트가 함께 표시되는지 확인
    Expected Result: 모델명 + 토큰 수 + 수평 바가 함께 렌더링
    Failure Indicators: 빈 문자열, 바 미표시, 잘못된 숫자 포맷
    Evidence: .sisyphus/evidence/task-4-model-token-bar.txt

  Scenario: 모델별 비용 바 차트 렌더링
    Tool: interactive_bash
    Preconditions: 그래프 뷰 진입 상태
    Steps:
      1. `tab` 입력하여 비용 탭으로 이동
      2. 화면에 `$` 기호와 비용 값이 표시되는지 확인
      3. 모델명별 수평 바가 보이는지 확인
    Expected Result: 모델별 비용과 바 차트 표시
    Failure Indicators: 비용 값 미표시, 바 누락, 탭 전환 실패
    Evidence: .sisyphus/evidence/task-4-model-cost-bar.txt
  ```

  **Commit**: NO

- [ ] 5. 일별 토큰 사용량 추이 라인 차트 구현

  **What to do**:
  - ntcharts line/time-series chart를 사용해 `renderDailyTokenTrendChart(snapshot service.GraphSnapshot, width int) string` 구현
  - 이번 달 1일부터 말일까지 일별 토큰 사용량을 X축으로 표시
  - 상위 N개 모델(예: 5개)만 개별 선으로 표시하고, 나머지는 `Other`로 합산하거나 생략
  - 범례(legend) 텍스트를 차트 하단에 렌더링: 모델명별 색상 매핑
  - 데이터가 없는 날짜는 0으로 채워 선이 끊기지 않게 함
  - 빈 데이터 시: `No daily token trend available for this month.`

  **Must NOT do**:
  - 하루 단위보다 더 촘촘한 시간 해상도를 추가하지 않는다
  - 모델이 너무 많다고 모든 선을 한 화면에 그리지 않는다 (가독성 보호)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 6)
  - **Blocks**: Task 7
  - **Blocked By**: Tasks 2, 3

  **References**:

  **Pattern References**:
  - `internal/adapters/tui/view.go:38-73` - renderDashboard 섹션 조합 패턴. 새 그래프 탭도 동일하게 header + chart + help 구조를 가진다.

  **External References**:
  - Context7 ntcharts docs - timeseries line chart에 `Push(TimePoint{date, value})`, `SetDataSetStyle(...)` 패턴 사용

  **Acceptance Criteria**:
  - [ ] 일별 토큰 추이 탭에서 라인 차트 렌더링
  - [ ] 최소 1개 이상의 모델 라인이 색상 구분되어 표시
  - [ ] 빈 날짜가 0으로 보간되어 차트가 연속 표시
  - [ ] 범례 표시

  **QA Scenarios**:

  ```
  Scenario: 일별 토큰 추이 차트 렌더링
    Tool: interactive_bash
    Preconditions: 그래프 뷰 구현 완료, 월 내 여러 날짜의 usage 데이터 존재
    Steps:
      1. `go run ./cmd/tui` 실행 후 `g` 입력
      2. `tab`으로 일별 추이 탭까지 이동
      3. 차트에 선 문자가 렌더링되는지 확인
      4. 하단에 범례 텍스트(모델명)가 표시되는지 확인
    Expected Result: 날짜 기반 라인 차트 + 범례 렌더링
    Failure Indicators: 빈 차트, 범례 누락, panic
    Evidence: .sisyphus/evidence/task-5-daily-token-trend.txt

  Scenario: 빈 데이터에서 일별 추이 empty state
    Tool: Bash
    Preconditions: 빈 DB 또는 테스트 월에 usage 데이터 없음
    Steps:
      1. 테스트용 빈 DB로 `go run ./cmd/tui --db /tmp/empty-graph.db` 실행
      2. `g` 입력 후 일별 추이 탭까지 이동
      3. `No daily token trend available for this month.` 문자열 확인
    Expected Result: 차트 대신 명확한 empty state 메시지 표시
    Failure Indicators: 공백 화면, panic, 잘못된 placeholder
    Evidence: .sisyphus/evidence/task-5-daily-token-empty.txt
  ```

  **Commit**: NO

- [ ] 6. 모델별 토큰 타입 비율 표시 구현

  **What to do**:
  - custom lipgloss 기반 렌더러 `renderModelTokenBreakdown(snapshot service.GraphSnapshot, width int) string` 구현
  - 모델별로 한 줄 또는 두 줄 블록 렌더링:
    - 모델명
    - 총 토큰 수
    - input/output/cache_read/cache_write 각각의 수치와 퍼센트
    - 비율 막대(예: 색상별 `█` 세그먼트)
  - 상위 5~10개 모델만 표시
  - 퍼센트는 소수점 1자리 또는 정수 반올림으로 고정
  - 빈 데이터 시: `No token breakdown data for this month.`

  **Must NOT do**:
  - pie chart/도넛 차트를 ASCII로 억지 구현하지 않는다
  - 퍼센트 합이 100을 넘거나 모자라게 계산하지 않는다

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5)
  - **Blocks**: Task 7
  - **Blocked By**: Tasks 2, 3

  **References**:

  **Pattern References**:
  - `internal/adapters/tui/view.go:15-21` - 기존 lipgloss 스타일 정의. 같은 팔레트 계열을 유지한다.
  - `internal/adapters/tui/view.go:203-225` - overview/providers처럼 텍스트+숫자 혼합 출력 패턴 참고.

  **API/Type References**:
  - `internal/domain/usage.go:37-43` - TokenUsage의 4개 구성요소가 비율 계산의 기준.

  **Acceptance Criteria**:
  - [ ] 각 모델에 대해 input/output/cache_read/cache_write 수치 표시
  - [ ] 비율 막대가 렌더링됨
  - [ ] 퍼센트 합이 100%에 수렴
  - [ ] 빈 상태 메시지 표시

  **QA Scenarios**:

  ```
  Scenario: 모델별 토큰 타입 비율 렌더링
    Tool: interactive_bash
    Preconditions: 그래프 뷰 구현 완료, usage 데이터 존재
    Steps:
      1. `go run ./cmd/tui` 실행 후 `g` 입력
      2. `tab`으로 토큰 타입 비율 탭까지 이동
      3. 각 모델 블록에 `input`, `output`, `cache` 관련 텍스트가 표시되는지 확인
      4. 색상 구분된 세그먼트 막대가 보이는지 확인
    Expected Result: 모델별 4가지 토큰 유형 수치 및 비율 표시
    Failure Indicators: 비율 누락, 음수 퍼센트, 막대 미표시
    Evidence: .sisyphus/evidence/task-6-token-breakdown.txt

  Scenario: 퍼센트 계산 검증
    Tool: Bash
    Preconditions: 테스트 코드 또는 deterministic fixture 존재
    Steps:
      1. `go test -v ./internal/service/ -run TestGraph.*Breakdown` 실행
      2. 출력에서 비율 합 검증 테스트 통과 확인
    Expected Result: PASS
    Failure Indicators: 합계 오차, FAIL
    Evidence: .sisyphus/evidence/task-6-breakdown-percentage-test.txt
  ```

  **Commit**: NO

---

- [ ] 7. 그래프 뷰 통합 + 빈 상태 처리 + 도움말/헤더 업데이트

  **What to do**:
  - `internal/adapters/tui/view.go`에 `renderView` switch에 `viewGraphs` 분기 추가
  - `renderGraphs(m *model, width int) string` 또는 `graph_view.go`의 통합 렌더 함수 연결
  - 그래프 뷰 헤더 모드명 추가: `Graphs`
  - `renderHelp(mode viewMode)`에 그래프 뷰 도움말 추가:
    - `Tab/Shift+Tab or ←→ switch graphs • Esc returns • r refresh • q quit`
  - 그래프 로딩 중 상태 메시지 추가
  - graphErr 발생 시 에러 패널 렌더링
  - 각 탭 empty state 일관화
  - `statusMessage`를 그래프 진입 시 적절히 설정 (`Viewing monthly model activity graphs.` 등)
  - 필요 시 `syncViewport()`에서 graph view의 content width/height를 고려해 차트 폭 계산 보정

  **Must NOT do**:
  - 기존 dashboard header/help 문구를 제거하거나 축소하지 않는다
  - graph view 전환 때문에 viewport scroll 동작을 깨뜨리지 않는다

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 8
  - **Blocked By**: Tasks 3, 4, 5, 6

  **References**:

  **Pattern References**:
  - `internal/adapters/tui/view.go:23-35` - renderView switch 구조. viewGraphs 분기 추가 위치.
  - `internal/adapters/tui/view.go:182-200` - renderHeader 패턴. 모드명과 statusMessage 표기 위치.
  - `internal/adapters/tui/view.go:320-330` - renderHelp 패턴. 새 모드 help string 추가.
  - `internal/adapters/tui/model.go:432-449` - syncViewport가 최종 content를 그리는 방식. graph view도 이 경로를 탄다.

  **Acceptance Criteria**:
  - [ ] graph view가 renderView switch에 연결됨
  - [ ] help text가 graph view에 맞게 출력됨
  - [ ] graphLoading/graphErr/empty state 모두 렌더링됨
  - [ ] viewport 내에서 차트가 잘리지 않고 표시됨

  **QA Scenarios**:

  ```
  Scenario: 그래프 뷰 도움말 및 헤더 검증
    Tool: interactive_bash
    Preconditions: Tasks 3-6 완료
    Steps:
      1. `go run ./cmd/tui` 실행 후 `g` 입력
      2. 헤더에 `Graphs` 또는 동등한 그래프 모드명이 있는지 확인
      3. 하단 help 문구에 `Esc returns`와 `switch graphs`가 포함되는지 확인
    Expected Result: 그래프 뷰 전용 헤더/도움말 표시
    Failure Indicators: dashboard help가 그대로 표시됨, 헤더 누락
    Evidence: .sisyphus/evidence/task-7-graph-help-header.txt

  Scenario: 그래프 에러/빈 상태 일관성 검증
    Tool: interactive_bash
    Preconditions: 빈 DB 또는 테스트 환경
    Steps:
      1. 빈 DB로 TUI 실행 후 graph view 진입
      2. 4개 탭을 모두 순회
      3. 각 탭에서 공백이 아닌 명시적 empty state 문구가 있는지 확인
    Expected Result: 각 탭마다 명확한 empty state 또는 에러 문구 표시
    Failure Indicators: blank viewport, panic, 잘못된 탭에서 이전 탭 내용 잔존
    Evidence: .sisyphus/evidence/task-7-empty-state-consistency.txt
  ```

  **Commit**: NO

- [ ] 8. 빌드 검증 + 전체 TUI 실행 QA

  **What to do**:
  - `go build ./cmd/tui` 실행
  - `go test ./...` 실행
  - `go vet ./...` 실행
  - 실제 TUI를 실행해 dashboard → graphs → dashboard 플로우 검증
  - 4개 탭 모두 스크린샷 또는 터미널 캡처 저장
  - 계획된 커밋 전략대로 정리

  **Must NOT do**:
  - 테스트 실패를 무시한 채 완료 처리하지 않는다
  - evidence 파일 없이 QA 완료 주장하지 않는다

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (final implementation task)
  - **Blocks**: Final Verification Wave
  - **Blocked By**: Task 7

  **References**:

  **Pattern References**:
  - `Makefile:3-7` - `test` / `build/tui` 명령 정의 참고
  - `internal/adapters/tui/run.go:18-58` - TUI 실행 엔트리포인트

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/tui` PASS
  - [ ] `go test ./...` PASS
  - [ ] `go vet ./...` PASS
  - [ ] 4개 그래프 탭 evidence 확보

  **QA Scenarios**:

  ```
  Scenario: 전체 빌드 및 테스트 통과
    Tool: Bash
    Preconditions: 모든 구현 task 완료
    Steps:
      1. `go build ./cmd/tui` 실행
      2. `go test ./...` 실행
      3. `go vet ./...` 실행
      4. 세 명령 모두 exit code 0 확인
    Expected Result: build/test/vet 모두 성공
    Failure Indicators: 컴파일 에러, 테스트 실패, vet warning
    Evidence: .sisyphus/evidence/task-8-build-test-vet.txt

  Scenario: end-to-end TUI 그래프 흐름 검증
    Tool: interactive_bash
    Preconditions: 빌드 성공
    Steps:
      1. `go run ./cmd/tui` 실행
      2. dashboard 진입 확인
      3. `g` 입력하여 graph view 진입
      4. `tab` 3회 입력하여 4개 탭 모두 순회
      5. 각 탭에서 모델/비용/추이/비율 관련 텍스트가 표시되는지 확인
      6. `esc` 입력하여 dashboard 복귀
      7. `q` 입력하여 종료
    Expected Result: 전체 플로우 성공, 4개 탭 모두 접근 가능
    Failure Indicators: 탭 접근 불가, 화면 깨짐, 복귀 실패, 종료 실패
    Evidence: .sisyphus/evidence/task-8-e2e-graph-flow.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): add monthly model activity and token usage graph views`
  - Files: `internal/adapters/tui/model.go`, `internal/adapters/tui/view.go`, `internal/adapters/tui/graph_view.go`, `internal/adapters/tui/run.go`
  - Pre-commit: `go build ./cmd/tui && go test ./... && go vet ./...`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./cmd/tui` + `go vet ./...` + `go test ./...`. Review all changed files for: unused imports, empty error handling, commented-out code, unused variables. Check for ntcharts API misuse. Verify lipgloss styles are consistent with existing codebase patterns.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Build TUI binary. Launch in tmux. Test: 'g' key opens graph view, Tab cycles through 4 graphs, Esc returns to dashboard. Verify each graph renders data or empty state. Take screenshots of each graph tab. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance: no mouse interaction, no DB migrations, no dashboard changes, no external API calls.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Group | Message | Files | Pre-commit |
|-------|---------|-------|------------|
| Task 1 | `chore(deps): add ntcharts v1 dependency for TUI graphs` | go.mod, go.sum | `go build ./...` |
| Task 2 | `feat(service): add graph aggregation query service for model/daily stats` | internal/service/graph_query.go, internal/service/graph_query_test.go | `go test ./internal/service/...` |
| Tasks 3-7 | `feat(tui): add token usage and model activity graph views` | internal/adapters/tui/model.go, internal/adapters/tui/view.go, internal/adapters/tui/graph_view.go | `go build ./cmd/tui && go test ./...` |

---

## Success Criteria

### Verification Commands
```bash
go build ./cmd/tui          # Expected: binary builds without errors
go test ./...               # Expected: all tests pass
go vet ./...                # Expected: no vet warnings
```

### Final Checklist
- [ ] 'g' 키로 그래프 뷰 진입 가능
- [ ] Tab으로 4개 그래프 탭 전환 가능
- [ ] Esc로 대시보드 복귀 가능
- [ ] 모델별 토큰 바 차트 정상 렌더링
- [ ] 모델별 비용 바 차트 정상 렌더링
- [ ] 일별 추이 라인 차트 정상 렌더링
- [ ] 토큰 타입 비율 정상 표시
- [ ] 빈 데이터 시 적절한 빈 상태 메시지
- [ ] 기존 대시보드 동작에 영향 없음
- [ ] All "Must NOT Have" absent
