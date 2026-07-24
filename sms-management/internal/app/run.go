package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servermanagementv1 "sms-management/gen/go/server_management/v1"
	"sms-management/internal/handler/grpc/middlewares"
	"sms-management/internal/infrastructure/logger"
)

func (a *App) Run() error {
	grpcAddr := fmt.Sprintf(":%s", a.Config.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Start Outbox Relay Worker
	if a.OutboxRelay != nil {
		a.OutboxRelay.Start()
	}

	// Start Status Consumer (listens for monitoring status changes)
	if a.StatusConsumer != nil {
		a.StatusConsumer.Start(context.Background())
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middlewares.AuthMetadataInterceptor(),
			middlewares.PermissionInterceptor(a.Authorizer, a.MethodPermissions),
		),
	)
	servermanagementv1.RegisterServerManagementServiceServer(grpcSrv, a.ServerHandler)

	go func() {
		logger.Log.Sugar().Infof("gRPC server listening on %s", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Log.Sugar().Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup gRPC Gateway
	gwmux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.HasPrefix(key, "X-User-") {
				return key, true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = servermanagementv1.RegisterServerManagementServiceHandlerFromEndpoint(ctx, gwmux, grpcAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to register gRPC gateway: %w", err)
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// REST API Handler for import/export
	mux.HandleFunc("/api/v1/servers/import", a.RESTImportExport.HandleImport)
	mux.HandleFunc("/api/v1/servers/export", a.RESTImportExport.HandleExport)

	// gRPC Gateway routes
	mux.Handle("/", gwmux)
	mux.Handle("/openapi/", http.StripPrefix("/openapi/", http.FileServer(http.Dir("./api/openapi"))))

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
	if a.OutboxRelay != nil {
		a.OutboxRelay.Stop()
	}
	grpcSrv.GracefulStop()
	httpSrv.Shutdown(ctx)
	logger.Log.Sugar().Info("Shutdown complete")

	return nil
}
