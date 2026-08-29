package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/config"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
)

// SttService 封装 StepFun StepAudio-2.5-ASR（Step Plan SSE 接口）。
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

// Step Plan SSE 请求格式
type sseRequest struct {
	Audio sseAudio `json:"audio"`
}

type sseAudio struct {
	Data  string   `json:"data"`
	Input sseInput `json:"input"`
}

type sseInput struct {
	Transcription sseTranscription `json:"transcription"`
	Format        sseFormat        `json:"format"`
}

type sseTranscription struct {
	Model     string `json:"model"`
	Language  string `json:"language"`
	EnableITN bool   `json:"enable_itn"`
}

type sseFormat struct {
	Type    string `json:"type"`
	Codec   string `json:"codec"`
	Rate    int    `json:"rate"`
	Bits    int    `json:"bits"`
	Channel int    `json:"channel"`
}

// SSE 响应格式
type sseEvent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Meta  json.RawMessage `json:"meta,omitempty"`
	Usage json.RawMessage `json:"usage,omitempty"`
}

// FilenameForMime 根据音频 MIME 推断文件名（保留用于兼容性）。
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

// SttOverrides 是单次转写的 per-user 覆盖项；空值字段回退全局默认。
type SttOverrides struct {
	APIKey    string
	BaseURL   string
	Model     string
	Language  string
	EnableITN *bool
}

// Transcribe 将音频字节提交到 StepFun Step Plan SSE 接口，返回转写文本。
func (s *SttService) Transcribe(ctx context.Context, data []byte, filename string) (string, error) {
	return s.TranscribeWithOverrides(ctx, data, filename, nil)
}

// TranscribeWithOverrides 在全局默认之上应用 per-user 覆盖后执行转写。
func (s *SttService) TranscribeWithOverrides(ctx context.Context, data []byte, filename string, o *SttOverrides) (string, error) {
	if len(data) == 0 {
		return "", apperrors.NewBadRequest("音频数据为空")
	}

	apiKey := s.apiKey
	baseURL := s.baseURL
	model := s.model
	language := "zh"
	enableITN := true
	if o != nil {
		if o.APIKey != "" {
			apiKey = o.APIKey
		}
		if o.BaseURL != "" {
			baseURL = strings.TrimRight(o.BaseURL, "/")
		}
		if o.Model != "" {
			model = o.Model
		}
		if o.Language != "" {
			language = o.Language
		}
		if o.EnableITN != nil {
			enableITN = *o.EnableITN
		}
	}

	// Base64 编码音频数据
	encodedAudio := base64.StdEncoding.EncodeToString(data)

	// 构造 SSE 请求
	reqBody := sseRequest{
		Audio: sseAudio{
			Data: encodedAudio,
			Input: sseInput{
				Transcription: sseTranscription{
					Model:     model,
					Language:  language,
					EnableITN: enableITN,
				},
				Format: sseFormat{
					Type:    "pcm",
					Codec:   "pcm_s16le",
					Rate:    16000,
					Bits:    16,
					Channel: 1,
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", apperrors.WrapInternal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/audio/asr/sse", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", apperrors.WrapInternal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", apperrors.NewInternal(fmt.Sprintf("STT 请求失败: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", apperrors.NewInternal(fmt.Sprintf("STT 返回错误 (HTTP %d): %s", resp.StatusCode, truncate(string(body), 500)))
	}

	// 解析 SSE 流
	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式: data: {...}
		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")

			// 跳过心跳
			if dataStr == "[DONE]" || dataStr == "" {
				continue
			}

			var event sseEvent
			if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
				continue
			}

			// 提取转写文本 (type 为 transcript.text.* 的事件)
			if strings.HasPrefix(event.Type, "transcript.text") {
				if event.Text != "" {
					fullText.WriteString(event.Text)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", apperrors.WrapInternal(fmt.Errorf("读取 SSE 流失败: %w", err))
	}

	result := strings.TrimSpace(fullText.String())
	// 允许返回空文本（静音音频是合法的）
	return result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
