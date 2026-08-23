package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	"github.com/chaojixinren/pulse/internal/service"
)

var (
	workerReminderCols = []string{"id", "user_id", "session_id", "identity_id", "type", "content", "due_at", "status", "created_at", "updated_at"}
	workerIdentityCols = []string{"id", "user_id", "name", "description", "color", "icon", "is_default", "created_at", "updated_at", "deleted_at"}
)

// newFakeAI 构建一个可脚本化的 AI 服务：依据系统提示词区分身份识别/信息提取两阶段。
func newFakeAI(t *testing.T, identityResp, extractResp map[string]interface{}) *service.AIService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		system := ""
		if msgs, ok := body["messages"].([]interface{}); ok && len(msgs) > 0 {
			if m, ok := msgs[0].(map[string]interface{}); ok {
				system, _ = m["content"].(string)
			}
		}
		resp := extractResp
		if strings.Contains(system, "身份识别") {
			resp = identityResp
		}
		content, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{"message": map[string]interface{}{"role": "assistant", "content": string(content)}},
			},
		})
	}))
	t.Cleanup(srv.Close)
	return service.NewAIService(&config.Config{
		AIBaseURL:             srv.URL,
		AIAPIKey:              "test-key",
		AIModel:               "test-model",
		AIConfidenceThreshold: 0.6,
	})
}

// TestAudioProcessorAnalyzeFullFlow 集成验收：转写完成 → 拉取候选身份 → AI 两阶段分析 → 回写会话 → 生成 todo 提醒与身份切换提醒。
func TestAudioProcessorAnalyzeFullFlow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sessions := repository.NewAudioSessionRepo(db)
	identities := repository.NewIdentityRepo(db)
	reminders := service.NewReminderService(repository.NewReminderRepo(db))
	ai := newFakeAI(t,
		map[string]interface{}{"identity_id": "i2", "confidence": 0.9},
		map[string]interface{}{
			"todos":       []interface{}{map[string]interface{}{"text": "买菜"}},
			"commitments": []interface{}{},
			"notes":       []interface{}{},
		},
	)

	w := &AudioProcessor{sessions: sessions, identities: identities, ai: ai, reminders: reminders}
	now := time.Now().UTC()

	// 1) 拉取候选身份
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(workerIdentityCols).
			AddRow("i1", "u1", "工作", nil, "#000000", "person", true, now, now, nil).
			AddRow("i2", "u1", "家庭", nil, "#000000", "person", false, now, now, nil))

	// 2) 上一条身份为 i1 → 与本次 i2 不同，触发身份切换
	mock.ExpectQuery(regexp.QuoteMeta("SELECT identity_id FROM audio_sessions")).
		WithArgs("u1", "s1").
		WillReturnRows(sqlmock.NewRows([]string{"identity_id"}).AddRow("i1"))

	// 3) 回写 AI 分析结果
	wantExtracted, err := json.Marshal(model.ExtractedData{
		Todos:       []model.Todo{{Text: "买菜"}},
		Commitments: []model.Commitment{},
		Notes:       []string{},
	})
	require.NoError(t, err)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET identity_id")).
		WithArgs("i2", 0.9, string(wantExtracted), sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 4) 生成 todo 提醒
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", "i2", "todo", "买菜", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 5) 身份切换提醒：查询该身份下未完成待办（空）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, session_id")).
		WithArgs("u1", "i2", "todo", "pending").
		WillReturnRows(sqlmock.NewRows(workerReminderCols))

	// 6) 写入身份切换提醒
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO reminders")).
		WithArgs(sqlmock.AnyArg(), "u1", "s1", "i2", "identity_switch", "身份已切换，该身份下暂无未完成待办。", nil, "pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	sess := &model.AudioSession{ID: "s1", UserID: "u1"}
	w.analyze(context.Background(), sess, "明天买菜")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAudioProcessorAnalyzeNoSwitch 集成验收：身份未变化且无待办/承诺时，不产生任何提醒。
func TestAudioProcessorAnalyzeNoSwitch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sessions := repository.NewAudioSessionRepo(db)
	identities := repository.NewIdentityRepo(db)
	reminders := service.NewReminderService(repository.NewReminderRepo(db))
	ai := newFakeAI(t,
		map[string]interface{}{"identity_id": "i1", "confidence": 0.9},
		map[string]interface{}{"todos": []interface{}{}, "commitments": []interface{}{}, "notes": []interface{}{}},
	)

	w := &AudioProcessor{sessions: sessions, identities: identities, ai: ai, reminders: reminders}
	now := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(workerIdentityCols).AddRow("i1", "u1", "工作", nil, "#000000", "person", true, now, now, nil))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT identity_id FROM audio_sessions")).
		WithArgs("u1", "s1").
		WillReturnRows(sqlmock.NewRows([]string{"identity_id"}).AddRow("i1"))

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
	reminders := service.NewReminderService(repository.NewReminderRepo(db))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	ai := service.NewAIService(&config.Config{AIBaseURL: srv.URL, AIAPIKey: "k", AIModel: "m", AIConfidenceThreshold: 0.6})

	w := &AudioProcessor{sessions: sessions, identities: identities, ai: ai, reminders: reminders}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, name")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows(workerIdentityCols))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT identity_id FROM audio_sessions")).
		WithArgs("u1", "s1").
		WillReturnRows(sqlmock.NewRows([]string{"identity_id"}))

	emptyJSON, err := json.Marshal(model.ExtractedData{Todos: []model.Todo{}, Commitments: []model.Commitment{}, Notes: []string{}})
	require.NoError(t, err)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET identity_id")).
		WithArgs(nil, 0.0, string(emptyJSON), sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	sess := &model.AudioSession{ID: "s1", UserID: "u1"}
	w.analyze(context.Background(), sess, "随便聊聊")

	assert.NoError(t, mock.ExpectationsWereMet())
}
