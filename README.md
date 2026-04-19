# LLM Budget Tracker

LLM API 토큰 사용료와 월 구독료를 한 곳에서 관리하는 로컬 우선 개인 지출 추적기

## 개요

LLM Budget Tracker는 개인 개발자와 LLM 사용자를 위한 경량 지출 추적 도구입니다. Claude Code, OpenAI Codex, Gemini CLI, OpenCode 등의 세션 로그를 자동으로 수집하거나 수동으로 API 사용량을 입력하여 비용을 한눈에 파악할 수 있습니다.

모든 데이터는 로컬 SQLite 데이터베이스에 저장되며, 프라이버시를 최우선으로 합니다. 프롬프트와 응답 텍스트는 저장하지 않고, 토큰 수, 비용, 해시 등 수치 데이터만 기록합니다.

9개의 주요 LLM 구독 요금제를 내장하고 있으며, 사용자 정의 구독을 추가할 수 있습니다. 월별 예산 설정, 임계치 알림, 8가지 낭비 패턴 자동 탐지 등으로 비용을 지능적으로 관리할 수 있습니다.

## 주요 기능

- **자동 세션 수집**: Claude Code, OpenAI Codex, Gemini CLI, OpenCode의 세션 로그를 자동으로 감지하고 수집
- **수동 API 입력**: 지원하지 않는 도구의 API 사용량을 수동으로 추가
- **구독 관리**: 9개 LLM 구독 프리셋 (ChatGPT Plus/Pro, Claude Pro/Max, Gemini Plus/Pro/Ultra) + 사용자 정의 구독 지원
- **예산 추적**: 월별 예산 설정, 초과 시 알림, 월말 비용 초과 예측
- **낭비 패턴 탐지**: 8가지 자동 탐지기 (Context Avalanche, Missed Prompt Caching, Planning Tax, Repeated File Reads, Retry Amplification, Zombie Loops, Over-Qualified Model, Tool Schema Bloat)
- **시각화**: 모델별 토큰 사용량, 모델별 비용, 일일 토큰 추세, 토큰 구성 등 4가지 차트
- **가격 관리**: 내장 JSON 가격표 + YAML 사용자 오버라이드 + OpenRouter 라이브 캐시
- **두 가지 UI**: TUI (Bubbletea 기반 터미널) + GUI (Wails 기반 데스크톱)

## 아키텍처

```
cmd/tui → internal/adapters/tui (Bubbletea)  ┐
cmd/gui → internal/adapters/gui (Wails)       ┘→ internal/service/* → internal/domain/*
                                                        ↕
                                                  internal/adapters/sqlite
                                                        ↕
                                                  internal/adapters/parsers
                                                  (Claude/Codex/Gemini/OpenCode)
```

헥사고날 아키텍처(Ports-and-Adapters)로 설계되었습니다. 도메인 계층은 비즈니스 로직을 순수하게 유지하고, 어댑터 계층은 UI, 데이터베이스, 파서 등 외부 시스템과의 상호작용을 담당합니다. 서비스 계층은 도메인 모델을 조합하여 사용 사례를 구현합니다.

## 요구 사항

- **Go**: 1.24.2 이상
- **플랫폼**: Linux, macOS, Windows
- **GUI 빌드 (선택사항)**:
  - Linux: WebKitGTK 4.0 또는 4.1 개발 헤더 필요
  - macOS: WebKit이 포함된 Wails 자동 설치
  - Windows: WebView2 런타임 필요

## 빠른 시작 (TUI)

TUI(터미널 사용자 인터페이스)로 즉시 시작할 수 있습니다:

```bash
go build ./cmd/tui
./tui
```

### 데이터베이스 경로

기본적으로 다음 위치에 저장됩니다:

- **Linux**: `~/.local/share/llmbudget/llmbudget.sqlite3`
- **macOS**: `~/Library/Application Support/llmbudget/llmbudget.sqlite3`
- **Windows**: `%LOCALAPPDATA%\llmbudget\llmbudget.sqlite3`

`--db` 플래그로 재정의할 수 있습니다:

```bash
./tui --db /custom/path/llmbudget.sqlite3
```

## GUI 빌드 (선택)

Wails 기반 GUI를 빌드할 수 있습니다. Ubuntu 24.04 예시:

```bash
# WebKitGTK 개발 헤더 설치
sudo apt install libwebkit2gtk-4.1-dev

# 프로덕션 빌드
make build/gui

# 개발 모드 (핫 리로드)
make run/gui-dev
```

## 테스트

모든 테스트를 실행하려면:

```bash
make test

# 또는
go test ./...
```

## 주요 의존성

| 라이브러리 | 버전 | 역할 |
|----------|------|------|
| charmbracelet/bubbletea | v1.2.4 | TUI 프레임워크 |
| charmbracelet/bubbles | v0.20.0 | TUI 컴포넌트 (입력, 테이블 등) |
| charmbracelet/lipgloss | v1.0.0 | TUI 스타일링 |
| NimbleMarkets/ntcharts | v0.5.1 | 터미널 차트 렌더링 |
| wailsapp/wails/v2 | v2.10.2 | 데스크톱 GUI 프레임워크 |
| modernc.org/sqlite | v1.34.2 | 순수 Go SQLite 드라이버 (CGO 불필요) |
| zalando/go-keyring | v0.2.6 | OS 키링 통합 (API 키 보안 저장) |

## 문서

- [Instruction.md](./Instruction.md) — 사용법 가이드
- [GUIDE.md](./GUIDE.md) — 전체 레퍼런스

## 라이선스

라이선스 정보는 추후 추가 예정입니다.
