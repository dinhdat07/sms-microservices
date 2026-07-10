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

	authv1 "sms-identity/gen/go/auth/v1"
	"sms-identity/internal/shared/logger"
)

func (a *App) Run() error {
	grpcAddr := fmt.Sprintf(":%s", a.Config.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcSrv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcSrv, a.AuthHandler)

	go func() {
		logger.Log.Sugar().Infof("gRPC server listening on %s", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Log.Sugar().Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, gwmux, grpcAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to register gRPC gateway: %w", err)
	}

	mux := http.NewServeMux()
	// Traefik ForwardAuth route
	mux.HandleFunc("/verify", a.ForwardAuthHandler.Verify)
	// gRPC Gateway routes
	mux.Handle("/", gwmux)

	httpAddr := fmt.Sprintf(":%s", a.Config.HTTPPort)
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	go func() {
		logger.Log.Sugar().Infof("HTTP server listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Sugar().Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Sugar().Info("Shutting down servers...")
	grpcSrv.GracefulStop()
	httpSrv.Shutdown(ctx)
	logger.Log.Sugar().Info("Shutdown complete")

	return nil
}
