# LLM Subscription Presets and Query Plan

## TL;DR

> **Quick Summary**: Extend the existing subscription save flow so users can either select one of the requested major LLM plans or keep using manual entry, while the system auto-generates hidden `planCode` and `subscriptionId`, accepts `StartsAt` as a date, and exposes a read-only subscription lookup flow.
>
> **Deliverables**:
> - Preset-backed + manual subscription registration on the existing subscription flow
> - Hidden deterministic `planCode` / `subscriptionId` generation for both paths
> - Date-only `StartsAt` input/output handling
> - Read-only subscription lookup/list capability following existing query/binding patterns
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 2 implementation waves + final verification
> **Critical Path**: T1/T2 → T4/T5 → T6/T7 → T8/T9 → T10

---

## Context

### Original Request
- as-is: 사용자가 직접 subscription form에서 subscription을 직접 작성
- to-be:
  - 사용자가 주요 LLM 구독(ChatGPT Plus/Pro 5x/Pro 20x, Anthropic Claude Pro/Max 5x/Max 20x, Gemini Plus/Pro/Ultra) 혹은 subscription fee 직접 등록 가능
  - 주요 LLM 구독은 자동 등록, 수동 등록은 현행 유지
  - 수동 등록 시에도 `plan code`, `subscription id`는 plan name/provider에 따라 자동 입력되고 사용자에게는 노출되지 않아야 함
  - `Starts At`은 RFC3339 문자열이 아니라 날짜로 등록
  - Subscription 조회 가능

### Interview / Research Summary
**Current flow**
- `internal/adapters/gui/forms_binding.go:179-233` saves subscriptions from the GUI binding.
- `internal/service/subscriptions.go:30-63` validates and persists subscriptions through the repository.
- `internal/adapters/sqlite/subscription_repository.go:14-130` upserts and lists subscriptions from SQLite.
- `internal/domain/subscription.go:67-165` currently requires `SubscriptionID`, `PlanCode`, `PlanName`, and normalized timestamps.

**Relevant existing patterns**
- `internal/config/settings.go:19-32,68-107` already models provider-specific subscription defaults for OpenAI / Claude / Gemini.
- `internal/catalog/embedded.go:1-13` shows the project already prefers embedded/static data over admin-managed catalogs.
- `internal/service/dashboard_query.go:71-157` + `internal/adapters/gui/dashboard_binding.go:15-78` define the preferred read/query architecture.
- `internal/adapters/gui/forms_binding.go:421-438` already accepts `YYYY-MM-DD` parsing, but the current form contract still exposes timestamp-style fields.

### Metis Review (addressed)
- Locked lookup scope to **read-only list v1**; no detail/edit/history expansion.
- Locked preset scope to the **9 named plans only**; no catalog CRUD, seed workflow, or remote price sync.
- Defaulted deterministic hidden identifiers to:
  - `planCode`: provider + normalized plan slug
  - `subscriptionId`: `planCode + startsAt(date)` deterministic internal id
- Defaulted preset flow to **auto-fill provider / plan / fee / renewal day**, while still allowing fee / renewal day / startsAt / isActive adjustment in the shared form.
- Defaulted date conversion to **UTC midnight** for date-only `StartsAt` values.

---

## Work Objectives

### Core Objective
Implement a single shared subscription registration experience that supports requested major LLM presets and manual entry without exposing internal identifiers, while also adding a minimal read-only subscription lookup flow that matches existing service/binding patterns.

### Concrete Deliverables
- Preset definitions for:
  - ChatGPT Plus / Pro 5x / Pro 20x
  - Claude Pro / Max 5x / Max 20x
  - Gemini Plus / Pro / Ultra
- Hidden deterministic `planCode` / `subscriptionId` generation
- Date-only `StartsAt` contract for GUI-facing subscription flows
- Subscription lookup/list query service + GUI binding
- Tests covering preset save, manual save, identifier generation, duplicate-safe behavior, date parsing, and empty/non-empty lookup states

### Definition of Done
- [ ] `go test ./...` passes
- [ ] Preset and manual save flows work without user-provided `planCode` / `subscriptionId`
- [ ] Subscription lookup returns persisted subscriptions through a dedicated read path
- [ ] `StartsAt` GUI contract is date-oriented, not RFC3339-oriented
- [ ] Requested preset list is available and no extra catalog management surface is introduced

### Must Have
- Preserve the existing manual registration path
- Use one shared save model for preset and manual subscriptions
- Keep generated identifiers hidden from the user
- Follow existing query-service + GUI binding architecture
- Keep verification agent-executable only

### Must NOT Have (Guardrails)
- No preset database table, preset CRUD UI, or pricing sync workflow
- No full subscription detail screen, edit console, or history browser in this plan
- No user-entered RFC3339 requirement for `StartsAt`
- No unrelated dashboard / ingestion / budget refactor
- No requirement for CI or browser-E2E infrastructure adoption in this scope

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: TDD
- **Framework**: Go `testing`
- **Primary command**: `go test ./...`

### QA Policy
Every implementation task must include agent-executed QA scenarios and concrete evidence paths.

- **Go/domain/service/repository verification**: Bash via `go test ./...` or targeted `go test ./path -run TestName`
- **Binding verification**: Go tests that exercise GUI bindings directly (no browser infra required)
- **Persistence verification**: temp SQLite round-trip tests using repository/bootstrap helpers
- **Evidence location**: `.sisyphus/evidence/task-{N}-{scenario-slug}.txt`

---

## Execution Strategy

### Parallel Execution Waves

```text
Wave 1 (foundation + test scaffolding)
├── T1: Save-flow TDD scaffolding [quick]
├── T2: Lookup-flow TDD scaffolding [quick]
├── T3: Static preset definitions [quick]
├── T4: Hidden identifier + date normalization rules [unspecified-high]
└── T5: GUI subscription DTO/state contract changes [quick]

Wave 2 (implementation + wiring)
├── T6: Shared preset/manual save binding [unspecified-high]
├── T7: Service + repository save semantics [unspecified-high]
├── T8: Subscription lookup query service [unspecified-high]
├── T9: Subscription lookup GUI binding + app wiring [quick]
└── T10: Persistence/integration regression pass [deep]

Critical Path: T1/T2 → T4/T5 → T6/T7 → T8/T9 → T10
Parallel Speedup: ~55-65% vs fully sequential execution
Max Concurrent: 5
```

### Dependency Matrix
- **T1**: Blocked By none → Blocks T6, T7, T10
- **T2**: Blocked By none → Blocks T8, T9, T10
- **T3**: Blocked By none → Blocks T6
- **T4**: Blocked By none → Blocks T6, T7, T10
- **T5**: Blocked By none → Blocks T6, T9
- **T6**: Blocked By T1, T3, T4, T5 → Blocks T10
- **T7**: Blocked By T1, T4 → Blocks T8, T10
- **T8**: Blocked By T2, T7 → Blocks T9, T10
- **T9**: Blocked By T2, T5, T8 → Blocks T10
- **T10**: Blocked By T1, T2, T4, T6, T7, T8, T9 → Blocks Final Verification

### Agent Dispatch Summary
- **Wave 1**
  - T1 → `quick`
  - T2 → `quick`
  - T3 → `quick`
  - T4 → `unspecified-high`
  - T5 → `quick`
- **Wave 2**
  - T6 → `unspecified-high`
  - T7 → `unspecified-high`
  - T8 → `unspecified-high`
  - T9 → `quick`
  - T10 → `deep`
- **Final Verification**
  - F1 → `oracle`
  - F2 → `unspecified-high`
  - F3 → `unspecified-high`
  - F4 → `deep`

---

## TODOs

- [ ] T1. Codify preset/manual save requirements with failing tests

  **What to do**:
  - Add/extend tests that define the expected save behavior for both preset-based and manual subscription registration.
  - Cover hidden `planCode` / `subscriptionId` generation, date-only `StartsAt`, editable preset-derived fee/renewal day, and validation failures.
  - Keep these tests red until the later implementation tasks satisfy them.

  **Must NOT do**:
  - Do not implement production logic in this task.
  - Do not widen scope into lookup/list behavior (handled separately in T2).

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: tightly scoped test-first changes in 1-2 existing test files.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `playwright`: no browser automation exists or is needed for binding-level Go tests.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T2, T3, T4, T5)
  - **Blocks**: T6, T7, T10
  - **Blocked By**: None

  **References**:
  - `internal/adapters/gui/forms_binding_test.go:20-195` - Existing end-to-end binding test pattern for settings + subscription + manual entry persistence assertions.
  - `internal/service/subscriptions_test.go:117-173` - Existing CRUD lifecycle expectations for save/update/disable behavior.
  - `internal/adapters/gui/forms_binding.go:179-233` - The current save path under test; use it to define the new expected input contract.
  - `internal/adapters/gui/forms_binding.go:421-450` - Existing timestamp parser already accepts `YYYY-MM-DD`; tests should lock this behavior to date-first usage.

  **Acceptance Criteria**:
  - [ ] Failing tests exist for preset save without user-supplied `subscriptionId` / `planCode`.
  - [ ] Failing tests exist for manual save without user-supplied `subscriptionId` / `planCode`.
  - [ ] Failing tests exist for invalid blank provider/plan and invalid date input.
  - [ ] Targeted test command fails before implementation and documents the expected behavior.

  **QA Scenarios**:
  ```text
  Scenario: preset save requirement is captured
    Tool: Bash (go test)
    Preconditions: New/updated tests added in forms_binding_test.go or subscriptions_test.go
    Steps:
      1. Run `go test ./internal/adapters/gui ./internal/service -run 'Test.*Subscription.*Preset|Test.*Subscription.*Manual'`
      2. Observe at least one failure proving the new preset/manual expectations are not yet implemented
      3. Confirm the failure text references missing/generated id/code or preset behavior
    Expected Result: command exits non-zero with assertion failures tied to the newly added requirements
    Failure Indicators: command passes immediately or failures are unrelated compile errors
    Evidence: .sisyphus/evidence/task-T1-preset-manual-red.txt

  Scenario: invalid input cases are specified
    Tool: Bash (go test)
    Preconditions: Negative-path tests are added
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*Subscription.*Invalid'`
      2. Inspect failure output for expected validation assertions around blank provider/plan or invalid date
    Expected Result: command exits non-zero because production validation has not yet been updated to satisfy the new assertions
    Failure Indicators: no matching tests, or only unrelated failures
    Evidence: .sisyphus/evidence/task-T1-invalid-input-red.txt
  ```

  **Commit**: YES
  - Message: `test(subscription): codify save flow requirements`
  - Files: `internal/adapters/gui/forms_binding_test.go`, `internal/service/subscriptions_test.go`
  - Pre-commit: `go test ./internal/adapters/gui ./internal/service`

- [ ] T2. Codify subscription lookup requirements with failing tests

  **What to do**:
  - Add tests that define the new read-only subscription lookup/list behavior.
  - Cover empty state, populated state, ordering, and the minimal fields the lookup response must expose.
  - Follow the existing dashboard query/binding test style.

  **Must NOT do**:
  - Do not implement the query service or binding in this task.
  - Do not expand into edit/detail/history behaviors.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: test scaffolding in a small number of files with an existing pattern to mirror.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `playwright`: unnecessary because query behavior is verified through Go binding tests.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T3, T4, T5)
  - **Blocks**: T8, T9, T10
  - **Blocked By**: None

  **References**:
  - `internal/adapters/gui/dashboard_binding_test.go:14-115` - Existing binding-level query tests covering empty and populated responses.
  - `internal/service/dashboard_query.go:71-157` - Current query-service orchestration pattern to emulate in tests.
  - `internal/adapters/gui/dashboard_binding.go:15-78` - Existing binding contract style for read-only responses.
  - `internal/ports/repository.go:37-43` - Current subscription filter surface that any lookup tests must align with or intentionally extend.

  **Acceptance Criteria**:
  - [ ] Failing tests define a read-only subscription list response.
  - [ ] Empty-state tests exist and assert zero rows without nil slices.
  - [ ] Populated-state tests assert ordering/field mapping for persisted subscriptions.
  - [ ] No test assumes detail or edit workflows.

  **QA Scenarios**:
  ```text
  Scenario: lookup populated-state requirement is captured
    Tool: Bash (go test)
    Preconditions: New lookup query/binding tests added
    Steps:
      1. Run `go test ./internal/adapters/gui ./internal/service -run 'Test.*Subscription.*Lookup|Test.*Subscription.*List'`
      2. Confirm at least one test fails because lookup logic/binding is not implemented yet
      3. Verify failure text references missing list rows, response fields, or empty state expectations
    Expected Result: non-zero exit with assertion failures tied to lookup behavior
    Failure Indicators: compile-only failure with no assertions, or all tests unexpectedly pass
    Evidence: .sisyphus/evidence/task-T2-lookup-red.txt

  Scenario: empty state requirement is captured
    Tool: Bash (go test)
    Preconditions: Empty-state assertions added
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*Subscription.*Empty'`
      2. Inspect output for the explicit empty-list expectation
    Expected Result: command exits non-zero until the empty-state response contract is implemented
    Failure Indicators: no empty-state test present or unrelated errors only
    Evidence: .sisyphus/evidence/task-T2-empty-red.txt
  ```

  **Commit**: YES
  - Message: `test(subscription): codify lookup requirements`
  - Files: `internal/adapters/gui/*subscription*_test.go` or `internal/adapters/gui/dashboard_binding_test.go`, `internal/service/*subscription*_test.go`
  - Pre-commit: `go test ./internal/adapters/gui ./internal/service`

- [ ] T3. Add static preset definitions for the requested LLM plans

  **What to do**:
  - Introduce a static preset source that enumerates exactly the requested 9 plans.
  - Reuse existing code-level configuration/catalog patterns rather than adding persistence-backed preset management.
  - Make the preset structure usable by the shared save flow and future lookup response mapping.

  **Must NOT do**:
  - Do not add a preset DB table, migration, remote fetcher, or CRUD interface.
  - Do not include providers/plans beyond the requested list.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: compact static-data/config work modeled after existing embedded/default patterns.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `writing`: not documentation work; this is code-structure planning only.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T4, T5)
  - **Blocks**: T6
  - **Blocked By**: None

  **References**:
  - `internal/config/settings.go:19-32` - Existing provider-level subscription default types that can anchor preset modeling.
  - `internal/config/settings.go:83-107` - Existing default subscription entries for OpenAI / Claude / Gemini; extend pattern rather than inventing a new store.
  - `internal/catalog/embedded.go:1-13` - Evidence that this codebase already prefers embedded/static catalog data patterns.
  - `internal/adapters/gui/forms_binding_test.go:40-64` - Existing tests already express provider default plan settings in GUI-facing state.

  **Acceptance Criteria**:
  - [ ] Exactly 9 presets are defined, matching the user request.
  - [ ] Presets are available from a static/code-level source.
  - [ ] No DB schema or runtime sync dependency is introduced.
  - [ ] Preset structure includes provider, plan name, fee, and renewal day data needed by the form.

  **QA Scenarios**:
  ```text
  Scenario: preset catalog includes exactly the requested plans
    Tool: Bash (go test)
    Preconditions: Static preset definitions and coverage tests are added
    Steps:
      1. Run `go test ./internal/config ./internal/service -run 'Test.*Subscription.*Preset'`
      2. Assert the test output confirms the expected provider/plan combinations only
    Expected Result: targeted tests pass and verify all 9 requested presets, with no extras
    Failure Indicators: missing requested plans, extra plans, or reliance on persistence/sync
    Evidence: .sisyphus/evidence/task-T3-preset-catalog.txt

  Scenario: no persistence-backed preset management is introduced
    Tool: Bash (go test)
    Preconditions: No migration or repo changes should be necessary for presets
    Steps:
      1. Run `go test ./...`
      2. Confirm no preset-management-specific schema/runtime errors appear
    Expected Result: tests that touch config/service layers pass without requiring DB-backed preset setup
    Failure Indicators: test failures referencing missing preset table/migration/sync state
    Evidence: .sisyphus/evidence/task-T3-no-preset-db.txt
  ```

  **Commit**: YES
  - Message: `feat(subscription): add static llm presets`
  - Files: `internal/config/settings.go`, `internal/service/*subscription*preset*.go` (or equivalent)
  - Pre-commit: `go test ./internal/config ./internal/service`

- [ ] T4. Define hidden identifier generation and date normalization rules

  **What to do**:
  - Add the canonical generation logic for `planCode` and `subscriptionId` so both preset and manual flows use the same deterministic rules.
  - Normalize date-only `StartsAt` values to UTC midnight semantics and make duplicate-safe behavior explicit.
  - Lock the rule in tests so future save-path changes cannot drift.

  **Must NOT do**:
  - Do not leak generated identifiers into GUI-facing required input fields.
  - Do not introduce random IDs or nondeterministic generation behavior.

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: domain/business-rule work where correctness and consistency matter more than raw speed.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `ultrabrain`: this is important but not logic-heavy enough to justify the heaviest profile.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T5)
  - **Blocks**: T6, T7, T10
  - **Blocked By**: None

  **References**:
  - `internal/domain/subscription.go:67-165` - Current domain validation requires `SubscriptionID` and `PlanCode`; update the authoritative rules here or adjacent helpers.
  - `internal/adapters/gui/forms_binding.go:179-215` - Current binding constructs `domain.Subscription` directly from user-entered identifiers; this is the leak point to eliminate.
  - `internal/adapters/gui/forms_binding.go:421-438` - Existing parser already normalizes `YYYY-MM-DD` to UTC midnight, which should become the explicit rule for GUI date input.
  - `db/migrations/0001_initial.sql:14-29` - Existing schema already stores `id`, `plan_code`, and `starts_at`; use this to avoid unnecessary migrations.

  **Acceptance Criteria**:
  - [ ] One deterministic generation rule exists for `planCode`.
  - [ ] One deterministic generation rule exists for `subscriptionId` using normalized date semantics.
  - [ ] Date-only `StartsAt` values normalize to UTC midnight.
  - [ ] Duplicate-safe behavior is defined and covered by tests.

  **QA Scenarios**:
  ```text
  Scenario: deterministic id/code generation works
    Tool: Bash (go test)
    Preconditions: Identifier-generation tests implemented
    Steps:
      1. Run `go test ./internal/service ./internal/domain -run 'Test.*Subscription.*Generate|Test.*Subscription.*Identifier'`
      2. Verify the same provider/plan/date input yields the same generated output across runs
      3. Verify a date change yields a different `subscriptionId` while preserving the same `planCode`
    Expected Result: tests pass and document deterministic generation behavior
    Failure Indicators: random output, flaky tests, or mismatched id/code expectations
    Evidence: .sisyphus/evidence/task-T4-generated-identifiers.txt

  Scenario: date-only normalization uses UTC midnight
    Tool: Bash (go test)
    Preconditions: Date-normalization coverage exists
    Steps:
      1. Run `go test ./internal/adapters/gui ./internal/domain -run 'Test.*StartsAt.*Date|Test.*Timestamp.*DateOnly'`
      2. Verify assertions expect `00:00:00Z` for date-only input
    Expected Result: tests pass with explicit UTC-midnight normalization semantics
    Failure Indicators: tests still accept/emit arbitrary timestamp time components for date-only input
    Evidence: .sisyphus/evidence/task-T4-date-normalization.txt
  ```

  **Commit**: YES
  - Message: `feat(subscription): define generated identifiers and date rules`
  - Files: `internal/domain/subscription.go`, `internal/service/*subscription*identity*.go` (or equivalent)
  - Pre-commit: `go test ./internal/domain ./internal/service ./internal/adapters/gui`

- [ ] T5. Update GUI subscription DTO/state contracts for presets and hidden fields

  **What to do**:
  - Redesign the GUI-facing subscription input/output contract so users no longer provide `subscriptionId` or `planCode` directly.
  - Add the minimum preset-selection signal needed by the binding while keeping one shared form model.
  - Make the `StartsAt` contract clearly date-oriented for GUI payloads and responses.

  **Must NOT do**:
  - Do not expose hidden generated identifiers back into editable GUI state.
  - Do not create separate, duplicated form contracts for preset vs manual paths.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: constrained contract work in DTO/state files with direct downstream usage.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `visual-engineering`: no frontend component code is present in-repo; this is binding contract work.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T4)
  - **Blocks**: T6, T9
  - **Blocked By**: None

  **References**:
  - `internal/adapters/gui/forms_types.go:91-118` - Current GUI input/output structs expose `SubscriptionID` and `PlanCode`; these are the primary contract surfaces to change.
  - `internal/adapters/gui/forms_binding.go:604-618` - Current response mapping emits full timestamp strings and generated identifiers; update the GUI-facing state appropriately.
  - `internal/adapters/gui/forms_binding.go:179-233` - The binding currently assumes the old DTO shape and must remain aligned with any contract changes.

  **Acceptance Criteria**:
  - [ ] GUI input contract no longer requires user-entered `subscriptionId` / `planCode`.
  - [ ] GUI contract can represent either preset selection or manual provider/plan entry in one shared model.
  - [ ] GUI-facing `StartsAt` is date-oriented.
  - [ ] Response mapping avoids re-exposing editable hidden identifiers.

  **QA Scenarios**:
  ```text
  Scenario: GUI contract hides generated fields
    Tool: Bash (go test)
    Preconditions: Binding/DTO tests exist
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*Subscription.*Contract|Test.*Subscription.*Binding'`
      2. Verify assertions confirm the save input works without `subscriptionId` / `planCode`
    Expected Result: tests pass with GUI contracts that no longer require hidden fields
    Failure Indicators: tests still construct inputs with user-supplied hidden ids or plan codes
    Evidence: .sisyphus/evidence/task-T5-hidden-fields.txt

  Scenario: GUI-facing StartsAt uses date-only semantics
    Tool: Bash (go test)
    Preconditions: Response/input contract assertions are updated
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*StartsAt.*Date|Test.*Subscription.*DateOnly'`
      2. Confirm the expected payloads/assertions use date-oriented values
    Expected Result: GUI contract tests pass with date-only expectations
    Failure Indicators: RFC3339 strings remain the required GUI contract in tests
    Evidence: .sisyphus/evidence/task-T5-date-contract.txt
  ```

  **Commit**: YES
  - Message: `feat(gui): hide subscription identifiers in form contracts`
  - Files: `internal/adapters/gui/forms_types.go`, `internal/adapters/gui/forms_binding.go`
  - Pre-commit: `go test ./internal/adapters/gui`

- [ ] T6. Implement one shared preset/manual save flow in FormsBinding

  **What to do**:
  - Rework `FormsBinding.SaveSubscription` so preset selection and manual entry converge into one authoritative save path.
  - Resolve preset data, merge editable user overrides, apply generated hidden identifiers, and persist through the existing service.
  - Preserve update/edit semantics where appropriate without requiring the user to type internal identifiers.

  **Must NOT do**:
  - Do not split preset save and manual save into unrelated code paths.
  - Do not bypass domain/service validation.

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: this is the main orchestration hotspot where contract, preset, and validation rules intersect.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `quick`: too much branching logic and cross-file dependency for the lowest-effort profile.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2
  - **Blocks**: T10
  - **Blocked By**: T1, T3, T4, T5

  **References**:
  - `internal/adapters/gui/forms_binding.go:179-233` - Current `SaveSubscription` implementation to refactor into the shared preset/manual resolver.
  - `internal/adapters/gui/forms_binding.go:199-202` - Existing lookup-by-subscription-id behavior used to preserve `CreatedAt`; update carefully because IDs become system-generated.
  - `internal/adapters/gui/forms_binding_test.go:93-174` - Existing binding save assertions and persistence checks to extend for the new flow.
  - `internal/config/settings.go:83-107` - Existing provider default plan metadata that can inform preset selection.

  **Acceptance Criteria**:
  - [ ] Preset selection auto-fills provider/plan/fee/renewal day through the shared save flow.
  - [ ] Manual save works without user-entered `subscriptionId` / `planCode`.
  - [ ] Editable preset overrides (fee, renewal day, startsAt, isActive) are preserved on save.
  - [ ] Binding continues to return persisted subscription state after save.

  **QA Scenarios**:
  ```text
  Scenario: preset selection saves through the shared binding path
    Tool: Bash (go test)
    Preconditions: FormsBinding save logic implemented
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*Subscription.*Preset'`
      2. Verify the targeted test saves a preset-backed subscription and asserts persisted provider/plan/fee data plus generated hidden ids
    Expected Result: targeted test passes
    Failure Indicators: preset path bypasses save flow, generated ids missing, or persisted fields mismatch
    Evidence: .sisyphus/evidence/task-T6-preset-save.txt

  Scenario: manual save succeeds without exposed identifiers
    Tool: Bash (go test)
    Preconditions: Manual-path tests updated
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*Subscription.*Manual'`
      2. Confirm assertions show success with no user-supplied `subscriptionId` / `planCode`
    Expected Result: targeted test passes and persisted record contains generated hidden values
    Failure Indicators: binding still requires/expects hidden identifiers from input
    Evidence: .sisyphus/evidence/task-T6-manual-save.txt
  ```

  **Commit**: YES
  - Message: `feat(subscription): unify preset and manual save flow`
  - Files: `internal/adapters/gui/forms_binding.go`, related save-flow tests
  - Pre-commit: `go test ./internal/adapters/gui`

- [ ] T7. Align service and repository save semantics with generated hidden identifiers

  **What to do**:
  - Move/generated-id assumptions out of the user-facing layer and ensure service/repository persistence works correctly with system-generated ids.
  - Preserve create/update behavior, `CreatedAt` / `UpdatedAt` handling, and duplicate-safe semantics under the new identifier policy.
  - Keep schema changes minimal; prefer reusing the current `subscriptions` table unless a truly minimal migration is proven necessary.

  **Must NOT do**:
  - Do not rely on GUI-only logic for invariant enforcement.
  - Do not introduce unnecessary schema churn if the current schema already supports the new behavior.

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: persistence semantics and service invariants need careful handling.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `quick`: repository/service coordination is too risk-prone for a minimal-effort profile.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T8 after contract alignment)
  - **Blocks**: T8, T10
  - **Blocked By**: T1, T4

  **References**:
  - `internal/service/subscriptions.go:30-63` - Current save/list behavior and timestamp stamping logic.
  - `internal/adapters/sqlite/subscription_repository.go:14-71` - Current upsert semantics keyed by subscription `id`.
  - `internal/adapters/sqlite/subscription_repository.go:74-130` - Current listing/order behavior that lookup will build upon.
  - `db/migrations/0001_initial.sql:14-29` - Existing subscriptions schema; validate reuse before considering any migration.
  - `internal/service/subscriptions_test.go:117-173` - Existing lifecycle/save-update-disable expectations to preserve.

  **Acceptance Criteria**:
  - [ ] Service/repository save behavior supports generated hidden identifiers end-to-end.
  - [ ] Existing create/update lifecycle semantics are preserved or explicitly redefined in tests.
  - [ ] Duplicate-safe semantics match the deterministic identifier policy.
  - [ ] No unnecessary migration is introduced if current schema is sufficient.

  **QA Scenarios**:
  ```text
  Scenario: service save path preserves lifecycle behavior
    Tool: Bash (go test)
    Preconditions: Service-layer tests updated
    Steps:
      1. Run `go test ./internal/service -run 'Test.*Subscription.*Lifecycle|Test.*Subscription.*Save'`
      2. Verify save/update behavior passes under generated-id rules
    Expected Result: targeted tests pass with correct created/updated timestamps and persisted identifiers
    Failure Indicators: updates create duplicate unintended rows, or lifecycle assertions fail
    Evidence: .sisyphus/evidence/task-T7-service-lifecycle.txt

  Scenario: repository list/upsert works with generated ids
    Tool: Bash (go test)
    Preconditions: Repository coverage exists for subscription persistence
    Steps:
      1. Run `go test ./internal/adapters/sqlite -run 'Test.*Subscription.*RoundTrip|Test.*Subscription.*Repository'`
      2. Verify saved records round-trip with generated ids and expected ordering
    Expected Result: repository tests pass
    Failure Indicators: generated ids are not persisted, list filters break, or ordering is unstable
    Evidence: .sisyphus/evidence/task-T7-repository-roundtrip.txt
  ```

  **Commit**: YES
  - Message: `feat(subscription): align service and repository save semantics`
  - Files: `internal/service/subscriptions.go`, `internal/adapters/sqlite/subscription_repository.go`, related tests
  - Pre-commit: `go test ./internal/service ./internal/adapters/sqlite`

- [ ] T8. Implement a read-only subscription lookup query service

  **What to do**:
  - Add a dedicated subscription lookup/list query service that follows the dashboard query-service pattern.
  - Reuse repository list/filter capabilities and return a response model designed for read-only subscription listing.
  - Keep v1 scope minimal: list/read only, with sensible ordering and empty-state behavior.

  **Must NOT do**:
  - Do not collapse lookup into the existing dashboard query service if that makes the boundary blurrier.
  - Do not introduce detail/edit/history semantics.

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: new read-path design should be explicit and pattern-aligned.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `deep`: a dedicated query service is new work, but still bounded and pattern-driven.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7)
  - **Blocks**: T9, T10
  - **Blocked By**: T2, T7

  **References**:
  - `internal/service/dashboard_query.go:71-157` - Preferred service-layer query orchestration pattern.
  - `internal/ports/repository.go:37-43` - Existing subscription filter inputs that can seed lookup querying.
  - `internal/adapters/sqlite/subscription_repository.go:74-130` - Current list implementation and default ordering by provider/plan_code/starts_at.
  - `internal/adapters/gui/dashboard_binding_test.go:14-104` - Empty vs populated query response expectations worth mirroring.

  **Acceptance Criteria**:
  - [ ] A dedicated read-only subscription query service exists.
  - [ ] Lookup returns empty and populated responses without nil-slice ambiguity.
  - [ ] Lookup ordering is deterministic.
  - [ ] Query scope remains list/read only.

  **QA Scenarios**:
  ```text
  Scenario: lookup query returns populated results
    Tool: Bash (go test)
    Preconditions: Query service and tests implemented
    Steps:
      1. Run `go test ./internal/service -run 'Test.*Subscription.*Lookup|Test.*Subscription.*List'`
      2. Confirm seeded subscriptions are returned in deterministic order with expected fields
    Expected Result: targeted query-service tests pass
    Failure Indicators: rows missing, unstable ordering, or empty-state mishandling
    Evidence: .sisyphus/evidence/task-T8-query-populated.txt

  Scenario: lookup query handles empty state cleanly
    Tool: Bash (go test)
    Preconditions: Empty-state tests exist
    Steps:
      1. Run `go test ./internal/service -run 'Test.*Subscription.*Empty'`
      2. Confirm zero-row scenarios return empty collections and not nil-driven special cases
    Expected Result: targeted tests pass
    Failure Indicators: nil slices, panic, or incorrect empty-state flags
    Evidence: .sisyphus/evidence/task-T8-query-empty.txt
  ```

  **Commit**: YES
  - Message: `feat(subscription): add lookup query service`
  - Files: `internal/service/*subscription*query*.go`, related tests, optional `internal/ports/repository.go`
  - Pre-commit: `go test ./internal/service`

- [ ] T9. Add subscription lookup GUI binding and wire it into the Wails app

  **What to do**:
  - Introduce a GUI binding for the new subscription lookup query service.
  - Follow the dashboard binding style for input parsing, context handling, and response mapping.
  - Register the new binding with the Wails app without disturbing existing dashboard/forms bindings.

  **Must NOT do**:
  - Do not bury lookup behavior inside `FormsBinding` if it makes the binding boundary muddy.
  - Do not add browser-specific code paths or unrelated app bootstrap changes.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: bounded adapter/wiring work following an established binding pattern.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `visual-engineering`: the repository lacks in-tree frontend component code; this task is Wails binding/wiring only.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2
  - **Blocks**: T10
  - **Blocked By**: T2, T5, T8

  **References**:
  - `internal/adapters/gui/dashboard_binding.go:15-78` - Binding design pattern for read-only service queries.
  - `internal/adapters/gui/app.go:14-55` - Current Wails binding registration surface.
  - `internal/adapters/gui/run.go:19-24` - App construction path where the new binding must be wired.
  - `internal/adapters/gui/dashboard_binding_test.go:106-125` - Existing test helper pattern for binding + store setup.

  **Acceptance Criteria**:
  - [ ] A dedicated GUI binding exposes subscription lookup/list behavior.
  - [ ] Binding is registered in the Wails app.
  - [ ] Empty and populated lookup responses are mapped into GUI-safe state.
  - [ ] Existing dashboard/forms bindings still initialize normally.

  **QA Scenarios**:
  ```text
  Scenario: lookup binding returns populated GUI-safe state
    Tool: Bash (go test)
    Preconditions: Binding and response mapping implemented
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*Subscription.*Lookup|Test.*Subscription.*Binding'`
      2. Verify assertions for populated lookup rows, ordering, and mapped fields
    Expected Result: targeted GUI binding tests pass
    Failure Indicators: binding not registered, wrong response shape, or incorrect mapping
    Evidence: .sisyphus/evidence/task-T9-binding-populated.txt

  Scenario: lookup binding handles empty results
    Tool: Bash (go test)
    Preconditions: Empty-state binding tests exist
    Steps:
      1. Run `go test ./internal/adapters/gui -run 'Test.*Subscription.*Empty'`
      2. Confirm empty result sets map cleanly without nil/panic behavior
    Expected Result: targeted tests pass
    Failure Indicators: nil slices, unexpected errors, or missing empty-state handling
    Evidence: .sisyphus/evidence/task-T9-binding-empty.txt
  ```

  **Commit**: YES
  - Message: `feat(gui): expose subscription lookup binding`
  - Files: `internal/adapters/gui/*subscription*_binding.go`, `internal/adapters/gui/app.go`, `internal/adapters/gui/run.go`
  - Pre-commit: `go test ./internal/adapters/gui`

- [ ] T10. Add persistence and integration regression coverage for the end-to-end subscription flow

  **What to do**:
  - Add/extend repository and integration-style tests to prove the full flow works against temp SQLite.
  - Validate preset save, manual save, invalid input, populated lookup, empty lookup, and deterministic ordering in one regression pass.
  - Use this task to reconcile any small mismatches left between service, binding, and repository layers.

  **Must NOT do**:
  - Do not add unrelated cleanup or refactors that are not required to make the regression suite pass.
  - Do not silently widen lookup scope beyond list/read-only.

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: this is the integration convergence point across save/query/persistence layers.
  - **Skills**: `[]`
  - **Skills Evaluated but Omitted**:
    - `quick`: final regression convergence needs broader cross-task awareness.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 tail / integration finish
  - **Blocks**: Final Verification
  - **Blocked By**: T1, T2, T4, T6, T7, T8, T9

  **References**:
  - `internal/adapters/gui/forms_binding_test.go:93-195` - Existing integration-style save/persist assertions through the GUI binding.
  - `internal/adapters/gui/dashboard_binding_test.go:14-104` - Query binding response assertions for empty/populated behavior.
  - `internal/adapters/sqlite/usage_repository_test.go:13-80` - Temp SQLite round-trip test style to mimic for subscription persistence.
  - `internal/adapters/sqlite/repository_test.go:117-132` - Shared `mustBootstrapStore` helper for repository-level SQLite tests.
  - `Makefile:1-5` - Canonical project test command (`go test ./...`).

  **Acceptance Criteria**:
  - [ ] Temp SQLite coverage proves preset and manual subscriptions persist correctly.
  - [ ] Lookup coverage proves persisted subscriptions are queryable after save.
  - [ ] Invalid blank/date scenarios fail with expected validation behavior.
  - [ ] `go test ./...` passes cleanly.

  **QA Scenarios**:
  ```text
  Scenario: full regression suite passes
    Tool: Bash (go test)
    Preconditions: All implementation tasks complete
    Steps:
      1. Run `go test ./...`
      2. Confirm exit status 0
      3. Confirm no subscription-related tests are skipped unexpectedly
    Expected Result: full suite passes
    Failure Indicators: non-zero exit, flaky tests, or newly failing unrelated areas caused by the change
    Evidence: .sisyphus/evidence/task-T10-go-test-all.txt

  Scenario: end-to-end save then lookup round-trip works
    Tool: Bash (go test)
    Preconditions: Integration or binding/repository round-trip tests added
    Steps:
      1. Run `go test ./internal/adapters/gui ./internal/adapters/sqlite -run 'Test.*Subscription.*RoundTrip|Test.*Subscription.*Lookup'`
      2. Verify a saved preset/manual subscription is returned by the lookup flow
    Expected Result: targeted tests pass and show persisted subscription data is queryable
    Failure Indicators: save succeeds but lookup misses the record, or ordering/fields are incorrect
    Evidence: .sisyphus/evidence/task-T10-save-lookup-roundtrip.txt
  ```

  **Commit**: YES
  - Message: `test(subscription): add persistence and integration regressions`
  - Files: `internal/adapters/gui/forms_binding_test.go`, `internal/adapters/gui/*subscription*_test.go`, `internal/adapters/sqlite/*subscription*_test.go`
  - Pre-commit: `go test ./...`

---

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit** — `oracle`
  - Verify the named preset list, hidden-id requirement, date-only contract, and read-only lookup scope all match the plan.
  - Confirm evidence files exist for every task scenario.
  - Output: `Must Have [N/N] | Must NOT Have [N/N] | VERDICT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  - Run `go test ./...`.
  - Review changed files for dead code, placeholder logic, leaked internal IDs in GUI contracts, and accidental scope creep.
  - Output: `Tests [PASS/FAIL] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real QA Execution** — `unspecified-high`
  - Execute every task QA scenario with Go tests / targeted commands.
  - Validate preset save, manual save, invalid date, blank provider/plan, collision-safe id generation, empty lookup, and populated lookup.
  - Save evidence to `.sisyphus/evidence/final-qa/`.
  - Output: `Scenarios [N/N pass] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  - Compare final diff to this plan.
  - Ensure no extra preset catalog management, no full detail screen, and no unrelated architecture refactor landed.
  - Output: `Tasks [N/N compliant] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Commit 1**: `test(subscription): codify preset and lookup requirements`
- **Commit 2**: `feat(subscription): add static llm presets and generated identifiers`
- **Commit 3**: `feat(gui): switch subscription flow to hidden ids and date input`
- **Commit 4**: `feat(subscription): add read-only lookup query flow`
- **Commit 5**: `test(subscription): add persistence regression coverage`

---

## Success Criteria

### Verification Commands
```bash
go test ./...   # Expected: exit status 0
```

### Final Checklist
- [ ] Requested 9 presets are present
- [ ] Manual registration still works
- [ ] `planCode` / `subscriptionId` are generated and hidden
- [ ] `StartsAt` is date-oriented for GUI input/output
- [ ] Lookup/list works in empty and populated states
- [ ] No preset catalog CRUD or detail-screen scope creep was introduced
