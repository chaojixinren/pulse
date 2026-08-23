package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

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
	r.Use(middleware.Logger(), middleware.ErrorHandler(), middleware.CORS(cfg.AllowedOrigins))

	// repositories
	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewRefreshTokenRepo(db)
	sessionRepo := repository.NewAudioSessionRepo(db)
	identityRepo := repository.NewIdentityRepo(db)
	deviceRepo := repository.NewDeviceRepo(db)
	reminderRepo := repository.NewReminderRepo(db)

	// services
	authService := service.NewAuthService(cfg, userRepo, tokenRepo)
	audioService := service.NewAudioService(cfg, sessionRepo)
	sttService := service.NewSttService(cfg)
	identityService := service.NewIdentityService(identityRepo)
	timelineService := service.NewTimelineService(sessionRepo)
	reportService := service.NewReportService(sessionRepo, identityRepo)
	aiService := service.NewAIService(cfg)
	deviceService := service.NewDeviceService(deviceRepo)
	reminderService := service.NewReminderService(reminderRepo)

	// handlers
	healthHandler := NewHealthHandler(db, rdb)
	authHandler := NewAuthHandler(authService)
	audioHandler := NewAudioHandler(audioService)
	identityHandler := NewIdentityHandler(identityService)
	timelineHandler := NewTimelineHandler(timelineService)
	reportHandler := NewReportHandler(reportService)
	deviceHandler := NewDeviceHandler(deviceService)
	reminderHandler := NewReminderHandler(reminderService)

	// worker
	processor := worker.NewAudioProcessor(sessionRepo, sttService, rdb, identityRepo, aiService, reminderService)

	r.GET("/health", healthHandler.Check)

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", middleware.Auth(cfg), authHandler.Me)
		}

		authed := v1.Group("")
		authed.Use(middleware.Auth(cfg))
		{
			authed.POST("/audio/upload", audioHandler.Upload)
			authed.POST("/audio/:id/retry", audioHandler.Retry)
			authed.GET("/identities", identityHandler.List)
			authed.POST("/identities", identityHandler.Create)
			authed.PUT("/identities/:id", identityHandler.Update)
			authed.DELETE("/identities/:id", identityHandler.Delete)
			authed.PUT("/identities/:id/default", identityHandler.SetDefault)
			authed.GET("/timeline", timelineHandler.List)
			authed.GET("/reports/daily", reportHandler.Daily)

			authed.POST("/devices/bind-code", deviceHandler.GenerateBindCode)
			authed.POST("/devices/bind", deviceHandler.Bind)
			authed.GET("/devices", deviceHandler.List)
			authed.GET("/devices/:id", deviceHandler.Get)
			authed.DELETE("/devices/:id", deviceHandler.Unbind)
			authed.POST("/devices/:id/heartbeat", deviceHandler.Heartbeat)
			authed.POST("/devices/:id/command", deviceHandler.Command)

			authed.GET("/reminders", reminderHandler.List)
			authed.PUT("/reminders/:id/done", reminderHandler.Done)
			authed.PUT("/reminders/:id/dismiss", reminderHandler.Dismiss)
		}
	}

	return r, processor
}
