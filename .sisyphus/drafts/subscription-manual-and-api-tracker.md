# Draft: Manual Subscription Input + API Tracker Maximization

## Requirements (confirmed)
- Subscription field input should become manual (currently unclear mechanism — needs clarification)
- Maximize features for OpenRouter / Bedrock / LLM first-party API tracking

## Research Findings

### Subscription System (current state)
- **Domain model**: `Subscription` has Provider (enum), PlanCode, PlanName, RenewalDay, StartsAt, EndsAt, FeeUSD, IsActive
- **Provider is an enum**: Anthropic, OpenAI, Gemini, OpenRouter — this likely constrains input to predefined choices
- **GUI binding**: `FormsBinding.SaveSubscription` receives `SubscriptionFormInput` and converts to domain struct
- **Repository**: SQLite with upsert pattern
- **Key files**: `internal/domain/subscription.go`, `internal/service/subscriptions.go`, `internal/adapters/gui/forms_binding.go`, `internal/adapters/gui/forms_types.go`

### API Tracking (current state)
- **OpenRouter**: Full API polling via `/activity` endpoint (direct API adapter)
- **Other providers (OpenAI, Anthropic, Gemini)**: Cost calculation via pricing catalog, but NO direct API polling
- **Log/File parsing**: Parsers exist for claude, gemini, codex, opencode local logs
- **Billing modes**: Subscription vs BYOK (Bring Your Own Key)
- **Cost calculation**: Centralized `CostCalculatorService` using `PriceCatalog`
- **Bedrock**: NOT supported at all currently — no adapter, no catalog data
- **Manual API entry**: Exists (`manual_api_entry.go`) — for manually adding usage entries

### Frontend/Dashboard (current state)
- **Wails v2** desktop app, pre-built SPA in `frontend/dist/`
- **Dashboard**: Monthly view with totals (variable + subscription), provider summaries, budgets, recent sessions
- **Settings**: Provider enable/disable, billing defaults, subscription defaults, budget limits, notification preferences
- **Alerts/Forecasting/Insights**: All implemented and persisted in SQLite

## Open Questions
- What exactly does "subscription field input should be manual now" mean? (see question below)
- What specific "maximize features" means for API tracking (see question below)
- Test strategy?

## Scope Boundaries
- INCLUDE: TBD
- EXCLUDE: TBD

## Technical Decisions
- TBD
