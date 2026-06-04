// Package provider defines the generic LLM API Provider interface and implementations.
package provider

import (
	"context"
	"go-llm-api-benchmark/internal/types"
)

// ChunkCallback is called for each text chunk received from a streaming SSE response.
type ChunkCallback func(chunk string)

// Provider is the abstract interface for LLM API backends.
// Each API style implements its own Provider.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// Model returns the model name (for display/report; may be empty if not applicable).
	Model() string

	// Chat sends a streaming chat request and returns the complete result.
	Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResult, error)

	// ChatStream sends a streaming chat request and pushes each token chunk via callback.
	ChatStream(ctx context.Context, req *types.ChatRequest, onChunk ChunkCallback) (*types.ChatResult, error)

	// ListModels returns the list of available models.
	ListModels(ctx context.Context) ([]string, error)
}
