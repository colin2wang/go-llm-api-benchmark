// Package app 编排整个压测流程：
// 加载配置 → 创建 Provider → 遍历测试用例 → 执行压测 → 输出报告。
package app

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-llm-api-benchmark/internal/config"
	"go-llm-api-benchmark/internal/logger"
	"go-llm-api-benchmark/internal/provider"
	"go-llm-api-benchmark/internal/report"
	"go-llm-api-benchmark/internal/runner"
	"go-llm-api-benchmark/internal/types"
)

const reportDir = "reports"

// Run 是应用入口函数
func Run() error {
	// 1. 加载配置文件
	cfgPath := config.DefaultConfigPath
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件 %s 不存在，请将 config.yaml.example 复制为 config.yaml 并填入真实参数", cfgPath)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 创建 Provider
	p, err := createProvider(cfg)
	if err != nil {
		return fmt.Errorf("创建 Provider 失败: %w", err)
	}
	fmt.Printf("✅ Provider: %s | Model: %s | BaseURL: %s\n", p.Name(), cfg.Model, cfg.BaseURL)

	// 3. 加载测试用例
	testFiles, err := config.LoadTestCases(config.DefaultTestCasesDir)
	if err != nil {
		return fmt.Errorf("加载测试用例失败: %w", err)
	}
	if len(testFiles) == 0 {
		return fmt.Errorf("test-cases 目录下没有找到 .yaml 文件")
	}
	fmt.Printf("📂 测试文件数: %d\n\n", len(testFiles))

	// 4. 创建日志记录器
	log, err := logger.New("log")
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}

	// 5. 创建 Runner 和 Reporter
	r := runner.NewRunner(p, log)
	consoleRep := report.NewConsoleReporter()
	markdownRep := report.NewMarkdownReporter(reportDir)

	// 6. 逐个测试文件执行
	for _, tf := range testFiles {
		fmt.Printf("═══════════════════════════════════════════════\n")
		fmt.Printf("  文件: %s\n", tf.FileName)
		fmt.Printf("═══════════════════════════════════════════════\n")

		var results []*types.TestCaseResult

		for _, tc := range tf.Suite.TestCases {
			// 合并配置：从 config 继承 model
			if tc.Model == "" {
				tc.Model = cfg.Model
			}
			// 生成随机 prompt
			tc.Prompt = buildPrompt(&tc)

			fmt.Printf("\n  ▶ 测试: %s\n", tc.Name)
			fmt.Printf("    Prompt: %s\n", truncateString(tc.Prompt, 80))

			result := r.RunTestCase(context.Background(), &tc)
			results = append(results, result)

			// 实时输出到终端
			consoleRep.Report(result)
		}

		// 每个测试文件生成一份 Markdown 报告
		baseName := strings.TrimSuffix(tf.FileName, filepath.Ext(tf.FileName))
		markdownRep.ReportAll(results, baseName+".md")
	}

	return nil
}

// createProvider 根据配置创建对应的 Provider 实例
func createProvider(cfg *types.Config) (provider.Provider, error) {
	switch cfg.Provider {
	case types.ProviderHuzhouAI:
		return provider.NewHuzhouAIProvider(cfg), nil
	case types.ProviderOpenAI:
		return provider.NewOpenAIProvider(cfg), nil
	default:
		return nil, fmt.Errorf("不支持的 Provider 类型: %s (可选: openai, huzhouai)", cfg.Provider)
	}
}

// buildPrompt 根据 TestCase 配置生成实际的 prompt 文本
func buildPrompt(tc *types.TestCase) string {
	if tc.Prompt != "" {
		return tc.Prompt
	}
	if tc.NumWords > 0 {
		return generateRandomWords(tc.NumWords)
	}
	return "Tell me a story."
}

// generateRandomWords 生成指定单词数的随机文本
func generateRandomWords(n int) string {
	if n <= 0 {
		return ""
	}

	// 常用英文单词池，用于生成多样化的测试 prompt
	wordPool := []string{
		"the", "a", "an", "and", "or", "but", "in", "on", "at", "to",
		"for", "of", "with", "by", "from", "as", "is", "was", "are", "were",
		"has", "have", "had", "do", "does", "did", "will", "would", "can", "could",
		"should", "may", "might", "shall", "need", "dare", "ought", "used",
		"this", "that", "these", "those", "what", "which", "who", "whom", "whose",
		"when", "where", "why", "how", "all", "each", "every", "both", "few", "many",
		"some", "any", "no", "none", "most", "other", "such", "only", "own", "same",
		"time", "year", "people", "way", "day", "man", "woman", "child", "world",
		"life", "hand", "part", "place", "case", "week", "company", "system",
		"program", "work", "government", "number", "night", "point", "home",
		"water", "room", "mother", "area", "money", "story", "fact", "month",
		"lot", "right", "study", "book", "eye", "job", "word", "business",
		"issue", "side", "kind", "head", "house", "service", "friend", "father",
		"power", "hour", "game", "line", "end", "member", "law", "car", "city",
		"community", "name", "president", "team", "minute", "idea", "kid",
		"body", "information", "back", "parent", "face", "others", "level",
		"office", "door", "health", "person", "art", "war", "history", "party",
		"result", "change", "morning", "reason", "research", "girl", "guy",
		"moment", "air", "teacher", "force", "education", "explain", "describe",
		"analyze", "compare", "contrast", "discuss", "summarize", "evaluate",
		"provide", "list", "detail", "elaborate", "define", "demonstrate",
		"illustrate", "outline", "review", "examine", "explore", "investigate",
	}

	words := make([]string, n)
	for i := 0; i < n; i++ {
		words[i] = wordPool[rand.Intn(len(wordPool))]
	}
	return strings.Join(words, " ")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
