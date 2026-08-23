package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
)

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "纯 JSON",
			input: "{\"key\": \"value\"}",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "带 markdown 代码块",
			input: "\x60\x60\x60json\n{\"key\": \"value\"}\n\x60\x60\x60",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "带前后文本",
			input: "这是结果：{\"key\": \"value\"} 完成",
			want:  "{\"key\": \"value\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeFences(tt.input)

			// 比较解析后的 JSON 以忽略空格差异
			var gotObj, wantObj interface{}
			if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
				t.Fatalf("无法解析返回的 JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantObj); err != nil {
				t.Fatalf("无法解析期望的 JSON: %v", err)
			}

			gotJSON, _ := json.Marshal(gotObj)
			wantJSON, _ := json.Marshal(wantObj)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("stripCodeFences() = %v, want %v", string(gotJSON), string(wantJSON))
			}
		})
	}
}

func TestBuildAnalysisResult(t *testing.T) {
	extracted := model.ExtractedData{
		Todos:       []model.Todo{{Text: "买菜"}},
		Commitments: []model.Commitment{},
		Notes:       []string{"会议定在周三"},
	}

	t.Run("高置信度绑定身份", func(t *testing.T) {
		r := buildAnalysisResult("i1", 0.9, 0.6, extracted)
		require.NotNil(t, r.IdentityID)
		assert.Equal(t, "i1", *r.IdentityID)
		assert.Equal(t, 0.9, r.Confidence)
		assert.Equal(t, extracted.Todos, r.Extracted.Todos)
		assert.Equal(t, extracted.Notes, r.Notes)
	})

	t.Run("低于阈值不绑定身份", func(t *testing.T) {
		r := buildAnalysisResult("i1", 0.5, 0.6, extracted)
		assert.Nil(t, r.IdentityID)
		assert.Equal(t, 0.5, r.Confidence)
	})

	t.Run("空身份不绑定", func(t *testing.T) {
		r := buildAnalysisResult("", 0.9, 0.6, extracted)
		assert.Nil(t, r.IdentityID)
	})

	t.Run("等于阈值绑定身份", func(t *testing.T) {
		r := buildAnalysisResult("i1", 0.6, 0.6, extracted)
		require.NotNil(t, r.IdentityID)
		assert.Equal(t, "i1", *r.IdentityID)
	})
}

func TestIdentityInList(t *testing.T) {
	ids := []model.Identity{{ID: "i1", Name: "工作"}, {ID: "i2", Name: "家庭"}}
	assert.True(t, identityInList("i1", ids))
	assert.True(t, identityInList("i2", ids))
	assert.False(t, identityInList("i3", ids))
	assert.False(t, identityInList("", ids))
	assert.False(t, identityInList("i1", nil))
}

func TestFormatIdentities(t *testing.T) {
	assert.Contains(t, formatIdentities(nil), "无候选身份")
	s := formatIdentities([]model.Identity{{ID: "i1", Name: "工作"}, {ID: "i2", Name: "家庭"}})
	assert.Contains(t, s, "i1: 工作")
	assert.Contains(t, s, "i2: 家庭")
}

func TestAnalyzeTranscriptHappyPath(t *testing.T) {
	svc := newFakeAIService(
		func() (string, error) { return "{\"identity_id\":\"i1\",\"confidence\":0.9}", nil },
		func() (string, error) {
			return "{\"todos\":[{\"text\":\"买菜\"}],\"commitments\":[{\"text\":\"帮小王改方案\",\"from\":\"我\",\"to\":\"小王\"}],\"notes\":[\"会议定在周三\"]}", nil
		},
	)
	res, err := svc.AnalyzeTranscript(context.Background(), "开会说下周一交报告", []model.Identity{{ID: "i1", Name: "工作"}, {ID: "i2", Name: "家庭"}})
	require.NoError(t, err)
	require.NotNil(t, res.IdentityID)
	assert.Equal(t, "i1", *res.IdentityID)
	assert.Equal(t, 0.9, res.Confidence)
	require.Len(t, res.Extracted.Todos, 1)
	assert.Equal(t, "买菜", res.Extracted.Todos[0].Text)
	require.Len(t, res.Extracted.Commitments, 1)
	assert.Equal(t, "我", res.Extracted.Commitments[0].From)
	require.Len(t, res.Extracted.Notes, 1)
}

func TestAnalyzeTranscriptLowConfidence(t *testing.T) {
	svc := newFakeAIService(
		func() (string, error) { return "{\"identity_id\":\"i1\",\"confidence\":0.5}", nil },
		func() (string, error) { return "{\"todos\":[],\"commitments\":[],\"notes\":[]}", nil },
	)
	res, err := svc.AnalyzeTranscript(context.Background(), "随便聊聊", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	assert.Nil(t, res.IdentityID)
	assert.Equal(t, 0.5, res.Confidence)
}

func TestAnalyzeTranscriptNonCandidateIdentity(t *testing.T) {
	svc := newFakeAIService(
		func() (string, error) { return "{\"identity_id\":\"i999\",\"confidence\":0.9}", nil },
		func() (string, error) { return "{\"todos\":[],\"commitments\":[],\"notes\":[]}", nil },
	)
	res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	assert.Nil(t, res.IdentityID)
}

func TestAnalyzeTranscriptInvalidJSONThenRetry(t *testing.T) {
	identityCalls := 0
	svc := newFakeAIService(
		func() (string, error) {
			identityCalls++
			if identityCalls == 1 {
				return "这不是合法 JSON", nil
			}
			return "{\"identity_id\":\"i1\",\"confidence\":0.9}", nil
		},
		func() (string, error) { return "{\"todos\":[],\"commitments\":[],\"notes\":[]}", nil },
	)
	res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	require.NotNil(t, res.IdentityID)
	assert.Equal(t, "i1", *res.IdentityID)
	assert.Equal(t, 2, identityCalls)
}

func TestAnalyzeTranscriptEmptyTranscript(t *testing.T) {
	svc := newAIService(&fakeLLM{name: "m"}, 0.6)
	_, err := svc.AnalyzeTranscript(context.Background(), "   ", nil)
	require.Error(t, err)
}
