package app

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	authgrpc "sms-identity/internal/modules/identity/handler/grpcserver"
	authrepo "sms-identity/internal/modules/identity/repository/impl"
	authsvc "sms-identity/internal/modules/identity/service"
	"sms-identity/internal/shared/config"
	"sms-identity/internal/shared/database"
	"sms-identity/internal/shared/logger"
	"sms-identity/internal/infrastructure/security"
	"sms-identity/internal/modules/identity/handler"
)

type App struct {
	Config             *config.Config
	DB                 *gorm.DB
	RedisClient        redis.UniversalClient
	AuthHandler        *authgrpc.AuthServer
	ForwardAuthHandler *handler.ForwardAuthHandler
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

	// 4. Init Identity Services
	userRepo := authrepo.NewUserRepository(db)
	sessionRepo := authrepo.NewAuthSessionRepository(db)
	refreshRepo := authrepo.NewRefreshTokenRepository(db)
	revoStore := authrepo.NewSessionRevocationStore(redisClient)
	tokenMgr := security.NewTokenManager(cfg.JWTSecret)

	authService := authsvc.NewAuthService(userRepo, sessionRepo, refreshRepo, revoStore, tokenMgr)
	authServer := authgrpc.NewAuthServer(authService)

	authenticator := security.NewAuthenticator(cfg.JWTSecret, redisClient)
	forwardAuthHandler := handler.NewForwardAuthHandler(authenticator)

	return &App{
		Config:             cfg,
		DB:                 db,
		RedisClient:        redisClient,
		AuthHandler:        authServer,
		ForwardAuthHandler: forwardAuthHandler,
	}, nil
}
