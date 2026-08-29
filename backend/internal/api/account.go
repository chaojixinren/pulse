package api

import (
	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/response"
)

type AccountHandler struct {
	svc *service.AccountService
}

func NewAccountHandler(svc *service.AccountService) *AccountHandler { return &AccountHandler{svc: svc} }

// Export 导出当前用户全部个人数据。
func (h *AccountHandler) Export(c *gin.Context) {
	userID := currentUserID(c)
	export, err := h.svc.Export(c.Request.Context(), userID)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, export)
}

// Delete 注销当前账户。
func (h *AccountHandler) Delete(c *gin.Context) {
	userID := currentUserID(c)
	if err := h.svc.Delete(c.Request.Context(), userID); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "账户已注销")
}

// GetAsr 返回当前用户的 ASR 配置（脱敏）。
func (h *AccountHandler) GetAsr(c *gin.Context) {
	view, err := h.svc.GetAsrSettings(c.Request.Context(), currentUserID(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, view)
}

// UpdateAsr 部分更新当前用户的 ASR 配置。
func (h *AccountHandler) UpdateAsr(c *gin.Context) {
	var req service.AsrSettingsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	view, err := h.svc.UpdateAsrSettings(c.Request.Context(), currentUserID(c), &req)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, view)
}

// GetAi 返回当前用户的 AI 分析配置（脱敏）。
func (h *AccountHandler) GetAi(c *gin.Context) {
	view, err := h.svc.GetAiSettings(c.Request.Context(), currentUserID(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, view)
}

// UpdateAi 部分更新当前用户的 AI 分析配置。
func (h *AccountHandler) UpdateAi(c *gin.Context) {
	var req service.AiSettingsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	view, err := h.svc.UpdateAiSettings(c.Request.Context(), currentUserID(c), &req)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, view)
}
