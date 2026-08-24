package service

import (
	"context"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// AccountExport 是用户数据导出的聚合结构。
// 敏感字段（password_hash / deleted_at / device_token_hash / audio_data）已由模型 JSON tag 排除。
type AccountExport struct {
	User       model.User           `json:"user"`
	Identities []model.Identity     `json:"identities"`
	Devices    []model.Device       `json:"devices"`
	Sessions   []model.AudioSession `json:"sessions"`
}

type AccountService struct {
	users      *repository.UserRepo
	identities *repository.IdentityRepo
	devices    *repository.DeviceRepo
	sessions   *repository.AudioSessionRepo
	tokens     *repository.RefreshTokenRepo
}

func NewAccountService(users *repository.UserRepo, identities *repository.IdentityRepo, devices *repository.DeviceRepo, sessions *repository.AudioSessionRepo, tokens *repository.RefreshTokenRepo) *AccountService {
	return &AccountService{users: users, identities: identities, devices: devices, sessions: sessions, tokens: tokens}
}

// Export 汇总用户全部个人数据（GDPR/个保法）。
func (s *AccountService) Export(ctx context.Context, userID string) (*AccountExport, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if user == nil {
		return nil, apperrors.NewNotFound("用户不存在")
	}

	identities, err := s.identities.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	devices, err := s.devices.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	sessions, err := s.sessions.ListAllByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}

	// 保证空切片序列化为 [] 而非 null。
	if identities == nil {
		identities = make([]model.Identity, 0)
	}
	if devices == nil {
		devices = make([]model.Device, 0)
	}
	if sessions == nil {
		sessions = make([]model.AudioSession, 0)
	}

	return &AccountExport{
		User:       *user,
		Identities: identities,
		Devices:    devices,
		Sessions:   sessions,
	}, nil
}

// Delete 注销账户：软删除用户并吊销全部 refresh token。
func (s *AccountService) Delete(ctx context.Context, userID string) error {
	if err := s.users.SoftDelete(ctx, userID); err != nil {
		return apperrors.WrapInternal(err)
	}
	if err := s.tokens.RevokeAllForUser(ctx, userID); err != nil {
		return apperrors.WrapInternal(err)
	}
	return nil
}
