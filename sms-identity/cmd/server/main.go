package main

import (
	"sms-identity/internal/shared/logger"
	"sms-identity/internal/app"
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
