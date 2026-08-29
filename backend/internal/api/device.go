package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/response"
)

type DeviceHandler struct {
	svc *service.DeviceService
}

func NewDeviceHandler(svc *service.DeviceService) *DeviceHandler { return &DeviceHandler{svc: svc} }

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

// DeviceHeartbeat 设备级心跳（Authorization: Device <token>）。
// 设备身份由 token 反解，不需要在 URL 里带 id。
// 响应捎带待执行指令与服务端时间，省掉一次独立请求。
func (h *DeviceHandler) DeviceHeartbeat(c *gin.Context) {
	deviceUUID := currentDeviceUUID(c)
	if deviceUUID == "" {
		fail(c, apperrors.ErrUnauthorized)
		return
	}
	var req heartbeatRequest
	// 心跳允许空 body：设备刚开机、PMU 还没出数时也应能报到。
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
			return
		}
	}
	device, commands, err := h.svc.DeviceHeartbeat(c.Request.Context(), deviceUUID, req.FirmwareVersion, req.BatteryLevel)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"device":   device,
		"commands": commands,
		// 设备 RTC 未校时时可直接用这个值兜底，省掉一次 NTP。
		"server_time": time.Now().UTC().Format(time.RFC3339),
	})
}

type ackCommandRequest struct {
	Status string `json:"status" binding:"required"`
}

// AckCommand 设备回执指令执行结果（done / failed）。
func (h *DeviceHandler) AckCommand(c *gin.Context) {
	deviceUUID := currentDeviceUUID(c)
	if deviceUUID == "" {
		fail(c, apperrors.ErrUnauthorized)
		return
	}
	var req ackCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	if err := h.svc.AckCommand(c.Request.Context(), deviceUUID, c.Param("id"), req.Status); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "回执已记录")
}

// ========== Device 创建/绑定（App 领养，一次性下发 token） ==========

type createDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Name     string `json:"name"`
}

// CreateDevice App 用户创建设备绑定，一次性返回 device_token 供抄录到硬件。
// POST /api/v1/devices
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	userID := currentUserID(c)
	if userID == "" {
		fail(c, apperrors.ErrUnauthorized)
		return
	}
	var req createDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	device, token, err := h.svc.CreateDevice(c.Request.Context(), userID, req.DeviceID, req.Name)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"device":       device,
		"device_token": token,
	})
}
