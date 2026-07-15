package app

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"sms-identity/internal/config"
	"sms-identity/internal/handler"
	authgrpc "sms-identity/internal/handler/grpcserver"
	"sms-identity/internal/infrastructure/database"
	"sms-identity/internal/infrastructure/logger"
	"sms-identity/internal/infrastructure/security"
	authrepo "sms-identity/internal/repository/impl"
	authsvc "sms-identity/internal/service"
)

type App struct {
	Config             *config.Config
	DB                 *gorm.DB
	RedisClient        redis.UniversalClient
	AuthHandler        *authgrpc.AuthServer
	ForwardAuthHandler *handler.ForwardAuthHandler
	CSRFManager        *security.CSRFManager
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

	// 4. Init Identity Services
	userRepo := authrepo.NewUserRepository(db)
	sessionRepo := authrepo.NewAuthSessionRepository(db)
	refreshRepo := authrepo.NewRefreshTokenRepository(db)
	revoStore := authrepo.NewSessionRevocationStore(redisClient)
	tokenMgr := security.NewTokenManager(cfg.JWTSecret)

	authService := authsvc.NewAuthService(userRepo, sessionRepo, refreshRepo, revoStore, tokenMgr)
	authServer := authgrpc.NewAuthServer(authService)

	authenticator := security.NewAuthenticator(cfg.JWTSecret, redisClient)
	csrfManager := security.NewCSRFManager()
	forwardAuthHandler := handler.NewForwardAuthHandler(authenticator, csrfManager)

	return &App{
		Config:             cfg,
		DB:                 db,
		RedisClient:        redisClient,
		AuthHandler:        authServer,
		ForwardAuthHandler: forwardAuthHandler,
		CSRFManager:        csrfManager,
	}, nil
}
