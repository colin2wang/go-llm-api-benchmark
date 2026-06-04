package provider

import (
	"context"

	"go-llm-api-benchmark/internal/types"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs.
// API endpoint: POST {base_url}/chat/completions
// This is a skeleton; full implementation to be added later.
type OpenAIProvider struct {
	config *types.OpenAIConfig
}

func NewOpenAIProvider(cfg *types.OpenAIConfig) *OpenAIProvider {
	return &OpenAIProvider{config: cfg}
}

func (p *OpenAIProvider) Name() string  { return "openai" }
func (p *OpenAIProvider) Model() string { return p.config.Model }

func (p *OpenAIProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResult, error) {
	return nil, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, req *types.ChatRequest, onChunk ChunkCallback) (*types.ChatResult, error) {
	return nil, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{p.config.Model}, nil
}
