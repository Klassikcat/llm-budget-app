package service

import (
	"context"

	"llm-budget-tracker/internal/ports"
)

type CatalogSyncService struct {
	source  ports.CatalogSyncSource
	catalog ports.PriceCatalog
}

func NewCatalogSyncService(source ports.CatalogSyncSource, catalog ports.PriceCatalog) *CatalogSyncService {
	return &CatalogSyncService{source: source, catalog: catalog}
}

func (s *CatalogSyncService) Sync(ctx context.Context) (ports.CatalogSnapshot, error) {
	if s == nil || s.source == nil {
		return ports.CatalogSnapshot{}, errCatalogSyncSourceRequired
	}

	if s.catalog == nil {
		return ports.CatalogSnapshot{}, errPriceCatalogRequired
	}

	snapshot, err := s.source.FetchCatalog(ctx)
	if err != nil {
		return ports.CatalogSnapshot{}, err
	}

	if err := s.catalog.ReplaceCatalog(ctx, snapshot); err != nil {
		return ports.CatalogSnapshot{}, err
	}

	return snapshot, nil
}
