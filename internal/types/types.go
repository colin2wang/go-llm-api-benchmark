// Package types 定义整个工具共享的数据结构和常量。
package types

import "time"

// ProviderType 支持的 Provider 类型
type ProviderType string

const (
	ProviderOpenAI   ProviderType = "openai"
	ProviderHuzhouAI ProviderType = "huzhouai"
)

// Config 顶层配置，对应 config.yaml
type Config struct {
	Provider ProviderType    `yaml:"provider"`
	BaseURL  string          `yaml:"base_url"`
	APIKey   string          `yaml:"api_key"`
	Model    string          `yaml:"model"`
	HuzhouAI *HuzhouAIConfig `yaml:"huzhouai,omitempty"`
}

// HuzhouAIConfig 湖州算力平台专属配置
type HuzhouAIConfig struct {
	DeptID    string `yaml:"dept_id"`
	ProjectID string `yaml:"project_id"`
}

// TestSuite 对应一个 test-case YAML 文件
type TestSuite struct {
	TestCases []TestCase `yaml:"test_cases"`
}

// TestCase 单个测试场景
type TestCase struct {
	Name        string `yaml:"name"`
	Model       string `yaml:"model,omitempty"` // 可选，覆盖 config 中的 model
	Prompt      string `yaml:"prompt"`
	MaxTokens   int    `yaml:"max_tokens"`
	NumWords    int    `yaml:"num_words"` // >0 时按字数随机生成 prompt
	Concurrency []int  `yaml:"concurrency"`
}

// TestFile 一个 YAML 文件及其解析出的测试套件
type TestFile struct {
	FileName string
	Suite    TestSuite
}

// ChatRequest 发送给 Provider 的单次聊天请求参数
type ChatRequest struct {
	Model     string
	Prompt    string
	MaxTokens int
	User      string
}

// ChatResult 单次聊天的完整结果
type ChatResult struct {
	Response         string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	TTFT             time.Duration // Time to First Token
	TotalLatency     time.Duration // 请求总耗时
	Error            error
}

// ConcurrencyResult 某一并发级别下的聚合结果
type ConcurrencyResult struct {
	Concurrency          int
	GenerationThroughput float64 // tokens/s
	PromptThroughput     float64 // tokens/s
	MinTTFT              time.Duration
	MaxTTFT              time.Duration
	AvgTTFT              time.Duration
	TotalRequests        int
	FailedRequests       int
	TotalDuration        time.Duration
}

// TestCaseResult 单个 TestCase 在各并发级别的完整结果
type TestCaseResult struct {
	TestCase    TestCase
	Concurrency []ConcurrencyResult
}
