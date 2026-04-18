package app

import (
	"context"

	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type billingModeFallbackParser struct {
	base         ports.SessionParser
	fallbackMode domain.BillingMode
}

func newBillingModeFallbackParser(base ports.SessionParser, fallback config.BillingMode) ports.SessionParser {
	if base == nil {
		return nil
	}

	mode := domainBillingMode(fallback)
	if mode == domain.BillingModeUnknown {
		return base
	}

	return &billingModeFallbackParser{base: base, fallbackMode: mode}
}

func (p *billingModeFallbackParser) ParserName() string {
	return p.base.ParserName()
}

func (p *billingModeFallbackParser) Parse(ctx context.Context, input ports.ParseInput) (ports.ParseResult, error) {
	result, err := p.base.Parse(ctx, input)
	if err != nil {
		return ports.ParseResult{}, err
	}

	for index := range result.Events {
		if result.Events[index].BillingModeHint == domain.BillingModeUnknown {
			result.Events[index].BillingModeHint = p.fallbackMode
		}
	}

	return result, nil
}

func domainBillingMode(mode config.BillingMode) domain.BillingMode {
	switch mode {
	case config.BillingModeSubscription:
		return domain.BillingModeSubscription
	case config.BillingModeBYOK:
		return domain.BillingModeBYOK
	default:
		return domain.BillingModeUnknown
	}
}
