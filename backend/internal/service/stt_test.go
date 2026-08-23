package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/config"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

func TestFilenameForMime(t *testing.T) {
	assert.Equal(t, "audio.mp3", FilenameForMime("audio/mpeg"))
	assert.Equal(t, "audio.mp3", FilenameForMime("audio/mp3"))
	assert.Equal(t, "audio.m4a", FilenameForMime("audio/mp4"))
	assert.Equal(t, "audio.m4a", FilenameForMime("audio/x-m4a"))
	assert.Equal(t, "audio.wav", FilenameForMime("audio/wav"))
	assert.Equal(t, "audio.wav", FilenameForMime(""))
}

func newSttServiceFor(t *testing.T, baseURL string) *SttService {
	t.Helper()
	return NewSttService(&config.Config{
		StepFunBaseURL:  baseURL,
		StepFunAPIKey:   "test-key",
		StepFunSTTModel: "test-model",
	})
}

func TestSttTranscribeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"type":"transcript.text.done","text":"hello world"}` + "\n\n"))
	}))
	defer srv.Close()

	svc := newSttServiceFor(t, srv.URL)
	text, err := svc.Transcribe(context.Background(), []byte("audio-data"), "clip.wav")
	require.NoError(t, err)
	assert.Equal(t, "hello world", text)
}

func TestSttTranscribeEmptyData(t *testing.T) {
	svc := newSttServiceFor(t, "http://127.0.0.1:1")
	_, err := svc.Transcribe(context.Background(), nil, "clip.wav")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 40000, appErr.Code)
}

func TestSttTranscribeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	svc := newSttServiceFor(t, srv.URL)
	_, err := svc.Transcribe(context.Background(), []byte("audio"), "clip.wav")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 50000, appErr.Code)
}

func TestSttTranscribeEmptyText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"type":"transcript.text.done","text":"  "}` + "\n\n"))
	}))
	defer srv.Close()

	svc := newSttServiceFor(t, srv.URL)
	text, err := svc.Transcribe(context.Background(), []byte("audio"), "clip.wav")
	require.NoError(t, err)
	assert.Equal(t, "", text)
}

func TestSttTranscribeInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: not-json\n\n"))
	}))
	defer srv.Close()

	svc := newSttServiceFor(t, srv.URL)
	text, err := svc.Transcribe(context.Background(), []byte("audio"), "clip.wav")
	require.NoError(t, err)
	assert.Equal(t, "", text)
}
