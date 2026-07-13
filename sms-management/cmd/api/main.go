package main

import (
	"sms-management/internal/infrastructure/logger"
	"sms-management/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		logger.Log.Sugar().Fatal(err)
	}

	if err := application.Run(); err != nil {
		logger.Log.Sugar().Fatal(err)
	}
}
