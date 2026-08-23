package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
)

func TestGenerateFromAnalysisNilResult(t *testing.T) {
	svc, mock := newReminderService(t)

	got, err := svc.GenerateFromAnalysis(context.Background(), "u1", "s1", nil, nil)
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFromAnalysisSkipsEmptyText(t *testing.T) {
	svc, mock := newReminderService(t)
	i1 := "i1"
	result := &model.AnalysisResult{
		IdentityID: &i1,
		Extracted: model.ExtractedData{
			Todos:       []model.Todo{{Text: "   "}, {Text: ""}},
			Commitments: []model.Commitment{{Text: "", From: "我", To: "小王"}},
			Notes:       []string{},
		},
	}

	got, err := svc.GenerateFromAnalysis(context.Background(), "u1", "s1", result, nil)
	require.NoError(t, err)
	assert.Empty(t, got, "空文本的待办/承诺不应生成提醒")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFromAnalysisNoSwitchWhenIdentityNil(t *testing.T) {
	svc, mock := newReminderService(t)
	prev := "i1"
	result := &model.AnalysisResult{
		IdentityID: nil, // 低置信度未绑定身份
		Extracted: model.ExtractedData{
			Todos:       []model.Todo{{Text: "买菜"}},
			Commitments: []model.Commitment{},
			Notes:       []string{},
		},
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", nil, "todo", "买菜", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	got, err := svc.GenerateFromAnalysis(context.Background(), "u1", "s1", result, &prev)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, model.ReminderTypeTodo, got[0].Type)
	assert.Nil(t, got[0].IdentityID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFromAnalysisIdentitySwitchNoPendingTodos(t *testing.T) {
	svc, mock := newReminderService(t)
	prev := "i1"
	i2 := "i2"
	result := &model.AnalysisResult{
		IdentityID: &i2,
		Extracted: model.ExtractedData{
			Todos:       []model.Todo{},
			Commitments: []model.Commitment{},
			Notes:       []string{},
		},
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("u1", "i2", "todo", "pending").
		WillReturnRows(sqlmock.NewRows(reminderCols))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", "i2", "identity_switch", "身份已切换，该身份下暂无未完成待办。", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	got, err := svc.GenerateFromAnalysis(context.Background(), "u1", "s1", result, &prev)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, model.ReminderTypeIdentitySwitch, got[0].Type)
	assert.Contains(t, got[0].Content, "暂无未完成待办")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReminderStatusUpdateNotFound(t *testing.T) {
	t.Run("done", func(t *testing.T) {
		svc, mock := newReminderService(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
			WithArgs("r1", "u1").
			WillReturnRows(sqlmock.NewRows(reminderCols))
		assertAppCode(t, svc.MarkDone(context.Background(), "u1", "r1"), 40400)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("dismiss", func(t *testing.T) {
		svc, mock := newReminderService(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
			WithArgs("r1", "u1").
			WillReturnRows(sqlmock.NewRows(reminderCols))
		assertAppCode(t, svc.Dismiss(context.Background(), "u1", "r1"), 40400)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
