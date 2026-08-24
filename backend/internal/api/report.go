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

// Weekly 返回以 week（YYYY-MM-DD）所在周一为起点的周报；week 为空时取当前周。
func (h *ReportHandler) Weekly(c *gin.Context) {
	userID := currentUserID(c)
	report, err := h.svc.Weekly(c.Request.Context(), userID, c.Query("week"))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, report)
}

// Stats 返回 [from, to]（含两端）的统计汇总；from 缺省取 to 前 30 天。
func (h *ReportHandler) Stats(c *gin.Context) {
	userID := currentUserID(c)

	to := c.Query("to")
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	from := c.Query("from")
	if from == "" {
		if toT, err := time.Parse("2006-01-02", to); err == nil {
			from = toT.AddDate(0, 0, -29).Format("2006-01-02")
		} else {
			from = to // 交给 service 校验并返回错误
		}
	}

	report, err := h.svc.Stats(c.Request.Context(), userID, from, to)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, report)
}
