package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chaojixinren/pulse/internal/config"
	"github.com/chaojixinren/pulse/internal/model"
	apperrors "github.com/chaojixinren/pulse/pkg/errors"
	"github.com/chaojixinren/pulse/pkg/prompt"

	adkmodel "google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/openaimodel"
	"google.golang.org/genai"
)

const (
	defaultAIBaseURL             = "https://api.openai.com/v1"
	defaultAIModel               = "gpt-4o-mini"
	defaultAIConfidenceThreshold = 0.6
)

// AIService 负责把转写文本交给 LLM，完成身份识别与信息提取。
// 使用 adk-go 的 openaimodel 接入 OpenAI 兼容模型，做显式两阶段编排：先身份识别，再信息提取。
type AIService struct {
	model     adkmodel.LLM
	threshold float64
	// 以下保存全局默认，供 per-user 覆盖时回退。
	apiKey    string
	baseURL   string
	modelName string
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
	llm, err := openaimodel.NewModel(context.Background(), modelName, &openaimodel.ClientConfig{
		APIKey:  cfg.AIAPIKey,
		BaseURL: baseURL,
	})
	if err != nil {
		// modelName 已在上方保证非空，此分支理论上不可达；兜底以防 SDK 行为变化。
		panic(fmt.Sprintf("创建 OpenAI 模型失败: %v", err))
	}
	threshold := cfg.AIConfidenceThreshold
	if threshold <= 0 {
		threshold = defaultAIConfidenceThreshold
	}
	s := newAIService(llm, threshold)
	s.apiKey = cfg.AIAPIKey
	s.baseURL = baseURL
	s.modelName = modelName
	return s
}

// NewAIServiceWithModel 注入自定义 model.LLM（测试/依赖注入用）。
func NewAIServiceWithModel(llm adkmodel.LLM, threshold float64) *AIService {
	return newAIService(llm, threshold)
}

// newAIService 构建 AI 服务，允许注入自定义 model.LLM（测试用）。
func newAIService(llm adkmodel.LLM, threshold float64) *AIService {
	if threshold <= 0 {
		threshold = defaultAIConfidenceThreshold
	}
	return &AIService{
		model:     llm,
		threshold: threshold,
	}
}

// AiOverrides 是单次 AI 分析的 per-user 覆盖项；空值字段回退全局默认。
type AiOverrides struct {
	APIKey    string
	BaseURL   string
	Model     string
	Threshold float64
}

// AnalyzeTranscript 转写文本 + 用户身份列表 → 分析结果（全局默认配置）。
// 两阶段：先身份识别（候选身份作为标签），再信息提取。
func (s *AIService) AnalyzeTranscript(ctx context.Context, transcript string, identities []model.Identity) (*model.AnalysisResult, error) {
	return s.AnalyzeTranscriptWithOverrides(ctx, transcript, identities, nil)
}

// AnalyzeTranscriptWithOverrides 在全局默认之上应用 per-user 覆盖后执行分析。
// 仅当用户自定义了 APIKey / BaseURL / Model 任一字段时才动态构建临时模型，否则复用全局实例。
func (s *AIService) AnalyzeTranscriptWithOverrides(ctx context.Context, transcript string, identities []model.Identity, o *AiOverrides) (*model.AnalysisResult, error) {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return nil, apperrors.NewBadRequest("转写文本为空，无法分析")
	}

	llm := s.model
	threshold := s.threshold
	if o != nil {
		if o.APIKey != "" || o.BaseURL != "" || o.Model != "" {
			var err error
			llm, err = s.buildLLM(o.APIKey, o.BaseURL, o.Model)
			if err != nil {
				return nil, apperrors.WrapInternal(err)
			}
		}
		if o.Threshold > 0 {
			threshold = o.Threshold
		}
	}

	identityID, confidence := s.recognizeIdentity(ctx, llm, transcript, identities)
	extracted := s.extract(ctx, llm, transcript)

	return buildAnalysisResult(identityID, confidence, threshold, extracted), nil
}

// buildLLM 用覆盖项构建 OpenAI 兼容模型；空字段回退全局默认。
func (s *AIService) buildLLM(apiKey, baseURL, modelName string) (adkmodel.LLM, error) {
	if apiKey == "" {
		apiKey = s.apiKey
	}
	if baseURL == "" {
		baseURL = s.baseURL
	}
	if modelName == "" {
		modelName = s.modelName
	}
	return openaimodel.NewModel(context.Background(), modelName, &openaimodel.ClientConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
	})
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

// call 通过指定的 model.LLM 执行一次生成，返回模型输出的文本。
func (s *AIService) call(ctx context.Context, llm adkmodel.LLM, systemPrompt, userPrompt string) (string, error) {
	req := &adkmodel.LLMRequest{
		Contents: []*genai.Content{
			{Role: "user", Parts: []*genai.Part{{Text: userPrompt}}},
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemPrompt}}},
		},
	}

	var out strings.Builder
	for resp, err := range llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", err
		}
		if resp == nil || resp.Content == nil {
			continue
		}
		for _, part := range resp.Content.Parts {
			if part != nil && part.Text != "" {
				out.WriteString(part.Text)
			}
		}
	}
	if strings.TrimSpace(out.String()) == "" {
		return "", fmt.Errorf("模型无返回内容")
	}
	return out.String(), nil
}

type identityWire struct {
	IdentityID string  `json:"identity_id"`
	Confidence float64 `json:"confidence"`
}

func (s *AIService) recognizeIdentity(ctx context.Context, llm adkmodel.LLM, transcript string, identities []model.Identity) (string, float64) {
	userPrompt := fmt.Sprintf(prompt.IdentityUserTemplate, formatIdentities(identities), transcript)

	text, err := s.call(ctx, llm, prompt.IdentitySystemPrompt, userPrompt)
	if err != nil {
		return "", 0
	}

	var wire identityWire
	if err := json.Unmarshal([]byte(stripCodeFences(text)), &wire); err != nil {
		// JSON 解析失败，重试一次
		retryText, retryErr := s.call(ctx, llm, prompt.IdentitySystemPrompt, userPrompt+prompt.IdentityRetrySuffix)
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

func (s *AIService) extract(ctx context.Context, llm adkmodel.LLM, transcript string) model.ExtractedData {
	userPrompt := fmt.Sprintf(prompt.ExtractionUserTemplate, transcript)

	text, err := s.call(ctx, llm, prompt.ExtractionSystemPrompt, userPrompt)
	if err != nil {
		return emptyExtracted()
	}

	var wire extractionWire
	if err := json.Unmarshal([]byte(stripCodeFences(text)), &wire); err != nil {
		// JSON 解析失败，重试一次
		retryText, retryErr := s.call(ctx, llm, prompt.ExtractionSystemPrompt, userPrompt+prompt.ExtractionRetrySuffix)
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
