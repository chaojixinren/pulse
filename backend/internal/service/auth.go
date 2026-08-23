package service

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

const (
	accessTokenTTL  = 1 * time.Hour
	refreshTokenTTL = 7 * 24 * time.Hour
)

type AuthService struct {
	cfg    *config.Config
	users  *repository.UserRepo
	tokens *repository.RefreshTokenRepo
}

func NewAuthService(cfg *config.Config, users *repository.UserRepo, tokens *repository.RefreshTokenRepo) *AuthService {
	return &AuthService{cfg: cfg, users: users, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (*model.User, error) {
	existing, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if existing != nil {
		return nil, apperrors.NewBadRequest("该邮箱已注册")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}

	user := &model.User{
		ID:           utils.NewUUID(),
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Settings:     "{}",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*model.TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if user == nil {
		return nil, apperrors.NewUnauthorized("邮箱或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, apperrors.NewUnauthorized("邮箱或密码错误")
	}
	return s.issueTokens(ctx, user.ID)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	hash := utils.SHA256Hex(refreshToken)
	record, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if record == nil || record.RevokedAt != nil || record.ExpiresAt.Before(time.Now().UTC()) {
		return nil, apperrors.NewUnauthorized("刷新令牌无效或已过期")
	}
	if err := s.tokens.Revoke(ctx, hash); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return s.issueTokens(ctx, record.UserID)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.tokens.Revoke(ctx, utils.SHA256Hex(refreshToken)); err != nil {
		return apperrors.WrapInternal(err)
	}
	return nil
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if user == nil {
		return nil, apperrors.NewNotFound("用户不存在")
	}
	return user, nil
}

func (s *AuthService) issueTokens(ctx context.Context, userID string) (*model.TokenPair, error) {
	access, err := utils.GenerateAccessToken(userID, s.cfg.JWTSecret, accessTokenTTL)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	refresh, err := utils.RandomToken(32)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	record := &model.RefreshToken{
		ID:        utils.NewUUID(),
		UserID:    userID,
		TokenHash: utils.SHA256Hex(refresh),
		ExpiresAt: time.Now().UTC().Add(refreshTokenTTL),
	}
	if err := s.tokens.Create(ctx, record); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return &model.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
