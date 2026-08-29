package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/model"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// 设备级鉴权写入 gin context 的键。
const (
	// CtxDeviceUUID 是 devices.id（主键 UUID）。
	CtxDeviceUUID = "device_uuid"
	// CtxDeviceBizID 是 devices.device_id（硬件唯一标识字符串）。
	CtxDeviceBizID = "device_biz_id"
)

// DeviceAuthenticator 由 service.DeviceService 实现。
// 中间件不直接依赖 repository，保持 middleware -> service -> repository 的分层。
type DeviceAuthenticator interface {
	AuthenticateDevice(ctx context.Context, token string) (*model.Device, error)
}

// DeviceAuth 校验设备 token（Authorization: Device <token>），
// 并把设备归属的 user_id 写入与用户态相同的 CtxUserID 键。
//
// 复用 CtxUserID 是刻意的：AudioHandler.Upload 等 handler 只依赖 currentUserID，
// 这样设备态无需改动这些 handler 即可复用。代价是设备 token 一旦挂到用户态路由组上
// 就能越权访问时间线、账号导出等接口，因此设备接口必须单独成组（见 router.go），
// 绝不能把本中间件加到 authed 组。
func DeviceAuth(a DeviceAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractSchemeToken(c, "Device")
		if token == "" {
			_ = c.Error(apperrors.NewUnauthorized("缺少设备凭据，应为 Authorization: Device <token>"))
			c.Abort()
			return
		}
		device, err := a.AuthenticateDevice(c.Request.Context(), token)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Set(CtxUserID, device.UserID)
		c.Set(CtxDeviceUUID, device.ID)
		c.Set(CtxDeviceBizID, device.DeviceID)
		c.Next()
	}
}
