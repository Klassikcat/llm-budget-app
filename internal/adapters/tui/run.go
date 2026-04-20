package tui

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/app"
	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

type RunOptions struct {
	NewWasteSummaryService func(store *sqlite.Store) *service.WasteSummaryService
}

func Run(opts RunOptions) error {
	return run(context.Background(), os.Args[1:], os.Stderr, opts)
}

func run(ctx context.Context, args []string, stderr io.Writer, opts RunOptions) error {
	flagSet := flag.NewFlagSet("llm-budget-tracker-tui", flag.ContinueOnError)
	flagSet.SetOutput(stderr)

	dbPath := flagSet.String("db", "", "path to SQLite database")
	bootstrapOnly := flagSet.Bool("bootstrap-only", false, "initialize config and SQLite database, then exit")
	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	resolvedDBPath, err := resolveDatabasePath(*dbPath)
	if err != nil {
		return err
	}

	graph, err := app.Start(ctx, app.Options{DatabasePath: resolvedDBPath, BootstrapOnly: *bootstrapOnly})
	if err != nil {
		return err
	}
	defer graph.Close()

	if *bootstrapOnly {
		return nil
	}

	period, err := domain.NewMonthlyPeriod(time.Now().UTC())
	if err != nil {
		return err
	}

	var wasteSummary wasteSummaryLoader
	if opts.NewWasteSummaryService != nil {
		wasteSummary = opts.NewWasteSummaryService(graph.Store)
	}

	program := tea.NewProgram(newModel(modelDependencies{
		loader:        graph.DashboardQueryService,
		graphs:        service.NewGraphQueryService(graph.Store),
		wasteSummary:  wasteSummary,
		manualEntries: graph.ManualEntryService,
		subscriptions: graph.SubscriptionService,
		insights:      graph.Store,
		alerts:        graph.Store,
	}, period))
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}

	return nil
}

func resolveDatabasePath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	paths, err := config.ResolvePaths(config.PathResolverOptions{})
	if err != nil {
		return "", err
	}

	return paths.DatabaseFile, nil
}
