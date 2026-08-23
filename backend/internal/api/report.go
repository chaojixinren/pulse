package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	"github.com/chaojixinren/pulse/pkg/response"
)

type ReportHandler struct {
	svc *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler { return &ReportHandler{svc: svc} }

func (h *ReportHandler) Daily(c *gin.Context) {
	userID := currentUserID(c)

	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	report, err := h.svc.Daily(c.Request.Context(), userID, date)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, report)
}
