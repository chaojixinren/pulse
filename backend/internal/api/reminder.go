package api

import (
	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	"github.com/chaojixinren/pulse/pkg/response"
)

type ReminderHandler struct {
	svc *service.ReminderService
}

func NewReminderHandler(svc *service.ReminderService) *ReminderHandler {
	return &ReminderHandler{svc: svc}
}

// List 返回待处理提醒列表。
func (h *ReminderHandler) List(c *gin.Context) {
	userID := currentUserID(c)
	list, err := h.svc.ListPending(c.Request.Context(), userID)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *ReminderHandler) Done(c *gin.Context) {
	userID := currentUserID(c)
	if err := h.svc.MarkDone(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "已标记完成")
}

func (h *ReminderHandler) Dismiss(c *gin.Context) {
	userID := currentUserID(c)
	if err := h.svc.Dismiss(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "已忽略")
}
