package main

import (
	"log"

	"llm-budget-tracker/internal/adapters/sqlite"
	apptui "llm-budget-tracker/internal/adapters/tui"
	"llm-budget-tracker/internal/service"
)

func main() {
	if err := apptui.Run(apptui.RunOptions{
		NewWasteSummaryService: func(store *sqlite.Store) *service.WasteSummaryService {
			return service.NewWasteSummaryService(store, store)
		},
	}); err != nil {
		log.Fatal(err)
	}
}
