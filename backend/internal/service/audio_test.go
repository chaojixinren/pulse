package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/repository"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

var sessionListCols = []string{"id", "user_id", "identity_id", "device_id", "audio_mime", "transcript", "duration", "file_size", "status", "error_message", "extracted_data", "ai_confidence", "recorded_at", "processed_at", "created_at", "updated_at"}

func newAudioService(t *testing.T, maxSize int64) (*AudioService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	cfg := &config.Config{MaxAudioSize: maxSize}
	return NewAudioService(cfg, repository.NewAudioSessionRepo(db)), mock
}

func TestExtToMime(t *testing.T) {
	assert.Equal(t, "audio/mpeg", extToMime(".mp3"))
	assert.Equal(t, "audio/mp4", extToMime(".m4a"))
	assert.Equal(t, "audio/wav", extToMime(".wav"))
	assert.Equal(t, "audio/wav", extToMime(".unknown"))
}

// validWAVBytes 返回一个最小可识别的 WAV 文件头（RIFF....WAVE）。
func validWAVBytes() []byte {
	return []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'A', 'V', 'E', 'f', 'm', 't', ' '}
}

func TestDetectAudioExt(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"wav", validWAVBytes(), ".wav"},
		{"mp3_id3", append([]byte("ID3"), make([]byte, 8)...), ".mp3"},
		{"mp3_frame_sync", []byte{0xFF, 0xFB, 0x90, 0x00}, ".mp3"},
		{"m4a_ftyp", []byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 'M', '4', 'A', ' '}, ".m4a"},
		{"garbage", []byte("hello world this is not audio"), ""},
		{"too_short", []byte("RI"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, detectAudioExt(tc.data))
		})
	}
}

func TestAudioUploadContentExtensionMismatch(t *testing.T) {
	svc, _ := newAudioService(t, 1024)

	// 扩展名 .wav，但内容是 MP3（ID3），应拒绝。
	_, err := svc.Upload(context.Background(), "u1", UploadInput{
		Data: append([]byte("ID3"), make([]byte, 8)...), Filename: "clip.wav",
	})
	require.Error(t, err)
	appErr, _ := apperrors.AsAppError(err)
	assert.Equal(t, 40000, appErr.Code)
}

func TestAudioUploadValidation(t *testing.T) {
	t.Run("empty_data", func(t *testing.T) {
		svc, _ := newAudioService(t, 1024)
		_, err := svc.Upload(context.Background(), "u1", UploadInput{Data: nil, Filename: "a.wav"})
		require.Error(t, err)
		appErr, _ := apperrors.AsAppError(err)
		assert.Equal(t, 40000, appErr.Code)
	})

	t.Run("too_large", func(t *testing.T) {
		svc, _ := newAudioService(t, 4)
		_, err := svc.Upload(context.Background(), "u1", UploadInput{Data: []byte("12345"), Filename: "a.wav"})
		require.Error(t, err)
		appErr, _ := apperrors.AsAppError(err)
		assert.Equal(t, 40000, appErr.Code)
	})

	t.Run("bad_extension", func(t *testing.T) {
		svc, _ := newAudioService(t, 1024)
		_, err := svc.Upload(context.Background(), "u1", UploadInput{Data: []byte("data"), Filename: "a.txt"})
		require.Error(t, err)
		appErr, _ := apperrors.AsAppError(err)
		assert.Equal(t, 40000, appErr.Code)
	})
}

func TestAudioUploadSuccess(t *testing.T) {
	svc, mock := newAudioService(t, 1024)
	recordedAt := time.Now().UTC()
	data := validWAVBytes()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audio_sessions")).
		WithArgs(sqlmock.AnyArg(), "u1", nil, nil, data, sqlmock.AnyArg(), nil, 10, int64(len(data)), "pending", nil, "{}", nil, recordedAt, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	s, err := svc.Upload(context.Background(), "u1", UploadInput{
		Data: data, Filename: "clip.wav", Duration: 10, RecordedAt: recordedAt,
	})
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, "pending", s.Status, "新上传会话应为 pending")
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, int64(len(data)), *s.FileSize)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAudioRetry(t *testing.T) {
	t.Run("only_failed_can_retry", func(t *testing.T) {
		svc, mock := newAudioService(t, 1024)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
			WithArgs("s1", "u1").
			WillReturnRows(sqlmock.NewRows(sessionListCols).
				AddRow("s1", "u1", nil, nil, "audio/wav", nil, 10, int64(5), "pending", nil, "{}", nil, now, nil, now, now))

		err := svc.Retry(context.Background(), "u1", "s1")
		require.Error(t, err)
		appErr, _ := apperrors.AsAppError(err)
		assert.Equal(t, 40000, appErr.Code)
	})

	t.Run("failed_transitions_to_processing", func(t *testing.T) {
		svc, mock := newAudioService(t, 1024)
		now := time.Now().UTC()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, identity_id, device_id, audio_mime")).
			WithArgs("s1", "u1").
			WillReturnRows(sqlmock.NewRows(sessionListCols).
				AddRow("s1", "u1", nil, nil, "audio/wav", nil, 10, int64(5), "failed", "boom", "{}", nil, now, nil, now, now))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM audio_sessions WHERE id = ? FOR UPDATE")).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("failed"))
		mock.ExpectExec(regexp.QuoteMeta("UPDATE audio_sessions SET status = ?, error_message = ?")).
			WithArgs("processing", nil, sqlmock.AnyArg(), "processing", "completed", sqlmock.AnyArg(), "s1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		require.NoError(t, svc.Retry(context.Background(), "u1", "s1"))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
