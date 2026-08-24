package api

import (
	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
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
