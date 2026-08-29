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

// Bind 用户在 App 内用 device_id + 一次性绑定码绑定设备，返回设备与设备级 token（只返回一次）。
func (s *DeviceService) Bind(ctx context.Context, userID, deviceID, name, bindCode string) (*model.Device, string, error) {
	return s.bindWithCode(ctx, userID, deviceID, name, bindCode, false)
}

// Claim 设备自助配对：设备拿自己的 device_id + 用户在 App 上生成的一次性绑定码换取 device_token，
// 换到后写入 NVS。用户身份从绑定码反解，因此设备侧不需要任何预置凭据，
// 免去了把 token 手抄进 TF 卡这一步。
//
// 安全边界：本方法对应免鉴权端点，唯一屏障是 6 位绑定码 + 10 分钟有效期，
// 且按产品决策**不做尝试次数限制**——穷举绑定码即可拿到对应账号的设备凭据。
// 后续若要补限流，在入口处按 bindCode / 来源 IP 计数即可，下面的逻辑无需改动。
func (s *DeviceService) Claim(ctx context.Context, deviceID, name, bindCode string) (*model.Device, string, error) {
	return s.bindWithCode(ctx, "", deviceID, name, bindCode, true)
}

// bindWithCode 是 Bind / Claim 的共用实现。
// expectUserID 非空时要求绑定码属于该用户（用户态绑定）；为空时用户身份由绑定码反解（设备自助配对）。
// allowRepair 为 true 时，同一用户重复配对同一设备会轮换 token 而非报错，
// 用于设备恢复出厂、NVS 被擦除后重新配对。
func (s *DeviceService) bindWithCode(ctx context.Context, expectUserID, deviceID, name, bindCode string, allowRepair bool) (*model.Device, string, error) {
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
	// 统一用同一句错误信息，避免泄漏"码存在但属于别人"这类信息。
	if code == nil || code.UsedAt != nil || code.ExpiresAt.Before(now) {
		return nil, "", apperrors.NewBadRequest("绑定码无效或已过期")
	}
	if expectUserID != "" && code.UserID != expectUserID {
		return nil, "", apperrors.NewBadRequest("绑定码无效或已过期")
	}
	userID := code.UserID

	token, err := utils.RandomToken(32)
	if err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}
	tokenHash := utils.SHA256Hex(token)

	existing, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, "", apperrors.WrapInternal(err)
	}
	if existing != nil {
		// 设备已存在时，只有"同一用户 + 允许重新配对"才轮换 token。
		// 否则一律拒绝，防止拿自己的绑定码劫持他人已绑定的设备。
		if !allowRepair || existing.UserID != userID {
			return nil, "", apperrors.NewBadRequest("该设备已被绑定")
		}
		if err := s.repo.MarkBindCodeUsed(ctx, bindCode); err != nil {
			return nil, "", apperrors.WrapInternal(err)
		}
		if err := s.repo.RotateToken(ctx, existing.ID, tokenHash); err != nil {
			return nil, "", apperrors.WrapInternal(err)
		}
		refreshed, err := s.repo.GetByIDOnly(ctx, existing.ID)
		if err != nil {
			return nil, "", apperrors.WrapInternal(err)
		}
		return refreshed, token, nil
	}

	if name == "" {
		name = defaultDeviceName
	}

	device := &model.Device{
		ID:              utils.NewUUID(),
		UserID:          userID,
		DeviceID:        deviceID,
		Name:            name,
		DeviceType:      defaultDeviceType,
		IsActive:        true,
		DeviceTokenHash: tokenHash,
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
