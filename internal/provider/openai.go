package provider

import (
	"context"

	"go-llm-api-benchmark/internal/types"
)

// OpenAIProvider 标准 OpenAI 兼容 API 的 Provider 实现。
// API 端点: POST {base_url}/chat/completions
// 暂为骨架实现，后续按需完善。
type OpenAIProvider struct {
	config *types.Config
}

func NewOpenAIProvider(cfg *types.Config) *OpenAIProvider {
	return &OpenAIProvider{config: cfg}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResult, error) {
	// TODO: 实现标准 OpenAI chat completion 流式调用
	// 参考: https://platform.openai.com/docs/api-reference/chat/create
	return nil, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	// TODO: 实现 GET {base_url}/models
	return []string{p.config.Model}, nil
}
