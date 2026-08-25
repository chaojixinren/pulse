package worker

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/model"
	"github.com/chaojixinren/pulse/internal/repository"
	"github.com/chaojixinren/pulse/internal/service"
	"github.com/chaojixinren/pulse/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init("test", "error")
	os.Exit(m.Run())
}

func newWorkerMockDB(t *testing.T) (*repository.AudioSessionRepo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewAudioSessionRepo(db), mock
}

// newFakeSTT 返回指向本地假服务的 SttService，可控制返回文本与 HTTP 状态。
func newFakeSTT(t *testing.T, text string, status int) *service.SttService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"stt down"}`))
			return
		}
		// 返回 SSE 格式
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"type":"transcript.text.done","text":"` + text + `"}` + "\n\n"))
	}))
	t.Cleanup(srv.Close)
	return service.NewSttService(&config.Config{
		StepFunBaseURL:  srv.URL,
		StepFunAPIKey:   "test-key",
		StepFunSTTModel: "test-model",
	})
}

func strPtr(s string) *string { return &s }

func TestAudioProcessorProcessOneSuccess(t *testing.T) {
	repo, mock := newWorkerMockDB(t)
	stt := newFakeSTT(t, "hello world", http.StatusOK)
	w := &AudioProcessor{sessions: repo, stt: stt}

	// 认领 → 写转写 → 置 completed
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?")).
		WithArgs("processing", sqlmock.AnyArg(), "s1", "pending", "processing").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET transcript")).
		WithArgs("hello world", sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE")).
		WithArgs("s1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("processing"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?, error_message = ?")).
		WithArgs("completed", nil, sqlmock.AnyArg(), "completed", "completed", sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	sess := &model.AudioSession{ID: "s1", AudioData: []byte("audio"), AudioMime: strPtr("audio/wav")}
	w.processOne(context.Background(), sess)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioProcessorProcessOneFailure(t *testing.T) {
	repo, mock := newWorkerMockDB(t)
	stt := newFakeSTT(t, "", http.StatusInternalServerError)
	w := &AudioProcessor{sessions: repo, stt: stt}

	// 认领 → 转写失败 → 置 failed（带 error_message）
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?")).
		WithArgs("processing", sqlmock.AnyArg(), "s1", "pending", "processing").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE")).
		WithArgs("s1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("processing"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?, error_message = ?")).
		WithArgs("failed", sqlmock.AnyArg(), sqlmock.AnyArg(), "failed", "completed", sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	sess := &model.AudioSession{ID: "s1", AudioData: []byte("audio"), AudioMime: strPtr("audio/wav")}
	w.processOne(context.Background(), sess)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioProcessorProcessOneSkippedWhenAlreadyClaimed(t *testing.T) {
	repo, mock := newWorkerMockDB(t)
	stt := newFakeSTT(t, "hello", http.StatusOK)
	w := &AudioProcessor{sessions: repo, stt: stt}

	// 认领失败（0 行受影响）→ 直接返回，不再转写/更新。
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?")).
		WithArgs("processing", sqlmock.AnyArg(), "s1", "pending", "processing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	sess := &model.AudioSession{ID: "s1", AudioData: []byte("audio"), AudioMime: strPtr("audio/wav")}
	w.processOne(context.Background(), sess)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioProcessorProcessBatch(t *testing.T) {
	repo, mock := newWorkerMockDB(t)
	stt := newFakeSTT(t, "hello", http.StatusOK)
	w := &AudioProcessor{sessions: repo, stt: stt, batchSize: 5}

	// 无 pending 会话时，不做任何处理。
	mock.ExpectQuery(regexp.QuoteMeta("FROM audio_sessions WHERE status IN (?, ?) ORDER BY created_at ASC LIMIT ?")).
		WithArgs("pending", "processing", 5).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "identity_id", "device_id", "audio_data", "audio_mime", "transcript", "duration",
			"file_size", "status", "error_message", "extracted_data", "ai_confidence", "recorded_at", "processed_at", "created_at", "updated_at",
		}))

	w.processBatch(context.Background())
	assert.NoError(t, mock.ExpectationsWereMet())
}

type fakeCompleter struct {
	responses []string
	calls     int
}

// TestAudioProcessorAnalyzeWithoutAI 验证未配置 AI 服务时，analyze 直接返回、不产生任何 DB 操作。
func TestAudioProcessorAnalyzeWithoutAI(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sessions := repository.NewAudioSessionRepo(db)

	w := &AudioProcessor{sessions: sessions, ai: nil}

	sess := &model.AudioSession{ID: "s1", UserID: "u1"}
	// AI 服务未配置时，analyze 应直接返回、不做任何 DB 访问。
	w.analyze(context.Background(), sess, "明天买菜")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// newCaptureSTT 返回一个会捕获请求体（音频字节）的假 STT 服务。
func newCaptureSTT(t *testing.T) (*service.SttService, chan []byte) {
	t.Helper()
	ch := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ch <- body
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"type":"transcript.text.done","text":"ok"}` + "\n\n"))
	}))
	t.Cleanup(srv.Close)
	return service.NewSttService(&config.Config{
		StepFunBaseURL:  srv.URL,
		StepFunAPIKey:   "test-key",
		StepFunSTTModel: "test-model",
	}), ch
}

func TestAudioProcessorDecryptsBeforeTranscribe(t *testing.T) {
	repo, mock := newWorkerMockDB(t)
	stt, captured := newCaptureSTT(t)

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plain := []byte("plain audio bytes")
	enc, err := service.EncryptAudio(plain, key)
	require.NoError(t, err)

	w := &AudioProcessor{sessions: repo, stt: stt, encryptionKey: key}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?")).
		WithArgs("processing", sqlmock.AnyArg(), "s1", "pending", "processing").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET transcript")).
		WithArgs("ok", sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE")).
		WithArgs("s1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("processing"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?, error_message = ?")).
		WithArgs("completed", nil, sqlmock.AnyArg(), "completed", "completed", sqlmock.AnyArg(), "s1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	sess := &model.AudioSession{ID: "s1", AudioData: enc, AudioMime: strPtr("audio/wav")}
	w.processOne(context.Background(), sess)

	assert.NoError(t, mock.ExpectationsWereMet())

	body := <-captured
	assert.Contains(t, string(body), base64.StdEncoding.EncodeToString(plain), "STT 请求体应包含明文音频的 base64")
	assert.NotContains(t, string(body), base64.StdEncoding.EncodeToString(enc), "STT 请求体不应包含密文")
}
