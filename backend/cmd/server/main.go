package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/chaojixinren/pulse/internal/api"
	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.GINMode, cfg.LogLevel)
	defer logger.Sync()

	db, err := config.InitDB(cfg.MySQLDSN)
	if err != nil {
		logger.Log.Fatal("连接 MySQL 失败", zap.Error(err))
	}
	defer db.Close()

	rdb, err := config.InitRedis(cfg.RedisURL)
	if err != nil {
		logger.Log.Fatal("连接 Redis 失败", zap.Error(err))
	}
	defer rdb.Close()

	router, processor := api.NewRouter(cfg, db, rdb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go processor.Run(ctx)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		logger.Log.Info("Pulse 后端启动", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("服务异常退出", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("正在关闭服务...")

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("服务关闭失败", zap.Error(err))
	}
	logger.Log.Info("服务已关闭")
}
