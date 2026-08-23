package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/utils"
)

type ReminderService struct {
	repo *repository.ReminderRepo
}

func NewReminderService(repo *repository.ReminderRepo) *ReminderService {
	return &ReminderService{repo: repo}
}

// ListPending 返回用户的待处理提醒列表。
func (s *ReminderService) ListPending(ctx context.Context, userID string) ([]model.Reminder, error) {
	list, err := s.repo.ListByUser(ctx, userID, model.ReminderStatusPending)
	if err != nil {
		return nil, apperrors.WrapInternal(err)
	}
	return list, nil
}

// MarkDone 将提醒标记为完成。
func (s *ReminderService) MarkDone(ctx context.Context, userID, id string) error {
	return s.updateStatus(ctx, userID, id, model.ReminderStatusDone)
}

// Dismiss 忽略提醒。
func (s *ReminderService) Dismiss(ctx context.Context, userID, id string) error {
	return s.updateStatus(ctx, userID, id, model.ReminderStatusDismissed)
}

func (s *ReminderService) updateStatus(ctx context.Context, userID, id, status string) error {
	m, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return apperrors.WrapInternal(err)
	}
	if m == nil {
		return apperrors.NewNotFound("提醒不存在")
	}
	if m.Status != model.ReminderStatusPending {
		return apperrors.NewBadRequest("只有待处理提醒才能更新状态")
	}
	if err := s.repo.UpdateStatus(ctx, id, userID, status); err != nil {
		return apperrors.WrapInternal(err)
	}
	return nil
}

// GenerateFromAnalysis 依据 AI 分析结果生成提醒。
// 每个 todo / commitment 生成一条提醒；身份相对上一条变化时生成 identity_switch 提醒。
func (s *ReminderService) GenerateFromAnalysis(ctx context.Context, userID, sessionID string, result *model.AnalysisResult, previousIdentityID *string) ([]model.Reminder, error) {
	if result == nil {
		return nil, nil
	}

	out := make([]model.Reminder, 0)
	var identityID *string
	if result.IdentityID != nil && *result.IdentityID != "" {
		identityID = result.IdentityID
	}

	for _, t := range result.Extracted.Todos {
		content := strings.TrimSpace(t.Text)
		if content == "" {
			continue
		}
		m := &model.Reminder{
			ID:         utils.NewUUID(),
			UserID:     userID,
			SessionID:  &sessionID,
			IdentityID: identityID,
			Type:       model.ReminderTypeTodo,
			Content:    content,
			DueAt:      t.DueAt,
			Status:     model.ReminderStatusPending,
		}
		if err := s.repo.Create(ctx, m); err != nil {
			return out, apperrors.WrapInternal(err)
		}
		out = append(out, *m)
	}

	for _, c := range result.Extracted.Commitments {
		content := commitmentContent(c)
		if content == "" {
			continue
		}
		m := &model.Reminder{
			ID:         utils.NewUUID(),
			UserID:     userID,
			SessionID:  &sessionID,
			IdentityID: identityID,
			Type:       model.ReminderTypeCommitment,
			Content:    content,
			DueAt:      c.DueAt,
			Status:     model.ReminderStatusPending,
		}
		if err := s.repo.Create(ctx, m); err != nil {
			return out, apperrors.WrapInternal(err)
		}
		out = append(out, *m)
	}

	// 身份切换提醒：新的身份识别结果与上一条不同。
	if identityID != nil && previousIdentityID != nil && *previousIdentityID != *identityID {
		todos, err := s.repo.ListPendingTodosByIdentity(ctx, userID, *identityID)
		if err != nil {
			return out, apperrors.WrapInternal(err)
		}
		m := &model.Reminder{
			ID:         utils.NewUUID(),
			UserID:     userID,
			SessionID:  &sessionID,
			IdentityID: identityID,
			Type:       model.ReminderTypeIdentitySwitch,
			Content:    identitySwitchContent(todos),
			Status:     model.ReminderStatusPending,
		}
		if err := s.repo.Create(ctx, m); err != nil {
			return out, apperrors.WrapInternal(err)
		}
		out = append(out, *m)
	}

	return out, nil
}

func commitmentContent(c model.Commitment) string {
	text := strings.TrimSpace(c.Text)
	from := strings.TrimSpace(c.From)
	to := strings.TrimSpace(c.To)
	if text == "" {
		return ""
	}
	switch {
	case from != "" && to != "":
		return fmt.Sprintf("%s → %s：%s", from, to, text)
	case from != "":
		return fmt.Sprintf("%s：%s", from, text)
	default:
		return text
	}
}

func identitySwitchContent(todos []model.Reminder) string {
	if len(todos) == 0 {
		return "身份已切换，该身份下暂无未完成待办。"
	}
	items := make([]string, 0, len(todos))
	for _, t := range todos {
		items = append(items, t.Content)
	}
	return "身份切换，上次未完成事项：" + strings.Join(items, "；")
}
