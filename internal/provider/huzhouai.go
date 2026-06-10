package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-llm-api-benchmark/internal/types"
)

// HuzhouAIProvider implements the Provider interface for the Huzhou AI platform (Dify-style API).
// API endpoint: POST {base_url}/chat-messages
// Streaming: SSE (text/event-stream)
//
// Session management:
//   - First request sends conversation_id = ""; the API returns a new conversation_id.
//   - Subsequent requests reuse the conversation_id from the latest response.
//   - Call ResetSession() to clear the stored id so the next request starts a new conversation.
type HuzhouAIProvider struct {
	config         *types.HuzhouAIConfig
	client         *http.Client
	conversationID string
	mu             sync.Mutex
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

// ResetSession clears the conversation_id so the next request starts a brand-new conversation
// (sends conversation_id = "").
func (p *HuzhouAIProvider) ResetSession() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conversationID = ""
	log.Printf("[huzhouai] session reset (next request will start a new conversation)")
	return ""
}

// ConversationID returns the current conversation_id from the last response,
// or empty string if no conversation has been started yet.
func (p *HuzhouAIProvider) ConversationID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conversationID
}

// setConversationID stores a conversation_id received from the API response.
func (p *HuzhouAIProvider) setConversationID(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id != "" && id != p.conversationID {
		p.conversationID = id
		fmt.Printf("\n  [session] conversation_id <- %s\n", id)
		log.Printf("[huzhouai] conversation_id updated from response: %s", id)
	}
}

// Chat sends a request and returns the complete result (streaming handled internally).
func (p *HuzhouAIProvider) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResult, error) {
	return p.ChatStream(ctx, req, nil)
}

// ChatStream sends a request and pushes each token chunk in real time via callback.
//
// The conversation_id is automatically managed:
//   - First call sends conversation_id = "" (empty string), the API creates a new conversation.
//   - The response's conversation_id is extracted and stored for subsequent calls.
//   - Call ResetSession() to clear the stored id so the next request starts a new conversation.
func (p *HuzhouAIProvider) ChatStream(ctx context.Context, req *types.ChatRequest, onChunk ChunkCallback) (*types.ChatResult, error) {
	// --- session management ---
	p.mu.Lock()
	cid := p.conversationID
	p.mu.Unlock()

	// --- build request body ---
	inputs := p.config.Inputs
	if inputs == nil {
		inputs = map[string]interface{}{}
	}

	user := p.config.User
	if user == "" {
		user = "unknown"
	}

	responseMode := p.config.ResponseMode
	if responseMode == "" {
		responseMode = "streaming"
	}

	bodyMap := map[string]interface{}{
		"inputs":          inputs,
		"query":           req.Prompt,
		"response_mode":   responseMode,
		"conversation_id": cid,
		"user":            user,
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
	if responseMode == "streaming" {
		httpReq.Header.Set("Accept", "text/event-stream")
	} else {
		httpReq.Header.Set("Accept", "application/json")
	}
	httpReq.Header.Set("x-motu-deptid", p.config.DeptID)
	httpReq.Header.Set("x-motu-projectId", p.config.ProjectID)

	// --- log full request (headers + body) to console and log file ---
	// Console: show a copy-paste ready curl command with full auth
	bodyPretty := string(bodyBytes)
	var curlBuf strings.Builder
	curlBuf.WriteString(fmt.Sprintf("curl -X POST '%s' \\\n", apiURL))
	curlBuf.WriteString(fmt.Sprintf("  --header 'Authorization: Bearer %s' \\\n", p.config.APIKey))
	curlBuf.WriteString("  --header 'Content-Type: application/json' \\\n")
	curlBuf.WriteString(fmt.Sprintf("  --header 'x-motu-deptid: %s' \\\n", p.config.DeptID))
	curlBuf.WriteString(fmt.Sprintf("  --header 'x-motu-projectId: %s' \\\n", p.config.ProjectID))
	// Escape single quotes inside the body for safe shell single-quote wrapping
	escapedBody := strings.ReplaceAll(bodyPretty, "'", "'\\''")
	curlBuf.WriteString(fmt.Sprintf("  --data-raw '%s'\n", escapedBody))
	fmt.Printf("\n--- Curl ---\n%s---------------\n", curlBuf.String())

	log.Printf("[huzhouai] %s %s | body: %s", httpReq.Method, apiURL, bodyPretty)

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

	if responseMode == "streaming" {
		result, parseErr := p.parseSSEStream(resp.Body, startTime, onChunk)
		if result != nil && parseErr == nil {
			p.logTiming("streaming", result)
		}
		return result, parseErr
	}
	result, parseErr := p.parseBlockingResponse(resp.Body, startTime, onChunk)
	if result != nil && parseErr == nil {
		p.logTiming("blocking", result)
	}
	return result, parseErr
}

// logTiming prints and logs timing metrics appropriate to the response mode.
func (p *HuzhouAIProvider) logTiming(mode string, r *types.ChatResult) {
	genTime := r.TotalLatency - r.TTFT
	var genSpeed float64
	if genTime > 0 && r.CompletionTokens > 0 {
		genSpeed = float64(r.CompletionTokens) / genTime.Seconds()
	}
	var overallSpeed float64
	if r.TotalLatency > 0 && r.CompletionTokens > 0 {
		overallSpeed = float64(r.CompletionTokens) / r.TotalLatency.Seconds()
	}

	if mode == "streaming" {
		log.Printf("[huzhouai] timing(streaming): ttft=%.3fs latency=%.3fs gen_time=%.3fs gen_speed=%.1f tok/s overall=%.1f tok/s tokens=%d",
			r.TTFT.Seconds(), r.TotalLatency.Seconds(), genTime.Seconds(), genSpeed, overallSpeed, r.CompletionTokens)
	} else {
		log.Printf("[huzhouai] timing(blocking): latency=%.3fs speed=%.1f tok/s tokens=%d",
			r.TotalLatency.Seconds(), overallSpeed, r.CompletionTokens)
	}
}

// parseBlockingResponse parses a single Dify blocking (non-SSE) JSON response.
func (p *HuzhouAIProvider) parseBlockingResponse(body io.Reader, startTime time.Time, onChunk ChunkCallback) (*types.ChatResult, error) {
	result := &types.ChatResult{}

	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read blocking response body: %w", err)
	}

	var resp struct {
		Event          string `json:"event"`
		ConversationID string `json:"conversation_id"`
		Answer         string `json:"answer"`
		Metadata       *struct {
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse blocking response JSON: %w", err)
	}

	// Extract conversation_id
	if resp.ConversationID != "" {
		p.setConversationID(resp.ConversationID)
	}

	// Full answer (no streaming chunks — TTFT is not applicable for blocking mode)
	result.TTFT = 0

	if resp.Answer != "" && onChunk != nil {
		onChunk(resp.Answer)
	}

	result.Response = resp.Answer
	result.TotalLatency = time.Since(startTime)

	// Token usage
	if resp.Metadata != nil && resp.Metadata.Usage != nil {
		result.PromptTokens = resp.Metadata.Usage.PromptTokens
		result.CompletionTokens = resp.Metadata.Usage.CompletionTokens
		result.TotalTokens = resp.Metadata.Usage.TotalTokens
	} else {
		wordCount := len(strings.Fields(resp.Answer))
		result.CompletionTokens = wordCount
		result.PromptTokens = int(float64(wordCount) * 1.3)
		if result.PromptTokens == 0 {
			result.PromptTokens = wordCount
		}
		result.TotalTokens = result.PromptTokens + result.CompletionTokens
	}

	return result, nil
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
			Event          string `json:"event"`
			ConversationID string `json:"conversation_id"`
			Answer         string `json:"answer"`
			Metadata       *struct {
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

		// Extract conversation_id from the response (first non-empty value wins)
		if event.ConversationID != "" {
			p.setConversationID(event.ConversationID)
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
