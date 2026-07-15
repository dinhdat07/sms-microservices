package app

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"sms-management/internal/config"
	"sms-management/internal/handler/grpcserver"
	resthandler "sms-management/internal/handler/rest"
	"sms-management/internal/infrastructure/database"
	"sms-management/internal/infrastructure/logger"
	"sms-management/internal/infrastructure/messagebroker"
	"sms-management/internal/repository/impl"
	"sms-management/internal/service"
	"sms-management/internal/worker"
)

type App struct {
	Config           *config.Config
	DB               *gorm.DB
	RedisClient      redis.UniversalClient
	ServerHandler    *grpcserver.ServerManagementServer
	RESTImportExport *resthandler.ImportExportHandler
	OutboxRelay      *worker.OutboxRelay
	StatusConsumer   *worker.StatusConsumer
}

func New() (*App, error) {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger.InitLogger(cfg.Logger, cfg.Env)

	// 2. Init DB
	db, err := database.GetInstance(cfg.DBUrl)
	if err != nil {
		logger.Log.Sugar().Errorf("Failed to connect to database: %v", err)
		return nil, err
	}

	// 3. AutoMigrate schemas
	if err := database.AutoMigrate(db); err != nil {
		return nil, err
	}

	// 4. Init Redis
	redisCfg, err := config.LoadRedisConfig()
	if err != nil {
		logger.Log.Sugar().Errorf("Failed to load redis config: %v", err)
	}
	redisClient := database.NewRedisClient(redisCfg)
	if redisClient != nil {
		if err := database.PingRedis(context.Background(), redisClient); err != nil {
			logger.Log.Sugar().Errorf("Redis ping failed: %v", err)
		}
	}

	// 4. Init Management Services
	serverRepo := impl.NewGormServerRepository(db)
	outboxRepo := impl.NewGormOutboxRepository(db)
	serverSvc := service.NewServerService(serverRepo, outboxRepo)

	serverHandler := grpcserver.NewServerManagementServer(serverSvc)
	restImportExport := resthandler.NewImportExportHandler(serverSvc)

	// Initialize Publisher
	publisher := messagebroker.NewRedisPublisher(redisClient)

	// Initialize Outbox Worker
	outboxRelay := worker.NewOutboxRelay(outboxRepo, publisher, cfg.Outbox)

	// Initialize Status Consumer (listens for status changes from sms-monitoring)
	subscriber := messagebroker.NewRedisSubscriber(redisClient)
	statusConsumer := worker.NewStatusConsumer(subscriber, serverRepo)

	return &App{
		Config:           cfg,
		DB:               db,
		RedisClient:      redisClient,
		ServerHandler:    serverHandler,
		RESTImportExport: restImportExport,
		OutboxRelay:      outboxRelay,
		StatusConsumer:   statusConsumer,
	}, nil
}
