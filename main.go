package main

import (
	"log"

	appgui "llm-budget-tracker/internal/adapters/gui"
)

func main() {
	if err := appgui.Run(); err != nil {
		log.Fatal(err)
	}
}
