package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/config"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// SttService 封装 StepFun StepAudio-2.5-ASR（OpenAI 兼容接口）。
type SttService struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewSttService(cfg *config.Config) *SttService {
	return &SttService{
		apiKey:  cfg.StepFunAPIKey,
		baseURL: strings.TrimRight(cfg.StepFunBaseURL, "/"),
		model:   cfg.StepFunSTTModel,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

type transcriptionResponse struct {
	Text string `json:"text"`
}

// FilenameForMime 根据音频 MIME 推断文件名，用于 multipart 上传。
func FilenameForMime(mime string) string {
	switch mime {
	case "audio/mpeg", "audio/mp3":
		return "audio.mp3"
	case "audio/mp4", "audio/x-m4a", "audio/m4a":
		return "audio.m4a"
	default:
		return "audio.wav"
	}
}

// Transcribe 将音频字节提交到 StepFun 语音识别，返回转写文本。
func (s *SttService) Transcribe(ctx context.Context, data []byte, filename string) (string, error) {
	if len(data) == 0 {
		return "", apperrors.NewBadRequest("音频数据为空")
	}
	if filename == "" {
		filename = "audio.wav"
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", apperrors.WrapInternal(err)
	}
	if _, err := part.Write(data); err != nil {
		return "", apperrors.WrapInternal(err)
	}
	_ = writer.WriteField("model", s.model)
	_ = writer.WriteField("response_format", "json")
	if err := writer.Close(); err != nil {
		return "", apperrors.WrapInternal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return "", apperrors.WrapInternal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", apperrors.NewInternal(fmt.Sprintf("STT 请求失败: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", apperrors.WrapInternal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", apperrors.NewInternal(fmt.Sprintf("STT 返回错误 (HTTP %d): %s", resp.StatusCode, truncate(string(body), 500)))
	}

	var result transcriptionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", apperrors.WrapInternal(fmt.Errorf("解析 STT 响应失败: %w", err))
	}
	if strings.TrimSpace(result.Text) == "" {
		return "", apperrors.NewInternal("STT 返回空文本")
	}
	return result.Text, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
