package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

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
	"sms-reporting/internal/repository/impl"
	"sms-reporting/internal/service"
)

type App struct {
	cfg         *config.Config
	grpcServer  *grpc.Server
	worker      service.ReportingWorker
	eventStream consumer.EventConsumer
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
	db.Exec("CREATE SCHEMA IF NOT EXISTS reporting_schema;")
	db.AutoMigrate(&domain.ReportRequest{}, &domain.ReportingServer{})

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
	grpcServer := grpc.NewServer()
	reportingv1.RegisterReportingServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	// Initialize Event Consumer
	subscriber := messagebroker.NewRedisSubscriber(redisClient)
	eventStream := consumer.NewEventConsumer(subscriber, repo)
	go eventStream.Start(context.Background())

	return &App{
		cfg:         cfg,
		grpcServer:  grpcServer,
		worker:      worker,
		eventStream: eventStream,
	}, nil
}

func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Worker
	a.worker.Start(ctx)

	// Start Consumer
	a.eventStream.Start(ctx)

	// Start gRPC Server
	errChan := make(chan error, 1)
	go func() {
		lis, err := net.Listen("tcp", ":"+a.cfg.GRPCPort)
		if err != nil {
			errChan <- fmt.Errorf("failed to listen on port %s: %w", a.cfg.GRPCPort, err)
			return
		}
		logger.Log.Sugar().Infof("gRPC server listening on port %s", a.cfg.GRPCPort)
		if err := a.grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("grpc server error: %w", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		logger.Log.Info("Shutting down Reporting Service...")
		cancel()
		a.grpcServer.GracefulStop()
		a.worker.Stop()
	}

	return nil
}
