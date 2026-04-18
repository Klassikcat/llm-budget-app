package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	priceCatalogSchemaV1  = "llm-budget-tracker.price_catalog.v1"
	openRouterCacheSource = "openrouter_cache"
)

var errModelPriceNotFound = errors.New("catalog model price not found")

type Warning struct {
	Path    string
	Message string
	Err     error
}

func (w Warning) Error() string {
	base := strings.TrimSpace(w.Message)
	if base == "" {
		base = "catalog warning"
	}
	if strings.TrimSpace(w.Path) != "" {
		base = fmt.Sprintf("%s (%s)", base, w.Path)
	}
	if w.Err == nil {
		return base
	}
	return fmt.Sprintf("%s: %v", base, w.Err)
}

func (w Warning) Unwrap() error {
	return w.Err
}

type Options struct {
	OverridePath string
}

type Catalog struct {
	mu sync.RWMutex

	overrides map[lookupKey]catalogEntry
	embedded  map[lookupKey]catalogEntry
	cache     map[lookupKey]catalogEntry

	cacheSnapshot ports.CatalogSnapshot
	warnings      []Warning
}

type lookupKey struct {
	provider domain.ProviderName
	lookup   string
}

func New(opts Options) (*Catalog, error) {
	embeddedDocs, err := loadEmbeddedDocuments()
	if err != nil {
		return nil, err
	}

	resolved := &Catalog{
		overrides: make(map[lookupKey]catalogEntry),
		embedded:  make(map[lookupKey]catalogEntry),
		cache:     make(map[lookupKey]catalogEntry),
	}

	for _, doc := range embeddedDocs {
		index := resolved.embedded
		if doc.Source == openRouterCacheSource {
			index = resolved.cache
			resolved.cacheSnapshot = doc.snapshot()
		}

		for _, entry := range doc.Entries {
			index[newLookupKey(entry.Provider, entry.LookupKey)] = entry
			index[newLookupKey(entry.Provider, entry.ModelID)] = entry
		}
	}

	if strings.TrimSpace(opts.OverridePath) == "" {
		return resolved, nil
	}

	data, err := os.ReadFile(opts.OverridePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return resolved, nil
		}
		return nil, fmt.Errorf("read override catalog: %w", err)
	}

	doc, err := parseCatalogDocument(data, opts.OverridePath)
	if err != nil {
		resolved.warnings = append(resolved.warnings, Warning{
			Path:    opts.OverridePath,
			Message: "ignoring malformed override catalog",
			Err:     err,
		})
		return resolved, nil
	}

	for _, entry := range doc.Entries {
		resolved.overrides[newLookupKey(entry.Provider, entry.LookupKey)] = entry
		resolved.overrides[newLookupKey(entry.Provider, entry.ModelID)] = entry
	}

	return resolved, nil
}

func (c *Catalog) LookupModelPrice(_ context.Context, ref domain.ModelPricingRef, _ time.Time) (ports.ModelPrice, error) {
	if c == nil {
		return ports.ModelPrice{}, errModelPriceNotFound
	}

	candidates := []lookupKey{
		newLookupKey(ref.Provider, ref.PricingLookupKey),
		newLookupKey(ref.Provider, ref.ModelID),
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, candidate := range candidates {
		if entry, ok := c.overrides[candidate]; ok {
			return entry.ModelPrice, nil
		}
	}

	for _, candidate := range candidates {
		if entry, ok := c.embedded[candidate]; ok {
			return entry.ModelPrice, nil
		}
	}

	if ref.Provider == domain.ProviderOpenRouter {
		for _, candidate := range candidates {
			if entry, ok := c.cache[candidate]; ok {
				return entry.ModelPrice, nil
			}
		}
	}

	return ports.ModelPrice{}, fmt.Errorf("%w: provider=%s lookup=%s", errModelPriceNotFound, ref.Provider, ref.PricingLookupKey)
}

func (c *Catalog) ListProviderPrices(_ context.Context, provider domain.ProviderName) ([]ports.ModelPrice, error) {
	if c == nil {
		return nil, nil
	}

	seen := make(map[string]ports.ModelPrice)

	c.mu.RLock()
	defer c.mu.RUnlock()

	collect := func(index map[lookupKey]catalogEntry) {
		for _, entry := range index {
			if entry.Provider != provider {
				continue
			}
			if _, ok := seen[entry.ModelID]; ok {
				continue
			}
			seen[entry.ModelID] = entry.ModelPrice
		}
	}

	collect(c.overrides)
	collect(c.embedded)
	collect(c.cache)

	prices := make([]ports.ModelPrice, 0, len(seen))
	for _, price := range seen {
		prices = append(prices, price)
	}

	sort.Slice(prices, func(i, j int) bool {
		if prices[i].ModelID == prices[j].ModelID {
			return prices[i].LookupKey < prices[j].LookupKey
		}
		return prices[i].ModelID < prices[j].ModelID
	})

	return prices, nil
}

func (c *Catalog) ReplaceCatalog(_ context.Context, snapshot ports.CatalogSnapshot) error {
	if c == nil {
		return errors.New("catalog is nil")
	}

	doc, err := documentFromSnapshot(snapshot)
	if err != nil {
		return err
	}

	cacheIndex := make(map[lookupKey]catalogEntry, len(doc.Entries)*2)
	for _, entry := range doc.Entries {
		cacheIndex[newLookupKey(entry.Provider, entry.LookupKey)] = entry
		cacheIndex[newLookupKey(entry.Provider, entry.ModelID)] = entry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = cacheIndex
	c.cacheSnapshot = doc.snapshot()

	return nil
}

func (c *Catalog) Warnings() []Warning {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]Warning, len(c.warnings))
	copy(out, c.warnings)
	return out
}

func (c *Catalog) CacheSnapshot() ports.CatalogSnapshot {
	if c == nil {
		return ports.CatalogSnapshot{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]ports.ModelPrice, len(c.cacheSnapshot.Entries))
	copy(entries, c.cacheSnapshot.Entries)

	return ports.CatalogSnapshot{
		Source:   c.cacheSnapshot.Source,
		Version:  c.cacheSnapshot.Version,
		SyncedAt: c.cacheSnapshot.SyncedAt,
		Entries:  entries,
	}
}

func loadEmbeddedDocuments() ([]catalogDocument, error) {
	documents := make([]catalogDocument, 0, len(embeddedCatalogPaths))
	for _, path := range embeddedCatalogPaths {
		data, err := fs.ReadFile(embeddedCatalogs, path)
		if err != nil {
			return nil, fmt.Errorf("read embedded catalog %s: %w", path, err)
		}

		doc, err := parseCatalogDocument(data, path)
		if err != nil {
			return nil, fmt.Errorf("load embedded catalog %s: %w", path, err)
		}

		documents = append(documents, doc)
	}

	sort.Slice(documents, func(i, j int) bool {
		return documents[i].Provider < documents[j].Provider
	})

	return documents, nil
}

func newLookupKey(provider domain.ProviderName, raw string) lookupKey {
	return lookupKey{
		provider: provider,
		lookup:   strings.ToLower(strings.TrimSpace(raw)),
	}
}
