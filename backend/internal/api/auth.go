package api

import (
	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/response"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	user, err := h.svc.Register(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, user)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, pair)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, pair)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "已注销")
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := currentUserID(c)
	if userID == "" {
		fail(c, apperrors.ErrUnauthorized)
		return
	}
	user, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, user)
}
