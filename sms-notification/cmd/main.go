package main

import (
	"log"

	"sms-notification/internal/app"
)

func main() {
	notificationApp, err := app.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize notification app: %v", err)
	}

	if err := notificationApp.Run(); err != nil {
		log.Fatalf("Notification App stopped with error: %v", err)
	}
}
