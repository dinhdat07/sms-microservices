package app

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	reportingv1 "sms-reporting/gen/go/reporting/v1"
	"sms-reporting/internal/config"
	"sms-reporting/internal/consumer"
	"sms-reporting/internal/domain"
	grpchandler "sms-reporting/internal/handler/grpc"
	"sms-reporting/internal/infrastructure/database"
	"sms-reporting/internal/infrastructure/elasticsearch"
	"sms-reporting/internal/infrastructure/logger"
	"sms-reporting/internal/infrastructure/messagebroker"
	"sms-reporting/internal/infrastructure/notifier"
	"sms-reporting/internal/infrastructure/security"
	"sms-reporting/internal/repository/impl"
	"sms-reporting/internal/service"
)

type App struct {
	cfg               *config.Config
	grpcServer        *grpc.Server
	grpcHandler       reportingv1.ReportingServiceServer
	worker            service.ReportingWorker
	eventStream       consumer.EventConsumer
	scheduler         *Scheduler
	authorizer        *security.Authorizer
	methodPermissions map[string]security.PermissionCode
}

func NewApp() (*App, error) {
	// Initialize logger
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	logger.InitLogger(cfg.Logger, "reporting")

	// Initialize Postgres
	db, err := database.GetInstance(cfg.DBUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	// Migrate schema
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS reporting_schema;").Error; err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}
	if err := db.AutoMigrate(&domain.ReportRequest{}, &domain.ReportingServer{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Initialize Redis
	redisCfg, err := config.LoadRedisConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load redis config: %w", err)
	}
	redisClient := database.NewRedisClient(redisCfg)
	if redisClient != nil {
		_ = database.PingRedis(context.Background(), redisClient)
	}

	// Initialize Elasticsearch
	esCfg := config.LoadElasticsearchConfig()
	esClient, err := database.NewElasticsearchClient(context.Background(), []string{esCfg.URL})
	if err != nil {
		return nil, fmt.Errorf("elasticsearch connection failed: %w", err)
	}

	// Initialize Notifier
	smtpConfig := notifier.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		UseAuth:  cfg.SMTP.UseAuth,
		UseTLS:   cfg.SMTP.UseTLS,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		FromName: cfg.SMTP.FromName,
	}
	smtpMailer := notifier.NewMailer(smtpConfig)

	// Initialize Repos & Services
	repo := impl.NewGormReportingRepository(db)
	uptimeCalc := elasticsearch.NewESUptimeCalculator(esClient, esCfg.ServerIndex)

	worker := service.NewReportingWorker(repo, uptimeCalc, cfg.Reporting.WorkerCount, cfg.Reporting.JobQueueSize, smtpMailer)
	svc := service.NewReportingService(repo, worker)
	handler := grpchandler.NewReportingGrpcHandler(svc)

	// Setup gRPC Server
	// Note: We don't initialize the grpc server here anymore. It will be initialized in run.go with interceptors.

	// Initialize Event Consumer
	subscriber := messagebroker.NewRedisSubscriber(redisClient)
	eventStream := consumer.NewEventConsumer(subscriber, repo, cfg.Consumer)

	// Initialize Scheduler
	sched := NewScheduler(worker, redisClient, cfg.Reporting.CronSpec, cfg.Reporting.AdminEmail)

	authorizer := security.NewAuthorizer()
	methodPermissions := map[string]security.PermissionCode{
		reportingv1.ReportingService_RequestReport_FullMethodName: security.PermReportRequest,
	}

	return &App{
		cfg:               cfg,
		grpcServer:        nil, // Set in run.go
		grpcHandler:       handler,
		worker:            worker,
		eventStream:       eventStream,
		scheduler:         sched,
		authorizer:        authorizer,
		methodPermissions: methodPermissions,
	}, nil
}
