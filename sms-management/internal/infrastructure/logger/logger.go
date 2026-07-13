package logger

import (
	"log"
	"os"

	"sms-management/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

func init() {
	Log, _ = zap.NewDevelopment()
}

func InitLogger(cfg config.LoggerConfig, appName string) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("failed to create logs directory: %v", err)
	}

	maxSize := cfg.LogMaxSize
	if maxSize <= 0 {
		maxSize = 10
	}
	maxBackups := cfg.LogMaxBackups
	if maxBackups < 0 {
		maxBackups = 3
	}
	maxAge := cfg.LogMaxAge
	if maxAge <= 0 {
		maxAge = 28
	}

	logFile := "logs/app.log"
	if appName != "" {
		logFile = "logs/" + appName + ".log"
	}

	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   cfg.LogCompress,
	})

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	prodConfig := zap.NewProductionEncoderConfig()
	prodConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(prodConfig)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.InfoLevel),
		zapcore.NewCore(fileEncoder, w, zap.InfoLevel),
	)

	Log = zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(Log)
	zap.RedirectStdLog(Log)
}
