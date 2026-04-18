package catalog

import (
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type catalogDocument struct {
	SchemaVersion  string
	CatalogVersion string
	Source         string
	Provider       domain.ProviderName
	Entries        []catalogEntry
	SyncedAt       time.Time
}

type catalogEntry struct {
	ports.ModelPrice
	CachedAt  time.Time
	ExpiresAt time.Time
}

func (d catalogDocument) snapshot() ports.CatalogSnapshot {
	entries := make([]ports.ModelPrice, 0, len(d.Entries))
	for _, entry := range d.Entries {
		entries = append(entries, entry.ModelPrice)
	}

	return ports.CatalogSnapshot{
		Source:   d.Source,
		Version:  d.CatalogVersion,
		SyncedAt: d.SyncedAt,
		Entries:  entries,
	}
}

func documentFromSnapshot(snapshot ports.CatalogSnapshot) (catalogDocument, error) {
	if strings.TrimSpace(snapshot.Source) == "" {
		return catalogDocument{}, fmt.Errorf("catalog snapshot source is required")
	}
	if strings.TrimSpace(snapshot.Version) == "" {
		return catalogDocument{}, fmt.Errorf("catalog snapshot version is required")
	}

	entries := make([]catalogEntry, 0, len(snapshot.Entries))
	provider := domain.ProviderName("")
	for i, price := range snapshot.Entries {
		entry, err := newCatalogEntry(rawCatalogEntry{
			Provider:             price.Provider.String(),
			ModelID:              price.ModelID,
			LookupKey:            price.LookupKey,
			InputUSDPer1M:        floatPointer(price.InputUSDPer1M),
			OutputUSDPer1M:       floatPointer(price.OutputUSDPer1M),
			CacheReadUSDPer1M:    floatPointer(price.CacheReadUSDPer1M),
			CacheWriteUSDPer1M:   floatPointer(price.CacheWriteUSDPer1M),
			ToolUSDPerInvocation: floatPointer(price.ToolUSDPerInvocation),
		}, domain.ProviderName(""))
		if err != nil {
			return catalogDocument{}, fmt.Errorf("catalog snapshot entry %d: %w", i, err)
		}
		entries = append(entries, entry)
		if provider == "" {
			provider = entry.Provider
		}
	}

	syncedAt := snapshot.SyncedAt.UTC()
	if snapshot.SyncedAt.IsZero() {
		syncedAt = time.Time{}
	}

	return catalogDocument{
		SchemaVersion:  priceCatalogSchemaV1,
		CatalogVersion: strings.TrimSpace(snapshot.Version),
		Source:         strings.TrimSpace(snapshot.Source),
		Provider:       provider,
		Entries:        entries,
		SyncedAt:       syncedAt,
	}, nil
}

func floatPointer(value float64) *float64 {
	return &value
}
