package main

import (
	"log"

	"sms-agent-handler/internal/app"
)

func main() {
	workerApp, err := app.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize agent handler app: %v", err)
	}

	if err := workerApp.Run(); err != nil {
		log.Fatalf("Agent Handler App stopped with error: %v", err)
	}
}
