package main

import (
	"log"

	"sms-reporting/internal/app"
)

func main() {
	reportingApp, err := app.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize reporting app: %v", err)
	}

	if err := reportingApp.Run(); err != nil {
		log.Fatalf("Reporting App stopped with error: %v", err)
	}
}
