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

// HuzhouAIProvider implements the Provider interface for the Huzhou AI platform (Dify-style API).
// API endpoint: POST {base_url}/chat-messages
// Streaming: SSE (text/event-stream)
type HuzhouAIProvider struct {
	config *types.HuzhouAIConfig
	client *http.Client
}

func NewHuzhouAIProvider(cfg *types.HuzhouAIConfig) *HuzhouAIProvider {
	return &HuzhouAIProvider{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (p *HuzhouAIProvider) Name() string { return "huzhouai" }
func (p *HuzhouAIProvider) Model() string {
	if p.config.Model != "" {
		return p.config.Model
	}
	return p.Name()
}

// Chat sends a request and returns the complete result (streaming handled internally).
func (p *HuzhouAIProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResult, error) {
	return p.ChatStream(ctx, req, nil)
}

// ChatStream sends a request and pushes each token chunk in real time via callback.
func (p *HuzhouAIProvider) ChatStream(ctx context.Context, req *types.ChatRequest, onChunk ChunkCallback) (*types.ChatResult, error) {
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
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiURL := strings.TrimRight(p.config.BaseURL, "/") + "/chat-messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-motu-deptid", p.config.DeptID)
	httpReq.Header.Set("x-motu-projectId", p.config.ProjectID)

	startTime := time.Now()
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyDump, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(bodyDump))
	}

	return p.parseSSEStream(resp.Body, startTime, onChunk)
}

// parseSSEStream parses the Dify SSE streaming response and invokes onChunk if set.
func (p *HuzhouAIProvider) parseSSEStream(body io.Reader, startTime time.Time, onChunk ChunkCallback) (*types.ChatResult, error) {
	result := &types.ChatResult{}
	var fullResponse strings.Builder
	var firstTokenTime time.Time
	wordCount := 0

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")

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
			continue
		}

		// Stream end event: extract token usage
		if event.Event == "message_end" || event.Event == "error" {
			if event.Metadata != nil && event.Metadata.Usage != nil {
				result.PromptTokens = event.Metadata.Usage.PromptTokens
				result.CompletionTokens = event.Metadata.Usage.CompletionTokens
				result.TotalTokens = event.Metadata.Usage.TotalTokens
			}
			break
		}

		// Message chunk event: accumulate response
		if event.Answer != "" {
			if firstTokenTime.IsZero() {
				firstTokenTime = time.Now()
				result.TTFT = firstTokenTime.Sub(startTime)
			}
			fullResponse.WriteString(event.Answer)
			wordCount += len(strings.Fields(event.Answer))

			if onChunk != nil {
				onChunk(event.Answer)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read SSE stream: %w", err)
	}

	result.Response = fullResponse.String()
	result.TotalLatency = time.Since(startTime)

	if result.CompletionTokens == 0 {
		result.CompletionTokens = wordCount
	}
	if result.PromptTokens == 0 {
		result.PromptTokens = int(float64(len(strings.Fields(result.Response))) * 1.3)
		if result.PromptTokens == 0 {
			result.PromptTokens = len(strings.Fields(result.Response))
		}
	}
	result.TotalTokens = result.PromptTokens + result.CompletionTokens

	return result, nil
}

func (p *HuzhouAIProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{p.config.Model}, nil
}
