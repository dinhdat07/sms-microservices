package app

import (
	"context"
	"fmt"

	"sms-notification/internal/config"
	"sms-notification/internal/consumer"
	"sms-notification/internal/infrastructure/logger"
	"sms-notification/internal/infrastructure/messagebroker"
	"sms-notification/internal/infrastructure/notifier"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type App struct {
	cfg                  config.Config
	redisClient          redis.UniversalClient
	notificationConsumer consumer.NotificationConsumer
}

func NewApp() (*App, error) {
	cfg := config.LoadConfig()
	logger.InitLogger(cfg.LogLevel, cfg.LogFormat, "logs/notification.log")

	var rdb redis.UniversalClient
	var subscriber messagebroker.Subscriber

	if cfg.RedisEnabled {
		rdb = redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:    []string{cfg.RedisAddr},
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		// Verify Redis connection
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
		logger.Log.Info("Connected to Redis", zap.String("addr", cfg.RedisAddr))

		subscriber = messagebroker.NewRedisSubscriber(rdb)
	} else {
		return nil, fmt.Errorf("redis is required for Notification Service to consume events")
	}

	// Setup SMTP Notifier
	smtpCfg := notifier.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		UseAuth:  cfg.SMTPUseAuth,
		UseTLS:   cfg.SMTPUseTLS,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		FromName: cfg.SMTPFromName,
	}

	// Ping SMTP
	if err := notifier.Ping(context.Background(), smtpCfg.Host, smtpCfg.Port); err != nil {
		logger.Log.Warn("Failed to ping SMTP server", zap.Error(err))
	} else {
		logger.Log.Info("SMTP Server connected successfully")
	}

	smtpMailer := notifier.NewMailer(smtpCfg)

	// Setup Consumer
	notificationConsumer := consumer.NewNotificationConsumer(subscriber, cfg, smtpMailer)

	return &App{
		cfg:                  cfg,
		redisClient:          rdb,
		notificationConsumer: notificationConsumer,
	}, nil
}
