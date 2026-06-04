package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-llm-api-benchmark/internal/types"
)

// DetailedMarkdownReporter generates a per-test-case detailed Markdown report
// containing per-request timing, prompt, and response.
type DetailedMarkdownReporter struct {
	outputDir string
}

func NewDetailedMarkdownReporter(outputDir string) *DetailedMarkdownReporter {
	return &DetailedMarkdownReporter{outputDir: outputDir}
}

// Report generates one detailed report for a single test case.
func (dr *DetailedMarkdownReporter) Report(result *types.TestCaseResult, filename string) {
	if err := os.MkdirAll(dr.outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create detailed report directory: %v\n", err)
		return
	}

	body := dr.generateBody(result)

	reportPath := filepath.Join(dr.outputDir, filename)
	if err := os.WriteFile(reportPath, []byte(body), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write detailed report %s: %v\n", reportPath, err)
		return
	}

	fmt.Printf("  Detailed report: %s\n", reportPath)
}

func (dr *DetailedMarkdownReporter) generateBody(result *types.TestCaseResult) string {
	var b strings.Builder

	tc := result.TestCase
	b.WriteString(fmt.Sprintf("# Detailed Benchmark Report: %s\n\n", tc.Name))
	b.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	b.WriteString("## Parameters\n\n")
	b.WriteString("| Param | Value |\n|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| Prompt | `%s` |\n", truncateStr(tc.Prompt, 80)))
	b.WriteString(fmt.Sprintf("| Max Tokens | %d |\n", tc.MaxTokens))
	b.WriteString(fmt.Sprintf("| Concurrency Levels | %s |\n\n", joinIntList(tc.Concurrency, ", ")))

	for _, cr := range result.Concurrency {
		b.WriteString(fmt.Sprintf("## Concurrency: %d\n\n", cr.Concurrency))
		b.WriteString(fmt.Sprintf("- Total requests: %d, Failed: %d, Duration: %.2fs\n", cr.TotalRequests+cr.FailedRequests, cr.FailedRequests, cr.TotalDuration.Seconds()))
		b.WriteString(fmt.Sprintf("- Gen throughput: %s tok/s, Prompt throughput: %s tok/s\n\n",
			fmtFloat(cr.GenerationThroughput), fmtFloat(cr.PromptThroughput)))

		b.WriteString("| # | Status | TTFT(s) | Latency(s) | Out Tokens | Prompt (first 80) | Response (first 80) |\n")
		b.WriteString("|---|--------|---------|------------|------------|-------------------|--------------------|\n")

		for i, req := range cr.Requests {
			status := "OK"
			if req.Error != nil {
				status = "FAIL"
			}

			ttft := fmtDur(req.TTFT)
			latency := fmtDur(req.TotalLatency)
			outTok := "-"
			if req.CompletionTokens > 0 {
				outTok = fmt.Sprintf("%d", req.CompletionTokens)
			}

			promptStr := truncateStr(req.Prompt, 80)
			respStr := truncateStr(req.Response, 80)

			if req.Error != nil {
				respStr = fmt.Sprintf("`%s`", req.Error.Error())
			}

			b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | `%s` | %s |\n",
				i+1, status, ttft, latency, outTok, promptStr, respStr))
		}
		b.WriteString("\n")
	}

	return b.String()
}
