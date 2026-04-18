package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

const dashboardMonthLayout = "2006-01"

type dashboardQuerier interface {
	QueryDashboard(ctx context.Context, query service.DashboardQuery) (service.DashboardSnapshot, error)
}

type DashboardBinding struct {
	queryService dashboardQuerier
	ctx          context.Context
	clock        func() time.Time
}

func NewDashboardBinding(queryService dashboardQuerier) *DashboardBinding {
	return &DashboardBinding{
		queryService: queryService,
		clock:        func() time.Time { return time.Now().UTC() },
	}
}

func (b *DashboardBinding) startup(ctx context.Context) {
	if b == nil {
		return
	}
	b.ctx = ctx
}

func (b *DashboardBinding) LoadDashboard(month string) (DashboardResponse, error) {
	if b == nil || b.queryService == nil {
		return DashboardResponse{}, fmt.Errorf("dashboard query service is not initialized")
	}

	period, err := b.resolvePeriod(month)
	if err != nil {
		return DashboardResponse{}, err
	}

	ctx := context.Background()
	if b.ctx != nil {
		ctx = b.ctx
	}

	snapshot, err := b.queryService.QueryDashboard(ctx, service.DashboardQuery{Period: period})
	if err != nil {
		return DashboardResponse{}, err
	}

	return toDashboardResponse(snapshot), nil
}

func (b *DashboardBinding) resolvePeriod(month string) (domain.MonthlyPeriod, error) {
	trimmed := strings.TrimSpace(month)
	if trimmed == "" {
		anchor := time.Now().UTC()
		if b.clock != nil {
			anchor = b.clock().UTC()
		}
		return domain.NewMonthlyPeriod(anchor)
	}

	parsed, err := time.Parse(dashboardMonthLayout, trimmed)
	if err != nil {
		return domain.MonthlyPeriod{}, fmt.Errorf("parse dashboard month %q: %w", trimmed, err)
	}

	return domain.NewMonthlyPeriod(parsed.UTC())
}
