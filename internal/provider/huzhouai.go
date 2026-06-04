package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-llm-api-benchmark/internal/types"
)

// HuzhouAIProvider 湖州算力平台 Dify 风格 API 的 Provider 实现。
// API 端点: POST {base_url}/chat-messages
// 流式响应: SSE (text/event-stream)
type HuzhouAIProvider struct {
	config *types.Config
	client *http.Client
}

func NewHuzhouAIProvider(cfg *types.Config) *HuzhouAIProvider {
	return &HuzhouAIProvider{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (p *HuzhouAIProvider) Name() string { return "huzhouai" }

func (p *HuzhouAIProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResult, error) {
	// 构建请求体（Dify Chat Messages API 格式）
	bodyMap := map[string]interface{}{
		"inputs":          map[string]interface{}{},
		"query":           req.Prompt,
		"response_mode":   "streaming",
		"conversation_id": "",
		"user":            req.User,
		"files":           []interface{}{},
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 构建 HTTP 请求
	apiURL := strings.TrimRight(p.config.BaseURL, "/") + "/chat-messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// 湖州平台专属请求头
	if p.config.HuzhouAI != nil {
		httpReq.Header.Set("x-motu-deptid", p.config.HuzhouAI.DeptID)
		httpReq.Header.Set("x-motu-projectId", p.config.HuzhouAI.ProjectID)
	}

	// 发送请求
	startTime := time.Now()
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyDump, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 返回非 200 状态码: %d, body: %s", resp.StatusCode, string(bodyDump))
	}

	return p.parseSSEStream(resp.Body, startTime)
}

// parseSSEStream 解析 Dify SSE 流式响应并提取指标
func (p *HuzhouAIProvider) parseSSEStream(body io.Reader, startTime time.Time) (*types.ChatResult, error) {
	result := &types.ChatResult{}
	var fullResponse strings.Builder
	var firstTokenTime time.Time
	wordCount := 0

	scanner := bufio.NewScanner(body)
	// SSE 行可能较长，设置较大的缓冲区
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和注释行
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 只处理 data: 前缀的行
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")

		// 解析事件
		var event struct {
			Event    string `json:"event"`
			Answer   string `json:"answer"`
			Metadata *struct {
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				} `json:"usage"`
			} `json:"metadata"`
		}

		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			continue // 跳过无法解析的行
		}

		// 流结束事件：提取 token 用量信息
		if event.Event == "message_end" || event.Event == "error" {
			if event.Metadata != nil && event.Metadata.Usage != nil {
				result.PromptTokens = event.Metadata.Usage.PromptTokens
				result.CompletionTokens = event.Metadata.Usage.CompletionTokens
				result.TotalTokens = event.Metadata.Usage.TotalTokens
			}
			break
		}

		// 消息块事件：累积响应内容
		if event.Answer != "" {
			if firstTokenTime.IsZero() {
				firstTokenTime = time.Now()
				result.TTFT = firstTokenTime.Sub(startTime)
			}
			fullResponse.WriteString(event.Answer)
			// 按空格分词近似统计 output token 数
			wordCount += len(strings.Fields(event.Answer))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 SSE 流失败: %w", err)
	}

	result.Response = fullResponse.String()
	result.TotalLatency = time.Since(startTime)

	// 如果 API 未返回 token 用量，使用近似值
	if result.CompletionTokens == 0 {
		result.CompletionTokens = wordCount
	}
	if result.PromptTokens == 0 {
		// 粗略估计：平均每词 1.3 个 token
		result.PromptTokens = int(float64(len(strings.Fields(result.Response))) * 1.3)
		if result.PromptTokens == 0 {
			result.PromptTokens = len(strings.Fields(result.Response))
		}
	}
	result.TotalTokens = result.PromptTokens + result.CompletionTokens

	return result, nil
}

func (p *HuzhouAIProvider) ListModels(ctx context.Context) ([]string, error) {
	// 湖州算力平台暂不支持模型列表查询，返回配置中指定的模型
	return []string{p.config.Model}, nil
}
