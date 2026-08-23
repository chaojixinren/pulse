package service

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

const (
	bindCodeTTL       = 10 * time.Minute
	defaultDeviceType = "wearable"
	defaultDeviceName = "我的设备"
)

// validCommands 是允许下发的指令集合。
var validCommands = map[string]bool{
	"start_recording": true,
	"stop_recording":  true,
}

type DeviceService struct {
	repo *repository.DeviceRepo
}

func NewDeviceService(repo *repository.DeviceRepo) *DeviceService {
	return &DeviceService{repo: repo}
}

// GenerateBindCode 生成一次性绑定码（10 分钟内有效），供前端展示、硬件首次连接时使用。
func (s *DeviceService) GenerateBindCode(ctx context.Context, userID string) (*model.DeviceBindCode, error) {
	code, err := randomBindCode(6)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	c := &model.DeviceBindCode{
		ID:        utils.NewUUID(),
		UserID:    userID,
		Code:      code,
		ExpiresAt: time.Now().UTC().Add(bindCodeTTL),
	}
	if err := s.repo.CreateBindCode(ctx, c); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return c, nil
}

// Bind 用户用 device_id + 一次性绑定码绑定设备，返回设备与设备级 token（只返回一次）。
func (s *DeviceService) Bind(ctx context.Context, userID, deviceID, name, bindCode string) (*model.Device, string, error) {
	deviceID = strings.TrimSpace(deviceID)
	bindCode = strings.TrimSpace(bindCode)
	name = strings.TrimSpace(name)

	if deviceID == "" {
		return nil, "", apperrors.NewBadRequest("设备标识不能为空")
	}
	if bindCode == "" {
		return nil, "", apperrors.NewBadRequest("绑定码不能为空")
	}

	code, err := s.repo.GetBindCodeByCode(ctx, bindCode)
	if err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}
	now := time.Now().UTC()
	if code == nil || code.UsedAt != nil || code.ExpiresAt.Before(now) || code.UserID != userID {
		return nil, "", apperrors.NewBadRequest("绑定码无效或已过期")
	}

	existing, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}
	if existing != nil {
		return nil, "", apperrors.NewBadRequest("该设备已被绑定")
	}

	if name == "" {
		name = defaultDeviceName
	}

	token, err := utils.RandomToken(32)
	if err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}

	device := &model.Device{
		ID:              utils.NewUUID(),
		UserID:          userID,
		DeviceID:        deviceID,
		Name:            name,
		DeviceType:      defaultDeviceType,
		IsActive:        true,
		DeviceTokenHash: utils.SHA256Hex(token),
	}

	// 先消费绑定码，再创建设备，保证一次性。
	if err := s.repo.MarkBindCodeUsed(ctx, bindCode); err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}
	if err := s.repo.Create(ctx, device); err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}
	return device, token, nil
}

func (s *DeviceService) List(ctx context.Context, userID string) ([]model.Device, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return list, nil
}

func (s *DeviceService) Get(ctx context.Context, userID, id string) (*model.Device, error) {
	d, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if d == nil {
		return nil, apperrors.NewNotFound("设备不存在")
	}
	return d, nil
}

func (s *DeviceService) Unbind(ctx context.Context, userID, id string) error {
	d, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return apperrors.WrapInternal(err)
	}
	if d == nil {
		return apperrors.NewNotFound("设备不存在")
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return apperrors.WrapInternal(err)
	}
	return nil
}

// Heartbeat 硬件上报心跳，更新 last_seen_at、电量与固件版本。
func (s *DeviceService) Heartbeat(ctx context.Context, userID, id string, firmware *string, battery *int) (*model.Device, error) {
	if battery != nil && (*battery < 0 || *battery > 100) {
		return nil, apperrors.NewBadRequest("电量应在 0-100 之间")
	}
	d, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if d == nil {
		return nil, apperrors.NewNotFound("设备不存在")
	}
	if err := s.repo.UpdateHeartbeat(ctx, id, userID, firmware, battery); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return s.repo.GetByID(ctx, id, userID)
}

// IssueCommand 下发指令（Phase 2 先落库，硬件按需拉取）。
func (s *DeviceService) IssueCommand(ctx context.Context, userID, id, command string) (*model.DeviceCommand, error) {
	command = strings.TrimSpace(command)
	if !validCommands[command] {
		return nil, apperrors.NewBadRequest("不支持的指令，仅支持 start_recording / stop_recording")
	}
	d, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if d == nil {
		return nil, apperrors.NewNotFound("设备不存在")
	}
	cmd := &model.DeviceCommand{
		ID:       utils.NewUUID(),
		DeviceID: d.ID,
		UserID:   userID,
		Command:  command,
		Status:   "pending",
	}
	if err := s.repo.CreateCommand(ctx, cmd); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return cmd, nil
}

// randomBindCode 生成 n 位数字绑定码。
func randomBindCode(n int) (string, error) {
	if n <= 0 {
		n = 6
	}
	const digits = "0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = digits[int(b)%len(digits)]
	}
	return string(out), nil
}
