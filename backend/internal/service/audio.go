package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

// UploadInput 音频上传入参。
type UploadInput struct {
	Data        []byte
	Filename    string
	ContentType string
	DeviceID    string
	Duration    int
	RecordedAt  time.Time
}

type AudioService struct {
	cfg      *config.Config
	sessions *repository.AudioSessionRepo
}

func NewAudioService(cfg *config.Config, sessions *repository.AudioSessionRepo) *AudioService {
	return &AudioService{cfg: cfg, sessions: sessions}
}

var allowedAudioExt = map[string]bool{
	".wav": true,
	".mp3": true,
	".m4a": true,
}

func extToMime(ext string) string {
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	default:
		return "audio/wav"
	}
}

// Upload 校验并直接把音频二进制落库到 audio_sessions。
func (s *AudioService) Upload(ctx context.Context, userID string, in UploadInput) (*model.AudioSession, error) {
	if len(in.Data) == 0 {
		return nil, apperrors.NewBadRequest("音频文件为空")
	}
	if int64(len(in.Data)) > s.cfg.MaxAudioSize {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("音频文件超过大小上限 %d 字节", s.cfg.MaxAudioSize))
	}

	ext := strings.ToLower(filepath.Ext(in.Filename))
	if !allowedAudioExt[ext] {
		return nil, apperrors.NewBadRequest("不支持的音频格式，仅支持 WAV/MP3/M4A")
	}

	fileSize := int64(len(in.Data))
	var deviceID *string
	if in.DeviceID != "" {
		deviceID = &in.DeviceID
	}

	mime := extToMime(ext)
	if in.ContentType != "" {
		mime = in.ContentType
	}

	session := &model.AudioSession{
		ID:            utils.NewUUID(),
		UserID:        userID,
		DeviceID:      deviceID,
		AudioData:     in.Data,
		AudioMime:     &mime,
		Duration:      in.Duration,
		FileSize:      &fileSize,
		Status:        model.StatusPending,
		ExtractedData: "{}",
		RecordedAt:    in.RecordedAt.UTC(),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return session, nil
}

func (s *AudioService) Get(ctx context.Context, userID, id string) (*model.AudioSession, error) {
	session, err := s.sessions.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if session == nil {
		return nil, apperrors.NewNotFound("语音会话不存在")
	}
	return session, nil
}

// Retry 将失败状态的会话重新放回处理队列。
func (s *AudioService) Retry(ctx context.Context, userID, id string) error {
	session, err := s.sessions.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return apperrors.WrapInternal(err)
	}
	if session == nil {
		return apperrors.NewNotFound("语音会话不存在")
	}
	if session.Status != model.StatusFailed {
		return apperrors.NewBadRequest("只有失败状态的会话才能重试")
	}
	if err := s.sessions.UpdateStatus(ctx, id, model.StatusPending, ""); err != nil {
		return err
	}
	return nil
}
