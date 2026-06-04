package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-llm-api-benchmark/internal/types"
)

// MarkdownReporter generates a summary Markdown report file with aggregated results.
type MarkdownReporter struct {
	outputDir string
}

func NewMarkdownReporter(outputDir string) *MarkdownReporter {
	return &MarkdownReporter{outputDir: outputDir}
}

// ReportAll writes results for all test cases in a test file into one Markdown file.
func (mr *MarkdownReporter) ReportAll(results []*types.TestCaseResult, filename string) {
	if len(results) == 0 {
		return
	}

	if err := os.MkdirAll(mr.outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create report directory: %v\n", err)
		return
	}

	body := mr.generateBody(results)

	reportPath := filepath.Join(mr.outputDir, filename)
	if err := os.WriteFile(reportPath, []byte(body), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write report file %s: %v\n", reportPath, err)
		return
	}

	fmt.Printf("  Report saved: %s\n", reportPath)
}

func (mr *MarkdownReporter) generateBody(results []*types.TestCaseResult) string {
	var b strings.Builder

	b.WriteString("# LLM API Benchmark Report\n\n")
	b.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	for i, result := range results {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		mr.writeTestCase(&b, result)
	}

	return b.String()
}

func (mr *MarkdownReporter) writeTestCase(b *strings.Builder, result *types.TestCaseResult) {
	tc := result.TestCase

	b.WriteString(fmt.Sprintf("## Test Case: %s\n\n", tc.Name))
	b.WriteString("### Parameters\n\n")
	b.WriteString("| Param | Value |\n|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| Prompt | `%s` |\n", truncateStr(tc.Prompt, 60)))
	b.WriteString(fmt.Sprintf("| Max Tokens | %d |\n", tc.MaxTokens))
	b.WriteString(fmt.Sprintf("| Num Words | %d |\n", tc.NumWords))
	b.WriteString(fmt.Sprintf("| Concurrency Levels | %s |\n\n", joinIntList(tc.Concurrency, ", ")))

	b.WriteString("### Results\n\n")
	b.WriteString("| Concurrency | Gen Throughput(tok/s) | Prompt Throughput(tok/s) | Min TTFT(s) | Max TTFT(s) | Avg TTFT(s) | Success/Total |\n")
	b.WriteString("|-------------|----------------------|-------------------------|-------------|-------------|-------------|---------------|\n")

	for _, cr := range result.Concurrency {
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s | %d/%d |\n",
			cr.Concurrency,
			fmtFloat(cr.GenerationThroughput),
			fmtFloat(cr.PromptThroughput),
			fmtDur(cr.MinTTFT),
			fmtDur(cr.MaxTTFT),
			fmtDur(cr.AvgTTFT),
			cr.TotalRequests, cr.TotalRequests+cr.FailedRequests,
		))
	}

	b.WriteString("\n")
}
