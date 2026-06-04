// Package types defines shared data structures and constants for the benchmark tool.
package types

import "time"

// ProviderType enumerates supported LLM provider types.
type ProviderType string

const (
	ProviderOpenAI   ProviderType = "openai"
	ProviderHuzhouAI ProviderType = "huzhouai"
)

// Config is the top-level configuration from config.yaml.
// It only declares the provider type and the path to its dedicated config file.
type Config struct {
	Provider   ProviderType `yaml:"provider"`
	ConfigFile string       `yaml:"config_file"`
}

// HuzhouAIConfig holds Huzhou AI platform-specific configuration.
// Corresponds to config.huzhouai.yaml.
type HuzhouAIConfig struct {
	BaseURL   string `yaml:"base_url"`
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model,omitempty"` // for display/report only, not sent to API
	DeptID    string `yaml:"dept_id"`
	ProjectID string `yaml:"project_id"`
}

// OpenAIConfig holds OpenAI-compatible API configuration.
// Corresponds to config.openai.yaml.
type OpenAIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

// TestSuite maps a single test-case YAML file.
type TestSuite struct {
	TestCases []TestCase `yaml:"test_cases"`
}

// TestCase defines a single benchmark scenario.
type TestCase struct {
	Name        string `yaml:"name"`
	Model       string `yaml:"model,omitempty"` // optional, overrides config model
	Prompt      string `yaml:"prompt"`
	MaxTokens   int    `yaml:"max_tokens"`
	NumWords    int    `yaml:"num_words"` // >0 generates a random prompt of this many words
	Concurrency []int  `yaml:"concurrency"`
}

// TestFile pairs a YAML filename with its parsed TestSuite.
type TestFile struct {
	FileName string
	Suite    TestSuite
}

// ChatRequest is sent to a Provider for a single chat completion.
type ChatRequest struct {
	Model     string
	Prompt    string
	MaxTokens int
	User      string
}

// ChatResult holds the complete result of a single chat request.
type ChatResult struct {
	RequestIndex     int
	Prompt           string
	Response         string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	TTFT             time.Duration // Time to First Token
	TotalLatency     time.Duration // Total request duration
	Error            error
}

// ConcurrencyResult aggregates results for one concurrency level.
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
	Requests             []ChatResult // per-request details for detailed reports
}

// TestCaseResult holds the full result for one test case across all concurrency levels.
type TestCaseResult struct {
	TestCase    TestCase
	Concurrency []ConcurrencyResult
}
