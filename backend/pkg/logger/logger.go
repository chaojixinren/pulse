package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log 是全局 zap 日志器；默认 no-op 避免在 Init 前调用时 panic，
// main 中调用 Init 后会替换为真实日志器。
var Log *zap.Logger = zap.NewNop()

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
