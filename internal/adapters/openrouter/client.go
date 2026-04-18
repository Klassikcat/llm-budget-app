package openrouter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	DefaultAPIBaseURL = "https://openrouter.ai/api/v1"
	CacheSource       = "openrouter_cache"

	pricingScalePerMillion = 1_000_000.0
	totalEpsilon           = 0.000000001
	dateLayout             = "2006-01-02"
)

type WarningCode string

const (
	WarningCodeMissingAPIKey WarningCode = "missing_api_key"
	WarningCodeInvalidAPIKey WarningCode = "invalid_api_key"
	WarningCodeAccessDenied  WarningCode = "access_denied"
)

type WarningState struct {
	Code       WarningCode
	SecretID   config.SecretID
	StatusCode int
	Message    string
	Err        error
}

func (w *WarningState) Error() string {
	if w == nil {
		return "<nil>"
	}

	if w.Err == nil {
		return w.Message
	}

	return fmt.Sprintf("%s: %v", w.Message, w.Err)
}

func (w *WarningState) Unwrap() error {
	if w == nil {
		return nil
	}
	return w.Err
}

type Options struct {
	APIKey     string
	APIBaseURL string
	HTTPClient *http.Client
	Now        func() time.Time
}

type Client struct {
	apiKey     string
	apiBaseURL string
	httpClient *http.Client
	now        func() time.Time
	warning    *WarningState
}

type UsageImport struct {
	OccurredAt       time.Time
	EntryID          string
	SessionID        string
	ExternalID       string
	ProjectName      string
	ProviderName     string
	Model            string
	ModelPermaslug   string
	PromptTokens     int64
	CompletionTokens int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ToolInvocations  int64
	Price            ports.ModelPrice
	UsageUSD         float64
	BYOKUsageUSD     float64
}

type modelsResponse struct {
	Data []modelPayload `json:"data"`
}

type modelPayload struct {
	ID            string         `json:"id"`
	CanonicalSlug string         `json:"canonical_slug"`
	Pricing       pricingPayload `json:"pricing"`
}

type pricingPayload struct {
	Prompt          string `json:"prompt"`
	Completion      string `json:"completion"`
	InputCacheRead  string `json:"input_cache_read"`
	InputCacheWrite string `json:"input_cache_write"`
	WebSearch       string `json:"web_search"`
}

type activityResponse struct {
	Data []activityPayload `json:"data"`
}

type activityPayload struct {
	Date               string  `json:"date"`
	Model              string  `json:"model"`
	ModelPermaslug     string  `json:"model_permaslug"`
	EndpointID         string  `json:"endpoint_id"`
	ProviderName       string  `json:"provider_name"`
	Usage              float64 `json:"usage"`
	BYOKUsageInference float64 `json:"byok_usage_inference"`
	Requests           int64   `json:"requests"`
	PromptTokens       int64   `json:"prompt_tokens"`
	CompletionTokens   int64   `json:"completion_tokens"`
	ReasoningTokens    int64   `json:"reasoning_tokens"`
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(opts Options) *Client {
	apiBaseURL := strings.TrimRight(strings.TrimSpace(opts.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = DefaultAPIBaseURL
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}

	client := &Client{
		apiKey:     strings.TrimSpace(opts.APIKey),
		apiBaseURL: apiBaseURL,
		httpClient: httpClient,
		now:        now,
	}

	if client.apiKey == "" {
		client.warning = &WarningState{
			Code:     WarningCodeMissingAPIKey,
			SecretID: config.SecretOpenRouterAPIKey,
			Message:  "OpenRouter sync is disabled until provider.openrouter.api_key is configured in the secret store",
		}
	}

	return client
}

func (c *Client) Configured() bool {
	return c != nil && strings.TrimSpace(c.apiKey) != ""
}

func (c *Client) WarningState() *WarningState {
	if c == nil || c.warning == nil {
		return nil
	}

	clone := *c.warning
	return &clone
}

func (c *Client) FetchCatalog(ctx context.Context) (ports.CatalogSnapshot, error) {
	if warning := c.configurationWarning(); warning != nil {
		return ports.CatalogSnapshot{}, warning
	}

	var payload modelsResponse
	if err := c.getJSON(ctx, "/models", nil, &payload); err != nil {
		return ports.CatalogSnapshot{}, err
	}

	syncedAt := c.now().UTC()
	entries := make([]ports.ModelPrice, 0, len(payload.Data))
	for i, item := range payload.Data {
		entry, err := normalizeModelPayload(item)
		if err != nil {
			return ports.CatalogSnapshot{}, fmt.Errorf("normalize OpenRouter model %d: %w", i, err)
		}
		entries = append(entries, entry)
	}

	return ports.CatalogSnapshot{
		Source:   CacheSource,
		Version:  fmt.Sprintf("sync-%s", syncedAt.Format(time.RFC3339)),
		SyncedAt: syncedAt,
		Entries:  entries,
	}, nil
}

func (c *Client) FetchUsageEntries(ctx context.Context, options ports.OpenRouterActivityOptions) ([]domain.UsageEntry, error) {
	if warning := c.configurationWarning(); warning != nil {
		return nil, warning
	}

	query := url.Values{}
	if !options.Date.IsZero() {
		query.Set("date", options.Date.UTC().Format(dateLayout))
	}
	if value := strings.TrimSpace(options.APIKeyHash); value != "" {
		query.Set("api_key_hash", value)
	}
	if value := strings.TrimSpace(options.UserID); value != "" {
		query.Set("user_id", value)
	}

	var payload activityResponse
	if err := c.getJSON(ctx, "/activity", query, &payload); err != nil {
		return nil, err
	}

	entries := make([]domain.UsageEntry, 0, len(payload.Data))
	for i, item := range payload.Data {
		entry, err := normalizeActivityPayload(item)
		if err != nil {
			return nil, fmt.Errorf("normalize OpenRouter activity %d: %w", i, err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (c *Client) NormalizeUsageImport(input UsageImport) (domain.UsageEntry, error) {
	return normalizeUsageImport(input)
}

func (c *Client) configurationWarning() error {
	if c == nil {
		return &WarningState{
			Code:     WarningCodeMissingAPIKey,
			SecretID: config.SecretOpenRouterAPIKey,
			Message:  "OpenRouter client is not initialized",
		}
	}
	if warning := c.WarningState(); warning != nil {
		return warning
	}
	return nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values, target any) error {
	requestURL, err := url.JoinPath(c.apiBaseURL, strings.TrimPrefix(endpoint, "/"))
	if err != nil {
		return err
	}
	if len(query) > 0 {
		requestURL = requestURL + "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request OpenRouter %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read OpenRouter %s response: %w", endpoint, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeWarning(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode OpenRouter %s response: %w", endpoint, err)
	}

	return nil
}

func decodeWarning(statusCode int, body []byte) error {
	message := strings.TrimSpace(string(body))
	var payload errorResponse
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Error.Message) != "" {
		message = strings.TrimSpace(payload.Error.Message)
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return &WarningState{
			Code:       WarningCodeInvalidAPIKey,
			SecretID:   config.SecretOpenRouterAPIKey,
			StatusCode: statusCode,
			Message:    "OpenRouter sync failed because the configured API key was rejected",
			Err:        errors.New(message),
		}
	case http.StatusForbidden:
		return &WarningState{
			Code:       WarningCodeAccessDenied,
			SecretID:   config.SecretOpenRouterAPIKey,
			StatusCode: statusCode,
			Message:    "OpenRouter sync failed because the configured key does not have management API access",
			Err:        errors.New(message),
		}
	default:
		return fmt.Errorf("OpenRouter request failed with status %d: %s", statusCode, message)
	}
}

func normalizeModelPayload(item modelPayload) (ports.ModelPrice, error) {
	lookupKey := strings.TrimSpace(item.ID)
	if lookupKey == "" {
		return ports.ModelPrice{}, fmt.Errorf("model id is required")
	}

	modelID := strings.TrimSpace(item.CanonicalSlug)
	if modelID == "" {
		modelID = lookupKey
	}

	prompt, err := parsePricePerMillion(item.Pricing.Prompt)
	if err != nil {
		return ports.ModelPrice{}, fmt.Errorf("prompt price: %w", err)
	}
	completion, err := parsePricePerMillion(item.Pricing.Completion)
	if err != nil {
		return ports.ModelPrice{}, fmt.Errorf("completion price: %w", err)
	}
	cacheRead, err := parsePricePerMillion(item.Pricing.InputCacheRead)
	if err != nil {
		return ports.ModelPrice{}, fmt.Errorf("input cache read price: %w", err)
	}
	cacheWrite, err := parsePricePerMillion(item.Pricing.InputCacheWrite)
	if err != nil {
		return ports.ModelPrice{}, fmt.Errorf("input cache write price: %w", err)
	}
	webSearch, err := parseUnitPrice(item.Pricing.WebSearch)
	if err != nil {
		return ports.ModelPrice{}, fmt.Errorf("web search price: %w", err)
	}

	price := ports.ModelPrice{
		Provider:             domain.ProviderOpenRouter,
		ModelID:              modelID,
		LookupKey:            lookupKey,
		InputUSDPer1M:        prompt,
		OutputUSDPer1M:       completion,
		CacheReadUSDPer1M:    cacheRead,
		CacheWriteUSDPer1M:   cacheWrite,
		ToolUSDPerInvocation: webSearch,
	}

	if _, err := price.Calculate(domain.TokenUsage{}, 0); err != nil {
		return ports.ModelPrice{}, err
	}

	return price, nil
}

func normalizeActivityPayload(item activityPayload) (domain.UsageEntry, error) {
	occurredAt, err := time.Parse(dateLayout, strings.TrimSpace(item.Date))
	if err != nil {
		return domain.UsageEntry{}, fmt.Errorf("activity date must be YYYY-MM-DD: %w", err)
	}

	priceRef, err := newPricingRef(strings.TrimSpace(item.ModelPermaslug), strings.TrimSpace(item.Model))
	if err != nil {
		return domain.UsageEntry{}, err
	}

	tokens, err := domain.NewTokenUsage(item.PromptTokens, item.CompletionTokens, 0, 0)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	breakdown, err := newTotalCostBreakdown(item.Usage + item.BYOKUsageInference)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	entryID := deterministicID("activity", item.Date, item.EndpointID, item.ModelPermaslug, item.Model)
	sessionID := strings.Join([]string{
		"openrouter",
		"activity",
		strings.TrimSpace(item.Date),
		sanitizeIDComponent(item.Model),
		sanitizeIDComponent(item.EndpointID),
	}, ":")

	return domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		Source:        domain.UsageSourceOpenRouter,
		Provider:      domain.ProviderOpenRouter,
		BillingMode:   domain.BillingModeOpenRouter,
		OccurredAt:    occurredAt.UTC(),
		SessionID:     sessionID,
		ExternalID:    strings.TrimSpace(item.EndpointID),
		ProjectName:   "",
		AgentName:     "",
		PricingRef:    &priceRef,
		Tokens:        tokens,
		CostBreakdown: breakdown,
	})
}

func normalizeUsageImport(input UsageImport) (domain.UsageEntry, error) {
	priceRef, err := newPricingRef(strings.TrimSpace(input.ModelPermaslug), strings.TrimSpace(input.Model))
	if err != nil {
		return domain.UsageEntry{}, err
	}

	tokens, err := domain.NewTokenUsage(input.PromptTokens, input.CompletionTokens, input.CacheReadTokens, input.CacheWriteTokens)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	breakdown, err := breakdownFromImport(input, tokens)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	entryID := strings.TrimSpace(input.EntryID)
	if entryID == "" {
		entryID = deterministicID(
			"usage",
			input.OccurredAt.UTC().Format(time.RFC3339),
			input.ExternalID,
			input.ModelPermaslug,
			input.Model,
		)
	}

	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		sessionID = strings.Join([]string{
			"openrouter",
			"usage",
			input.OccurredAt.UTC().Format(dateLayout),
			sanitizeIDComponent(input.Model),
		}, ":")
	}

	return domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		Source:        domain.UsageSourceOpenRouter,
		Provider:      domain.ProviderOpenRouter,
		BillingMode:   domain.BillingModeOpenRouter,
		OccurredAt:    input.OccurredAt.UTC(),
		SessionID:     sessionID,
		ExternalID:    strings.TrimSpace(input.ExternalID),
		ProjectName:   strings.TrimSpace(input.ProjectName),
		AgentName:     "",
		PricingRef:    &priceRef,
		Tokens:        tokens,
		CostBreakdown: breakdown,
	})
}

func breakdownFromImport(input UsageImport, tokens domain.TokenUsage) (domain.CostBreakdown, error) {
	totalUSD := input.UsageUSD + input.BYOKUsageUSD
	if totalUSD < 0 {
		return domain.CostBreakdown{}, fmt.Errorf("total usage cost must be non-negative")
	}

	price := input.Price
	if price.Provider == "" {
		price.Provider = domain.ProviderOpenRouter
	}
	if strings.TrimSpace(price.ModelID) == "" {
		price.ModelID = firstNonEmpty(strings.TrimSpace(input.ModelPermaslug), strings.TrimSpace(input.Model))
	}
	if strings.TrimSpace(price.LookupKey) == "" {
		price.LookupKey = firstNonEmpty(strings.TrimSpace(input.Model), strings.TrimSpace(input.ModelPermaslug), price.ModelID)
	}

	if hasExplicitPricing(price) {
		calculated, err := price.Calculate(tokens, input.ToolInvocations)
		if err != nil {
			return domain.CostBreakdown{}, err
		}

		if totalUSD <= 0 {
			return calculated, nil
		}

		if totalUSD+totalEpsilon < calculated.TotalUSD {
			return newTotalCostBreakdown(totalUSD)
		}

		flatAdjustment := totalUSD - calculated.TotalUSD
		if math.Abs(flatAdjustment) <= totalEpsilon {
			flatAdjustment = 0
		}

		return domain.NewCostBreakdown(
			calculated.InputUSD,
			calculated.OutputUSD,
			calculated.CacheReadUSD,
			calculated.CacheWriteUSD,
			calculated.ToolUSD,
			flatAdjustment,
		)
	}

	return newTotalCostBreakdown(totalUSD)
}

func newPricingRef(modelID, lookupKey string) (domain.ModelPricingRef, error) {
	resolvedModelID := firstNonEmpty(modelID, lookupKey)
	resolvedLookupKey := firstNonEmpty(lookupKey, modelID)
	return domain.NewModelPricingRef(domain.ProviderOpenRouter, resolvedModelID, resolvedLookupKey)
}

func newTotalCostBreakdown(totalUSD float64) (domain.CostBreakdown, error) {
	return domain.NewCostBreakdown(0, 0, 0, 0, 0, totalUSD)
}

func parsePricePerMillion(raw string) (float64, error) {
	unitValue, err := parseUnitPrice(raw)
	if err != nil {
		return 0, err
	}
	return unitValue * pricingScalePerMillion, nil
}

func parseUnitPrice(raw string) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}

	value, err := strconvParseFloat(trimmed)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", trimmed, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("price must be non-negative")
	}
	return value, nil
}

func hasExplicitPricing(price ports.ModelPrice) bool {
	return price.InputUSDPer1M > 0 || price.OutputUSDPer1M > 0 || price.CacheReadUSDPer1M > 0 || price.CacheWriteUSDPer1M > 0 || price.ToolUSDPerInvocation > 0
}

func deterministicID(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "|")))
	return "openrouter-" + hex.EncodeToString(hash[:])
}

func sanitizeIDComponent(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	trimmed = strings.ReplaceAll(trimmed, "/", "-")
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	trimmed = strings.ReplaceAll(trimmed, ":", "-")
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func strconvParseFloat(raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid floating-point value")
	}
	return value, nil
}

var _ ports.CatalogSyncSource = (*Client)(nil)
var _ ports.OpenRouterActivitySource = (*Client)(nil)
