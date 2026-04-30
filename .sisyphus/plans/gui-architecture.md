# GUI Architecture & Implementation: LLM Budget Tracker

## TL;DR

> **Quick Summary**: Wails 기반 GUI를 Svelte + TypeScript로 그린필드 재구축. Grafana/Datadog 스타일의 데이터 밀집 대시보드로 TUI의 모든 기능을 "아름답게" 구현. 다크/라이트 테마 + 시스템 알림 추가.
> 
> **Deliverables**:
> - 완전한 Svelte + TypeScript 프론트엔드 (기존 유실된 프론트엔드 대체)
> - Grafana/Datadog 스타일 대시보드 UI
> - TUI 기능 100% 패리티 + 다크/라이트 테마 + 알림 시스템
> - TDD 기반 테스트 스위트 (Vitest + Playwright)
> - Linux + Mac 크로스 플랫폼 지원
> 
> **Estimated Effort**: Large
> **Parallel Execution**: YES - 5 waves
> **Critical Path**: Task 1 → Task 4 → Task 7 → Task 13 → Task 19 → Task 22 → FINAL

---

## Context

### Original Request
GUI 버전을 새로 제작. 핵심은 TUI 버전의 모든 기능을 "아름답게" 만드는 것. Linux + Mac 지원.

### Interview Summary
**Key Discussions**:
- GUI Framework: **Wails** — 기존 인프라(백엔드 바인딩, wails.json) 재사용, 프론트엔드만 새로 구축
- Frontend: **Svelte + TypeScript** — 작은 번들, 컴파일 타임 최적화
- Design: **Grafana/Datadog 스타일** — 데이터 밀집, 다중 패널, 실시간 지표
- TUI/GUI: **공존** — 같은 SQLite DB 공유, 독립 실행
- Test: **TDD** (Vitest + Svelte Testing Library + Playwright)
- 추가 기능: 다크/라이트 테마 전환, 알림 시스템

**Research Findings**:
- 기존 Wails 백엔드 바인딩 존재: DashboardBinding, FormsBinding, SubscriptionLookupBinding
- 프론트엔드 소스코드 유실 — dist/ 산출물만 남아있어 완전한 재구축 필요
- TUI는 6개 뷰 제공: Dashboard, ManualEntryForm, SubscriptionForm, SubscriptionList, InsightList, Graphs
- Hexagonal Architecture로 GUI 어댑터 교체가 도메인/서비스 계층에 영향 없음

### Metis Review
**Identified Gaps** (addressed):
- TUI→GUI 기능 패리티 매트릭스 명시 필요 → 각 태스크에 패리티 검증 포함
- MVP vs 확장 기능 범위 모호 → "Must NOT Have" 섹션으로 명시적 제외
- 실시간 데이터 갱신 의미론 미정 → Mutation-triggered refresh + 수동 새로고침으로 결정
- 알림 의미론 미정 → 예산 임계값 초과 시 1회 알림, 동일 임계값 중복 방지
- SQLite 동시성 검증 필요 → WAL 모드 검증 태스크 포함
- 바인딩 감사(audit) 필요 → Wave 1에 포함
- 프론트엔드 아키텍처 상세 필요 → 본 계획에 명시

---

## TUI → GUI Feature Parity Matrix

| TUI View | TUI Feature | GUI Route/Panel | Status |
|----------|------------|-----------------|--------|
| Dashboard | Overview (spend, tokens, budget) | `/` Main Dashboard | NEW |
| Dashboard | Providers breakdown | `/` Providers Panel | NEW |
| Dashboard | Budgets summary | `/` Budget Panel | NEW |
| Dashboard | Recent Sessions | `/` Sessions Panel | NEW |
| ManualEntryForm | 9-field entry form | `/usage/new` Form Page | NEW |
| SubscriptionForm | Preset selector + 7-field form | `/subscriptions/new` Form Page | NEW |
| SubscriptionList | List with delete | `/subscriptions` List Page | NEW |
| InsightList | Dashboard + Logs tabs | `/insights` Page | NEW |
| InsightDetail | Insight detail view | `/insights/:id` Detail Modal | NEW |
| Graphs | 4 chart tabs | `/graphs` Page | NEW |
| *(NEW)* | Dark/Light theme toggle | Global toggle in sidebar | GUI-ONLY |
| *(NEW)* | System notifications | Background service | GUI-ONLY |
| *(NEW)* | Settings/Preferences | `/settings` Page | GUI-ONLY |

---

## Work Objectives

### Core Objective
Wails + Svelte + TypeScript로 Grafana/Datadog 스타일의 데이터 밀집 대시보드 GUI를 구축하여, TUI의 모든 기능을 아름답게 구현하고 다크/라이트 테마 + 알림 시스템을 추가한다.

### Concrete Deliverables
- `internal/adapters/gui/frontend/` — 완전한 Svelte + TypeScript 프론트엔드
- `internal/adapters/gui/bindings_client.go` — 프론트엔드용 바인딩 클라이언트 레이어 (필요시)
- `.svelte-kit/` 또는 Svelte 빌드 산출물
- Playwright E2E 테스트 스위트
- Vitest 유닛/컴포넌트 테스트 스위트

### Definition of Done
- [ ] `wails dev` 실행 시 GUI가 정상적으로 렌더링됨
- [ ] TUI의 6개 뷰 모두에 GUI 동등 기능 존재
- [ ] 다크/라이트 테마 전환 동작
- [ ] 예산 임계값 초과 시 시스템 알림 발생
- [ ] `npm test` (Vitest) 전체 통과
- [ ] Playwright E2E 전체 통과
- [ ] Linux + Mac에서 `wails build` 성공

### Must Have
- TUI 기능 100% 패리티 (Dashboard, Manual Entry, Subscription CRUD, Insights, Graphs)
- Grafana/Datadog 스타일 데이터 밀집 대시보드
- 다크/라이트 테마 토글 (디자인 토큰 기반)
- 시스템 알림 (예산 임계값 초과)
- TDD (Vitest + Playwright)
- Linux + Mac 지원
- 기존 Go 백엔드 바인딩 재사용
- TUI와 GUI의 SQLite DB 공유

### Must NOT Have (Guardrails)
- 드래그앤드롭 대시보드 패널 재배치
- 커스텀 대시보드 빌더 / 저장된 레이아웃
- 고급 필터링 / 쿼리 빌더
- CSV/PDF/이미지 내보내기
- 클라우드 동기화
- 인증 / 사용자 계정
- 백그라운드 알림 데몬
- 알림 스누즈 / 규칙 엔진
- 모바일 / 반응형 디자인
- 플러그인 아키텍처
- macOS 패키징 서명 / 공증 (별도 작업)
- 자동 업데이트 메커니즘
- Go 백엔드 도메인 로직 수정 (바인딩 추가만 허용)
- TypeScript에 비즈니스 로직 중복 구현

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: NO (프론트엔드 소스 유실로 재구축 필요)
- **Automated tests**: TDD (Red-Green-Refactor)
- **Framework**: Vitest + Svelte Testing Library (컴포넌트), Playwright (E2E)
- **TDD Flow**: 각 태스크는 RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Frontend/UI**: Playwright — Navigate, interact, assert DOM, screenshot
- **Backend Bindings**: Bash (curl to Wails dev server) — Send requests, assert response
- **Components**: Vitest + Svelte Testing Library — Render, interact, assert output

### Refresh Semantics (Defined)
- **Primary**: Mutation-triggered refresh — 데이터 변경 후 관련 스토어 자동 갱신
- **Secondary**: 수동 새로고침 버튼 — 사용자가 언제든 F5 또는 버튼으로 갱신
- **No polling**: 주기적 폴링 없음 (배터리/CPU 절약)

### Notification Semantics (Defined)
- **Trigger**: 예산 임계값 초과 시 (BudgetMonitor 서비스 기준)
- **Dedup**: 동일 임계값 레벨에 대해 앱 시작당 1회만 알림
- **Dismiss**: 사용자가 수동으로 닫거나 앱 재시작 시 초기화
- **No persistence**: 알림 상태는 DB에 저장하지 않음 (재계산 가능)

---

## Execution Strategy

### Frontend Architecture

```
internal/adapters/gui/frontend/
├── src/
│   ├── lib/
│   │   ├── bindings/          # Wails Go 바인딩 래퍼
│   │   │   ├── index.ts       # 바인딩 클라이언트 진입점
│   │   │   ├── dashboard.ts   # DashboardBinding 래핑
│   │   │   ├── forms.ts       # FormsBinding 래핑
│   │   │   └── subscriptions.ts # SubscriptionLookupBinding 래핑
│   │   ├── stores/            # Svelte 스토어 (상태 관리)
│   │   │   ├── budget.ts
│   │   │   ├── usage.ts
│   │   │   ├── subscription.ts
│   │   │   ├── waste.ts
│   │   │   ├── theme.ts
│   │   │   └── notification.ts
│   │   ├── components/        # 재사용 UI 컴포넌트
│   │   │   ├── ui/            # 기본 UI 프리미티브
│   │   │   ├── charts/        # ECharts 래퍼 컴포넌트
│   │   │   ├── tables/        # 데이터 테이블
│   │   │   └── forms/         # 폼 필드 + 검증
│   │   ├── styles/            # 디자인 토큰 + 글로벌 스타일
│   │   │   ├── tokens.css     # CSS 변수 (색상, 간격, 타이포)
│   │   │   └── tailwind.css   # TailwindCSS 진입점
│   │   └── utils/             # 유틸리티 함수
│   ├── routes/                # Svelte 라우팅 (SPA)
│   │   ├── +layout.svelte     # 사이드바 + 메인 영역
│   │   ├── +page.svelte       # 대시보드 (홈)
│   │   ├── /usage/
│   │   │   └── +page.svelte   # 사용량 추적 + 수동 입력
│   │   ├── /subscriptions/
│   │   │   ├── +page.svelte   # 구독 목록
│   │   │   └── /new/+page.svelte  # 구독 추가 폼
│   │   ├── /budgets/
│   │   │   └── +page.svelte   # 예산 관리
│   │   ├── /insights/
│   │   │   └── +page.svelte   # 낭비 감지 인사이트
│   │   ├── /graphs/
│   │   │   └── +page.svelte   # 4개 차트 탭
│   │   └── /settings/
│   │       └── +page.svelte   # 설정 + 테마 전환
│   └── app.html               # HTML 진입점
├── static/                    # 정적 에셋
├── tests/                     # 테스트
│   ├── unit/                  # Vitest 유닛 테스트
│   ├── component/             # Svelte Testing Library
│   └── e2e/                   # Playwright E2E
├── package.json
├── svelte.config.js
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
└── playwright.config.ts
```

### Technology Choices

| Concern | Choice | Rationale |
|---------|--------|-----------|
| UI Framework | Svelte 5 + TypeScript | 작은 번들, 적은 코드, 반응성 |
| Styling | TailwindCSS 4 | 디자인 토큰 기반, 다크/라이트 쉽게 전환 |
| Charts | ECharts (via svelte-echarts) | Grafana/Datadog 수준의 차트, 30+ 차트 타입 |
| UI Primitives | Skeleton UI v2 | Svelte 네이티브, 다크/라이트 내장 |
| Icons | Lucide Svelte | 가벼운 SVG 아이콘 세트 |
| Routing | SvelteKit (SPA mode) | 파일 기반 라우팅, Wails 호환 |
| State | Svelte Stores (writable/readable) | 내장 상태 관리, 충분함 |
| Forms | Superforms + Zod | SvelteKit 폼 검증 |
| Testing (Unit) | Vitest + Svelte Testing Library | 빠른 TDD 사이클 |
| Testing (E2E) | Playwright | 크로스 브라우저, 스크린샷 |
| Date | date-fns | 가벼운 날짜 처리 |

### Parallel Execution Waves

```
Wave 1 (Start Immediately — foundation + scaffolding):
├── Task 1:  Svelte + Wails 프론트엔드 스캐폴딩 + 빌드 설정 [quick]
├── Task 2:  디자인 시스템 토큰 (TailwindCSS + CSS 변수) [quick]
├── Task 3:  TypeScript 타입 정의 (Go 도메인 모델 미러링) [quick]
├── Task 4:  Wails 바인딩 감사 + 클라이언트 레이어 [deep]
├── Task 5:  테마 시스템 (다크/라이트 토글) [quick]
├── Task 6:  레이아웃 셸 (사이드바 + 메인 영역) [visual-engineering]
└── Task 7:  테스트 인프라 설정 (Vitest + Playwright) [quick]

Wave 2 (After Wave 1 — core component library):
├── Task 8:  ECharts 차트 컴포넌트 라이브러리 (depends: 2, 3, 4) [visual-engineering]
├── Task 9:  데이터 테이블 컴포넌트 (depends: 2, 3) [visual-engineering]
├── Task 10: 폼 컴포넌트 (Input, Select, Toggle, DatePicker) (depends: 2, 3) [visual-engineering]
├── Task 11: 카드/패널 컴포넌트 (depends: 2) [quick]
├── Task 12: Svelte 스토어 (budget, usage, subscription, waste) (depends: 3, 4) [deep]
└── Task 13: 알림 서비스 (depends: 4, 5) [unspecified-high]

Wave 3 (After Wave 2 — feature screens):
├── Task 14: 대시보드 화면 (Overview + Providers + Budgets + Sessions) (depends: 8, 9, 11, 12) [deep]
├── Task 15: 사용량 추적 화면 (수동 입력 + 히스토리) (depends: 10, 12) [unspecified-high]
├── Task 16: 구독 관리 화면 (목록 + 추가 폼) (depends: 9, 10, 12) [unspecified-high]
├── Task 17: 예산 관리 화면 (설정 + 모니터링) (depends: 10, 12, 13) [unspecified-high]
├── Task 18: 인사이트 화면 (낭비 감지 대시보드) (depends: 8, 9, 11, 12) [deep]
└── Task 19: 그래프 화면 (4개 차트 탭) (depends: 8, 12) [visual-engineering]

Wave 4 (After Wave 3 — integration + cross-platform):
├── Task 20: 설정 화면 + 테마/알림 환경설정 (depends: 5, 10, 13) [quick]
├── Task 21: Wails 바인딩 전체 연동 + mutation-triggered refresh (depends: 14-19) [deep]
├── Task 22: Playwright E2E 테스트 스위트 (depends: 21) [unspecified-high]
└── Task 23: Linux + Mac 크로스 플랫폼 빌드 + 검증 (depends: 22) [deep]

Wave FINAL (After ALL tasks — 4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Task 1 → Task 4 → Task 12 → Task 14 → Task 21 → Task 22 → Task 23 → F1-F4 → user okay
Parallel Speedup: ~65% faster than sequential
Max Concurrent: 7 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | - | 4, 7 | 1 |
| 2 | - | 5, 8, 9, 10, 11 | 1 |
| 3 | - | 4, 8, 9, 10, 12 | 1 |
| 4 | 1, 3 | 8, 12, 13 | 1 |
| 5 | 2 | 13, 20 | 1 |
| 6 | 2 | 14-19 | 1 |
| 7 | 1 | 22 | 1 |
| 8 | 2, 3, 4 | 14, 18, 19 | 2 |
| 9 | 2, 3 | 14, 16, 18 | 2 |
| 10 | 2, 3 | 15, 16, 17, 20 | 2 |
| 11 | 2 | 14, 18 | 2 |
| 12 | 3, 4 | 14-19 | 2 |
| 13 | 4, 5 | 17, 20 | 2 |
| 14 | 8, 9, 11, 12 | 21 | 3 |
| 15 | 10, 12 | 21 | 3 |
| 16 | 9, 10, 12 | 21 | 3 |
| 17 | 10, 12, 13 | 21 | 3 |
| 18 | 8, 9, 11, 12 | 21 | 3 |
| 19 | 8, 12 | 21 | 3 |
| 20 | 5, 10, 13 | 21 | 4 |
| 21 | 14-20 | 22 | 4 |
| 22 | 21, 7 | 23 | 4 |
| 23 | 22 | FINAL | 4 |

### Agent Dispatch Summary

- **Wave 1**: **7 tasks** — T1,7 → `quick`, T2,3,5 → `quick`, T4 → `deep`, T6 → `visual-engineering`
- **Wave 2**: **6 tasks** — T8,9,10 → `visual-engineering`, T11 → `quick`, T12 → `deep`, T13 → `unspecified-high`
- **Wave 3**: **6 tasks** — T14,18 → `deep`, T15,16,17 → `unspecified-high`, T19 → `visual-engineering`
- **Wave 4**: **4 tasks** — T20 → `quick`, T21 → `deep`, T22 → `unspecified-high`, T23 → `deep`
- **FINAL**: **4 tasks** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Svelte + Wails 프론트엔드 스캐폴딩 + 빌드 설정

  **What to do**:
  - `internal/adapters/gui/frontend/` 디렉토리에 SvelteKit SPA 프로젝트 초기화 (`npm create svelte@latest`)
  - TypeScript 활성화
  - `wails.json`의 `frontend:install` 및 `frontend:build` 명령어가 새 Svelte 빌드와 호환되도록 업데이트
  - `package.json`에 필수 의존성 추가: tailwindcss, echarts, svelte-echarts, lucide-svelte, date-fns, zod
  - `vite.config.ts`에 Wails 플러그인 설정 (`@wailsapp/wails/vite`)
  - 기존 `dist/` 산출물 삭제 후 새 빌드 파이프라인 확인
  - `npm run build` 성공 확인
  - TDD: `vitest` 설정 파일 작성, 샘플 테스트 1개 작성하여 통과 확인

  **Must NOT do**:
  - 기존 `internal/adapters/gui/app.go` 수정 금지
  - 기존 `cmd/gui/main.go` 수정 금지
  - TUI 코드 수정 금지

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 스캐폴딩과 설정은 표준화된 작업
  - **Skills**: [`/frontend-ui-ux`]
    - `/frontend-ui-ux`: Svelte + Wails 설정 전문 지식

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 5)
  - **Blocks**: Tasks 4, 7
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/adapters/gui/app.go` — 기존 Wails 앱 설정, Bind 배열에 등록된 바인딩 구조체 확인
  - `cmd/gui/main.go` — GUI 엔트리포인트, Run() 호출 방식
  - `wails.json` — 프론트엔드 빌드 설정 (frontendDir, build command)

  **External References**:
  - Wails Svelte 템플릿: https://wails.io/docs/guides/svelte
  - SvelteKit SPA 모드: https://kit.svelte.dev/docs/single-page-apps

  **WHY Each Reference Matters**:
  - `app.go`: 바인딩 배열 구조를 알아야 프론트엔드에서 호출 가능한 메서드 파악
  - `wails.json`: 빌드 명령어가 새 프론트엔드와 호환되어야 함

  **Acceptance Criteria**:
  - [ ] `internal/adapters/gui/frontend/package.json` 존재
  - [ ] `npm run build` 성공 (에러 없이 dist/ 생성)
  - [ ] 샘플 Vitest 테스트 1개 통과
  - [ ] `wails dev` 실행 시 빈 창이라도 Wails 앱 로드됨

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: Svelte 프로젝트 빌드 성공
    Tool: Bash
    Preconditions: internal/adapters/gui/frontend/ 디렉토리에 package.json 존재
    Steps:
      1. cd internal/adapters/gui/frontend && npm install
      2. npm run build
    Expected Result: 빌드 에러 없이 dist/ 디렉토리 생성됨
    Failure Indicators: 빌드 실패, TypeScript 에러, 번들링 에러
    Evidence: .sisyphus/evidence/task-1-svelte-build.txt

  Scenario: Vitest 테스트 실행 성공
    Tool: Bash
    Preconditions: vitest 설정 완료
    Steps:
      1. cd internal/adapters/gui/frontend && npm test
    Expected Result: 최소 1개 테스트 통과, 0 failures
    Failure Indicators: 테스트 러너 실행 실패, 0 tests found
    Evidence: .sisyphus/evidence/task-1-vitest-run.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(gui): scaffold Svelte + Wails frontend`
  - Files: `internal/adapters/gui/frontend/*`, `wails.json`
  - Pre-commit: `cd internal/adapters/gui/frontend && npm run build`

- [x] 2. 디자인 시스템 토큰 (TailwindCSS + CSS 변수)

  **What to do**:
  - TailwindCSS 4 설치 및 설정 (`tailwind.config.ts`)
  - CSS 변수 기반 디자인 토큰 정의:
    - **색상**: Primary (blue/cyan 계열), Success (green), Warning (amber), Danger (red), Muted (gray)
    - **다크 테마**: 배경 #0f1117 (Grafana 스타일), 카드 #1a1d23, 텍스트 #d8d9da
    - **라이트 테마**: 배경 #f5f5f5, 카드 #ffffff, 텍스트 #1a1d23
    - **간격**: xs(2px), sm(4px), md(8px), lg(16px), xl(24px), 2xl(32px)
    - **타이포**: 모노스페이스 (JetBrains Mono), 산스 (Inter)
  - Grafana/Datadog 스타일에 맞는 데이터 밀집 레이아웃 토큰 (패널 보더, 패딩, 글로우 효과)
  - 상태 색상 토큰: 정상(green), 경고(yellow), 위험(red), 비활성(gray)
  - TDD: 테마 토큰 값 검증 테스트 작성

  **Must NOT do**:
  - 인라인 스타일 사용 금지 (모두 토큰/클래스로)
  - 하드코딩 색상값 사용 금지

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: CSS 변수 설정은 정형화된 작업
  - **Skills**: [`/frontend-ui-ux`]
    - `/frontend-ui-ux`: 디자인 시스템 구축 전문 지식

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 5)
  - **Blocks**: Tasks 5, 8, 9, 10, 11
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/adapters/tui/view.go` — TUI 색상 체계 (lipgloss 색상 241, 86, 244, 203) 참고용

  **External References**:
  - Grafana UI 색상: https://developers.grafana.com/ui/latest/
  - TailwindCSS Dark Mode: https://tailwindcss.com/docs/dark-mode

  **WHY Each Reference Matters**:
  - TUI 색상 체계를 참고하여 GUI에서도 유사한 시각적 일관성 유지

  **Acceptance Criteria**:
  - [ ] `src/lib/styles/tokens.css` 에 CSS 변수 정의됨 (최소 30개 변수)
  - [ ] `tailwind.config.ts` 에 커스텀 색상/간격/타포 설정됨
  - [ ] 다크 테마 기본값 적용 시 배경색 #0f1117 표시
  - [ ] 라이트 테마 전환 시 배경색 #f5f5f5 표시

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: 다크 테마 기본값 렌더링
    Tool: Playwright
    Preconditions: Svelte 앱 실행 중 (wails dev 또는 vite dev)
    Steps:
      1. 브라우저에서 localhost:34115 접속
      2. body 요소의 background-color 확인
    Expected Result: background-color가 #0f1117 (또는 rgb(15,17,23))
    Failure Indicators: 배경색이 흰색이거나 다른 값
    Evidence: .sisyphus/evidence/task-2-dark-theme.png

  Scenario: 라이트 테마 전환
    Tool: Playwright
    Preconditions: 다크 테마 렌더링 상태
    Steps:
      1. 테마 토글 요소 클릭 (아직 UI 없으므로 document.documentElement.classList 토글)
      2. body 요소의 background-color 확인
    Expected Result: background-color가 #f5f5f5 (또는 rgb(245,245,245))
    Evidence: .sisyphus/evidence/task-2-light-theme.png
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(gui): add design system tokens and TailwindCSS`
  - Files: `internal/adapters/gui/frontend/src/lib/styles/*`, `tailwind.config.ts`

- [x] 3. TypeScript 타입 정의 (Go 도메인 모델 미러링)

  **What to do**:
  - Go 도메인 모델을 TypeScript 인터페이스로 미러링:
    - `MonthlyBudget`, `BudgetStatus`, `ForecastSnapshot`, `BudgetState`
    - `UsageEntry`, `TokenUsage`, `CostBreakdown`
    - `Subscription`, `SubscriptionFee`
    - `WasteSummary` (waste metrics, trends, top causes)
  - Wails 바인딩 응답 타입 정의
  - 폼 입력 타입 정의 (ManualEntryInput, SubscriptionInput, BudgetInput)
  - 알림 타입 정의 (ThresholdAlert, NotificationState)
  - TDD: 타입 가드 및 변환 함수 테스트

  **Must NOT do**:
  - 비즈니스 로직을 TypeScript에 구현 금지 (Go가 단일 진실 공급원)
  - Go에 없는 타입 추가 금지 (바인딩 응답 래퍼 제외)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: 타입 정의는 기계적 변환 작업
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 5)
  - **Blocks**: Tasks 4, 8, 9, 10, 12
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/domain/budget.go` — MonthlyBudget, BudgetStatus, ForecastSnapshot, BudgetState 구조체
  - `internal/domain/usage.go` — UsageEntry, TokenUsage, CostBreakdown 구조체
  - `internal/domain/subscription.go` — Subscription, SubscriptionFee 구조체
  - `internal/domain/waste_summary.go` — WasteSummary 구조체

  **WHY Each Reference Matters**:
  - 각 Go 구조체의 필드명, 타입, JSON 태그를 정확히 미러링해야 Wails 바인딩 직렬화가 호환됨

  **Acceptance Criteria**:
  - [ ] `src/lib/types/domain.ts` 파일에 4개 이상의 주요 인터페이스 정의
  - [ ] `src/lib/types/forms.ts` 파일에 폼 입력 타입 정의
  - [ ] `src/lib/types/notifications.ts` 파일에 알림 타입 정의
  - [ ] 모든 인터페이스가 Go 구조체와 필드명 일치 (JSON 태그 기준)
  - [ ] `npm test` 타입 가드 테스트 통과

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: 타입 정의 파일 존재 및 유효성
    Tool: Bash
    Preconditions: 타입 파일 작성 완료
    Steps:
      1. npx tsc --noEmit internal/adapters/gui/frontend/src/lib/types/*.ts
    Expected Result: TypeScript 컴파일 에러 없음
    Failure Indicators: 타입 에러, 순환 참조 에러
    Evidence: .sisyphus/evidence/task-3-typecheck.txt

  Scenario: Go-TypeScript 필드명 일치 검증
    Tool: Bash
    Preconditions: Go 도메인 모델과 TS 타입 모두 존재
    Steps:
      1. Go 구조체 JSON 태그에서 필드명 추출
      2. TS 인터페이스에서 동일 필드명 존재 확인
    Expected Result: MonthlyBudget의 모든 JSON 필드가 TS 인터페이스에 존재
    Failure Indicators: 누락된 필드, 불일치한 이름
    Evidence: .sisyphus/evidence/task-3-field-parity.txt
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(gui): add TypeScript type definitions mirroring Go domain`
  - Files: `internal/adapters/gui/frontend/src/lib/types/*.ts`

- [x] 4. Wails 바인딩 감사 + 클라이언트 레이어

  **What to do**:
  - 기존 바인딩 전수 조사:
    - `internal/adapters/gui/dashboard_binding.go` — 메서드 시그니처, 파라미터, 반환 타입
    - `internal/adapters/gui/forms_binding.go` — 메서드 시그니처, 파라미터, 반환 타입
    - `internal/adapters/gui/app.go` — Bind 배열에 등록된 모든 바인딩
  - TUI 기능 대비 누락된 바인딩 식별:
    - InsightList/InsightDetail 조회 바인딩 존재 여부
    - Graph 데이터 조회 바인딩 존재 여부
    - Usage 히스토리 조회 바인딩 존재 여부
  - 누락된 바인딩이 있으면 `bindings_client.go`에 신규 바인딩 추가
  - 프론트엔드 바인딩 클라이언트 레이어 작성:
    - `src/lib/bindings/index.ts` — Wails JS 바인딩 임포트
    - `src/lib/bindings/dashboard.ts` — DashboardBinding 래핑
    - `src/lib/bindings/forms.ts` — FormsBinding 래핑
    - `src/lib/bindings/subscriptions.ts` — SubscriptionLookupBinding 래핑
  - 각 래퍼에 에러 핸들링 및 타입 캐스팅 적용
  - TDD: 모든 바인딩 래퍼에 대한 모킹 테스트

  **Must NOT do**:
  - Go 백엔드 비즈니스 로직 수정 금지
  - TUI 코드 수정 금지
  - 기존 바인딩 메서드 시그니처 변경 금지 (추가만 허용)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: 바인딩 감사는 분석이 필요하고, 누락 식별 후 신규 바인딩 추가는 Go 코드 작성 포함
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Task 1, 3)
  - **Parallel Group**: Wave 1 (after Tasks 1, 3)
  - **Blocks**: Tasks 8, 12, 13
  - **Blocked By**: Tasks 1, 3

  **References**:

  **Pattern References**:
  - `internal/adapters/gui/app.go` — Wails 앱 설정, Bind: []interface{} 배열
  - `internal/adapters/gui/dashboard_binding.go` — DashboardBinding 메서드
  - `internal/adapters/gui/forms_binding.go` — FormsBinding 메서드

  **API/Type References**:
  - `internal/ports/repository.go` — Repository 인터페이스 (바인딩이 호출해야 하는 메서드)
  - `internal/ports/ingestion.go` — IngestionService, InsightDetector 인터페이스
  - `internal/service/` — 모든 서비스 메서드 (바인딩에서 노출 가능한 것)

  **WHY Each Reference Matters**:
  - `app.go`: 현재 등록된 바인딩만 프론트엔드에서 호출 가능
  - `forms_binding.go`: 어떤 폼 기능이 이미 바인딩되어 있는지 파악
  - `ports/`: 서비스 계층이 제공하는 기능 전체 파악 → 누락 식별

  **Acceptance Criteria**:
  - [ ] 바인딩 감사 문서 작성: 각 바인딩의 메서드, 파라미터, 반환 타입
  - [ ] TUI 기능 대비 누락 바인딩 리스트업
  - [ ] 누락된 바인딩에 대한 Go 코드 추가 (필요시)
  - [ ] `src/lib/bindings/` 에 모든 바인딩 래퍼 TypeScript 파일 존재
  - [ ] 각 래퍼에 에러 핸들링 포함
  - [ ] 모킹 기반 Vitest 테스트 통과

  **QA Scenarios (MANDATORY):**

  ```
  Scenario: 바인딩 래퍼 타입 안전성
    Tool: Bash
    Preconditions: 바인딩 래퍼 파일 존재
    Steps:
      1. npx tsc --noEmit internal/adapters/gui/frontend/src/lib/bindings/*.ts
    Expected Result: TypeScript 컴파일 에러 없음
    Failure Indicators: any 타입 에러, 누락된 타입
    Evidence: .sisyphus/evidence/task-4-binding-types.txt

  Scenario: 누락 바인딩 식별 완료
    Tool: Bash
    Preconditions: 감사 완료
    Steps:
      1. 바인딩 감사 결과 파일에서 "MISSING" 항목 확인
    Expected Result: 모든 TUI 기능에 대해 바인딩 존재 또는 추가됨으로 표시
    Failure Indicators: TUI 기능에 해당하는 바인딩이 MISSING 상태로 남아있음
    Evidence: .sisyphus/evidence/task-4-binding-audit.md
  ```

  **Commit**: YES (groups with Wave 1)
  - Message: `feat(gui): audit Wails bindings and add client layer`
  - Files: `internal/adapters/gui/frontend/src/lib/bindings/*`, `internal/adapters/gui/*_binding.go` (필요시)

- [x] 5. 테마 시스템 (다크/라이트 토글)

  **What to do**:
  - `src/lib/stores/theme.ts` — Svelte writable store로 테마 상태 관리
  - `localStorage`에 테마 설정 영속화
  - OS 시스템 다크모드 감지 (`prefers-color-scheme` media query)
  - 초기 로드 시 저장된 테마 또는 OS 설정 기반 자동 선택
  - 테마 토글 함수: `toggleTheme()`, `setTheme('dark'|'light')`
  - `<html>` 요소의 `class` 에 `dark`/`light` 토글
  - TDD: 테마 store 동작 테스트 (초기값, 토글, 영속화)

  **Must NOT do**:
  - 컴포넌트별 인라인 테마 로직 금지
  - CSS 변수 외의 하드코딩 색상 금지

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (depends only on Task 2)
  - **Parallel Group**: Wave 1 (after Task 2)
  - **Blocks**: Tasks 13, 20
  - **Blocked By**: Task 2

  **References**:
  - `src/lib/styles/tokens.css` (Task 2 산출물) — CSS 변수 정의

  **Acceptance Criteria**:
  - [ ] `src/lib/stores/theme.ts` 존재
  - [ ] 테마 토글 시 `document.documentElement.classList` 변경됨
  - [ ] localStorage에 테마 설정 저장됨
  - [ ] OS 다크모드 감지 동작
  - [ ] Vitest 테스트 통과

  **QA Scenarios:**
  ```
  Scenario: 테마 토글 동작
    Tool: Playwright
    Steps:
      1. 페이지 로드
      2. 초기 테마 상태 확인 (localStorage)
      3. 테마 토글 실행 (스토어 직접 호출)
      4. html classList에 'dark'/'light' 전환 확인
    Expected Result: dark ↔ light 전환, localStorage 업데이트
    Evidence: .sisyphus/evidence/task-5-theme-toggle.png

  Scenario: OS 다크모드 자동 감지
    Tool: Playwright
    Steps:
      1. prefers-color-scheme: dark로 에뮬레이션
      2. localStorage 클리어 후 페이지 로드
      3. 초기 테마가 dark인지 확인
    Expected Result: 자동으로 dark 테마 적용
    Evidence: .sisyphus/evidence/task-5-os-dark-detect.txt
  ```

  **Commit**: YES
  - Message: `feat(gui): add theme system with dark/light toggle`
  - Files: `src/lib/stores/theme.ts`

- [x] 6. 레이아웃 셸 (사이드바 + 메인 영역)

  **What to do**:
  - Grafana/Datadog 스타일 레이아웃 구현:
    - **왼쪽 사이드바** (64px collapsed / 240px expanded):
      - 앱 로고 + 이름
      - 네비게이션 메뉴: Dashboard, Usage, Subscriptions, Budgets, Insights, Graphs, Settings
      - 각 메뉴 아이콘 (Lucide) + 라벨
      - 현재 활성 메뉴 하이라이트
      - 하단에 테마 토글 버튼
    - **메인 영역** (나머지 공간):
      - 상단 헤더바 (현재 페이지 타이틀 + 새로고침 버튼)
      - 콘텐츠 영역 (라우트별 페이지 렌더링)
  - `+layout.svelte` 에 레이아웃 셸 구현
  - SvelteKit SPA 라우팅 설정 (`/`, `/usage`, `/subscriptions`, `/budgets`, `/insights`, `/graphs`, `/settings`)
  - 반응형 사이드바 (접기/펼치기 토글)
  - TDD: 레이아웃 컴포넌트 렌더링 테스트

  **Must NOT do**:
  - 드래그앤드롭 패널 재배치 금지
  - 커스텀 대시보드 레이아웃 저장 금지

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: 레이아웃은 시각적 품질이 핵심
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES (depends only on Task 2)
  - **Parallel Group**: Wave 1 (after Task 2)
  - **Blocks**: Tasks 14-19
  - **Blocked By**: Task 2

  **References**:
  - `internal/adapters/tui/model.go` — TUI의 viewMode enum (viewDashboard, viewManualEntryForm, viewSubscriptionForm, etc.) — GUI 라우트와 1:1 매핑

  **Acceptance Criteria**:
  - [ ] 사이드바에 7개 네비게이션 메뉴 표시
  - [ ] 메뉴 클릭 시 해당 라우트로 이동
  - [ ] 활성 메뉴 하이라이트 표시
  - [ ] 사이드바 접기/펼치기 동작
  - [ ] 테마 토글 버튼 표시
  - [ ] Vitest 렌더링 테스트 통과

  **QA Scenarios:**
  ```
  Scenario: 사이드바 네비게이션 동작
    Tool: Playwright
    Steps:
      1. 앱 로드
      2. 사이드바에서 "Subscriptions" 클릭
      3. URL이 /subscriptions로 변경 확인
      4. 활성 메뉴 하이라이트 확인
    Expected Result: /subscriptions 라우트로 이동, 해당 메뉴 활성 표시
    Evidence: .sisyphus/evidence/task-6-sidebar-nav.png

  Scenario: 사이드바 접기/펼치기
    Tool: Playwright
    Steps:
      1. 사이드바 펼침 상태에서 토글 버튼 클릭
      2. 사이드바 너비가 64px로 변경 확인
      3. 라벨 숨김, 아이콘만 표시 확인
    Expected Result: 사이드바 collapsed 상태 전환
    Evidence: .sisyphus/evidence/task-6-sidebar-collapse.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add layout shell with sidebar navigation`
  - Files: `src/routes/+layout.svelte`, `src/lib/components/layout/*`

- [x] 7. 테스트 인프라 설정 (Vitest + Playwright)

  **What to do**:
  - Vitest 설정:
    - `vitest.config.ts` — Svelte 플러그인, jsdom 환경
    - `@testing-library/svelte` 설정
    - Wails 바인딩 모킹 유틸리티 (`__mocks__/wailsjs/`)
    - 글로벌 setup 파일
  - Playwright 설정:
    - `playwright.config.ts` — Wails dev 서버 대상
    - 테스트 헬퍼: 인증, 시드 데이터, 페이지 객체
    - 스크린샷/비디오 캡처 설정
  - CI 명령어: `npm test` (Vitest), `npx playwright test` (E2E)
  - 테스트 커버리지 설정 (v8 provider)
  - 샘플 테스트 각각 1개씩 작성하여 인프라 검증

  **Must NOT do**:
  - 실제 Wails 바이너리 빌드를 E2E 테스트에서 요구 금지 (dev 서버 사용)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (depends only on Task 1)
  - **Parallel Group**: Wave 1 (after Task 1)
  - **Blocks**: Task 22
  - **Blocked By**: Task 1

  **References**:
  - `wails.json` — Wails dev 서버 포트 (기본 34115)

  **Acceptance Criteria**:
  - [ ] `vitest.config.ts` 존재, `npm test` 실행 가능
  - [ ] `playwright.config.ts` 존재, `npx playwright test` 실행 가능
  - [ ] Wails 바인딩 모킹 유틸리티 존재
  - [ ] 샘플 Vitest 테스트 1개 통과
  - [ ] 샘플 Playwright 테스트 1개 통과

  **QA Scenarios:**
  ```
  Scenario: Vitest 인프라 동작
    Tool: Bash
    Steps:
      1. cd internal/adapters/gui/frontend && npm test
    Expected Result: 테스트 러너 실행, 최소 1개 테스트 통과
    Evidence: .sisyphus/evidence/task-7-vitest-infra.txt

  Scenario: Playwright 인프라 동작
    Tool: Bash
    Steps:
      1. npx playwright test --config=playwright.config.ts
    Expected Result: 테스트 러너 실행, 브라우저 연동
    Evidence: .sisyphus/evidence/task-7-playwright-infra.txt
  ```

  **Commit**: YES
  - Message: `feat(gui): add Vitest + Playwright test infrastructure`
  - Files: `vitest.config.ts`, `playwright.config.ts`, `__mocks__/*`, `tests/*`

- [x] 8. ECharts 차트 컴포넌트 라이브러리

  **What to do**:
  - ECharts 기반 재사용 차트 컴포넌트 작성:
    - `LineChart.svelte` — 시계열 라인 차트 (일별 토큰 트렌드, 낭비 트렌드)
    - `BarChart.svelte` — 바 차트 (모델별 토큰 사용량, 비용)
    - `PieChart.svelte` — 파이/도넛 차트 (모델 토큰 비율)
    - `StackedBarChart.svelte` — 스택 바 차트 (모델 토큰 분해)
  - 모든 차트 공통 기능:
    - 다크/라이트 테마 자동 전환 (ECharts theme 객체)
    - 반응형 크기 조절 (ResizeObserver)
    - 로딩 상태 표시
    - 빈 데이터 상태 표시 ("No data available")
    - 툴팁 포맷팅
  - `ChartContainer.svelte` — 공통 래퍼 (제목, 로딩, 에러 처리)
  - TDD: 각 차트 컴포넌트 렌더링 테스트 (모킹 데이터)

  **Must NOT do**:
  - D3 원본 사용 금지 (ECharts로 통일)
  - 차트에 비즈니스 로직 포함 금지

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 14, 18, 19
  - **Blocked By**: Tasks 2, 3, 4

  **References**:
  - `internal/adapters/tui/insights_dashboard_view.go` — TUI 차트 렌더링 (ntcharts 사용), 어떤 데이터를 어떤 차트로 표시하는지
  - `internal/domain/waste_summary.go` — 차트에 표시할 낭비 데이터 구조

  **Acceptance Criteria**:
  - [ ] 4개 차트 컴포넌트 파일 존재
  - [ ] ChartContainer.svelte 존재
  - [ ] 모든 차트 다크/라이트 테마 전환 동작
  - [ ] 빈 데이터 상태 렌더링
  - [ ] 반응형 리사이즈 동작
  - [ ] Vitest 테스트 통과 (각 차트 1개 이상)

  **QA Scenarios:**
  ```
  Scenario: LineChart 렌더링
    Tool: Playwright
    Steps:
      1. 차트 컴포넌트가 있는 테스트 페이지 로드
      2. 모킹 데이터로 LineChart 렌더링
      3. canvas 요소 존재 확인
      4. 차트 제목 텍스트 확인
    Expected Result: 차트 canvas 렌더링, 제목 표시, 툴팁 동작
    Evidence: .sisyphus/evidence/task-8-line-chart.png

  Scenario: 빈 데이터 상태
    Tool: Playwright
    Steps:
      1. 빈 데이터로 LineChart 렌더링
      2. "No data available" 텍스트 확인
    Expected Result: 에러 없이 빈 상태 메시지 표시
    Evidence: .sisyphus/evidence/task-8-empty-chart.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add ECharts chart component library`
  - Files: `src/lib/components/charts/*.svelte`

- [x] 9. 데이터 테이블 컴포넌트

  **What to do**:
  - Grafana/Datadog 스타일 데이터 테이블:
    - `DataTable.svelte` — 제네릭 데이터 테이블
      - 컬럼 정의 (키, 라벨, 정렬 가능, 포맷터)
      - 행 클릭 핸들러
      - 빈 상태 표시
      - 로딩 스켈레톤
      - 교대 행 색상 (다크/라이트 테마 지원)
    - `StatusBadge.svelte` — 상태 표시 뱃지 (Active, Expired, Over Budget 등)
    - `CurrencyCell.svelte` — 통화 포맷팅 셀 ($20.00)
    - `TokenCell.svelte` — 토큰 수 포맷팅 셀 (1.2K, 1.5M)
    - `DateCell.svelte` — 날짜 포맷팅 셀
  - TDD: 각 셀 컴포넌트 포맷팅 테스트

  **Must NOT do**:
  - 고급 필터링/검색/페이지네이션 금지 (TUI에 없음)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 14, 16, 18
  - **Blocked By**: Tasks 2, 3

  **References**:
  - `internal/adapters/tui/view.go` — TUI 테이블 렌더링 방식 (구독 목록 등)
  - `internal/domain/subscription.go` — Subscription 필드 (Date, Provider, Plan, Fee, Renewal Day, Active)

  **Acceptance Criteria**:
  - [ ] DataTable.svelte 존재, 제네릭 타입 지원
  - [ ] StatusBadge, CurrencyCell, TokenCell, DateCell 존재
  - [ ] 모든 셀 다크/라이트 테마 지원
  - [ ] 빈 상태 표시
  - [ ] Vitest 테스트 통과

  **QA Scenarios:**
  ```
  Scenario: DataTable 렌더링
    Tool: Playwright
    Steps:
      1. 모킹 데이터 3행으로 DataTable 렌더링
      2. 테이블 행 3개 표시 확인
      3. 교대 행 색상 확인
    Expected Result: 3개 행 렌더링, 교대 색상 적용
    Evidence: .sisyphus/evidence/task-9-datatable.png

  Scenario: CurrencyCell 포맷팅
    Tool: Bash (Vitest)
    Steps:
      1. CurrencyCell 컴포넌트 테스트 실행
    Expected Result: 20 → "$20.00", 1234.5 → "$1,234.50"
    Evidence: .sisyphus/evidence/task-9-currency-cell.txt
  ```

  **Commit**: YES
  - Message: `feat(gui): add DataTable and cell formatter components`
  - Files: `src/lib/components/tables/*.svelte`

- [x] 10. 폼 컴포넌트 (Input, Select, Toggle, DatePicker)

  **What to do**:
  - Grafana/Datadog 스타일 폼 프리미티브:
    - `TextInput.svelte` — 텍스트 입력 (라벨, 플레이스홀더, 검증 에러)
    - `NumberInput.svelte` — 숫자 입력 (min, max, step)
    - `SelectInput.svelte` — 셀렉트 드롭다운 (옵션 리스트)
    - `Toggle.svelte` — 불리언 토글 스위치
    - `DatePicker.svelte` — 날짜 선택 (native date input + 포맷팅)
    - `Form.svelte` — 폼 래퍼 (제출 핸들링, 검증 에러 표시)
    - `FormField.svelte` — 필드 래퍼 (라벨 + 에러 메시지)
  - Zod 스키마 기반 폼 검증 연동
  - 모든 폼 컴포넌트 다크/라이트 테마 지원
  - TDD: 각 입력 컴포넌트 렌더링 + 검증 테스트

  **Must NOT do**:
  - 커스텀 date picker 위젯 구현 금지 (native 사용)
  - 폼에 비즈니스 로직 포함 금지

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 15, 16, 17, 20
  - **Blocked By**: Tasks 2, 3

  **References**:
  - `internal/adapters/tui/view.go` — TUI 폼 필드 구조 (ManualEntryForm 9필드, SubscriptionForm 7필드)
  - `internal/domain/usage.go` — UsageEntry 필드 (Provider, Model, InputTokens, OutputTokens, etc.)

  **Acceptance Criteria**:
  - [ ] 6개 폼 컴포넌트 + Form 래퍼 존재
  - [ ] Zod 검증 연동 동작
  - [ ] 검증 에러 표시
  - [ ] 다크/라이트 테마 지원
  - [ ] Vitest 테스트 통과

  **QA Scenarios:**
  ```
  Scenario: TextInput 검증 에러 표시
    Tool: Playwright
    Steps:
      1. 필수 TextInput 필드가 있는 폼 렌더링
      2. 빈 값으로 제출
      3. 에러 메시지 표시 확인
    Expected Result: "This field is required" 에러 메시지 표시
    Evidence: .sisyphus/evidence/task-10-form-validation.png

  Scenario: SelectInput 옵션 선택
    Tool: Playwright
    Steps:
      1. 옵션이 있는 SelectInput 렌더링
      2. 특정 옵션 선택
      3. 선택값 확인
    Expected Result: 선택한 옵션값 반영
    Evidence: .sisyphus/evidence/task-10-select-input.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add form components with Zod validation`
  - Files: `src/lib/components/forms/*.svelte`

- [x] 11. 카드/패널 컴포넌트

  **What to do**:
  - Grafana/Datadog 스타일 패널:
    - `Panel.svelte` — 기본 패널 컨테이너 (제목, 액션 버튼, 콘텐츠 슬롯)
    - `StatCard.svelte` — 숫자 지표 카드 (값, 라벨, 트렌드 화살표, 색상)
    - `SparklineCard.svelte` — 미니 차트가 있는 지표 카드
    - `AlertCard.svelte` — 경고/알림 카드 (아이콘 + 메시지)
  - 모든 카드 다크/라이트 테마 지원
  - 패널 보더, 그림자, 글로우 효과 (테마별)
  - TDD: 각 카드 렌더링 테스트

  **Must NOT do**:
  - 드래그앤드롭 리사이즈 금지
  - 커스텀 패널 저장 금지

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 14, 18
  - **Blocked By**: Task 2

  **References**:
  - `internal/adapters/tui/insights_dashboard_view.go` — TUI 인사이트 대시보드 카드 (Waste Headline, Waste %, Projected Waste, Weekly Waste)

  **Acceptance Criteria**:
  - [ ] Panel, StatCard, SparklineCard, AlertCard 존재
  - [ ] 다크/라이트 테마 지원
  - [ ] Vitest 테스트 통과

  **QA Scenarios:**
  ```
  Scenario: StatCard 렌더링
    Tool: Playwright
    Steps:
      1. StatCard에 값 "$42.50", 라벨 "Total Spend", 트렌드 "up" 전달
      2. 카드 렌더링 확인
    Expected Result: "$42.50" 텍스트, "Total Spend" 라벨, 상승 화살표 표시
    Evidence: .sisyphus/evidence/task-11-stat-card.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add Panel and StatCard components`
  - Files: `src/lib/components/ui/panel.svelte`, `src/lib/components/ui/stat-card.svelte`

- [x] 12. Svelte 스토어 (budget, usage, subscription, waste)

  **What to do**:
  - 바인딩 클라이언트 레이어를 호출하는 Svelte 스토어:
    - `src/lib/stores/budget.ts` — 예산 데이터 (MonthlyBudget, BudgetStatus, Forecast)
    - `src/lib/stores/usage.ts` — 사용량 데이터 (UsageEntry[], 토큰/비용 요약)
    - `src/lib/stores/subscription.ts` — 구독 데이터 (Subscription[], 프리셋)
    - `src/lib/stores/waste.ts` — 낭비 데이터 (WasteSummary, 인사이트)
  - 각 스토어 기능:
    - `load()` — 바인딩에서 데이터 로드
    - `refresh()` — 강제 새로고침
    - 로딩/에러 상태 관리
    - mutation 후 자동 관련 스토어 갱신
  - TDD: 각 스토어의 로드/리프레시/에러 처리 테스트 (모킹 바인딩)

  **Must NOT do**:
  - 스토어에 비즈니스 로직 구현 금지 (Go 서비스가 담당)
  - 폴링 타이머 구현 금지 (mutation-triggered만)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 14-19
  - **Blocked By**: Tasks 3, 4

  **References**:
  - `src/lib/bindings/*.ts` (Task 4 산출물) — 바인딩 클라이언트 래퍼
  - `src/lib/types/domain.ts` (Task 3 산출물) — TypeScript 타입

  **Acceptance Criteria**:
  - [ ] 4개 스토어 파일 존재
  - [ ] 각 스토어에 load(), refresh() 메서드
  - [ ] 로딩/에러 상태 관리
  - [ ] Vitest 모킹 테스트 통과

  **QA Scenarios:**
  ```
  Scenario: Budget 스토어 로드
    Tool: Bash (Vitest)
    Steps:
      1. 모킹 바인딩으로 budget store load() 호출
      2. 스토어 상태 확인
    Expected Result: 로딩 → 완료 전환, 데이터 존재
    Evidence: .sisyphus/evidence/task-12-budget-store.txt

  Scenario: 에러 처리
    Tool: Bash (Vitest)
    Steps:
      1. 모킹 바인딩이 에러를 throw하도록 설정
      2. budget store load() 호출
    Expected Result: 에러 상태 설정, 에러 메시지 저장
    Evidence: .sisyphus/evidence/task-12-error-handling.txt
  ```

  **Commit**: YES
  - Message: `feat(gui): add Svelte stores for budget, usage, subscription, waste`
  - Files: `src/lib/stores/*.ts`

- [x] 13. 알림 서비스

  **What to do**:
  - 시스템 알림 서비스 구현:
    - `src/lib/stores/notification.ts` — 알림 상태 관리
    - `src/lib/services/notification.ts` — 알림 서비스
      - 예산 임계값 초과 감지 (BudgetMonitor 서비스 결과 기반)
      - Wails 런타임을 통한 시스템 알림 발송 (`runtime.MessageDialog`)
      - 알림 중복 방지 (동일 임계값 레벨에 대해 세션당 1회)
      - 알림 기록 (현재 세션의 발송된 알림 목록)
  - 알림 UI:
    - `NotificationToast.svelte` — 앱 내 토스트 알림 (자동 사라짐)
    - `NotificationCenter.svelte` — 알림 목록 패널 (사이드바 하단)
  - TDD: 알림 중복 방지, 임계값 감지 테스트

  **Must NOT do**:
  - 백그라운드 알림 데몬 구현 금지
  - 알림 스누즈/규칙 엔진 금지
  - 알림 상태 DB 영속화 금지

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 17, 20
  - **Blocked By**: Tasks 4, 5

  **References**:
  - `internal/service/budget_monitor.go` — BudgetMonitorService, MonitorPeriod 메서드, BudgetMonitorResult (alerts 포함)
  - `internal/domain/budget.go` — BudgetStatus (triggered thresholds), MonthlyBudget

  **Acceptance Criteria**:
  - [ ] `notification.ts` 스토어 + 서비스 존재
  - [ ] NotificationToast 컴포넌트 존재
  - [ ] 임계값 초과 시 토스트 알림 표시
  - [ ] 동일 임계값 중복 방지 동작
  - [ ] Vitest 테스트 통과

  **QA Scenarios:**
  ```
  Scenario: 임계값 초과 알림
    Tool: Bash (Vitest)
    Steps:
      1. budget store에 임계값 초과 상태 설정
      2. notification service check 호출
    Expected Result: 토스트 알림 표시됨
    Evidence: .sisyphus/evidence/task-13-threshold-alert.txt

  Scenario: 중복 알림 방지
    Tool: Bash (Vitest)
    Steps:
      1. 동일 임계값에 대해 check 두 번 호출
      2. 발송된 알림 수 확인
    Expected Result: 1개 알림만 발송됨
    Evidence: .sisyphus/evidence/task-13-dedup-alert.txt
  ```

  **Commit**: YES
  - Message: `feat(gui): add notification service with threshold alerts`
  - Files: `src/lib/stores/notification.ts`, `src/lib/services/notification.ts`, `src/lib/components/ui/notification-toast.svelte`

- [x] 14. 대시보드 화면 (Overview + Providers + Budgets + Sessions)

  **What to do**:
  - Grafana/Datadog 스타일 메인 대시보드 구현 (`/` 라우트):
    - **상단 지표 카드 행** (4개 StatCard):
      - 총 지출 (Total Spend) — 이번 달, 전월 대비 트렌드
      - 총 토큰 사용량 (Total Tokens) — input + output 합
      - 구독 비용 (Subscription Cost) — 활성 구독 월간 합
      - 낭비율 (Waste %) — 전체 대비 낭비 비율
    - **좌측 패널** (2/3 너비):
      - 일별 비용 트렌드 라인 차트 (최근 30일)
      - 프로바이더별 비용 바 차트
    - **우측 패널** (1/3 너비):
      - 예산 상태 패널 (사용량/한도 프로그레스 바)
      - 최근 세션 테이블 (최근 10개)
  - 빈 상태 처리 (DB 비어있을 때 안내 메시지)
  - 페이지 로드 시 스토어에서 데이터 자동 로드
  - 새로고침 버튼으로 수동 갱신
  - TDD: 대시보드 컴포넌트 렌더링 테스트

  **Must NOT do**:
  - 실시간 폴링 금지 (mutation-triggered + 수동만)
  - 커스텀 대시보드 저장 금지

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 8, 9, 11, 12

  **References**:
  - `internal/adapters/tui/view.go` — TUI 대시보드 렌더링 (renderDashboard 함수)
  - `internal/adapters/tui/model.go` — focusSection enum: Overview, Providers, Budgets, RecentSessions — GUI 패널과 1:1 매핑
  - `internal/adapters/gui/dashboard_binding.go` — DashboardBinding 메서드

  **Acceptance Criteria**:
  - [ ] `/` 라우트에서 대시보드 렌더링
  - [ ] 4개 지표 카드 표시
  - [ ] 비용 트렌드 라인 차트 표시
  - [ ] 프로바이더별 바 차트 표시
  - [ ] 예산 프로그레스 바 표시
  - [ ] 최근 세션 테이블 표시
  - [ ] 빈 DB 상태 처리

  **QA Scenarios:**
  ```
  Scenario: 대시보드 전체 렌더링
    Tool: Playwright
    Steps:
      1. 모킹 데이터로 대시보드 로드
      2. 4개 StatCard 표시 확인
      3. 차트 2개 렌더링 확인 (canvas 요소 존재)
      4. 예산 프로그레스 바 표시 확인
      5. 세션 테이블 행 표시 확인
    Expected Result: 모든 패널이 데이터와 함께 렌더링됨
    Evidence: .sisyphus/evidence/task-14-dashboard-full.png

  Scenario: 빈 DB 대시보드
    Tool: Playwright
    Steps:
      1. 빈 데이터로 대시보드 로드
      2. 각 패널의 빈 상태 메시지 확인
    Expected Result: "No data available" 또는 안내 메시지 표시, 에러 없음
    Evidence: .sisyphus/evidence/task-14-dashboard-empty.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add dashboard screen`
  - Files: `src/routes/+page.svelte`, `src/routes/_components/dashboard/*`

- [x] 15. 사용량 추적 화면 (수동 입력 + 히스토리)

  **What to do**:
  - 사용량 관리 화면 (`/usage` 라우트):
    - **수동 입력 폼**:
      - Provider (SelectInput: Claude Code, Codex, Gemini, OpenCode, Other)
      - Model ID (TextInput)
      - Occurred At (DatePicker)
      - Input Tokens (NumberInput)
      - Output Tokens (NumberInput)
      - Cached Tokens (NumberInput)
      - Cache Write Tokens (NumberInput)
      - Project Name (TextInput)
      - 저장 버튼 + 취소 버튼
    - **사용량 히스토리 테이블**:
      - DataTable으로 최근 사용량 표시
      - 컬럼: Date, Provider, Model, Input Tokens, Output Tokens, Cost
  - Zod 폼 검증 (필수 필드, 토큰 음수 불가)
  - 저장 성공 시 토스트 알림 + 스토어 갱신
  - TDD: 폼 제출 + 검증 테스트

  **Must NOT do**:
  - 사용량 수정/삭제 기능 금지 (TUI에 없음 — 나중에 추가 가능)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 10, 12

  **References**:
  - `internal/adapters/tui/view.go` — TUI ManualEntryForm 9개 필드 정확한 구조
  - `internal/domain/usage.go` — UsageEntry 구조체, TokenUsage, CostBreakdown

  **Acceptance Criteria**:
  - [ ] `/usage` 라우트에서 사용량 화면 렌더링
  - [ ] 8개 폼 필드 표시
  - [ ] Zod 검증 동작 (필수 필드, 음수 토큰 거부)
  - [ ] 저장 시 바인딩 호출 + 스토어 갱신
  - [ ] 히스토리 테이블 표시

  **QA Scenarios:**
  ```
  Scenario: 수동 입력 폼 제출
    Tool: Playwright
    Steps:
      1. /usage 페이지 로드
      2. Provider 선택 "Claude Code"
      3. Model ID 입력 "claude-3.5-sonnet"
      4. Input Tokens 입력 "1000"
      5. Output Tokens 입력 "500"
      6. 저장 버튼 클릭
    Expected Result: 토스트 "Usage entry saved", 히스토리 테이블에 새 행 추가
    Evidence: .sisyphus/evidence/task-15-manual-entry.png

  Scenario: 폼 검증 에러
    Tool: Playwright
    Steps:
      1. /usage 페이지 로드
      2. 빈 폼으로 저장 버튼 클릭
    Expected Result: 필수 필드 에러 메시지 표시, 바인딩 호출 안 됨
    Evidence: .sisyphus/evidence/task-15-form-validation.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add usage tracking screen`
  - Files: `src/routes/usage/+page.svelte`

- [x] 16. 구독 관리 화면 (목록 + 추가 폼)

  **What to do**:
  - 구독 관리 화면:
    - **목록 뷰** (`/subscriptions`):
      - DataTable: Date, Provider, Plan, Fee, Renewal Day, Active 상태
      - StatusBadge로 Active/Inactive 표시
      - 행 클릭 시 편집 모드 (또는 상세 패널)
      - 삭제 버튼 (확인 다이얼로그 포함)
    - **추가 폼** (`/subscriptions/new`):
      - Provider (SelectInput)
      - Plan Name (TextInput)
      - Renewal Day (NumberInput 1-31)
      - Starts At (DatePicker)
      - Fee USD (NumberInput)
      - Active (Toggle)
      - Ends At (DatePicker, optional)
      - **프리셋 선택기**: 9개 내장 프리셋에서 빠른 선택
        - ChatGPT Plus, ChatGPT Pro, Claude Pro, Claude Max, Gemini Advanced, etc.
  - TDD: 구독 CRUD 테스트

  **Must NOT do**:
  - 구독 편집은 추가/삭제로 대체 (TUI와 동일한 방식)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 9, 10, 12

  **References**:
  - `internal/adapters/tui/view.go` — TUI SubscriptionForm (preset selector + 7 fields), SubscriptionList (delete with 'd' key)
  - `internal/domain/subscription.go` — Subscription, SubscriptionFee

  **Acceptance Criteria**:
  - [ ] `/subscriptions` 에 목록 렌더링
  - [ ] `/subscriptions/new` 에 추가 폼 렌더링
  - [ ] 프리셋 선택기 동작
  - [ ] 삭제 확인 다이얼로그 표시
  - [ ] 저장 시 바인딩 호출 + 스토어 갱신

  **QA Scenarios:**
  ```
  Scenario: 구독 추가
    Tool: Playwright
    Steps:
      1. /subscriptions/new 페이지 로드
      2. "ChatGPT Plus" 프리셋 선택
      3. 폼 자동 채워짐 확인
      4. 저장 버튼 클릭
    Expected Result: 토스트 "Subscription saved", 목록에 새 행 추가
    Evidence: .sisyphus/evidence/task-16-subscription-add.png

  Scenario: 구독 삭제
    Tool: Playwright
    Steps:
      1. /subscriptions 페이지에서 기존 구독 행의 삭제 버튼 클릭
      2. 확인 다이얼로그에서 "Delete" 클릭
    Expected Result: 구독이 목록에서 제거됨
    Evidence: .sisyphus/evidence/task-16-subscription-delete.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add subscription management screen`
  - Files: `src/routes/subscriptions/+page.svelte`, `src/routes/subscriptions/new/+page.svelte`

- [x] 17. 예산 관리 화면 (설정 + 모니터링)

  **What to do**:
  - 예산 관리 화면 (`/budgets` 라우트):
    - **예산 설정 폼**:
      - Monthly Limit (NumberInput, USD)
      - Warning Threshold (NumberInput, %, 예: 80)
      - Critical Threshold (NumberInput, %, 예: 100)
      - 저장 버튼
    - **예산 모니터링 패널**:
      - 현재 사용량 vs 한도 프로그레스 바 (색상: green → yellow → red)
      - ForecastSnapshot 표시 (예상 월말 지출)
      - 트리거된 임계값 경고 목록 (AlertCard)
    - **월간 지출 요약**:
      - 프로바이더별 비용 파이 차트
      - 일별 누적 지출 라인 차트
  - 알림 서비스와 연동 (임계값 초과 시 알림)
  - TDD: 예산 설정 + 임계값 감지 테스트

  **Must NOT do**:
  - 과거 예산 이력 조회 금지 (현재 월만)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 10, 12, 13

  **References**:
  - `internal/domain/budget.go` — MonthlyBudget, BudgetStatus, ForecastSnapshot
  - `internal/service/budget_monitor.go` — BudgetMonitorService, MonitorPeriod 메서드

  **Acceptance Criteria**:
  - [ ] `/budgets` 에 예산 화면 렌더링
  - [ ] 예산 설정 폼 동작
  - [ ] 프로그레스 바 표시 (색상 임계값에 따라)
  - [ ] 예측 지출 표시
  - [ ] 임계값 경고 표시
  - [ ] 차트 2개 렌더링

  **QA Scenarios:**
  ```
  Scenario: 예산 설정
    Tool: Playwright
    Steps:
      1. /budgets 페이지 로드
      2. Monthly Limit에 "100" 입력
      3. Warning Threshold에 "80" 입력
      4. 저장 버튼 클릭
    Expected Result: 토스트 "Budget saved", 프로그레스 바 업데이트
    Evidence: .sisyphus/evidence/task-17-budget-set.png

  Scenario: 임계값 초과 경고
    Tool: Playwright
    Steps:
      1. 예산 한도 $10, 사용량 $9인 상태 로드
      2. Warning 경고 AlertCard 표시 확인
    Expected Result: "Warning: 90% of budget used" AlertCard 표시
    Evidence: .sisyphus/evidence/task-17-budget-warning.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add budget management screen`
  - Files: `src/routes/budgets/+page.svelte`

- [x] 18. 인사이트 화면 (낭비 감지 대시보드)

  **What to do**:
  - 낭비 감지 인사이트 화면 (`/insights` 라우트):
    - **상단 요약 카드** (4개 StatCard):
      - Waste Headline (가장 큰 낭비 원인)
      - Waste % (전체 대비 낭비 비율)
      - Projected Waste (예상 낭비 금액)
      - Weekly Waste (주간 낭비 금액)
    - **Top Waste Causes** 바 차트 (8개 낭지 감지기 결과)
    - **Daily Waste Trend** 라인 차트
    - **인사이트 로그 테이블**:
      - DataTable로 감지된 인사이트 목록
      - 컬럼: Detector, Severity, Description, Detected At
      - 행 클릭 시 상세 모달 (인사이트 디테일)
  - TDD: 인사이트 데이터 렌더링 테스트

  **Must NOT do**:
  - 인사이트 필터링/검색 금지 (TUI에 없음)
  - 커스텀 낭지 감지기 추가 금지

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 8, 9, 11, 12

  **References**:
  - `internal/adapters/tui/insights_dashboard_view.go` — TUI 인사이트 대시보드 (Waste Headline, Waste %, Projected Waste, Weekly Waste, Top Causes, Daily Trend)
  - `internal/domain/waste_summary.go` — WasteSummary 구조체
  - `internal/service/waste_summary.go` — WasteSummaryService, QueryWasteSummary 메서드

  **Acceptance Criteria**:
  - [ ] `/insights` 에 인사이트 화면 렌더링
  - [ ] 4개 요약 카드 표시
  - [ ] 낭비 원인 바 차트 표시
  - [ ] 일별 낭지 트렌드 라인 차트 표시
  - [ ] 인사이트 로그 테이블 표시
  - [ ] 행 클릭 시 상세 모달 표시

  **QA Scenarios:**
  ```
  Scenario: 인사이트 대시보드 전체 렌더링
    Tool: Playwright
    Steps:
      1. 모킹 낭비 데이터로 /insights 로드
      2. 4개 StatCard 값 확인
      3. 바 차트 렌더링 확인
      4. 라인 차트 렌더링 확인
      5. 인사이트 테이블 행 확인
    Expected Result: 모든 패널과 차트가 데이터와 함께 렌더링됨
    Evidence: .sisyphus/evidence/task-18-insights-full.png

  Scenario: 인사이트 상세 모달
    Tool: Playwright
    Steps:
      1. 인사이트 테이블의 첫 번째 행 클릭
      2. 모달 표시 확인
      3. 상세 내용 표시 확인
      4. 닫기 버튼 클릭
    Expected Result: 모달 열림 → 상세 내용 → 모달 닫힘
    Evidence: .sisyphus/evidence/task-18-insight-detail.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add insights screen with waste detection`
  - Files: `src/routes/insights/+page.svelte`

- [x] 19. 그래프 화면 (4개 차트 탭)

  **What to do**:
  - 그래프 화면 (`/graphs` 라우트):
    - **탭 네비게이션** (4개 탭, TUI와 동일):
      1. **Model Token Usage** — 모델별 토큰 사용량 바 차트
      2. **Model Cost** — 모델별 비용 바 차트
      3. **Daily Token Trend** — 일별 토큰 사용량 라인 차트
      4. **Model Token Breakdown** — 모델별 토큰 분해 스택 바 차트
    - 각 탭:
      - 전체 너비 차트
      - 시간 범위 선택 (최근 7일 / 30일 / 전체)
      - 차트 로딩/빈 상태
  - TDD: 탭 전환 + 차트 렌더링 테스트

  **Must NOT do**:
  - 커스텀 차트 구성 금지
  - 차트 이미지 내보내기 금지

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 8, 12

  **References**:
  - `internal/adapters/tui/model.go` — TUI Graphs viewMode, 4개 탭: Model Token Usage, Model Cost, Daily Token Trend, Model Token Breakdown

  **Acceptance Criteria**:
  - [ ] `/graphs` 에 그래프 화면 렌더링
  - [ ] 4개 탭 전환 동작
  - [ ] 각 탭에 해당 차트 렌더링
  - [ ] 시간 범위 선택 동작
  - [ ] 빈 데이터 상태 처리

  **QA Scenarios:**
  ```
  Scenario: 그래프 탭 전환
    Tool: Playwright
    Steps:
      1. /graphs 페이지 로드
      2. "Model Cost" 탭 클릭
      3. 차트 렌더링 확인
      4. "Daily Token Trend" 탭 클릭
      5. 라인 차트 렌더링 확인
    Expected Result: 탭 전환 시 해당 차트가 렌더링됨
    Evidence: .sisyphus/evidence/task-19-graph-tabs.png

  Scenario: 시간 범위 변경
    Tool: Playwright
    Steps:
      1. 기본 "30일" 범위에서 "7일" 선택
      2. 차트 데이터 갱신 확인
    Expected Result: 차트가 최근 7일 데이터로 갱신됨
    Evidence: .sisyphus/evidence/task-19-time-range.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add graphs screen with 4 chart tabs`
  - Files: `src/routes/graphs/+page.svelte`

- [x] 20. 설정 화면 + 테마/알림 환경설정

  **What to do**:
  - 설정 화면 (`/settings` 라우트):
    - **테마 설정**: 다크/라이트 토글 스위치 (현재 상태 표시)
    - **알림 설정**:
      - 예산 임계값 알림 활성화/비활성화 토글
      - 시스템 알림 권한 상태 표시
    - **데이터 관리**:
      - DB 경로 표시 (읽기 전용)
      - 마지막 갱신 시간 표시
    - **앱 정보**: 버전, 빌드 정보
  - TDD: 설정 페이지 렌더링 테스트

  **Must NOT do**:
  - 고급 설정 에디터 금지
  - DB 초기화/리셋 기능 금지 (위험)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`/frontend-ui-ux`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 21-23)
  - **Blocks**: Task 21
  - **Blocked By**: Tasks 5, 10, 13

  **References**:
  - `src/lib/stores/theme.ts` (Task 5) — 테마 스토어
  - `src/lib/stores/notification.ts` (Task 13) — 알림 스토어

  **Acceptance Criteria**:
  - [ ] `/settings` 에 설정 화면 렌더링
  - [ ] 테마 토글 동작
  - [ ] 알림 설정 토글 동작
  - [ ] DB 경로 표시

  **QA Scenarios:**
  ```
  Scenario: 설정 화면 렌더링
    Tool: Playwright
    Steps:
      1. /settings 페이지 로드
      2. 테마 토글 스위치 존재 확인
      3. 알림 설정 토글 존재 확인
      4. DB 경로 표시 확인
    Expected Result: 모든 설정 섹션 렌더링
    Evidence: .sisyphus/evidence/task-20-settings.png

  Scenario: 테마 토글 동작
    Tool: Playwright
    Steps:
      1. 다크 테마 상태에서 토글 클릭
      2. 라이트 테마로 전환 확인
    Expected Result: 전체 UI 테마 전환
    Evidence: .sisyphus/evidence/task-20-theme-toggle-settings.png
  ```

  **Commit**: YES
  - Message: `feat(gui): add settings screen with theme and notification preferences`
  - Files: `src/routes/settings/+page.svelte`

- [x] 21. Wails 바인딩 전체 연동 + mutation-triggered refresh

  **What to do**:
  - 모든 화면의 바인딩 연동:
    - 대시보드 → DashboardBinding.GetAllDashboardData()
    - 사용량 폼 → FormsBinding.AddUsageEntry()
    - 구독 CRUD → FormsBinding.AddSubscription(), SubscriptionLookupBinding
    - 예산 설정 → FormsBinding.SetBudget()
    - 인사이트 → DashboardBinding 또는 신규 바인딩
    - 그래프 → DashboardBinding 또는 신규 바인딩
  - Mutation-triggered refresh 구현:
    - 데이터 변경(저장/삭제) 후 관련 스토어 자동 refresh()
    - 예: 구독 추가 → subscriptionStore.refresh() + budgetStore.refresh()
  - 에러 처리: 바인딩 호출 실패 시 사용자에게 에러 토스트 표시
  - 로딩 상태: 각 화면 첫 로드 시 스피너/스켈레톤 표시
  - TDD: 연동 테스트 (모킹 바인딩으로 전체 플로우)

  **Must NOT do**:
  - 실시간 폴링 추가 금지
  - Go 비즈니스 로직 수정 금지

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (sequential, after Wave 3)
  - **Blocks**: Task 22
  - **Blocked By**: Tasks 14-20

  **References**:
  - `src/lib/bindings/*.ts` (Task 4) — 바인딩 클라이언트 래퍼
  - `src/lib/stores/*.ts` (Task 12) — Svelte 스토어
  - `internal/adapters/gui/app.go` — Bind 배열

  **Acceptance Criteria**:
  - [ ] 모든 화면에서 실제 바인딩 호출 동작 (wails dev 환경)
  - [ ] 데이터 변경 후 관련 스토어 자동 갱신
  - [ ] 바인딩 에러 시 에러 토스트 표시
  - [ ] 로딩 상태 표시

  **QA Scenarios:**
  ```
  Scenario: 전체 연동 플로우
    Tool: Playwright
    Steps:
      1. wails dev 실행
      2. 대시보드 로드 → 실제 데이터 표시
      3. 수동 입력 폼 → 데이터 저장
      4. 대시보드로 돌아가기 → 데이터 갱신 확인
    Expected Result: 모든 CRUD 동작 + 자동 갱신 동작
    Evidence: .sisyphus/evidence/task-21-full-integration.png

  Scenario: 바인딩 에러 처리
    Tool: Playwright
    Steps:
      1. 바인딩이 에러를 반환하는 상황 시뮬레이션
      2. 에러 토스트 표시 확인
    Expected Result: "Failed to save" 토스트 알림 표시
    Evidence: .sisyphus/evidence/task-21-binding-error.png
  ```

  **Commit**: YES
  - Message: `feat(gui): integrate all Wails bindings with mutation-triggered refresh`
  - Files: `src/routes/**/*.svelte`, `src/lib/stores/*.ts`

- [x] 22. Playwright E2E 테스트 스위트

  **What to do**:
  - 전체 GUI 기능에 대한 E2E 테스트 작성:
    - **대시보드**: 로드, 지표 카드, 차트, 새로고침
    - **수동 입력**: 폼 제출, 검증 에러, 성공 토스트
    - **구독 관리**: 목록, 추가, 삭제, 프리셋 선택
    - **예산 관리**: 설정, 프로그레스 바, 임계값 경고
    - **인사이트**: 요약 카드, 차트, 테이블, 모달
    - **그래프**: 탭 전환, 시간 범위, 차트 렌더링
    - **설정**: 테마 토글, 알림 설정
    - **크로스 화면**: 사이드바 네비게이션, 빈 DB 상태
  - 시드 데이터 스크립트: E2E 테스트용 결정론적 데이터
  - 각 테스트에 스크린샷 캡처
  - 실행: `npx playwright test`

  **Must NOT do**:
  - 실제 Wails 바이너리 빌드 필요 금지 (dev 서버 사용)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`/playwright`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after Task 21)
  - **Blocks**: Task 23
  - **Blocked By**: Tasks 21, 7

  **References**:
  - `src/routes/**/*.svelte` — 모든 화면 컴포넌트
  - `playwright.config.ts` (Task 7) — Playwright 설정

  **Acceptance Criteria**:
  - [ ] 최소 15개 E2E 테스트 작성
  - [ ] 모든 주요 화면 커버
  - [ ] `npx playwright test` 전체 통과
  - [ ] 스크린샷 캡처됨

  **QA Scenarios:**
  ```
  Scenario: E2E 테스트 스위트 실행
    Tool: Bash
    Steps:
      1. cd internal/adapters/gui/frontend && npx playwright test
    Expected Result: 15+ 테스트 모두 통과, 0 failures
    Failure Indicators: 타임아웃, 요소 미발견, assertion 실패
    Evidence: .sisyphus/evidence/task-22-e2e-results.txt
  ```

  **Commit**: YES
  - Message: `test(gui): add Playwright E2E test suite`
  - Files: `tests/e2e/*.spec.ts`

- [x] 23. Linux + Mac 크로스 플랫폼 빌드 + 검증

  **What to do**:
  - 크로스 플랫폼 빌드 설정:
    - Makefile에 Linux + Mac 빌드 타겟 추가
    - `wails build -platform linux/amd64,darwin/universal`
  - SQLite WAL 모드 검증:
    - TUI와 GUI 동시 실행 시 DB 무결성 확인
    - GUI에서 쓰기 → TUI에서 읽기 일관성 확인
  - 플랫폼별 이슈 검증:
    - Linux: WebKit2 의존성, 시스템 알림 동작
    - Mac: 코드 사이닝 없이 빌드, 시스템 알림 동작
  - 최종 `wails build` 성공 확인
  - 빈 DB 첫 실행 시나리오 테스트

  **Must NOT do**:
  - macOS 공증/사이닝 금지 (별도 작업)
  - Windows 빌드 금지

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after Task 22)
  - **Blocks**: FINAL
  - **Blocked By**: Task 22

  **References**:
  - `wails.json` — 빌드 설정
  - `Makefile` — 기존 빌드 타겟

  **Acceptance Criteria**:
  - [ ] `wails build` Linux에서 성공
  - [ ] `wails build` Mac에서 성공
  - [ ] 빈 DB 첫 실행 시 정상 동작
  - [ ] SQLite 동시성 검증 통과

  **QA Scenarios:**
  ```
  Scenario: Linux 빌드 성공
    Tool: Bash
    Steps:
      1. wails build -platform linux/amd64
      2. 생성된 바이너리 실행
      3. GUI 창 표시 확인
    Expected Result: 바이너리 생성, 실행 시 GUI 표시
    Evidence: .sisyphus/evidence/task-23-linux-build.txt

  Scenario: 빈 DB 첫 실행
    Tool: Playwright
    Steps:
      1. 기존 DB 파일 백업 후 삭제
      2. GUI 실행
      3. 모든 화면 에러 없이 로드 확인
    Expected Result: 빈 상태 UI 표시, 크래시 없음
    Evidence: .sisyphus/evidence/task-23-first-run.png
  ```

  **Commit**: YES
  - Message: `build(gui): cross-platform Linux + Mac build and verification`
  - Files: `Makefile`, `wails.json`

---

## Final Verification Wave

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `npm run check` (svelte-check) + `npm run lint` + `npm test`. Review all changed files for: `any` types, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high` (+ `playwright` skill)
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration. Test edge cases: empty DB, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Wave 1**: `feat(gui): scaffold Svelte + Wails frontend` - 스캐폴딩 파일들
- **Wave 1**: `feat(gui): add design system tokens and theme` - 스타일 파일들
- **Wave 1**: `feat(gui): add TypeScript types and binding client` - 타입 + 바인딩
- **Wave 1**: `feat(gui): add layout shell with sidebar` - 레이아웃
- **Wave 2**: `feat(gui): add chart components` - 차트 라이브러리
- **Wave 2**: `feat(gui): add table, form, card components` - UI 컴포넌트
- **Wave 2**: `feat(gui): add Svelte stores and notification service` - 상태 + 알림
- **Wave 3**: `feat(gui): add dashboard screen` - 대시보드
- **Wave 3**: `feat(gui): add usage tracking screen` - 사용량
- **Wave 3**: `feat(gui): add subscription management screen` - 구독
- **Wave 3**: `feat(gui): add budget management screen` - 예산
- **Wave 3**: `feat(gui): add insights screen` - 인사이트
- **Wave 3**: `feat(gui): add graphs screen` - 그래프
- **Wave 4**: `feat(gui): add settings screen` - 설정
- **Wave 4**: `feat(gui): integrate all bindings and refresh` - 연동
- **Wave 4**: `test(gui): add Playwright E2E suite` - E2E 테스트
- **Wave 4**: `build(gui): cross-platform Linux + Mac build` - 빌드

---

## Success Criteria

### Verification Commands
```bash
cd internal/adapters/gui/frontend && npm test          # Expected: All Vitest tests pass
cd internal/adapters/gui/frontend && npm run build     # Expected: Build succeeds
cd internal/adapters/gui/frontend && npx playwright test  # Expected: All E2E tests pass
wails build                                            # Expected: Binary created
```

### Final Checklist
- [ ] All "Must Have" present (TUI 100% 패리티, 다크/라이트, 알림)
- [ ] All "Must NOT Have" absent (드래그앤드롭, 클라우드 동기화, 모바일 등)
- [ ] All Vitest tests pass
- [ ] All Playwright E2E tests pass
- [ ] `wails dev` 실행 시 GUI 정상 렌더링
- [ ] Linux + Mac 빌드 성공
