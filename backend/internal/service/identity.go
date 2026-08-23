package service

import (
	"context"
	"strings"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

type IdentityService struct {
	repo *repository.IdentityRepo
}

func NewIdentityService(repo *repository.IdentityRepo) *IdentityService {
	return &IdentityService{repo: repo}
}

// IdentityInput 身份创建/更新入参。
type IdentityInput struct {
	Name        string
	Description *string
	Color       string
	Icon        string
	IsDefault   bool
}

func (s *IdentityService) List(ctx context.Context, userID string) ([]model.Identity, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return list, nil
}

func (s *IdentityService) Create(ctx context.Context, userID string, in IdentityInput) (*model.Identity, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperrors.NewBadRequest("身份名称不能为空")
	}
	color := in.Color
	if color == "" {
		color = "#000000"
	}
	icon := in.Icon
	if icon == "" {
		icon = "person"
	}

	identity := &model.Identity{
		ID:          utils.NewUUID(),
		UserID:      userID,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Color:       color,
		Icon:        icon,
		IsDefault:   false,
	}
	if err := s.repo.Create(ctx, identity); err != nil {
		return nil, apperrors.WrapInternal(err)
	}

	count, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if in.IsDefault || count == 1 {
		if err := s.repo.SetDefault(ctx, userID, identity.ID); err != nil {
			return nil, apperrors.WrapInternal(err)
		}
	}
	return s.repo.GetByID(ctx, identity.ID, userID)
}

func (s *IdentityService) Update(ctx context.Context, userID, id string, in IdentityInput) (*model.Identity, error) {
	existing, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if existing == nil {
		return nil, apperrors.NewNotFound("身份不存在")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperrors.NewBadRequest("身份名称不能为空")
	}

	existing.Name = strings.TrimSpace(in.Name)
	if in.Description != nil {
		existing.Description = in.Description
	}
	if in.Color != "" {
		existing.Color = in.Color
	}
	if in.Icon != "" {
		existing.Icon = in.Icon
	}
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	if in.IsDefault {
		if err := s.repo.SetDefault(ctx, userID, id); err != nil {
			return nil, apperrors.WrapInternal(err)
		}
	}
	return s.repo.GetByID(ctx, id, userID)
}

func (s *IdentityService) Delete(ctx context.Context, userID, id string) error {
	existing, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return apperrors.WrapInternal(err)
	}
	if existing == nil {
		return apperrors.NewNotFound("身份不存在")
	}
	if existing.IsDefault {
		return apperrors.NewBadRequest("默认身份不可删除，请先将其他身份设为默认")
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return apperrors.WrapInternal(err)
	}
	return nil
}

func (s *IdentityService) SetDefault(ctx context.Context, userID, id string) error {
	existing, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return apperrors.WrapInternal(err)
	}
	if existing == nil {
		return apperrors.NewNotFound("身份不存在")
	}
	if err := s.repo.SetDefault(ctx, userID, id); err != nil {
		return apperrors.WrapInternal(err)
	}
	return nil
}
