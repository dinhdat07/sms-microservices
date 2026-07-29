package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"sms-agent-handler/internal/handler/rest"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (a *App) Run() error {


	// Handle OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Initialize Handlers
	heartbeatHandler := rest.NewHeartbeatHandler(a.RedisClient, a.logger)
	authMiddleware := rest.NewMasterKeyAuthMiddleware(a.cfg.MasterKey, a.logger)

	// Setup Router
	r := mux.NewRouter()
	
	// Create subrouter for agent API
	agentRouter := r.PathPrefix("/api/v1/agent").Subrouter()
	agentRouter.Use(authMiddleware.Middleware)
	agentRouter.HandleFunc("/heartbeat", heartbeatHandler.HandleHeartbeat).Methods("POST")

	// HTTP Server
	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.cfg.Port),
		Handler: r,
	}

	// Start server
	go func() {
		a.logger.Info("Starting HTTP Server for Agent Handler", zap.String("port", a.cfg.Port))
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	<-sigCh

	a.Shutdown(context.Background())

	return nil
}

func (a *App) Shutdown(ctx context.Context) {
	a.logger.Info("Shutting down Agent Handler...")

	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			a.logger.Error("Server forced to shutdown", zap.Error(err))
		}
	}

	if a.RedisClient != nil {
		a.RedisClient.Close()
	}

	a.logger.Info("Agent Handler stopped")
}
