package service

import (
	"context"
	"iter"
	"strings"

	adkmodel "google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

// fakeLLM 是一个可脚本化的 model.LLM 实现，用于测试。
type fakeLLM struct {
	name string
	fn   func(req *adkmodel.LLMRequest) (string, error)
}

func (f *fakeLLM) Name() string { return f.name }

func (f *fakeLLM) GenerateContent(_ context.Context, req *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		text, err := f.fn(req)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(&adkmodel.LLMResponse{
			Content: &genai.Content{
				Role:  "model",
				Parts: []*genai.Part{{Text: text}},
			},
		}, nil)
	}
}

// fakeSystemText 提取请求中的系统指令文本。
func fakeSystemText(req *adkmodel.LLMRequest) string {
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

// newFakeAIService 构建一个按系统提示词分发到身份识别/信息提取的脚本化 AI 服务。
func newFakeAIService(identityFn, extractFn func() (string, error)) *AIService {
	return newAIService(&fakeLLM{
		name: "test-model",
		fn: func(req *adkmodel.LLMRequest) (string, error) {
			if strings.Contains(fakeSystemText(req), "身份识别") {
				return identityFn()
			}
			return extractFn()
		},
	}, 0.6)
}
