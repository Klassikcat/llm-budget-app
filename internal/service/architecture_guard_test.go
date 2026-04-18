package service

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCorePackagesAvoidAdapterAndFrameworkImports(t *testing.T) {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate architecture guard test file")
	}

	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	targets := []string{
		filepath.Join(moduleRoot, "internal", "domain"),
		filepath.Join(moduleRoot, "internal", "ports"),
	}

	forbiddenPrefixes := []string{
		"llm-budget-tracker/internal/adapters",
		"github.com/wailsapp/",
		"github.com/charmbracelet/bubbletea",
		"github.com/charmbracelet/bubbles",
		"github.com/fsnotify/",
		"modernc.org/sqlite",
		"github.com/mattn/go-sqlite3",
		"github.com/glebarez/sqlite",
	}

	for _, target := range targets {
		entries, err := os.ReadDir(target)
		if err != nil {
			t.Fatalf("read target %s: %v", target, err)
		}
		if len(entries) == 0 {
			t.Fatalf("architecture guard target %s is unexpectedly empty", target)
		}

		if err := filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}

			fileSet := token.NewFileSet()
			parsed, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			for _, imported := range parsed.Imports {
				importPath := strings.Trim(imported.Path.Value, "\"")
				for _, prefix := range forbiddenPrefixes {
					if strings.HasPrefix(importPath, prefix) {
						t.Fatalf("forbidden import %q found in %s", importPath, path)
					}
				}
			}

			return nil
		}); err != nil {
			t.Fatalf("scan target %s: %v", target, err)
		}
	}
}
