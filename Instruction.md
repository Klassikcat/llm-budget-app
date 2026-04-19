# 사용법 가이드 (Instruction)

LLM Budget Tracker TUI에서 LLM API 비용과 구독료를 관리하는 방법을 단계별로 설명합니다.

## 시작하기

TUI를 실행하려면 먼저 빌드한 후 다음 명령어를 사용하세요.

```bash
# 기본 DB 경로 사용
./tui

# 사용자 지정 DB 경로
./tui -db /path/to/llmbudget.sqlite3
```

최초 실행 시 데이터베이스와 필요한 디렉터리가 자동으로 생성됩니다. 기본 위치는 README.md를 참고하세요.

## 화면 구성

TUI 대시보드는 다음 요소들로 구성됩니다.

**상단 헤더**
- 현재 월 (UTC 기준)
- 최근 AlertEvent 배너: critical은 빨간색, warning은 시안/초록색으로 표시
- 시스템 상태 메시지

**4개 섹션** (Tab으로 이동)
1. **Overview**: 이번 달 총 비용, 예산 대비 사용률, 활성 구독 수 등 요약
2. **Provider Summary**: 제공자별(OpenAI, Claude, Gemini 등) 토큰 사용량과 비용
3. **Budgets**: 월별 예산 설정 현황과 경고 상태
4. **Recent Sessions**: 최근 8개 자동 수집된 세션 목록 (도구별 타임스탬프)

**하단 힌트**
- 주요 키 바인딩 요약

섹션 간 이동: `Tab`, `Shift+Tab`, `↑`, `↓`, `j`, `k` 사용.

## 키 바인딩 요약 (대시보드)

| 키 | 동작 |
|----|------|
| `Tab`, `↓`, `j` | 다음 섹션으로 포커스 이동 |
| `Shift+Tab`, `↑`, `k` | 이전 섹션으로 포커스 이동 |
| `m` | 수동 API 사용 입력 폼 열기 |
| `s` | 구독 요금 입력 폼 열기 |
| `l` | 구독 목록 보기 |
| `i` | 인사이트 목록 보기 |
| `g` | 그래프 화면 열기 |
| `r` | 대시보드, 알림, 인사이트 새로고침 |
| `q`, `Ctrl+C` | TUI 종료 |

## 구독 추가하기

구독 요금을 추가하는 가장 중요한 기능입니다. 대시보드에서 `s`를 누르면 구독 폼이 열립니다.

### 5.1 프리셋으로 추가하기 (권장)

구독 폼을 열면 9개의 LLM 구독 프리셋 목록이 표시됩니다.

| 플랜 | 제공자 | 기본 요금 (USD/월) | 갱신일 |
|------|--------|-------------------|-------|
| ChatGPT Plus | openai | $20 | 1 |
| ChatGPT Pro 5x | openai | $100 | 1 |
| ChatGPT Pro 20x | openai | $200 | 1 |
| Claude Pro | claude | $20 | 1 |
| Claude Max 5x | claude | $100 | 1 |
| Claude Max 20x | claude | $200 | 1 |
| Gemini Plus | gemini | $7.99 | 1 |
| Gemini Pro | gemini | $19.99 | 1 |
| Gemini Ultra | gemini | $249.99 | 1 |

**조작 방법**
- `↑`, `↓` (또는 `←`, `→`)로 프리셋 목록 이동
- `Enter`로 선택된 프리셋 활성화 (여러 개 동시 선택 가능)
- 선택된 프리셋 개수는 상단에 표시됨
- `Ctrl+S`로 선택된 모든 프리셋 저장
- `Esc`로 취소

프리셋 모드는 기본값(요금, 갱신일, 활성 상태, 기본 시작일)으로 구독을 저장합니다.

### 5.2 수동 입력으로 추가하기

"Others (Manual)" 프리셋을 선택하면 필드 기반 폼으로 전환됩니다.

**입력 필드**
- Provider: 제공자 선택 (`openai`, `claude`, `gemini`, `anthropic`, `openrouter` 등)
- Plan Name: 구독 플랜명 (예: "Claude Pro", "Custom Plan")
- Monthly Fee (USD): 월 요금 (양수)
- Renewal Day (1–28): 매월 갱신일
- Starts At (YYYY-MM-DD): 시작일 (선택, 비워두면 오늘)
- Ends At (YYYY-MM-DD): 종료일 (선택, 비활성 구독은 필수)
- Active (true/false): 활성 여부

**조작 방법**
- `Tab`, `Shift+Tab`으로 필드 이동
- 각 필드에 값 입력 (텍스트, 숫자, 날짜 형식 자동 검증)
- `Ctrl+S`로 저장 또는 마지막 필드에서 `Enter`로 제출
- `Esc`로 취소
- 에러 발생 시 문제 필드가 강조되며 수정 안내 표시

**참고**
- 비활성(`active: false`) 구독은 `ends_at` 필드가 필수입니다.
- 동일한 provider + plan name + starts_at 조합은 멱등(upsert)으로 처리되어 중복되지 않습니다.

## 구독 목록 관리

대시보드에서 `l`을 누르면 등록된 모든 구독을 확인할 수 있습니다.

**기능**
- 모든 구독이 리스트로 표시됨 (활성/비활성 상태 포함)
- `↑`, `k` / `↓`, `j`로 선택 이동
- `d`로 선택한 구독 비활성화 (완전 삭제가 아니라 inactive로 전환, 이력 보존)
- `r`로 목록 새로고침
- `Esc`, `Backspace`로 대시보드 복귀

비활성화된 구독은 이전 데이터를 보존하면서 더 이상 월별 롤업에 포함되지 않습니다.

## 수동 API 사용 입력

대시보드에서 `m`을 누르면 수동 API 사용 입력 폼이 열립니다. 자동 수집이 지원되지 않는 도구나 과거 기록 소급 입력에 사용합니다.

**입력 필드**
- Entry ID: 기록 ID (선택, 비워두면 자동 생성; 동일 ID로 재입력 시 덮어쓰기)
- Provider: 제공자 (`anthropic`, `openai`, `gemini`, `openrouter`, `claude`, `codex`, `opencode` 등)
- Model ID: 모델명 (예: `claude-sonnet-4-5`, `gpt-4o`)
- Occurred At: 발생 시간 (ISO 8601 또는 `YYYY-MM-DD HH:MM:SS`)
- Input Tokens: 입력 토큰 수 (선택)
- Output Tokens: 출력 토큰 수 (선택)
- Cached Tokens: 캐시된 입력 토큰 (선택, 해당하는 경우)
- Cache Write Tokens: 캐시 쓰기 토큰 (선택, 해당하는 경우)
- Project Name: 프로젝트명 (선택)

**조작 방법**
- `Tab`, `Shift+Tab`으로 필드 이동
- 각 필드에 값 입력 (형식 자동 검증)
- `Ctrl+S` 또는 마지막 필드에서 `Enter`로 제출
- `Esc`로 취소
- 에러 발생 시 문제 필드 강조 및 수정 안내

**참고**
- 비용은 내장된 가격 카탈로그에서 자동 계산됩니다.
- 폼은 "프롬프트/응답 텍스트를 절대 저장하지 않습니다"는 안내를 표시합니다. 해시, 카운트, 수치 메트릭만 기록됩니다.

## 인사이트 확인

대시보드에서 `i`를 누르면 탐지된 낭비 패턴 목록이 표시됩니다.

**8가지 자동 탐지 패턴**
1. Context Avalanche: 컨텍스트가 계속 증가하는 패턴
2. Missed Prompt Caching: 캐시할 수 있는 프롬프트를 캐시하지 않음
3. Planning Tax: 계획 관련 토큰 오버헤드
4. Repeated File Reads: 같은 파일을 반복해서 읽음
5. Retry Amplification: 재시도로 인한 토큰 증폭
6. Zombie Loops: 무한 또는 불필요한 루프
7. Over-Qualified Model: 더 저렴한 모델로 충분한데 고급 모델 사용
8. Tool Schema Bloat: 도구 스키마가 과도하게 큼

**조작 방법**
- `↑`, `k` / `↓`, `j`로 목록 이동
- `Enter`로 선택한 인사이트 상세 보기 진입
- `r`로 인사이트 재탐지 실행 (대시보드 전체 새로고침)
- `Esc`, `Backspace`로 대시보드 복귀

**상세 화면**
- 심각도 (low, medium, high)
- 범주 (8가지 탐지 패턴 중 하나)
- 탐지 시점
- 페이로드 요약: 영향받은 세션/기록 ID, 수치 메트릭, 해시값

**프라이버시**
- 프롬프트/응답 텍스트는 절대 저장되지 않습니다.
- 해시, 카운트, 수치 메트릭만 저장됩니다.

## 그래프 보기

대시보드에서 `g`를 누르면 그래프 화면이 열립니다. 4가지 시각화 탭을 제공합니다.

**탭 이동**
- `Tab`, `→`, `l`: 다음 탭으로 이동
- `Shift+Tab`, `←`, `h`: 이전 탭으로 이동

**4가지 탭**
1. **Model Token Usage**: 모델별 토큰 사용량 막대 차트
2. **Model Cost**: 모델별 비용 막대 차트
3. **Daily Token Trend**: 일별 토큰 사용 추이 라인 차트
4. **Token Breakdown**: 입력 / 출력 / 캐시 토큰 비율

**기타 키**
- `r`: 데이터 새로고침
- `Esc`, `Backspace`: 대시보드 복귀

그래프는 현재 월(UTC 기준) 데이터만 표시합니다. 이전 월 데이터를 보려면 GUI를 사용하세요.

## 자동 세션 수집 확인

TUI는 지원되는 도구에서 자동으로 세션 로그를 수집합니다.

**지원 도구**
- Claude Code: JSONL 세션 로그 (`~/.llm-budget-tracker/claude-code.jsonl`)
- OpenAI Codex: 자동 감지 및 파싱
- Gemini CLI: 자동 감지 및 파싱
- OpenCode: SQLite 데이터베이스 자동 감지

**확인 방법**
- 대시보드 "Recent Sessions" 섹션에 최대 8개 최근 세션 표시
- 각 세션에 타임스탬프와 도구명 표시
- 데이터가 보이지 않으면:
  1. 해당 도구에서 최근에 세션을 실행했는지 확인
  2. 로그 파일이 기본 경로에 있는지 확인 (README.md 참고)
  3. TUI에서 `r`을 눌러 대시보드 새로고침

자동 수집은 백그라운드에서 계속 모니터링되므로 도구 사용 후 TUI를 새로고침하면 데이터가 자동으로 반영됩니다.

## 예산과 알림

월별 예산 설정과 알림은 TUI의 제한된 기능입니다. 상세 설정은 GUI에서 합니다.

**TUI에서의 동작**
- 월별 예산 초과 상황이 대시보드 상단 "Budgets" 섹션과 알림 배너에 표시됨
- 알림 심각도:
  - `critical` (빨강): 예산 초과
  - `warning` (시안/초록): 임계치 도달 또는 예측 초과
  - `info`: 정보성 알림

**알림 종류**
- Budget Threshold: 예산의 특정 퍼센티지(예: 80%)에 도달
- Budget Overrun: 예산 초과
- Forecast Overrun: 월말 예측 비용이 예산 초과
- Insight Detected: 낭비 패턴 탐지

**예산 설정 변경**
- TUI에서는 직접 설정 불가 (추후 개선 예정)
- GUI에서 월별 예산을 설정하면 TUI에 자동 반영

## 자주 겪는 문제

**Q: 데이터베이스 파일이 어디에 있어요?**
A: 플랫폼에 따라 다릅니다. README.md의 "데이터베이스 경로" 섹션을 참고하세요. 기본 위치:
- Linux: `~/.local/share/llmbudget/llmbudget.sqlite3`
- macOS: `~/Library/Application Support/llmbudget/llmbudget.sqlite3`
- Windows: `%LOCALAPPDATA%\llmbudget\llmbudget.sqlite3`

**Q: 구독을 삭제하고 싶어요.**
A: TUI에서는 완전 삭제가 아니라 비활성화합니다. `l`로 구독 목록을 열고 삭제할 구독을 선택한 후 `d`를 누르면 비활성화됩니다. 이력이 보존됩니다. 완전 삭제는 DB에서 직접 SQL로 처리해야 합니다.

**Q: 가격이 최신이 아닌 것 같아요.**
A: 내장 가격표는 정기적으로 업데이트됩니다. 사용자 정의 가격 오버라이드는 `~/.config/llmbudget/prices.yaml`에서 설정하거나, OpenRouter 라이브 동기화 기능을 사용하세요. 자세한 내용은 GUIDE.md를 참고하세요.

**Q: TUI가 응답하지 않거나 멈췄어요.**
A: 다음을 시도하세요.
1. `Ctrl+C`로 TUI 종료
2. 몇 초 대기
3. `./tui`로 다시 실행
만약 데이터베이스 락이 남아 있으면 TUI 프로세스를 강제 종료한 후(`kill` 명령) 재실행하세요.

**Q: 자동 세션 수집이 동작하지 않아요.**
A: 다음을 확인하세요.
1. 해당 도구(Claude Code, OpenAI Codex 등)가 최근에 세션을 생성했는지 확인
2. 로그 파일이 기본 경로에 있는지 확인
3. TUI를 `r`로 새로고침
4. 여전히 안 되면 GUIDE.md에서 로그 경로 설정 방법을 참고하세요.

**Q: 수동으로 입력한 API 사용량이 저장되지 않았어요.**
A: 다음을 확인하세요.
1. 폼 하단의 에러 메시지 확인 (어느 필드가 문제인지 표시됨)
2. Provider와 Model ID가 지원되는 값인지 확인
3. 날짜/시간 형식이 올바른지 확인 (ISO 8601 또는 `YYYY-MM-DD HH:MM:SS`)
4. 토큰 값이 음수가 아닌지 확인
5. 재시도 전에 에러 필드를 수정하세요.
