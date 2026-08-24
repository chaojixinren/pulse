package api

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/middleware"
	"github.com/chaojixinren/pulse/internal/repository"
	"github.com/chaojixinren/pulse/internal/service"
	"github.com/chaojixinren/pulse/internal/worker"
)

// NewRouter 组装所有依赖、注册路由，并返回 HTTP 引擎与后台转写 worker。
func NewRouter(cfg *config.Config, db *sql.DB, rdb *redis.Client) (*gin.Engine, *worker.AudioProcessor) {
	gin.SetMode(cfg.GINMode)
	r := gin.New()
	r.Use(middleware.Trace(), middleware.Logger(), middleware.Metrics(), middleware.ErrorHandler(), middleware.CORS(cfg.AllowedOrigins))

	// repositories
	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewRefreshTokenRepo(db)
	sessionRepo := repository.NewAudioSessionRepo(db)
	identityRepo := repository.NewIdentityRepo(db)
	deviceRepo := repository.NewDeviceRepo(db)

	// services
	authService := service.NewAuthService(cfg, userRepo, tokenRepo)
	audioService := service.NewAudioService(cfg, sessionRepo)
	sttService := service.NewSttService(cfg)
	identityService := service.NewIdentityService(identityRepo)
	timelineService := service.NewTimelineService(sessionRepo)
	reportService := service.NewReportService(sessionRepo, identityRepo, rdb)
	aiService := service.NewAIService(cfg)
	deviceService := service.NewDeviceService(deviceRepo)
	accountService := service.NewAccountService(userRepo, identityRepo, deviceRepo, sessionRepo, tokenRepo)

	// handlers
	healthHandler := NewHealthHandler(db, rdb)
	authHandler := NewAuthHandler(authService)
	audioHandler := NewAudioHandler(audioService)
	identityHandler := NewIdentityHandler(identityService)
	timelineHandler := NewTimelineHandler(timelineService)
	reportHandler := NewReportHandler(reportService)
	deviceHandler := NewDeviceHandler(deviceService)
	accountHandler := NewAccountHandler(accountService)

	// worker
	processor := worker.NewAudioProcessor(sessionRepo, sttService, rdb, identityRepo, aiService, cfg.AudioEncryptionKey)

	r.GET("/health", healthHandler.Check)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", middleware.RateLimitByIP(rdb, "auth", cfg.RateLimitAuthPerMin, time.Minute), authHandler.Register)
			auth.POST("/login", middleware.RateLimitByIP(rdb, "auth", cfg.RateLimitAuthPerMin, time.Minute), authHandler.Login)
			auth.POST("/refresh", middleware.RateLimitByIP(rdb, "auth", cfg.RateLimitAuthPerMin, time.Minute), authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", middleware.Auth(cfg), authHandler.Me)
		}

		authed := v1.Group("")
		authed.Use(middleware.Auth(cfg))
		{
			authed.POST("/audio/upload", middleware.RateLimitByUser(rdb, "upload", cfg.RateLimitUploadPerMin, time.Minute), audioHandler.Upload)
			authed.POST("/audio/:id/retry", audioHandler.Retry)
			authed.GET("/identities", identityHandler.List)
			authed.POST("/identities", identityHandler.Create)
			authed.PUT("/identities/:id", identityHandler.Update)
			authed.DELETE("/identities/:id", identityHandler.Delete)
			authed.PUT("/identities/:id/default", identityHandler.SetDefault)
			authed.GET("/timeline", timelineHandler.List)
			authed.GET("/reports/daily", reportHandler.Daily)
			authed.GET("/reports/weekly", reportHandler.Weekly)
			authed.GET("/reports/stats", reportHandler.Stats)
			authed.GET("/account/export", accountHandler.Export)
			authed.DELETE("/account", accountHandler.Delete)

			authed.POST("/devices/bind-code", deviceHandler.GenerateBindCode)
			authed.POST("/devices/bind", deviceHandler.Bind)
			authed.GET("/devices", deviceHandler.List)
			authed.GET("/devices/:id", deviceHandler.Get)
			authed.DELETE("/devices/:id", deviceHandler.Unbind)
			authed.POST("/devices/:id/heartbeat", deviceHandler.Heartbeat)
			authed.POST("/devices/:id/command", deviceHandler.Command)
		}
	}

	return r, processor
}
