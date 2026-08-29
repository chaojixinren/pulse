package service

import (
	"context"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

const (
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

// commandTTL 是指令的最长存活时间。start/stop_recording 是即时语义，
// 设备离线太久之后再补发已经没有意义，过期比迟到更安全。
const commandTTL = 10 * time.Minute

// validAckStatus 是设备回执允许的终态。
var validAckStatus = map[string]bool{
	"done":   true,
	"failed": true,
}

// AuthenticateDevice 按设备 token 反查设备，实现 middleware.DeviceAuthenticator。
// 设备 token 不设过期：嵌入式设备没有交互式续期的条件，
// 凭据吊销靠用户解绑（DELETE devices）或重新配对轮换 token。
func (s *DeviceService) AuthenticateDevice(ctx context.Context, token string) (*model.Device, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, apperrors.NewUnauthorized("设备凭据无效")
	}
	d, err := s.repo.GetByTokenHash(ctx, utils.SHA256Hex(token))
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if d == nil {
		return nil, apperrors.NewUnauthorized("设备凭据无效或设备已解绑")
	}
	return d, nil
}

// DeviceHeartbeat 设备自报心跳，并在同一次响应里捎带待执行指令。
//
// 控制面走心跳返回而不是独立轮询端点：设备是间歇联网的，
// 每多一次请求就多一次射频唤醒，直接影响续航。
func (s *DeviceService) DeviceHeartbeat(ctx context.Context, deviceUUID string, firmware *string, battery *int) (*model.Device, []model.DeviceCommand, error) {
	if battery != nil && (*battery < 0 || *battery > 100) {
		return nil, nil, apperrors.NewBadRequest("电量应在 0-100 之间")
	}
	if err := s.repo.UpdateHeartbeatByID(ctx, deviceUUID, firmware, battery); err != nil {
		return nil, nil, apperrors.WrapInternal(err)
	}
	// 先淘汰过期指令，再取待办，避免陈年指令在每次心跳里被反复投递。
	if err := s.repo.ExpireStaleCommands(ctx, deviceUUID, time.Now().UTC().Add(-commandTTL)); err != nil {
		return nil, nil, apperrors.WrapInternal(err)
	}
	cmds, err := s.repo.ListPendingCommands(ctx, deviceUUID)
	if err != nil {
		return nil, nil, apperrors.WrapInternal(err)
	}
	device, err := s.repo.GetByIDOnly(ctx, deviceUUID)
	if err != nil {
		return nil, nil, apperrors.WrapInternal(err)
	}
	if device == nil {
		return nil, nil, apperrors.NewNotFound("设备不存在")
	}
	return device, cmds, nil
}

// AckCommand 设备回执指令执行结果。
// 指令在收到回执前一直是 pending，因此会在后续心跳里重复下发；
// start/stop_recording 本身幂等，重复执行无副作用，这样换取的是丢包时不丢指令。
func (s *DeviceService) AckCommand(ctx context.Context, deviceUUID, commandID, status string) error {
	status = strings.TrimSpace(status)
	if !validAckStatus[status] {
		return apperrors.NewBadRequest("回执状态无效，仅支持 done / failed")
	}
	ok, err := s.repo.AckCommand(ctx, commandID, deviceUUID, status)
	if err != nil {
		return apperrors.WrapInternal(err)
	}
	if !ok {
		return apperrors.NewNotFound("指令不存在或已处理")
	}
	return nil
}

// ========== Device 创建/绑定（App 领养，一次性下发 token） ==========

// CreateDevice App 用户创建设备绑定并生成设备令牌。
//
// 硬件回退为「纯上传 + token 手抄」后，配对退化为：用户在 App 填 device_id
// 创建设备，后端一次性返回明文 device_token，用户抄录进硬件 config.json。
// 同一 device_id 重复创建视为重新配对：轮换 token，旧 token 立即失效。
func (s *DeviceService) CreateDevice(ctx context.Context, userID, deviceID, deviceName string) (*model.Device, string, error) {
	userID = strings.TrimSpace(userID)
	deviceID = strings.TrimSpace(deviceID)
	deviceName = strings.TrimSpace(deviceName)
	if userID == "" || deviceID == "" {
		return nil, "", apperrors.NewBadRequest("user_id 和 device_id 不能为空")
	}

	existing, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}

	var device *model.Device
	var deviceToken string

	if existing != nil {
		// 设备已存在，只允许原用户重新配对（轮换 token，旧 token 立即失效）。
		if existing.UserID != userID {
			return nil, "", apperrors.NewBadRequest("该设备已被其他用户绑定")
		}
		token, err := utils.RandomToken(32)
		if err != nil {
			return nil, "", apperrors.WrapInternal(err)
		}
		if err := s.repo.RotateToken(ctx, existing.ID, utils.SHA256Hex(token)); err != nil {
			return nil, "", apperrors.WrapInternal(err)
		}
		device = existing
		deviceToken = token
	} else {
		// 新设备，创建绑定。
		token, err := utils.RandomToken(32)
		if err != nil {
			return nil, "", apperrors.WrapInternal(err)
		}
		if deviceName == "" {
			deviceName = defaultDeviceName
		}
		device = &model.Device{
			ID:              utils.NewUUID(),
			UserID:          userID,
			DeviceID:        deviceID,
			Name:            deviceName,
			DeviceType:      defaultDeviceType,
			IsActive:        true,
			DeviceTokenHash: utils.SHA256Hex(token),
		}
		if err := s.repo.Create(ctx, device); err != nil {
			return nil, "", apperrors.WrapInternal(err)
		}
		deviceToken = token
	}

	return device, deviceToken, nil
}
