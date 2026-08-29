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

type claimDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Name     string `json:"name"`
	BindCode string `json:"bind_code" binding:"required"`
}

// Claim 设备自助配对（免鉴权）：device_id + 一次性绑定码 → device_token。
// 设备把返回的 token 写进 NVS 即可，无需人工把 token 抄进 TF 卡。
//
// 该端点不带任何鉴权，屏障只有绑定码本身（6 位数字 / 10 分钟 / 一次性），
// 且按产品决策未加尝试次数限制。
func (h *DeviceHandler) Claim(c *gin.Context) {
	var req claimDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, apperrors.NewBadRequest("参数错误: "+err.Error()))
		return
	}
	device, token, err := h.svc.Claim(c.Request.Context(), req.DeviceID, req.Name, req.BindCode)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"device": device, "device_token": token})
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
