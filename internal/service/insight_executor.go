package service

import (
	"context"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type InsightExecutionResult struct {
	Insights []domain.Insight
}

type InsightExecutorService struct {
	detectors   []ports.InsightDetector
	sessionRepo ports.SessionRepository
	usageRepo   ports.UsageEntryRepository
	insightRepo ports.InsightRepository
}

func NewInsightExecutorService(detectors []ports.InsightDetector, sessionRepo ports.SessionRepository, usageRepo ports.UsageEntryRepository, insightRepo ports.InsightRepository) *InsightExecutorService {
	cloned := append([]ports.InsightDetector(nil), detectors...)
	return &InsightExecutorService{
		detectors:   cloned,
		sessionRepo: sessionRepo,
		usageRepo:   usageRepo,
		insightRepo: insightRepo,
	}
}

func (s *InsightExecutorService) Execute(ctx context.Context, period domain.MonthlyPeriod) (InsightExecutionResult, error) {
	if s == nil || s.sessionRepo == nil {
		return InsightExecutionResult{}, errSessionRepositoryRequired
	}

	if s.usageRepo == nil {
		return InsightExecutionResult{}, errUsageEntryRepositoryRequired
	}

	if s.insightRepo == nil {
		return InsightExecutionResult{}, errInsightRepositoryRequired
	}

	sessions, err := s.sessionRepo.ListSessions(ctx, ports.SessionFilter{Period: &period})
	if err != nil {
		return InsightExecutionResult{}, err
	}

	usageEntries, err := s.usageRepo.ListUsageEntries(ctx, ports.UsageFilter{Period: &period})
	if err != nil {
		return InsightExecutionResult{}, err
	}

	insights := make([]domain.Insight, 0)
	for _, detector := range s.detectors {
		if detector == nil {
			continue
		}

		detected, err := detector.Detect(ctx, period, sessions, usageEntries)
		if err != nil {
			return InsightExecutionResult{}, err
		}

		insights = append(insights, detected...)
	}

	if len(insights) > 0 {
		if err := s.insightRepo.UpsertInsights(ctx, insights); err != nil {
			return InsightExecutionResult{}, err
		}
	}

	return InsightExecutionResult{Insights: insights}, nil
}
