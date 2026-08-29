package service

import (
	"context"
	"strings"

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
	// secretKey 用于加密用户自带的模型 API Key（ASR/AI）。空表示明文存储。
	secretKey []byte
}

func NewAccountService(users *repository.UserRepo, identities *repository.IdentityRepo, devices *repository.DeviceRepo, sessions *repository.AudioSessionRepo, tokens *repository.RefreshTokenRepo, secretKey []byte) *AccountService {
	return &AccountService{users: users, identities: identities, devices: devices, sessions: sessions, tokens: tokens, secretKey: secretKey}
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

// ===== 用户级模型配置（ASR / AI） =====

// AsrSettingsView 是 ASR 配置的回显结构：API Key 不回传明文，仅给掩码。
type AsrSettingsView struct {
	BaseURL      string `json:"base_url"`
	Model        string `json:"model"`
	Language     string `json:"language"`
	EnableITN    bool   `json:"enable_itn"`
	HasAPIKey    bool   `json:"has_api_key"`
	APIKeyMasked string `json:"api_key_masked"`
}

// AiSettingsView 是 AI 分析配置的回显结构。
type AiSettingsView struct {
	BaseURL             string  `json:"base_url"`
	Model               string  `json:"model"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	HasAPIKey           bool    `json:"has_api_key"`
	APIKeyMasked        string  `json:"api_key_masked"`
}

// AsrSettingsInput 是 ASR 配置的更新入参；指针字段 nil 表示「不改变」，
// api_key 传空字符串表示「清除密钥」。
type AsrSettingsInput struct {
	APIKey    *string `json:"api_key,omitempty"`
	BaseURL   *string `json:"base_url,omitempty"`
	Model     *string `json:"model,omitempty"`
	Language  *string `json:"language,omitempty"`
	EnableITN *bool   `json:"enable_itn,omitempty"`
}

// AiSettingsInput 是 AI 配置的更新入参。
type AiSettingsInput struct {
	APIKey              *string  `json:"api_key,omitempty"`
	BaseURL             *string  `json:"base_url,omitempty"`
	Model               *string  `json:"model,omitempty"`
	ConfidenceThreshold *float64 `json:"confidence_threshold,omitempty"`
}

// loadSettings 读取用户 settings JSON（用户不存在时由上层 GetByID 保证存在）。
func (s *AccountService) loadSettings(ctx context.Context, userID string) (*userSettings, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if u == nil {
		return nil, apperrors.NewNotFound("用户不存在")
	}
	return parseUserSettings(u.Settings), nil
}

func (s *AccountService) saveSettings(ctx context.Context, userID string, st *userSettings) error {
	raw, err := serializeUserSettings(st)
	if err != nil {
		return apperrors.WrapInternal(err)
	}
	if err := s.users.UpdateSettings(ctx, userID, raw); err != nil {
		return apperrors.WrapInternal(err)
	}
	return nil
}

// GetAsrSettings 返回脱敏后的 ASR 配置。
func (s *AccountService) GetAsrSettings(ctx context.Context, userID string) (*AsrSettingsView, error) {
	st, err := s.loadSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	asr := st.ASR
	view := &AsrSettingsView{EnableITN: true}
	if asr == nil {
		return view, nil
	}
	view.BaseURL = asr.BaseURL
	view.Model = asr.Model
	view.Language = asr.Language
	if asr.EnableITN != nil {
		view.EnableITN = *asr.EnableITN
	}
	if key := decryptKey(asr.APIKey, s.secretKey); key != "" {
		view.HasAPIKey = true
		view.APIKeyMasked = maskSecret(key)
	}
	return view, nil
}

// UpdateAsrSettings 应用部分更新并落库，返回更新后的脱敏配置。
func (s *AccountService) UpdateAsrSettings(ctx context.Context, userID string, in *AsrSettingsInput) (*AsrSettingsView, error) {
	st, err := s.loadSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	asr := st.ASR
	if asr == nil {
		asr = &AsrSettings{}
	}

	if in.APIKey != nil {
		enc, err := EncryptSecret(strings.TrimSpace(*in.APIKey), s.secretKey)
		if err != nil {
			return nil, apperrors.WrapInternal(err)
		}
		asr.APIKey = enc
	}
	if in.BaseURL != nil {
		asr.BaseURL = strings.TrimSpace(*in.BaseURL)
	}
	if in.Model != nil {
		asr.Model = strings.TrimSpace(*in.Model)
	}
	if in.Language != nil {
		asr.Language = strings.TrimSpace(*in.Language)
	}
	if in.EnableITN != nil {
		asr.EnableITN = in.EnableITN
	}

	st.ASR = asr
	if err := s.saveSettings(ctx, userID, st); err != nil {
		return nil, err
	}
	return s.GetAsrSettings(ctx, userID)
}

// GetAiSettings 返回脱敏后的 AI 配置。
func (s *AccountService) GetAiSettings(ctx context.Context, userID string) (*AiSettingsView, error) {
	st, err := s.loadSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	ai := st.AI
	view := &AiSettingsView{ConfidenceThreshold: defaultAIConfidenceThreshold}
	if ai == nil {
		return view, nil
	}
	view.BaseURL = ai.BaseURL
	view.Model = ai.Model
	if ai.ConfidenceThreshold != nil {
		view.ConfidenceThreshold = *ai.ConfidenceThreshold
	}
	if key := decryptKey(ai.APIKey, s.secretKey); key != "" {
		view.HasAPIKey = true
		view.APIKeyMasked = maskSecret(key)
	}
	return view, nil
}

// UpdateAiSettings 应用部分更新并落库，返回更新后的脱敏配置。
func (s *AccountService) UpdateAiSettings(ctx context.Context, userID string, in *AiSettingsInput) (*AiSettingsView, error) {
	st, err := s.loadSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	ai := st.AI
	if ai == nil {
		ai = &AiSettings{}
	}

	if in.APIKey != nil {
		enc, err := EncryptSecret(strings.TrimSpace(*in.APIKey), s.secretKey)
		if err != nil {
			return nil, apperrors.WrapInternal(err)
		}
		ai.APIKey = enc
	}
	if in.BaseURL != nil {
		ai.BaseURL = strings.TrimSpace(*in.BaseURL)
	}
	if in.Model != nil {
		ai.Model = strings.TrimSpace(*in.Model)
	}
	if in.ConfidenceThreshold != nil {
		ai.ConfidenceThreshold = in.ConfidenceThreshold
	}

	st.AI = ai
	if err := s.saveSettings(ctx, userID, st); err != nil {
		return nil, err
	}
	return s.GetAiSettings(ctx, userID)
}

// decryptKey 解密存储的 API Key，供脱敏回显与运行时覆盖使用。
func decryptKey(enc string, key []byte) string {
	plain, err := DecryptSecret(enc, key)
	if err != nil {
		return ""
	}
	return plain
}

// SttOverridesForUser 返回用户 ASR 运行时覆盖项（API Key 已解密）。
// 用户未配置任何 ASR 项时返回 nil，由调用方回退全局默认。
func (s *AccountService) SttOverridesForUser(ctx context.Context, userID string) (*SttOverrides, error) {
	st, err := s.loadSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	asr := st.ASR
	if asr == nil {
		return nil, nil
	}
	return &SttOverrides{
		APIKey:    decryptKey(asr.APIKey, s.secretKey),
		BaseURL:   asr.BaseURL,
		Model:     asr.Model,
		Language:  asr.Language,
		EnableITN: asr.EnableITN,
	}, nil
}

// AiOverridesForUser 返回用户 AI 分析运行时覆盖项（API Key 已解密）。
// 用户未配置任何 AI 项时返回 nil，由调用方回退全局默认。
func (s *AccountService) AiOverridesForUser(ctx context.Context, userID string) (*AiOverrides, error) {
	st, err := s.loadSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	ai := st.AI
	if ai == nil {
		return nil, nil
	}
	o := &AiOverrides{
		APIKey:  decryptKey(ai.APIKey, s.secretKey),
		BaseURL: ai.BaseURL,
		Model:   ai.Model,
	}
	if ai.ConfidenceThreshold != nil {
		o.Threshold = *ai.ConfidenceThreshold
	}
	return o, nil
}
