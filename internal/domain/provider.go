package domain

import (
	"regexp"
	"strings"
)

var providerNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type ProviderName string

const (
	ProviderAnthropic  ProviderName = "anthropic"
	ProviderOpenAI     ProviderName = "openai"
	ProviderGemini     ProviderName = "gemini"
	ProviderOpenRouter ProviderName = "openrouter"
	ProviderClaude     ProviderName = "claude"
	ProviderCodex      ProviderName = "codex"
	ProviderOpenCode   ProviderName = "opencode"
)

type Provider struct {
	Name ProviderName
}

func NewProvider(raw string) (Provider, error) {
	name, err := NewProviderName(raw)
	if err != nil {
		return Provider{}, err
	}

	return Provider{Name: name}, nil
}

func NewProviderName(raw string) (ProviderName, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", requiredError("provider")
	}

	if !providerNamePattern.MatchString(normalized) {
		return "", &ValidationError{
			Code:    ValidationCodeInvalidProviderName,
			Field:   "provider",
			Message: "provider names must be lowercase slugs like 'openai' or 'openrouter'",
		}
	}

	return ProviderName(normalized), nil
}

func (p ProviderName) String() string {
	return string(p)
}
