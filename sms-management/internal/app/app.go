package app

import (
	"context"
	"net/http"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	servermanagementv1 "sms-management/gen/go/server_management/v1"
	"sms-management/internal/config"
	grpchandler "sms-management/internal/handler/grpc"
	resthandler "sms-management/internal/handler/rest"
	"sms-management/internal/infrastructure/database"
	"sms-management/internal/infrastructure/logger"
	"sms-management/internal/infrastructure/messagebroker"
	"sms-management/internal/infrastructure/security"
	"sms-management/internal/repository/impl"
	"sms-management/internal/service"
	"sms-management/internal/worker"
)

type App struct {
	Config            *config.Config
	DB                *gorm.DB
	RedisClient       redis.UniversalClient
	ServerHandler     *grpchandler.ServerManagementServer
	RESTImportExport  *resthandler.ImportExportHandler
	OutboxRelay       *worker.OutboxRelay
	StatusConsumer    *worker.StatusConsumer
	Authorizer        *security.Authorizer
	MethodPermissions map[string]security.PermissionCode
	grpcServer        *grpc.Server
	httpServer        *http.Server
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
	serverSvc := service.NewServerService(serverRepo, outboxRepo, cfg.AgentSigningSecret, cfg.OccMaxRetries)

	serverHandler := grpchandler.NewServerManagementServer(serverSvc)
	restImportExport := resthandler.NewImportExportHandler(serverSvc)

	// Initialize Publisher
	publisher := messagebroker.NewRedisPublisher(redisClient, cfg.Outbox.MaxLen)

	// Initialize Outbox Worker
	outboxRelay := worker.NewOutboxRelay(outboxRepo, publisher, cfg.Outbox)

	// Initialize Status Consumer (listens for status changes from sms-monitoring)
	subscriber := messagebroker.NewRedisSubscriber(redisClient)
	statusConsumer := worker.NewStatusConsumer(subscriber, serverRepo)

	authorizer := security.NewAuthorizer()
	methodPermissions := map[string]security.PermissionCode{
		servermanagementv1.ServerManagementService_CreateServer_FullMethodName:  security.PermServerCreate,
		servermanagementv1.ServerManagementService_ViewServers_FullMethodName:   security.PermServerRead,
		servermanagementv1.ServerManagementService_UpdateServer_FullMethodName:  security.PermServerUpdate,
		servermanagementv1.ServerManagementService_DeleteServer_FullMethodName:  security.PermServerDelete,
		servermanagementv1.ServerManagementService_ImportServers_FullMethodName: security.PermServerImport,
		servermanagementv1.ServerManagementService_ExportServers_FullMethodName: security.PermServerExport,
	}

	return &App{
		Config:            cfg,
		DB:                db,
		RedisClient:       redisClient,
		ServerHandler:     serverHandler,
		RESTImportExport:  restImportExport,
		OutboxRelay:       outboxRelay,
		StatusConsumer:    statusConsumer,
		Authorizer:        authorizer,
		MethodPermissions: methodPermissions,
	}, nil
}
