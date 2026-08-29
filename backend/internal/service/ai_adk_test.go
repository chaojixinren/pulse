package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chaojixinren/pulse/internal/config"

	adkmodel "google.golang.org/adk/v2/model"
)

// TestCallBuildsLLMRequest 验证 call() 将 system/user 正确映射到 adk-go 的 LLMRequest。
func TestCallBuildsLLMRequest(t *testing.T) {
	var captured *adkmodel.LLMRequest
	m := &fakeLLM{
		name: "m",
		fn: func(req *adkmodel.LLMRequest) (string, error) {
			captured = req
			return "模型输出", nil
		},
	}
	svc := newAIService(m, 0.6)

	text, err := svc.call(context.Background(), m, "系统提示", "用户输入")
	require.NoError(t, err)
	assert.Equal(t, "模型输出", text)

	require.NotNil(t, captured)
	require.NotNil(t, captured.Config)
	require.NotNil(t, captured.Config.SystemInstruction)
	require.NotEmpty(t, captured.Config.SystemInstruction.Parts)
	assert.Equal(t, "系统提示", captured.Config.SystemInstruction.Parts[0].Text)

	require.Len(t, captured.Contents, 1)
	assert.Equal(t, "user", captured.Contents[0].Role)
	require.Len(t, captured.Contents[0].Parts, 1)
	assert.Equal(t, "用户输入", captured.Contents[0].Parts[0].Text)
}

// TestCallEmptyContentReturnsError 验证模型返回空内容时 call() 报错。
func TestCallEmptyContentReturnsError(t *testing.T) {
	m := &fakeLLM{
		name: "m",
		fn:   func(req *adkmodel.LLMRequest) (string, error) { return "", nil },
	}
	svc := newAIService(m, 0.6)
	_, err := svc.call(context.Background(), m, "s", "u")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无返回内容")
}

// TestCallPropagatesModelError 验证模型报错时 call() 透传错误。
func TestCallPropagatesModelError(t *testing.T) {
	boom := errors.New("boom")
	m := &fakeLLM{
		name: "m",
		fn:   func(req *adkmodel.LLMRequest) (string, error) { return "", boom },
	}
	svc := newAIService(m, 0.6)
	_, err := svc.call(context.Background(), m, "s", "u")
	require.ErrorIs(t, err, boom)
}

// TestNewAIServiceDefaults 验证 NewAIService 使用默认配置构建（模型名回填，不 panic）。
func TestNewAIServiceDefaults(t *testing.T) {
	svc := NewAIService(&config.Config{})
	require.NotNil(t, svc)
	assert.Equal(t, 0.6, svc.threshold)
	assert.Equal(t, "gpt-4o-mini", svc.model.Name())
}

// TestNewAIServiceWithModel 验证依赖注入构造函数。
func TestNewAIServiceWithModel(t *testing.T) {
	m := &fakeLLM{name: "fake"}
	svc := NewAIServiceWithModel(m, 0.8)
	require.NotNil(t, svc)
	assert.Equal(t, 0.8, svc.threshold)
	assert.Equal(t, m, svc.model)
}
