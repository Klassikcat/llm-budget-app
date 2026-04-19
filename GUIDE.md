# 전체 레퍼런스 (GUIDE)

LLM Budget Tracker 핵심 기술자 레퍼런스. 디렉터리 구조, 키 바인딩, 프리셋, 탐지기, DB 스키마, 가격 카탈로그 오버라이드를 단계적으로 설명.

---

## 아키텍처 개요

육각형 아키텍처(Ports & Adapters) 기반의 정렬된 설계:

```
cmd/tui        → internal/adapters/tui (Bubbletea)    ┐
cmd/gui        → internal/adapters/gui (Wails)         ├→ internal/service/*  →  internal/domain/*
                                                        │
                                    internal/adapters/sqlite (저장소)
                                                        │
                                    internal/adapters/parsers (세션 수집)
                                    ↑
                    internal/adapters/fsnotify (파일 감지)
                    ↑
                    internal/catalog (가격 카탈로그)
                    internal/config (경로, 설정)
```

**아키텍처 흐름**:
- **TUI/GUI**: 두 UI는 동일한 `internal/service/` 비즈니스 로직 위에 병렬로 올라간 어댑터
- **세션 파서**: `claude.go`, `codex.go`, `gemini.go`, `opencode.go`가 로그 파일을 정규화해 `SessionSummary` + `UsageEntry`로 변환
- **저장소**: SQLite 어댑터가 모든 데이터 persistence 담당
- **가격 카탈로그**: 3단 우선순위 (사용자 YAML 오버라이드 > 임베디드 JSON > OpenRouter 라이브 캐시)
- **파일 워처**: `fsnotify` 래퍼가 세션 로그 변경을 감지해 증분 수집

---

## 디렉터리 구조

```
cmd/
  gui/
    main.go              # Wails GUI 바이너리 진입점
  tui/
    main.go              # Bubbletea TUI 바이너리 진입점

internal/
  adapters/
    gui/                 # Wails 어댑터
      dashboard_binding.go
      notifier.go
      types.go
    tui/                 # Bubbletea 어댑터
      model.go           # Elm 아키텍처 모델 (상태, 업데이트)
      view.go            # 렌더링 로직
      graph_view.go      # 그래프 탭 렌더링
      model_test.go
      run.go             # CLI 진입점
    sqlite/              # SQLite 저장소 구현
      repository.go
      migrations.go
      bootstrap.go
    parsers/             # 세션 로그 파서
      claude.go          # Claude Code JSONL 파서
      codex.go           # OpenAI Codex JSONL 파서
      gemini.go          # Gemini CLI JSON 파서
      opencode.go        # OpenCode SQLite DB 파서
    fsnotify/
      watcher.go         # 파일 워처 래퍼
    openrouter/
      client.go          # OpenRouter HTTP 클라이언트

  catalog/               # 가격 카탈로그 (3단 우선순위)
    loader.go
    catalog.go
    document.go
    embedded.go          # 임베디드 JSON 데이터
    data/
      anthropic.json
      openai.json
      gemini.json
      openrouter-cache.json

  config/                # 환경 및 설정 경로
    paths.go             # XDG 경로 해석 (Linux/macOS/Windows)
    secrets.go           # OS 키링 통합
    settings.go
    errors.go

  domain/                # 순수 도메인 모델 (비즈니스 로직 불포함)
    provider.go
    session.go
    usage.go
    subscription.go
    insight.go
    alert.go
    budget.go
    billing.go
    time.go

  ports/                 # 어댑터 경계 인터페이스
    parser.go            # SessionParser 인터페이스
    repository.go        # 저장소 인터페이스들
    catalog.go           # PriceCatalog 인터페이스
    notifier.go
    openrouter.go
    ingestion.go

  service/               # 비즈니스 로직 (구독, 탐지, 질의, 워처)
    subscriptions.go
    detector_set_a.go    # Context Avalanche, Missed Prompt Caching, Planning Tax
    detector_set_b.go    # Repeated File Reads, Retry Amplification, Zombie Loops
    detector_set_c.go    # Over-Qualified Model, Tool Schema Bloat
    dashboard_query.go
    graph_query.go
    manual_api_entry.go
    watcher.go
    cost_calculator.go
    budget_monitor.go
    insight_executor.go
    session_normalizer.go
    attribution.go
    subscription_presets.go  # 9개 기본 프리셋 정의

  app/
    parser_billing_mode.go

db/
  migrations/
    0001_initial.sql                    # 전체 스키마
    0002_usage_entries_manual_fields.sql # 복합 인덱스
    0003_insights_privacy_safe.sql       # 프라이버시 마이그레이션
    0004_budget_monitoring.sql          # 예산 감시 테이블

Makefile                # 빌드 타겟
go.mod / go.sum
```

---

## TUI 화면 모드

| 모드 | 상수 | 진입 방법 | 설명 |
|------|------|----------|------|
| Dashboard | `viewDashboard` | 앱 시작 / 다른 모드에서 `Esc` | 개요, 제공자 요약, 예산, 최근 세션 |
| Manual API Entry | `viewManualEntryForm` | 대시보드에서 `m` | 수동 API 사용량 입력 폼 |
| Subscription Fee Form | `viewSubscriptionForm` | 대시보드에서 `s` | 구독 프리셋 선택 또는 수동 입력 |
| Subscription List | `viewSubscriptionList` | 대시보드에서 `l` | 저장된 구독 레코드 목록, 비활성화 옵션 |
| Insight List | `viewInsightList` | 대시보드에서 `i` | 탐지된 낭비 패턴 목록 |
| Insight Detail | `viewInsightDetail` | 인사이트 목록에서 `Enter` | 선택 인사이트의 상세 정보 |
| Graphs | `viewGraphs` | 대시보드에서 `g` | 모델 사용량, 비용, 일일 추세, 모델별 토큰 분석 |

---

## 전체 키 바인딩 레퍼런스

### Dashboard (대시보드)

| 키 | 동작 |
|----|------|
| `Tab`, `→`, `↓`, `l`, `j` | 다음 섹션으로 포커스 이동 |
| `Shift+Tab`, `←`, `↑`, `h`, `k` | 이전 섹션으로 포커스 이동 |
| `m` | 수동 API 사용량 입력 폼으로 이동 |
| `s` | 구독 요금 입력 폼으로 이동 |
| `l` | 저장된 구독 목록으로 이동 |
| `i` | 탐지된 인사이트 목록으로 이동 |
| `g` | 그래프 탭으로 이동 |
| `r` | 대시보드, 인사이트, 알림 새로고침 |
| `q`, `Ctrl+C` | 앱 종료 |

### Manual Entry Form (수동 입력 폼)

| 키 | 동작 |
|----|------|
| `Tab`, `↓` | 다음 필드로 포커스 이동 |
| `Shift+Tab`, `↑` | 이전 필드로 포커스 이동 |
| `Ctrl+S` | 폼 제출 (현재 필드 유효성 검사) |
| 마지막 필드에서 `Enter` | 폼 제출 |
| `Esc` | 취소, 대시보드로 돌아가기 |

### Subscription Form (구독 입력 폼)

| 키 | 동작 |
|----|------|
| `↑`, `k` | 커서를 위로 이동 (프리셋 또는 필드) |
| `↓`, `j` | 커서를 아래로 이동 (프리셋 또는 필드) |
| `←`, `h` | 왼쪽으로 이동 (프리셋 그리드) |
| `→`, `l` | 오른쪽으로 이동 (프리셋 그리드) |
| `Enter` | 프리셋 선택 토글 또는 필드로 진입 |
| `Tab` | Manual 모드에서 다음 필드로 이동 |
| `Shift+Tab` | Manual 모드에서 이전 필드로 이동 |
| `Ctrl+S` | 선택된 모든 프리셋 또는 수동 폼 제출 |
| `Esc` | 취소, 대시보드로 돌아가기 |

### Subscription List (구독 목록)

| 키 | 동작 |
|----|------|
| `↑`, `k` | 선택 커서를 위로 이동 |
| `↓`, `j` | 선택 커서를 아래로 이동 |
| `d` | 선택된 구독을 비활성화 |
| `r` | 구독 목록 새로고침 |
| `Esc`, `Backspace` | 대시보드로 돌아가기 |

### Insight List (인사이트 목록)

| 키 | 동작 |
|----|------|
| `↑`, `k` | 선택 커서를 위로 이동 |
| `↓`, `j` | 선택 커서를 아래로 이동 |
| `Enter` | 선택된 인사이트 상세 보기로 이동 |
| `r` | 인사이트 목록 새로고침 (대시보드 전체 새로고침) |
| `Esc`, `Backspace` | 대시보드로 돌아가기 |

### Insight Detail (인사이트 상세)

| 키 | 동작 |
|----|------|
| `Esc`, `Backspace` | 인사이트 목록으로 돌아가기 |

### Graphs (그래프)

| 키 | 동작 |
|----|------|
| `Tab`, `→`, `l` | 다음 그래프 탭으로 이동 |
| `Shift+Tab`, `←`, `h` | 이전 그래프 탭으로 이동 |
| `r` | 그래프 데이터 새로고침 |
| `Esc`, `Backspace` | 대시보드로 돌아가기 |

---

## 구독 프리셋 전체 목록

9개의 기본 프리셋은 `internal/service/subscription_presets.go`에서 정의됨. 각 프리셋은 결정적 방식으로 저장되며, 같은 조합 재저장은 upsert (멱등):

| Key | Provider | Plan Name | 기본 요금 (USD/월) | Renewal Day |
|-----|----------|-----------|-------|-------------|
| `chatgpt-plus` | `openai` | ChatGPT Plus | 20.00 | 1 |
| `chatgpt-pro-5x` | `openai` | ChatGPT Pro 5x | 100.00 | 1 |
| `chatgpt-pro-20x` | `openai` | ChatGPT Pro 20x | 200.00 | 1 |
| `claude-pro` | `claude` | Claude Pro | 20.00 | 1 |
| `claude-max-5x` | `claude` | Claude Max 5x | 100.00 | 1 |
| `claude-max-20x` | `claude` | Claude Max 20x | 200.00 | 1 |
| `gemini-plus` | `gemini` | Gemini Plus | 7.99 | 1 |
| `gemini-pro` | `gemini` | Gemini Pro | 19.99 | 1 |
| `gemini-ultra` | `gemini` | Gemini Ultra | 249.99 | 1 |

**Plan Code 생성 규칙**:
- `planCode` = provider + 정규화된 plan-slug (예: `openai-chatgpt-plus`)
- `subscriptionID` = planCode + starts_at 날짜 해시 (결정적 생성)
- 같은 planCode + 날짜 조합 재저장 = upsert (기존 레코드 갱신)

---

## 인사이트 탐지기 (낭비 패턴 8종)

모든 탐지기는 토큰 수, 비용, 해시된 식별자만 사용. **프롬프트/응답 텍스트 접근 불가** (마이그레이션 `0003_insights_privacy_safe.sql`에서 관련 컬럼 제거):

| Detector | Category Key | 트리거 조건 | 소스 파일 |
|----------|-------------|-------------|----------|
| Context Avalanche | `context_avalanche` | input/output 비율 ≥ 4.0 + 초과 토큰 ≥ 2000 | `detector_set_a.go` |
| Missed Prompt Caching | `missed_prompt_caching` | 반복 토큰 ≥ 2000 × ≥ 3회 캐시 미사용 | `detector_set_a.go` |
| Planning Tax | `planning_tax` | reasoning/output 비율 ≥ 2.0 + tool call ≤ 1 | `detector_set_a.go` |
| Repeated File Reads | `repeated_file_reads` | 동일 파일 ≥ 4회 읽음 | `detector_set_b.go` |
| Retry Amplification | `retry_amplification` | 연속 tool call 실패 ≥ 3회 | `detector_set_b.go` |
| Zombie Loops | `zombie_loops` | 거의 동일한 스텝 ≥ 5회 반복 | `detector_set_b.go` |
| Over-Qualified Model | `over_qualified_model_choice` | 고비용 모델 + 출력 ≤ 300 토큰 + tool call ≤ 1 | `detector_set_c.go` |
| Tool Schema Bloat | `tool_schema_bloat` | 스키마 ≥ 8192 bytes + 입력 비율 ≥ 25% | `detector_set_c.go` |

**Detector 인터페이스** (`internal/ports/parser.go`):
```go
type InsightDetector interface {
    Category() domain.DetectorCategory
    Detect(ctx context.Context, period domain.MonthlyPeriod, 
           sessions []domain.SessionSummary, 
           usageEntries []domain.UsageEntry) ([]domain.Insight, error)
}
```

---

## 데이터베이스 파일 위치

플랫폼별 경로 (XDG 표준 준수):

| 플랫폼 | Config 디렉터리 | DB 파일 경로 | 데이터 디렉터리 |
|--------|----------------|-----------|------------|
| Linux | `~/.config/llmbudget/` | `~/.local/share/llmbudget/llmbudget.sqlite3` | `~/.local/share/llmbudget/` |
| macOS | `~/Library/Application Support/llmbudget/` | `~/Library/Application Support/llmbudget/llmbudget.sqlite3` | `~/Library/Application Support/llmbudget/` |
| Windows | `%APPDATA%\llmbudget\` | `%LOCALAPPDATA%\llmbudget\llmbudget.sqlite3` | `%LOCALAPPDATA%\llmbudget\` |

**환경 변수 오버라이드** (Linux):
- `XDG_CONFIG_HOME` — config 디렉터리 위치
- `XDG_DATA_HOME` — data 디렉터리 위치

**TUI 플래그**:
- `--db <path>` — SQLite DB 경로 명시

DB 및 디렉터리는 최초 실행 시 자동 생성. 모든 마이그레이션은 `internal/adapters/sqlite/migrations.go`가 순차 자동 적용.

---

## 데이터베이스 스키마

주요 테이블 (마이그레이션에서 순차 생성):

### pricing_catalog_cache
OpenRouter 라이브 가격 캐시 저장소. 온라인 동기화 결과를 캐싱해 오프라인 모드 지원.
- `provider TEXT` — API 제공자 (anthropic, openai, gemini, openrouter)
- `model_key TEXT` — 모델 식별자 (예: claude-opus-4-5)
- `input_per_million_usd REAL` — 백만 입력 토큰당 USD
- `output_per_million_usd REAL` — 백만 출력 토큰당 USD
- `cached_at TEXT` — 캐시 시간
- `expires_at TEXT` — 캐시 만료 시간

### subscriptions
구독 요금 레코드.
- `id TEXT PRIMARY KEY` — 구독 ID
- `provider TEXT` — 서비스 제공자 (openai, claude, gemini)
- `plan_code TEXT` — 계획 코드 (예: openai-chatgpt-plus)
- `plan_name TEXT` — 인간 가독형 계획 이름
- `renewal_day INTEGER` — 월별 갱신 날짜 (1-31)
- `amount_usd REAL` — 월 요금 (USD)
- `starts_at TEXT` — 구독 시작 날짜 (ISO 8601)
- `ends_at TEXT` — 구독 종료 날짜 (NULL = 활성)
- `is_active INTEGER` — 활성 여부 (1/0)
- `created_at TEXT`, `updated_at TEXT` — 타임스탐프

### subscription_fees
월별 요금 계산 결과 (구독 레코드에서 파생).
- `subscription_id TEXT` — 구독 ID (FK)
- `provider TEXT`, `plan_code TEXT`
- `charged_at TEXT` — 청구 날짜
- `period_start_at TEXT`, `period_end_exclusive TEXT` — 요금 기간
- `fee_usd REAL` — 계산된 월 요금

### sessions
세션 요약 (파서에서 정규화).
- `session_id TEXT UNIQUE` — 세션 식별자
- `source_type TEXT` — 파서 유형 (claude, codex, gemini, opencode)
- `provider TEXT` — API 제공자 (anthropic, openai, gemini, openrouter)
- `tool_name TEXT` — 도구 이름 (Claude Code, Codex, ...)
- `billing_mode TEXT` — 청구 모드 (monthly, per_api_call)
- `project_name TEXT`, `model_name TEXT`
- `pricing_lookup_key TEXT` — 가격 카탈로그 조회 키
- `started_at TEXT`, `ended_at TEXT` — 시간 범위
- `input_tokens`, `output_tokens`, `cache_creation_tokens`, `cache_read_tokens` — 토큰 카운트
- `input_cost_usd`, `output_cost_usd`, `cache_creation_cost_usd`, `cache_read_cost_usd`, `tool_cost_usd`, `flat_cost_usd`, `total_cost_usd` — 비용 분석

### usage_entries
개별 API 호출 사용량.
- `entry_id TEXT UNIQUE` — 입력 ID
- `session_key TEXT` — 세션 ID (FK)
- `provider TEXT`, `source_type TEXT`, `billing_mode TEXT`
- `recorded_at TEXT` — 기록 시간
- `external_id TEXT` — 외부 ID (세션 로그에서)
- `project_name TEXT`, `agent_name TEXT`, `model_name TEXT`
- `pricing_lookup_key TEXT`
- `input_tokens`, `output_tokens`, `cache_creation_tokens`, `cache_read_tokens`
- `input_cost_usd`, `output_cost_usd`, ..., `cost_usd`
- `metadata_json TEXT` — JSON 메타데이터 (탐지기 사용)

### insights
탐지 결과 (프라이버시 안전 페이로드만).
- `id INTEGER PRIMARY KEY`
- `session_id INTEGER` — 세션 ID (FK)
- `rule_key TEXT` — 탐지기 카테고리 (context_avalanche, ...)
- `severity TEXT` — low, medium, high
- `title TEXT`, `summary TEXT` — 설명
- `detected_at TEXT` — 탐지 시간
- `estimated_waste_usd REAL`, `estimated_waste_tokens INTEGER`
- `dismissed_at TEXT` — 확인 시간 (NULL = 미확인)

### watcher_offsets
세션 파서 tail 오프셋 (증분 수집 추적).
- `watcher_key TEXT PRIMARY KEY` — 워처 키 (예: claude_code_sessions.jsonl)
- `source_path TEXT` — 소스 파일 경로
- `file_identity TEXT` — 파일 ID (inode, ...)
- `byte_offset INTEGER` — 마지막 읽은 바이트 오프셋
- `last_marker TEXT` — 마지막 읽은 라인 마커

### settings_snapshots
설정 스냅샷 (감사 추적).
- `schema_version INTEGER`
- `captured_at TEXT`
- `provider_*_enabled INTEGER` — 각 제공자 활성화 상태
- `default_*_billing_mode TEXT` — 각 소스의 기본 청구 모드
- `monthly_budget_usd REAL` — 전체 월 예산
- `monthly_subscription_budget_usd REAL`, `monthly_usage_budget_usd REAL`
- `warning_threshold_percent INTEGER`, `critical_threshold_percent INTEGER`
- `notifications_* INTEGER` — 알림 설정

**마이그레이션 버전**:
- `0001_initial.sql` — 전체 스키마 생성
- `0002_usage_entries_manual_fields.sql` — 복합 인덱스 추가
- `0003_insights_privacy_safe.sql` — insights 테이블 재구성 (프롬프트/응답 텍스트 컬럼 제거)
- `0004_budget_monitoring.sql` — 예산 감시 테이블 추가

---

## 가격 카탈로그 오버라이드

사용자가 가격을 재정의하는 방법. 우선순위:

1. **사용자 YAML 오버라이드** (최우선)
2. **임베디드 JSON** 
3. **OpenRouter 라이브 캐시** (최후)

### 오버라이드 파일 위치

경로: `~/.config/llmbudget/prices.yaml` (Linux 기준; 플랫폼별 config 디렉터리)

### YAML 포맷 예시

```yaml
version: 1
entries:
  - provider: anthropic
    model: claude-sonnet-4-5
    input_usd_per_mtok: 3.00
    output_usd_per_mtok: 15.00
    cache_write_usd_per_mtok: 3.75
    cache_read_usd_per_mtok: 0.30
    effective_from: "2025-01-01T00:00:00Z"
  - provider: openai
    model: gpt-4o
    input_usd_per_mtok: 5.00
    output_usd_per_mtok: 15.00
```

**필드 설명**:
- `provider` — API 제공자 (anthropic, openai, gemini, openrouter)
- `model` — 모델 이름
- `input_usd_per_mtok` — 백만 입력 토큰당 USD
- `output_usd_per_mtok` — 백만 출력 토큰당 USD
- `cache_write_usd_per_mtok` (선택사항) — 캐시 쓰기 가격
- `cache_read_usd_per_mtok` (선택사항) — 캐시 읽기 가격
- `effective_from` (선택사항) — 적용 시작 시간

**오류 처리**:
- 잘못된 YAML은 무시되고 경고 로그만 출력. 임베디드 데이터 사용 계속.

**임베디드 데이터 파일**:
- `internal/catalog/data/anthropic.json`
- `internal/catalog/data/openai.json`
- `internal/catalog/data/gemini.json`
- `internal/catalog/data/openrouter-cache.json`

---

## 세션 파서 지원 도구

모든 파서는 `internal/ports/SessionParser` 인터페이스 구현:

| 도구 | 로그 형식 | 파서 파일 | 특징 |
|------|----------|----------|------|
| Claude Code | `.jsonl` (현재 + 레거시) | `parsers/claude.go` | 증분 tail 지원, fsnotify 통합 |
| OpenAI Codex | `.jsonl` | `parsers/codex.go` | |
| Gemini CLI | `.json` | `parsers/gemini.go` | |
| OpenCode | SQLite DB | `parsers/opencode.go` | 자체 DB 직접 읽음 |

**워처 서비스** (`internal/service/watcher.go`):
- `fsnotify`로 파일 변경 감지
- 바이트 오프셋 추적 (incremental tail)
- 세션 파서를 비동기 호출
- 오프셋을 `watcher_offsets` 테이블에 저장

---

## 알림 종류

| AlertKind | 심각도 | 트리거 조건 |
|-----------|--------|-----------|
| `budget_threshold` | info, warning | 예산 임계치(% 기반) 교차 |
| `budget_overrun` | critical | 현재 지출이 예산 한도 초과 |
| `forecast_overrun` | warning | 예측된 월말 지출이 한도 초과 |
| `insight_detected` | varies | 낭비 패턴 탐지기 발동 |

**심각도**: `info`, `warning`, `critical`

---

## Provider 식별자

7가지 provider 식별자:

| Identifier | 역할 | 사용 예 |
|-----------|------|--------|
| `anthropic` | API 제공자 | 가격 카탈로그, 세션 가격 계산 |
| `openai` | API 제공자 | 가격 카탈로그, 세션 가격 계산 |
| `gemini` | API 제공자 | 가격 카탈로그, 세션 가격 계산 |
| `openrouter` | API 제공자 | 라이브 가격 캐시 |
| `claude` | 세션 로그 소스 | Claude Code 세션 파서 |
| `codex` | 세션 로그 소스 | OpenAI Codex 파서 |
| `opencode` | 세션 로그 소스 | OpenCode DB 파서 |

API 제공자는 가격 카탈로그와 비용 계산에 사용. 세션 소스는 로그 파일 형식을 구분.

---

## 주요 의존성 상세

| 라이브러리 | 버전 | 역할 |
|-----------|------|------|
| `charmbracelet/bubbletea` | v1.2.4 | Elm 아키텍처 TUI 프레임워크 |
| `charmbracelet/bubbles` | v0.20.0 | textinput, viewport 컴포넌트 |
| `charmbracelet/lipgloss` | v1.0.0 | TUI 스타일링 |
| `NimbleMarkets/ntcharts` | v0.5.1 | 차트 렌더링 (bar, line) |
| `wailsapp/wails/v2` | v2.10.2 | 데스크톱 GUI (Go + WebView) |
| `modernc.org/sqlite` | v1.34.2 | Pure Go SQLite (CGO 불필요) |
| `zalando/go-keyring` | v0.2.6 | OS 키링 (API 키 저장) |
| `google/uuid` | v1.6.0 | UUID 생성 |
| `gorilla/websocket` | v1.5.3 | Wails 개발 모드 핫리로드 |

**Go 버전**: **1.24.2** (또는 이상)

---

## 빌드 타겟 참고

### Makefile 타겟

```bash
make test              # go test ./... 실행
make build/gui         # Wails GUI 빌드 (webkit2gtk 4.1 기반, Ubuntu 24.04)
make run/gui-dev       # Wails 개발 모드 (핫리로드)
```

### TUI 직접 빌드

```bash
go build ./cmd/tui     # TUI 바이너리 빌드
go run ./cmd/tui       # TUI 직접 실행
```

### GUI 직접 빌드

```bash
wails build -debug     # 디버그 빌드
wails dev              # 개발 모드
```

---

## 확장 포인트

개발자가 새로운 기능을 추가하는 방법:

### 새 세션 파서 추가

1. `internal/adapters/parsers/` 디렉터리에 새 파일 생성 (예: `newparser.go`)
2. `internal/ports/SessionParser` 인터페이스 구현:
   ```go
   type SessionParser interface {
       Parse(ctx context.Context, sourcePath string) ([]SessionNormalized, error)
   }
   ```
3. `internal/adapters/tui/run.go` 또는 `cmd/gui/` 진입점에서 등록
4. `fsnotify` 워처 설정 (필요시)

### 새 탐지기 추가

1. `internal/service/detector_set_x.go`에 새 파일 생성 (또는 기존 파일에 추가)
2. `internal/ports/InsightDetector` 인터페이스 구현:
   ```go
   type InsightDetector interface {
       Category() domain.DetectorCategory
       Detect(ctx context.Context, period domain.MonthlyPeriod, 
              sessions []domain.SessionSummary, 
              usageEntries []domain.UsageEntry) ([]domain.Insight, error)
   }
   ```
3. `internal/domain/insight.go`에 새 `DetectorCategory` 상수 정의
4. `internal/service/insight_executor.go`에서 `NewDetectorSetX()` 호출 (필요시)

### 새 구독 프리셋 추가

1. `internal/service/subscription_presets.go` 파일 열기
2. `subscriptionPresets` 슬라이스에 새 항목 추가:
   ```go
   {Key: "new-key", Provider: domain.ProviderXXX, PlanName: "Plan Name", DefaultFeeUSD: 29.99, DefaultRenewalDay: 1},
   ```

### 새 마이그레이션 추가

1. `db/migrations/` 디렉터리에 새 파일 생성: `NNNN_description.sql` (NNNN = 4자리 순번)
2. SQL 마이그레이션 작성
3. 자동 적용됨 (`internal/adapters/sqlite/migrations.go` 참조)

---

## 주요 인터페이스 요약

### ports.SessionParser
세션 로그를 정규화된 `SessionNormalized` 항목으로 변환.

### ports.InsightDetector
세션과 사용량 항목에서 낭비 패턴 탐지.

### ports.PriceCatalog
모델별 가격 조회 (3단 우선순위 처리).

### ports.Repository
모든 데이터 CRUD 작업 (SQLite 구현).

### ports.AlertFilter, ports.SubscriptionFilter
조회 조건 필터링.

---

## 명령어 줄 플래그

### TUI

```bash
llm-budget-tracker-tui --db /path/to/database.sqlite3
```

- `--db` — SQLite 데이터베이스 경로 (선택사항, 기본값은 XDG 경로)

### GUI

```bash
llm-budget-tracker-gui
```

(Wails 애플리케이션, 플래그 없음)

---

## 테스트 실행

```bash
go test ./...              # 전체 패키지 테스트
go test ./internal/...     # internal 패키지만
go test -v ./...           # 상세 출력
go test -cover ./...       # 커버리지 포함
```

각 파일에는 `*_test.go` 페어가 있음.

---

최종 수정: 2026-04-19
