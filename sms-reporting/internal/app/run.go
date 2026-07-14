package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	reportingv1 "sms-reporting/gen/go/reporting/v1"
	"sms-reporting/internal/infrastructure/logger"
)

func (a *App) Run() error {
	grpcAddr := fmt.Sprintf(":%s", a.cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", a.cfg.GRPCPort, err)
	}

	// Start Worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.worker.Start(ctx)

	// Start Event Consumer
	a.eventStream.Start(ctx)

	go func() {
		logger.Log.Sugar().Infof("gRPC server listening on %s", grpcAddr)
		if err := a.grpcServer.Serve(lis); err != nil {
			logger.Log.Sugar().Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Setup gRPC Gateway
	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	
	// Register the generated handler for gRPC Gateway
	err = reportingv1.RegisterReportingServiceHandlerFromEndpoint(ctx, gwmux, grpcAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to register gRPC gateway: %w", err)
	}

	mux := http.NewServeMux()
	// Route all API traffic to gRPC gateway
	mux.Handle("/", gwmux)

	httpAddr := fmt.Sprintf(":%s", a.cfg.HTTPPort)
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	go func() {
		logger.Log.Sugar().Infof("HTTP REST gateway listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Sugar().Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	logger.Log.Info("Shutting down Reporting Service...")
	
	cancel()
	a.grpcServer.GracefulStop()
	httpSrv.Shutdown(ctx)
	a.worker.Stop()
	
	logger.Log.Info("Reporting Service shutdown complete")
	return nil
}
