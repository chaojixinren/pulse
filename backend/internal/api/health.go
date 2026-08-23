package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/chaojixinren/pulse/pkg/response"
)

type HealthHandler struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewHealthHandler(db *sql.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{"mysql": "up", "redis": "up"}
	healthy := true

	if err := h.db.PingContext(ctx); err != nil {
		checks["mysql"] = "down"
		healthy = false
	}
	if h.rdb != nil {
		if err := h.rdb.Ping(ctx).Err(); err != nil {
			checks["redis"] = "down"
			healthy = false
		}
	}

	if healthy {
		response.OK(c, nil)
		return
	}
	c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50300, Message: "服务不可用", Data: checks})
}
