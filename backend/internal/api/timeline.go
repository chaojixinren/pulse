package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	"github.com/chaojixinren/pulse/pkg/response"
)

type TimelineHandler struct {
	svc *service.TimelineService
}

func NewTimelineHandler(svc *service.TimelineService) *TimelineHandler {
	return &TimelineHandler{svc: svc}
}

func (h *TimelineHandler) List(c *gin.Context) {
	userID := currentUserID(c)

	var filter service.TimelineFilter
	if v := c.Query("identity_id"); v != "" {
		filter.IdentityID = &v
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.To = &t
		}
	}
	if v := c.Query("status"); v != "" {
		filter.Status = &v
	}

	page := parseIntDefault(c.Query("page"), 1)
	size := parseIntDefault(c.Query("size"), 20)

	items, total, err := h.svc.List(c.Request.Context(), userID, filter, page, size)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"items": items,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
