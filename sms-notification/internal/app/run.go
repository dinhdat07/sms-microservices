package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"sms-notification/internal/infrastructure/logger"
)

func (a *App) Run() error {

	logger.Log.Info("Starting Notification Service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.notificationConsumer.Start(ctx)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()
	a.Shutdown(context.Background())

	return nil
}

func (a *App) Shutdown(ctx context.Context) {
	logger.Log.Info("Shutting down Notification Service...")

	if a.redisClient != nil {
		a.redisClient.Close()
	}

	logger.Log.Info("Notification Service stopped")
}
