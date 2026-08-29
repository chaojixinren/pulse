package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/middleware"
	"github.com/chaojixinren/pulse/internal/repository"
	"github.com/chaojixinren/pulse/internal/service"
	"github.com/chaojixinren/pulse/internal/worker"
)

// NewRouter 组装所有依赖、注册路由，并返回 HTTP 引擎与后台转写 worker。
func NewRouter(cfg *config.Config, db *sql.DB) (*gin.Engine, *worker.AudioProcessor) {
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
	reportService := service.NewReportService(sessionRepo, identityRepo)
	aiService := service.NewAIService(cfg)
	deviceService := service.NewDeviceService(deviceRepo)
	accountService := service.NewAccountService(userRepo, identityRepo, deviceRepo, sessionRepo, tokenRepo, cfg.AudioEncryptionKey)

	// handlers
	healthHandler := NewHealthHandler(db)
	authHandler := NewAuthHandler(authService)
	audioHandler := NewAudioHandler(audioService)
	identityHandler := NewIdentityHandler(identityService)
	timelineHandler := NewTimelineHandler(timelineService)
	reportHandler := NewReportHandler(reportService)
	deviceHandler := NewDeviceHandler(deviceService)
	accountHandler := NewAccountHandler(accountService)

	// worker
	processor := worker.NewAudioProcessor(sessionRepo, sttService, identityRepo, aiService, accountService, cfg.AudioEncryptionKey)

	r.GET("/health", healthHandler.Check)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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
			authed.GET("/reports/weekly", reportHandler.Weekly)
			authed.GET("/reports/stats", reportHandler.Stats)
			authed.GET("/account/export", accountHandler.Export)
			authed.DELETE("/account", accountHandler.Delete)
			authed.GET("/account/asr", accountHandler.GetAsr)
			authed.PUT("/account/asr", accountHandler.UpdateAsr)
			authed.GET("/account/ai", accountHandler.GetAi)
			authed.PUT("/account/ai", accountHandler.UpdateAi)

			authed.GET("/devices", deviceHandler.List)
			authed.POST("/devices", deviceHandler.CreateDevice)
			authed.GET("/devices/:id", deviceHandler.Get)
			authed.DELETE("/devices/:id", deviceHandler.Unbind)
			authed.POST("/devices/:id/heartbeat", deviceHandler.Heartbeat)
			authed.POST("/devices/:id/command", deviceHandler.Command)
		}

		// 设备态路由组。必须与 authed 分开：DeviceAuth 会写入与用户态相同的
		// CtxUserID，一旦把设备接口并进 authed，设备 token 就能访问时间线、
		// 账号导出乃至删号。这里只暴露设备真正需要的三个接口。
		device := v1.Group("/device")
		{
			authedDevice := device.Group("")
			authedDevice.Use(middleware.DeviceAuth(deviceService))
			{
				authedDevice.POST("/audio/upload", audioHandler.Upload)
				authedDevice.POST("/heartbeat", deviceHandler.DeviceHeartbeat)
				authedDevice.POST("/commands/:id/ack", deviceHandler.AckCommand)
			}
		}
	}

	return r, processor
}
