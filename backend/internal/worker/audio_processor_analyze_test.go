package worker

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	"github.com/chaojixinren/pulse/internal/service"

	adkmodel "google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

var workerIdentityCols = []string{"id", "user_id", "name", "description", "color", "icon", "is_default", "created_at", "updated_at", "deleted_at"}

// fakeWorkerLLM 是 worker 测试用的可脚本化 model.LLM：依据系统提示词区分身份识别/信息提取两阶段。
type fakeWorkerLLM struct {
	identity string
	extract  string
	fail     bool
}

func (f *fakeWorkerLLM) Name() string { return "test-model" }

func (f *fakeWorkerLLM) GenerateContent(_ context.Context, req *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		if f.fail {
			yield(nil, errors.New("model down"))
			return
		}
		text := f.extract
		if strings.Contains(workerSystemText(req), "身份识别") {
			text = f.identity
		}
		yield(&adkmodel.LLMResponse{
			Content: &genai.Content{Role: "model", Parts: []*genai.Part{{Text: text}}},
		}, nil)
	}
}

func workerSystemText(req *adkmodel.LLMRequest) string {
	if req == nil || req.Config == nil || req.Config.SystemInstruction == nil {
		return ""
	}
	var b strings.Builder
	for _, p := range req.Config.SystemInstruction.Parts {
		if p != nil {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// newFakeAI 构建一个可脚本化的 AI 服务：依据系统提示词区分身份识别/信息提取两阶段。
func newFakeAI(identityResp, extractResp map[string]interface{}) *service.AIService {
	idJSON, _ := json.Marshal(identityResp)
	exJSON, _ := json.Marshal(extractResp)
	return service.NewAIServiceWithModel(&fakeWorkerLLM{identity: string(idJSON), extract: string(exJSON)}, 0.6)
}

// TestAudioProcessorAnalyzeFullFlow 集成验收：转写完成 → 拉取候选身份 → AI 两阶段分析 → 回写会话。
func TestAudioProcessorAnalyzeFullFlow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sessions := repository.NewAudioSessionRepo(db)
	identities := repository.NewIdentityRepo(db)
	ai := newFakeAI(
		map[string]interface{}{"identity_id": "i2", "confidence": 0.9},
		map[string]interface{}{
			"todos":       []interface{}{map[string]interface{}{"text": "买菜"}},
			"commitments": []interface{}{},
			"notes":       []interface{}{},
		},
	)

	w := &AudioProcessor{sessions: sessions, identities: identities, ai: ai}
	now := time.Now().UTC()

	// 1) 拉取候选身份
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(workerIdentityCols).
			AddRow("i1", "u1", "工作", nil, "#000000", "person", true, now, now, nil).
			AddRow("i2", "u1", "家庭", nil, "#000000", "person", false, now, now, nil))

	// 2) 回写 AI 分析结果
	wantExtracted, err := json.Marshal(model.ExtractedData{
		Todos:       []model.Todo{{Text: "买菜"}},
		Commitments: []model.Commitment{},
		Notes:       []string{},
	})
	require.NoError(t, err)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET identity_id")).
		WithArgs("i2", 0.9, string(wantExtracted), sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	sess := &model.AudioSession{ID: "s1", UserID: "u1"}
	w.analyze(context.Background(), sess, "明天买菜")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAudioProcessorAnalyzeEmptyExtraction 集成验收：识别到身份但未提取到待办/承诺时，仍回写身份与空提取结果。
func TestAudioProcessorAnalyzeEmptyExtraction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sessions := repository.NewAudioSessionRepo(db)
	identities := repository.NewIdentityRepo(db)
	ai := newFakeAI(
		map[string]interface{}{"identity_id": "i1", "confidence": 0.9},
		map[string]interface{}{"todos": []interface{}{}, "commitments": []interface{}{}, "notes": []interface{}{}},
	)

	w := &AudioProcessor{sessions: sessions, identities: identities, ai: ai}
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(workerIdentityCols).AddRow("i1", "u1", "工作", nil, "#000000", "person", true, now, now, nil))

	wantExtracted, err := json.Marshal(model.ExtractedData{Todos: []model.Todo{}, Commitments: []model.Commitment{}, Notes: []string{}})
	require.NoError(t, err)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET identity_id")).
		WithArgs("i1", 0.9, string(wantExtracted), sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	sess := &model.AudioSession{ID: "s1", UserID: "u1"}
	w.analyze(context.Background(), sess, "随便聊聊")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAudioProcessorAnalyzeDegradesOnAIError 集成验收：AI 服务不可用时降级回写（identity 为空、confidence=0、空提取），会话不丢失。
func TestAudioProcessorAnalyzeDegradesOnAIError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sessions := repository.NewAudioSessionRepo(db)
	identities := repository.NewIdentityRepo(db)
	ai := service.NewAIServiceWithModel(&fakeWorkerLLM{fail: true}, 0.6)

	w := &AudioProcessor{sessions: sessions, identities: identities, ai: ai}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(workerIdentityCols))

	emptyJSON, err := json.Marshal(model.ExtractedData{Todos: []model.Todo{}, Commitments: []model.Commitment{}, Notes: []string{}})
	require.NoError(t, err)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET identity_id")).
		WithArgs(nil, 0.0, string(emptyJSON), sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	sess := &model.AudioSession{ID: "s1", UserID: "u1"}
	w.analyze(context.Background(), sess, "随便聊聊")

	assert.NoError(t, mock.ExpectationsWereMet())
}
