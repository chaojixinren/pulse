package api

import (
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chaojixinren/pulse/internal/service"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/response"
)

type AudioHandler struct {
	svc *service.AudioService
}

func NewAudioHandler(svc *service.AudioService) *AudioHandler { return &AudioHandler{svc: svc} }

func (h *AudioHandler) Upload(c *gin.Context) {
	userID := currentUserID(c)
	if userID == "" {
		fail(c, apperrors.ErrUnauthorized)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		fail(c, apperrors.NewBadRequest("缺少上传文件字段 file"))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		fail(c, apperrors.WrapInternal(err))
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		fail(c, apperrors.WrapInternal(err))
		return
	}

	recordedAt := time.Now().UTC()
	if v := c.PostForm("recorded_at"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			recordedAt = t
		}
	}

	session, err := h.svc.Upload(c.Request.Context(), userID, service.UploadInput{
		Data:        data,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		DeviceID:    c.PostForm("device_id"),
		Duration:    parseIntDefault(c.PostForm("duration"), 0),
		RecordedAt:  recordedAt,
	})
	if err != nil {
		fail(c, err)
		return
	}

	response.OK(c, gin.H{
		"session_id": session.ID,
		"status":     session.Status,
	})
}

func (h *AudioHandler) Retry(c *gin.Context) {
	userID := currentUserID(c)
	if userID == "" {
		fail(c, apperrors.ErrUnauthorized)
		return
	}
	if err := h.svc.Retry(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, err)
		return
	}
	response.OKMessage(c, "已重新加入处理队列")
}
