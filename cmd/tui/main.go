package main

import (
	"log"

	apptui "llm-budget-tracker/internal/adapters/tui"
)

func main() {
	if err := apptui.Run(); err != nil {
		log.Fatal(err)
	}
}
