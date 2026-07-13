package app

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"sms-management/internal/modules/server_management/handler/grpcserver"
	resthandler "sms-management/internal/modules/server_management/handler/rest"
	"sms-management/internal/modules/server_management/repository/impl"
	"sms-management/internal/modules/server_management/service"
	"sms-management/internal/shared/config"
	"sms-management/internal/shared/database"
	"sms-management/internal/shared/logger"
	infraRedis "sms-management/internal/infrastructure/redis"
)

type App struct {
	Config           *config.Config
	DB               *gorm.DB
	RedisClient      redis.UniversalClient
	ServerHandler    *grpcserver.ServerManagementServer
	RESTImportExport *resthandler.ImportExportHandler
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

	// 3. Init Redis
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
	serverCache := infraRedis.NewServerCache(redisClient)
	serverSvc := service.NewServerService(serverRepo, serverCache)
	
	serverHandler := grpcserver.NewServerManagementServer(serverSvc)
	restImportExport := resthandler.NewImportExportHandler(serverSvc)

	return &App{
		Config:           cfg,
		DB:               db,
		RedisClient:      redisClient,
		ServerHandler:    serverHandler,
		RESTImportExport: restImportExport,
	}, nil
}
