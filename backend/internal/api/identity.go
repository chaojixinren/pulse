package api

import (
	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/response"
)

type IdentityHandler struct {
	svc *service.IdentityService
}

func NewIdentityHandler(svc *service.IdentityService) *IdentityHandler {
	return &IdentityHandler{svc: svc}
}

type identityRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Color       string  `json:"color"`
	Icon        string  `json:"icon"`
	IsDefault   bool    `json:"is_default"`
}

func (h *IdentityHandler) List(c *gin.Context) {
	userID := currentUserID(c)
	list, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *IdentityHandler) Create(c *gin.Context) {
	userID := currentUserID(c)
	var req identityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	identity, err := h.svc.Create(c.Request.Context(), userID, identityInput(req))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, identity)
}

func (h *IdentityHandler) Update(c *gin.Context) {
	userID := currentUserID(c)
	var req identityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	identity, err := h.svc.Update(c.Request.Context(), userID, c.Param("id"), identityInput(req))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, identity)
}

func (h *IdentityHandler) Delete(c *gin.Context) {
	userID := currentUserID(c)
	if err := h.svc.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "删除成功")
}

func (h *IdentityHandler) SetDefault(c *gin.Context) {
	userID := currentUserID(c)
	if err := h.svc.SetDefault(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "已设为默认身份")
}

func identityInput(req identityRequest) service.IdentityInput {
	return service.IdentityInput{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		IsDefault:   req.IsDefault,
	}
}
