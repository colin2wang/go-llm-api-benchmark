// Package provider 定义了 LLM API 的通用 Provider 接口及其实现。
package provider

import (
	"context"
	"go-llm-api-benchmark/internal/types"
)

// Provider 是 LLM API 的抽象接口
// 每个不同的 API 风格实现一个 Provider。
type Provider interface {
	// Name 返回 Provider 标识名
	Name() string

	// Chat 发送一次聊天请求（流式），返回完整结果
	// 实现方负责在内部处理 SSE 流式解析，并测量 TTFT。
	Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResult, error)

	// ListModels 列出可用的模型列表
	ListModels(ctx context.Context) ([]string, error)
}
