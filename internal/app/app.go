// Package app orchestrates the full benchmark workflow:
// load config -> create Provider -> show interactive menu -> manual test / auto benchmark.
package app

import (
	"bufio"
	"context"
	"fmt"
	"log"
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

// Run is the application entry point.
func Run() error {
	// 1. Load main config
	cfgPath := config.DefaultConfigPath
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("config file %s not found; copy config.yaml.example to config.yaml and fill in your settings", cfgPath)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Create Provider (loads provider-specific config automatically)
	p, providerModel, err := createProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	fmt.Printf("Provider: %s | Model: %s\n", p.Name(), providerModel)

	// 3. Create logger
	appLog, err := logger.New("log")
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// 4. Interactive menu
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n================================================")
		fmt.Println("  LLM API Benchmark")
		fmt.Println("================================================")
		fmt.Println("  1. Manual Test")
		fmt.Println("  2. Auto Benchmark")
		fmt.Println("  3. Exit")
		fmt.Println("================================================")
		fmt.Print("Select [1-3]: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			runManualTest(p, providerModel)
		case "2":
			runAutoBenchmark(p, providerModel, appLog)
		case "3":
			fmt.Println("Goodbye.")
			return nil
		default:
			fmt.Println("Invalid input. Please enter 1, 2, or 3.")
		}
	}
}

// runAutoBenchmark runs the automated benchmark flow using test-case YAML files.
func runAutoBenchmark(p provider.Provider, model string, appLog *log.Logger) {
	testFiles, err := config.LoadTestCases(config.DefaultTestCasesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to load test cases: %v\n", err)
		return
	}
	if len(testFiles) == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: no .yaml files found in test-cases directory")
		return
	}
	fmt.Printf("Test files: %d\n\n", len(testFiles))

	r := runner.NewRunner(p, appLog)
	consoleRep := report.NewConsoleReporter()
	markdownRep := report.NewMarkdownReporter(reportDir)
	detailedRep := report.NewDetailedMarkdownReporter(filepath.Join(reportDir, "detailed"))

	for _, tf := range testFiles {
		appLog.Printf("---------- Test file: %s ----------", tf.FileName)
		fmt.Printf("================================================\n")
		fmt.Printf("  File: %s\n", tf.FileName)
		fmt.Printf("================================================\n")

		var results []*types.TestCaseResult

		for _, tc := range tf.Suite.TestCases {
			if tc.Model == "" {
				tc.Model = model
			}
			tc.Prompt = buildPrompt(&tc)

			appLog.Printf("Test [%s] concurrency=%v max_tokens=%d prompt_len=%d",
				tc.Name, tc.Concurrency, tc.MaxTokens, len(tc.Prompt))
			fmt.Printf("\n  > Test: %s\n", tc.Name)
			fmt.Printf("    Prompt: %s\n", truncateString(tc.Prompt, 80))

			result := r.RunTestCase(context.Background(), &tc)
			results = append(results, result)

			consoleRep.Report(result)
		}

		baseName := strings.TrimSuffix(tf.FileName, filepath.Ext(tf.FileName))
		markdownRep.ReportAll(results, baseName+".md")

		for _, result := range results {
			detailName := fmt.Sprintf("%s_%s_detailed.md",
				baseName, sanitizeName(result.TestCase.Name))
			detailedRep.Report(result, detailName)
		}
	}

	appLog.Printf("========== Auto Benchmark Completed ==========")
	fmt.Println("\nAuto benchmark completed.")
}

// createProvider loads the provider-specific config and instantiates the corresponding Provider.
func createProvider(cfg *types.Config) (provider.Provider, string, error) {
	configFile := cfg.ConfigFile
	if configFile == "" {
		configFile = fmt.Sprintf("config.%s.yaml", cfg.Provider)
	}

	switch cfg.Provider {
	case types.ProviderHuzhouAI:
		var hzCfg types.HuzhouAIConfig
		if err := config.LoadProviderConfig(configFile, &hzCfg); err != nil {
			return nil, "", err
		}
		if hzCfg.BaseURL == "" {
			return nil, "", fmt.Errorf("base_url must not be empty (check %s)", configFile)
		}
		return provider.NewHuzhouAIProvider(&hzCfg), hzCfg.Model, nil

	case types.ProviderOpenAI:
		var oaiCfg types.OpenAIConfig
		if err := config.LoadProviderConfig(configFile, &oaiCfg); err != nil {
			return nil, "", err
		}
		if oaiCfg.BaseURL == "" {
			return nil, "", fmt.Errorf("base_url must not be empty (check %s)", configFile)
		}
		return provider.NewOpenAIProvider(&oaiCfg), oaiCfg.Model, nil

	default:
		return nil, "", fmt.Errorf("unsupported provider type: %s (options: openai, huzhouai)", cfg.Provider)
	}
}

// buildPrompt generates the final prompt text based on test case config.
func buildPrompt(tc *types.TestCase) string {
	if tc.Prompt != "" {
		return tc.Prompt
	}
	if tc.NumWords > 0 {
		return generateRandomWords(tc.NumWords)
	}
	return "Tell me a story."
}

// generateRandomWords generates a random prompt with the given word count.
func generateRandomWords(n int) string {
	if n <= 0 {
		return ""
	}

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

// sanitizeName replaces special characters with underscores for safe filenames.
func sanitizeName(name string) string {
	r := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return r.Replace(name)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
