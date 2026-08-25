package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/pkg/response"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{"mysql": "up"}
	healthy := true

	if err := h.db.PingContext(ctx); err != nil {
		checks["mysql"] = "down"
		healthy = false
	}

	if healthy {
		response.OK(c, nil)
		return
	}
	c.JSON(http.StatusServiceUnavailable, response.Response{Code: 50300, Message: "服务不可用", Data: checks})
}
