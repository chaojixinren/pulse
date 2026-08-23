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
	type capturedReq struct {
		path     string
		auth     string
		model    string
		filename string
		hasFile  bool
	}
	got := make(chan capturedReq, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		c := capturedReq{path: r.URL.Path, auth: r.Header.Get("Authorization"), model: r.FormValue("model")}
		file, fh, err := r.FormFile("file")
		if err == nil {
			c.hasFile = true
			c.filename = fh.Filename
			_ = file.Close()
		}
		got <- c
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"hello world"}`))
	}))
	defer srv.Close()

	svc := newSttServiceFor(t, srv.URL)
	text, err := svc.Transcribe(context.Background(), []byte("audio-data"), "clip.wav")
	require.NoError(t, err)
	assert.Equal(t, "hello world", text)

	req := <-got
	assert.Equal(t, "/audio/transcriptions", req.path)
	assert.Equal(t, "Bearer test-key", req.auth)
	assert.Equal(t, "test-model", req.model)
	assert.True(t, req.hasFile)
	assert.Equal(t, "clip.wav", req.filename)
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
		_, _ = w.Write([]byte(`{"text":"  "}`))
	}))
	defer srv.Close()

	svc := newSttServiceFor(t, srv.URL)
	_, err := svc.Transcribe(context.Background(), []byte("audio"), "clip.wav")
	require.Error(t, err)
}

func TestSttTranscribeInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	svc := newSttServiceFor(t, srv.URL)
	_, err := svc.Transcribe(context.Background(), []byte("audio"), "clip.wav")
	require.Error(t, err)
}
