package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/config"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

// CtxUserID 是 gin context 中存放 user_id 的键。
const CtxUserID = "user_id"

// Auth 校验 JWT 并将 user_id 写入 context。
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			_ = c.Error(apperrors.ErrUnauthorized)
			c.Abort()
			return
		}
		userID, err := utils.ParseAccessToken(token, cfg.JWTSecret)
		if err != nil {
			_ = c.Error(apperrors.NewUnauthorized("token 无效或已过期"))
			c.Abort()
			return
		}
		c.Set(CtxUserID, userID)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	return extractSchemeToken(c, "Bearer")
}

// extractSchemeToken 解析 "Authorization: <scheme> <token>"，scheme 不匹配时返回空串。
// 用户态用 Bearer，设备态用 Device，两者共用同一套解析逻辑。
func extractSchemeToken(c *gin.Context, scheme string) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], scheme) {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
