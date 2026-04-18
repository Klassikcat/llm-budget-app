package catalog

import "embed"

//go:embed data/*.json
var embeddedCatalogs embed.FS

var embeddedCatalogPaths = []string{
	"data/anthropic.json",
	"data/openai.json",
	"data/gemini.json",
	"data/openrouter-cache.json",
}
