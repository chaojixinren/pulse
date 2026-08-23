package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log 是全局 zap 日志器，在 main 中调用 Init 后可用。
var Log *zap.Logger

// Init 根据运行环境初始化 zap 日志器。
func Init(environment, level string) {
	var cfg zap.Config
	if environment == "production" || environment == "release" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if lvl, err := zapcore.ParseLevel(level); err == nil {
		cfg.Level.SetLevel(lvl)
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	Log = logger
}

// Sync 刷新日志缓冲。
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
