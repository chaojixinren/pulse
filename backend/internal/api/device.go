package api

import (
	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/response"
)

type DeviceHandler struct {
	svc *service.DeviceService
}

func NewDeviceHandler(svc *service.DeviceService) *DeviceHandler { return &DeviceHandler{svc: svc} }

// GenerateBindCode 生成一次性绑定码。
func (h *DeviceHandler) GenerateBindCode(c *gin.Context) {
	userID := currentUserID(c)
	code, err := h.svc.GenerateBindCode(c.Request.Context(), userID)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, code)
}

type bindDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Name     string `json:"name"`
	BindCode string `json:"bind_code" binding:"required"`
}

// Bind 绑定设备，返回设备信息与一次性设备 token。
func (h *DeviceHandler) Bind(c *gin.Context) {
	userID := currentUserID(c)
	var req bindDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	device, token, err := h.svc.Bind(c.Request.Context(), userID, req.DeviceID, req.Name, req.BindCode)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"device": device, "device_token": token})
}

func (h *DeviceHandler) List(c *gin.Context) {
	userID := currentUserID(c)
	list, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, list)
}

func (h *DeviceHandler) Get(c *gin.Context) {
	userID := currentUserID(c)
	device, err := h.svc.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, device)
}

func (h *DeviceHandler) Unbind(c *gin.Context) {
	userID := currentUserID(c)
	if err := h.svc.Unbind(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "解绑成功")
}

type heartbeatRequest struct {
	BatteryLevel    *int    `json:"battery_level"`
	FirmwareVersion *string `json:"firmware_version"`
}

// Heartbeat 硬件上报心跳。
func (h *DeviceHandler) Heartbeat(c *gin.Context) {
	userID := currentUserID(c)
	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	device, err := h.svc.Heartbeat(c.Request.Context(), userID, c.Param("id"), req.FirmwareVersion, req.BatteryLevel)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, device)
}

type commandRequest struct {
	Command string `json:"command" binding:"required"`
}

// Command 下发指令（Phase 2 先落库）。
func (h *DeviceHandler) Command(c *gin.Context) {
	userID := currentUserID(c)
	var req commandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	cmd, err := h.svc.IssueCommand(c.Request.Context(), userID, c.Param("id"), req.Command)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, cmd)
}
