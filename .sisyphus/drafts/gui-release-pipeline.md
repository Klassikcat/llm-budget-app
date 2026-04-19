2026-04-19T12:20:00+09:00 - TODO: GUI CI/CD는 Wails/WebKitGTK 패키징 전략이 정리될 때까지 보류한다.

## Deferred scope
- GitHub Issue: #1 - Add GUI release pipeline for Wails build artifacts
- Current CI/CD scope: release tag push -> ensure/create release for the tag -> checkout tagged commit -> build TUI -> upload/update release assets for that tag

## Follow-up checklist
- [ ] GUI 타깃 플랫폼별 산출물 형식 정의 (app bundle, installer, zip 등)
- [ ] Wails 기반 GUI 빌드에 필요한 시스템 의존성 정리
- [ ] 코드 서명/노터리제이션 필요 여부 결정
- [ ] 태그 릴리스에 GUI 아티팩트 업로드 단계 추가
