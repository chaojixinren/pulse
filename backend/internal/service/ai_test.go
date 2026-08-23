package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
)

// newTestAIService 启动一个可脚本化的 httptest 服务器并构建 AI 服务。
func newTestAIService(t *testing.T, handler http.HandlerFunc, threshold float64) *AIService {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return newAIService(srv.URL, "test-key", "test-model", srv.Client(), threshold)
}

// chatHandler 依据请求里的系统提示词分发到身份识别 / 信息提取响应。
func chatHandler(identityFn, extractFn func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		system := ""
		if len(req.Messages) > 0 {
			system = req.Messages[0].Content
		}
		content := extractFn()
		if strings.Contains(system, "身份识别") {
			content = identityFn()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": content}},
			},
		})
	}
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "纯 JSON",
			input: `{"key": "value"}`,

			want: `{"key": "value"}`,
		},
		{
			name:  "带 markdown 代码块",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "带前后文本",
			input: "这是结果：{\"key\": \"value\"} 完成",
			want:  `{"key": "value"}`,
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
	svc := newTestAIService(t, chatHandler(
		func() string { return `{"identity_id":"i1","confidence":0.9}` },
		func() string {
			return `{"todos":[{"text":"买菜"}],"commitments":[{"text":"帮小王改方案","from":"我","to":"小王"}],"notes":["会议定在周三"]}`
		},
	), 0.6)
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
	svc := newTestAIService(t, chatHandler(
		func() string { return `{"identity_id":"i1","confidence":0.5}` },
		func() string { return `{"todos":[],"commitments":[],"notes":[]}` },
	), 0.6)
	res, err := svc.AnalyzeTranscript(context.Background(), "随便聊聊", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	assert.Nil(t, res.IdentityID)
	assert.Equal(t, 0.5, res.Confidence)
}

func TestAnalyzeTranscriptNonCandidateIdentity(t *testing.T) {
	svc := newTestAIService(t, chatHandler(
		func() string { return `{"identity_id":"i999","confidence":0.9}` },
		func() string { return `{"todos":[],"commitments":[],"notes":[]}` },
	), 0.6)
	res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	assert.Nil(t, res.IdentityID)
}

func TestAnalyzeTranscriptInvalidJSONThenRetry(t *testing.T) {
	identityCalls := 0
	svc := newTestAIService(t, chatHandler(
		func() string {
			identityCalls++
			if identityCalls == 1 {
				return "这不是合法 JSON"
			}
			return `{"identity_id":"i1","confidence":0.9}`
		},
		func() string { return `{"todos":[],"commitments":[],"notes":[]}` },
	), 0.6)
	res, err := svc.AnalyzeTranscript(context.Background(), "开会", []model.Identity{{ID: "i1", Name: "工作"}})
	require.NoError(t, err)
	require.NotNil(t, res.IdentityID)
	assert.Equal(t, "i1", *res.IdentityID)
	assert.Equal(t, 2, identityCalls)
}

func TestAnalyzeTranscriptEmptyTranscript(t *testing.T) {
	svc := newAIService("http://unused", "", "m", &http.Client{}, 0.6)
	_, err := svc.AnalyzeTranscript(context.Background(), "   ", nil)
	require.Error(t, err)
}
