package main

import (
	"log"

	"sms-monitoring/internal/app"
)

func main() {
	workerApp, err := app.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize monitoring app: %v", err)
	}
	defer workerApp.Shutdown()

	if err := workerApp.Run(); err != nil {
		log.Fatalf("Monitoring App stopped with error: %v", err)
	}
}
