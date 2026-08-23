package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
)

func aiTestStrPtr(s string) *string        { return &s }
func aiTestTimePtr(t time.Time) *time.Time { return &t }

func TestParseDueAt(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  *time.Time
	}{
		{"nil 输入", nil, nil},
		{"空字符串", aiTestStrPtr(""), nil},
		{"纯空白", aiTestStrPtr("   "), nil},
		{"RFC3339", aiTestStrPtr("2024-06-01T10:00:00Z"), aiTestTimePtr(time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC))},
		{"日期时间", aiTestStrPtr("2024-06-01 10:00:00"), aiTestTimePtr(time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC))},
		{"仅日期", aiTestStrPtr("2024-06-01"), aiTestTimePtr(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))},
		{"非法时间", aiTestStrPtr("not-a-time"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDueAt(tt.input)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.True(t, got.Equal(*tt.want), "got %v, want %v", got, *tt.want)
		})
	}
}

// TestAnalyzeTranscriptWorkMeetingExtraction 验收：给定「工作会议」转写，识别身份并抽出格式正确的待办/承诺/笔记（含 due_at）。
func TestAnalyzeTranscriptWorkMeetingExtraction(t *testing.T) {
	svc := newFakeAIService(
		func() (string, error) { return "{\"identity_id\":\"i1\",\"confidence\":0.85}", nil },
		func() (string, error) {
			return "{\"todos\":[{\"text\":\"周五前交周报\",\"due_at\":\"2024-06-07T18:00:00Z\"}],\"commitments\":[{\"text\":\"帮市场部改方案\",\"from\":\"我\",\"to\":\"小王\",\"due_at\":\"2024-06-05\"}],\"notes\":[\"下次站会改到周一\"]}", nil
		},
	)

	res, err := svc.AnalyzeTranscript(context.Background(),
		"我们讨论一下 Q2 目标，我周五前交周报，另外我承诺帮小王在市场部方案上出一版。",
		[]model.Identity{{ID: "i1", Name: "工作"}, {ID: "i2", Name: "家庭"}})
	require.NoError(t, err)
	require.NotNil(t, res.IdentityID)
	assert.Equal(t, "i1", *res.IdentityID)
	assert.Equal(t, 0.85, res.Confidence)

	require.Len(t, res.Extracted.Todos, 1)
	require.NotNil(t, res.Extracted.Todos[0].DueAt, "待办 due_at 应被解析")

	require.Len(t, res.Extracted.Commitments, 1)
	assert.Equal(t, "我", res.Extracted.Commitments[0].From)
	assert.Equal(t, "小王", res.Extracted.Commitments[0].To)
	require.NotNil(t, res.Extracted.Commitments[0].DueAt, "承诺 due_at 应被解析")

	require.Len(t, res.Extracted.Notes, 1)
	assert.Equal(t, "下次站会改到周一", res.Extracted.Notes[0])
}

// TestAnalyzeTranscriptExtractionInvalidJSONThenRetry 验收：提取阶段返回非法 JSON 时重试一次。
func TestAnalyzeTranscriptExtractionInvalidJSONThenRetry(t *testing.T) {
	extractCalls := 0
	svc := newFakeAIService(
		func() (string, error) { return "{\"identity_id\":\"i1\",\"confidence\":0.9}", nil },
		func() (string, error) {
			extractCalls++
			if extractCalls == 1 {
				return "这不是合法 JSON", nil
			}
			return "{\"todos\":[{\"text\":\"买菜\",\"due_at\":\"2024-06-01T10:00:00Z\"}],\"commitments\":[],\"notes\":[]}", nil
		},
	)

	res, err := svc.AnalyzeTranscript(context.Background(), "开会说明天买菜", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	require.Len(t, res.Extracted.Todos, 1)
	assert.Equal(t, "买菜", res.Extracted.Todos[0].Text)
	require.NotNil(t, res.Extracted.Todos[0].DueAt)
	assert.Equal(t, 2, extractCalls, "提取阶段解析失败应重试一次")
}

// TestAnalyzeTranscriptExtractionDegradesOnError 验收：提取阶段模型失败时不崩溃、降级为空提取，身份识别不受影响。
func TestAnalyzeTranscriptExtractionDegradesOnError(t *testing.T) {
	svc := newFakeAIService(
		func() (string, error) { return "{\"identity_id\":\"i1\",\"confidence\":0.9}", nil },
		func() (string, error) { return "", errors.New("model down") },
	)

	res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err, "提取失败不应导致整体崩溃")
	require.NotNil(t, res.IdentityID, "身份识别成功仍应绑定身份")
	assert.Empty(t, res.Extracted.Todos)
	assert.Empty(t, res.Extracted.Commitments)
	assert.Empty(t, res.Extracted.Notes)
}

// TestAnalyzeTranscriptIdentityDegradesOnError 验收：身份识别失败时降级为未绑定（confidence=0），提取阶段不受影响。
func TestAnalyzeTranscriptIdentityDegradesOnError(t *testing.T) {
	svc := newFakeAIService(
		func() (string, error) { return "", errors.New("model down") },
		func() (string, error) {
			return "{\"todos\":[],\"commitments\":[],\"notes\":[\"会议定在周三\"]}", nil
		},
	)

	res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	assert.Nil(t, res.IdentityID, "身份识别失败应降级为未绑定")
	assert.Equal(t, 0.0, res.Confidence)
	require.Len(t, res.Extracted.Notes, 1, "提取阶段不受身份识别失败影响")
	assert.Equal(t, "会议定在周三", res.Extracted.Notes[0])
}

// TestRecognizeIdentityConfidenceClamped 验收：LLM 返回越界置信度时被截断到 [0,1]。
func TestRecognizeIdentityConfidenceClamped(t *testing.T) {
	t.Run("超过 1 截断为 1", func(t *testing.T) {
		svc := newFakeAIService(
			func() (string, error) { return "{\"identity_id\":\"i1\",\"confidence\":1.5}", nil },
			func() (string, error) { return "{\"todos\":[],\"commitments\":[],\"notes\":[]}", nil },
		)
		res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
		require.NoError(t, err)
		assert.Equal(t, 1.0, res.Confidence)
		require.NotNil(t, res.IdentityID)
	})

	t.Run("小于 0 截断为 0", func(t *testing.T) {
		svc := newFakeAIService(
			func() (string, error) { return "{\"identity_id\":\"i1\",\"confidence\":-0.5}", nil },
			func() (string, error) { return "{\"todos\":[],\"commitments\":[],\"notes\":[]}", nil },
		)
		res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
		require.NoError(t, err)
		assert.Equal(t, 0.0, res.Confidence)
		assert.Nil(t, res.IdentityID, "置信度 0 低于阈值不绑定")
	})
}
