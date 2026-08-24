package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
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

// EncryptAudio 使用 AES-256-GCM 加密音频，返回 nonce || ciphertext。
// key 必须为 32 字节；调用方应在 key 为空时跳过加密。
func EncryptAudio(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// DecryptAudio 解密 EncryptAudio 产生的密文（nonce || ciphertext）。
func DecryptAudio(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return nil, errors.New("密文过短，无法解密")
	}
	return gcm.Open(nil, data[:ns], data[ns:], nil)
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

// detectAudioExt 根据文件头魔数识别音频格式，返回 ".wav"/".mp3"/".m4a"，无法识别返回 ""。
func detectAudioExt(data []byte) string {
	switch {
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WAVE":
		return ".wav"
	case len(data) >= 3 && string(data[0:3]) == "ID3":
		return ".mp3"
	case len(data) >= 2 && data[0] == 0xFF && data[1]&0xE0 == 0xE0:
		return ".mp3"
	case len(data) >= 8 && string(data[4:8]) == "ftyp":
		return ".m4a"
	default:
		return ""
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

	// 校验文件内容（魔数），避免仅凭扩展名放行非音频数据。
	detected := detectAudioExt(in.Data)
	if detected == "" {
		return nil, apperrors.NewBadRequest("无法识别的音频文件内容，仅支持 WAV/MP3/M4A")
	}
	if detected != ext {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("文件内容与扩展名不符：内容为 %s，扩展名为 %s", detected, ext))
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

	// Phase 3 模块 2：音频落库前 AES-256-GCM 加密（配置密钥时）。
	stored := in.Data
	if len(s.cfg.AudioEncryptionKey) > 0 {
		encrypted, err := EncryptAudio(in.Data, s.cfg.AudioEncryptionKey)
		if err != nil {
			return nil, apperrors.WrapInternal(err)
		}
		stored = encrypted
	}

	session := &model.AudioSession{
		ID:            utils.NewUUID(),
		UserID:        userID,
		DeviceID:      deviceID,
		AudioData:     stored,
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
	// 重试直接置为 processing，由 worker 认领后重新转写。
	if err := s.sessions.UpdateStatus(ctx, id, model.StatusProcessing, ""); err != nil {
		return err
	}
	return nil
}
