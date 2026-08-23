package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/model"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/prompt"
)

const (
	defaultAIBaseURL             = "https://api.openai.com/v1/chat/completions"
	defaultAIModel               = "gpt-4o-mini"
	defaultAIConfidenceThreshold = 0.6
	aiHTTPTimeout                = 60 * time.Second
)

// AIService 负责把转写文本交给 LLM，完成身份识别与信息提取。
// 使用 OpenAI 兼容的 chat/completions 客户端做显式两阶段编排：先身份识别，再信息提取。
type AIService struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	modelName  string
	threshold  float64
}

// NewAIService 根据配置构建 AI 分析服务。
func NewAIService(cfg *config.Config) *AIService {
	baseURL := cfg.AIBaseURL
	if baseURL == "" {
		baseURL = defaultAIBaseURL
	}
	modelName := cfg.AIModel
	if modelName == "" {
		modelName = defaultAIModel
	}
	threshold := cfg.AIConfidenceThreshold
	if threshold <= 0 {
		threshold = defaultAIConfidenceThreshold
	}
	return newAIService(baseURL, cfg.AIAPIKey, modelName, &http.Client{Timeout: aiHTTPTimeout}, threshold)
}

// newAIService 构建 AI 服务，允许注入自定义 HTTP 客户端（测试用）。
func newAIService(baseURL, apiKey, modelName string, httpClient *http.Client, threshold float64) *AIService {
	if threshold <= 0 {
		threshold = defaultAIConfidenceThreshold
	}
	return &AIService{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKey:     apiKey,
		modelName:  modelName,
		threshold:  threshold,
	}
}

// AnalyzeTranscript 转写文本 + 用户身份列表 → 分析结果。
// 两阶段：先身份识别（候选身份作为标签），再信息提取。
func (s *AIService) AnalyzeTranscript(ctx context.Context, transcript string, identities []model.Identity) (*model.AnalysisResult, error) {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return nil, apperrors.NewBadRequest("转写文本为空，无法分析")
	}

	identityID, confidence := s.recognizeIdentity(ctx, transcript, identities)
	extracted := s.extract(ctx, transcript)

	return buildAnalysisResult(identityID, confidence, s.threshold, extracted), nil
}

// buildAnalysisResult 依据识别结果、置信度与阈值组装最终结果。
// 仅当置信度达到阈值且身份非空时才绑定身份，否则 identity_id 留空交用户手动标注。
func buildAnalysisResult(identityID string, confidence float64, threshold float64, extracted model.ExtractedData) *model.AnalysisResult {
	result := &model.AnalysisResult{
		Extracted:   extracted,
		Todos:       extracted.Todos,
		Commitments: extracted.Commitments,
		Notes:       extracted.Notes,
		Confidence:  confidence,
	}
	if confidence >= threshold && identityID != "" {
		id := identityID
		result.IdentityID = &id
	}
	return result
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// call 执行一次 chat/completions 调用，返回模型输出的文本。
func (s *AIService) call(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatCompletionRequest{
		Model: s.modelName,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat completion 请求失败: status=%d body=%s", resp.StatusCode, string(body))
	}

	var cr chatCompletionResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", err
	}
	if cr.Error != nil && cr.Error.Message != "" {
		return "", fmt.Errorf("chat completion 返回错误: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("chat completion 无返回内容")
	}
	return cr.Choices[0].Message.Content, nil
}

type identityWire struct {
	IdentityID string  `json:"identity_id"`
	Confidence float64 `json:"confidence"`
}

func (s *AIService) recognizeIdentity(ctx context.Context, transcript string, identities []model.Identity) (string, float64) {
	userPrompt := fmt.Sprintf(prompt.IdentityUserTemplate, formatIdentities(identities), transcript)

	text, err := s.call(ctx, prompt.IdentitySystemPrompt, userPrompt)
	if err != nil {
		return "", 0
	}

	var wire identityWire
	if err := json.Unmarshal([]byte(stripCodeFences(text)), &wire); err != nil {
		// JSON 解析失败，重试一次
		retryText, retryErr := s.call(ctx, prompt.IdentitySystemPrompt, userPrompt+prompt.IdentityRetrySuffix)
		if retryErr != nil {
			return "", 0
		}
		if err := json.Unmarshal([]byte(stripCodeFences(retryText)), &wire); err != nil {
			return "", 0
		}
	}

	// 校验返回的身份 id 必须在候选列表中
	if wire.IdentityID != "" && !identityInList(wire.IdentityID, identities) {
		return "", 0
	}
	if wire.Confidence < 0 {
		wire.Confidence = 0
	}
	if wire.Confidence > 1 {
		wire.Confidence = 1
	}
	return wire.IdentityID, wire.Confidence
}

type todoWire struct {
	Text  string  `json:"text"`
	DueAt *string `json:"due_at"`
}

type commitmentWire struct {
	Text  string  `json:"text"`
	From  string  `json:"from"`
	To    string  `json:"to"`
	DueAt *string `json:"due_at"`
}

type extractionWire struct {
	Todos       []todoWire       `json:"todos"`
	Commitments []commitmentWire `json:"commitments"`
	Notes       []string         `json:"notes"`
}

func (s *AIService) extract(ctx context.Context, transcript string) model.ExtractedData {
	userPrompt := fmt.Sprintf(prompt.ExtractionUserTemplate, transcript)

	text, err := s.call(ctx, prompt.ExtractionSystemPrompt, userPrompt)
	if err != nil {
		return emptyExtracted()
	}

	var wire extractionWire
	if err := json.Unmarshal([]byte(stripCodeFences(text)), &wire); err != nil {
		// JSON 解析失败，重试一次
		retryText, retryErr := s.call(ctx, prompt.ExtractionSystemPrompt, userPrompt+prompt.ExtractionRetrySuffix)
		if retryErr != nil {
			return emptyExtracted()
		}
		if err := json.Unmarshal([]byte(stripCodeFences(retryText)), &wire); err != nil {
			return emptyExtracted()
		}
	}

	out := model.ExtractedData{
		Todos:       make([]model.Todo, 0, len(wire.Todos)),
		Commitments: make([]model.Commitment, 0, len(wire.Commitments)),
		Notes:       make([]string, 0, len(wire.Notes)),
	}
	for _, t := range wire.Todos {
		out.Todos = append(out.Todos, model.Todo{Text: strings.TrimSpace(t.Text), DueAt: parseDueAt(t.DueAt)})
	}
	for _, c := range wire.Commitments {
		out.Commitments = append(out.Commitments, model.Commitment{
			Text:  strings.TrimSpace(c.Text),
			From:  strings.TrimSpace(c.From),
			To:    strings.TrimSpace(c.To),
			DueAt: parseDueAt(c.DueAt),
		})
	}
	if wire.Notes != nil {
		out.Notes = wire.Notes
	}
	return out
}

func emptyExtracted() model.ExtractedData {
	return model.ExtractedData{
		Todos:       []model.Todo{},
		Commitments: []model.Commitment{},
		Notes:       []string{},
	}
}

// stripCodeFences 去掉 LLM 常见的 markdown 代码块包裹，并截取首个 JSON 对象。
func stripCodeFences(s string) string {
	const fence = "\x60\x60\x60"
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, fence) {
		lines := strings.Split(s, "\n")
		if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), fence) {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == fence {
			lines = lines[:len(lines)-1]
		}
		s = strings.Join(lines, "\n")
	}
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return s
}

func parseDueAt(v *string) *time.Time {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	s := strings.TrimSpace(*v)
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			t = t.UTC()
			return &t
		}
	}
	return nil
}

func formatIdentities(identities []model.Identity) string {
	if len(identities) == 0 {
		return "（无候选身份）"
	}
	var b strings.Builder
	for _, id := range identities {
		fmt.Fprintf(&b, "- %s: %s\n", id.ID, id.Name)
	}
	return b.String()
}

func identityInList(id string, identities []model.Identity) bool {
	for _, v := range identities {
		if v.ID == id {
			return true
		}
	}
	return false
}
