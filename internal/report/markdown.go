package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-llm-api-benchmark/internal/types"
)

// MarkdownReporter 生成 Markdown 格式的压测报告文件
type MarkdownReporter struct {
	outputDir string
}

func NewMarkdownReporter(outputDir string) *MarkdownReporter {
	return &MarkdownReporter{outputDir: outputDir}
}

// ReportAll 将多个测试用例结果写入一个 Markdown 文件中
// filename 是报告文件名（不含路径），例如: quick-smoke.md
func (mr *MarkdownReporter) ReportAll(results []*types.TestCaseResult, filename string) {
	if len(results) == 0 {
		return
	}

	// 确保输出目录存在
	if err := os.MkdirAll(mr.outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建报告目录失败: %v\n", err)
		return
	}

	body := mr.generateBody(results)

	reportPath := filepath.Join(mr.outputDir, filename)
	if err := os.WriteFile(reportPath, []byte(body), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入报告文件 %s 失败: %v\n", reportPath, err)
		return
	}

	fmt.Printf("  报告已保存: %s\n", reportPath)
}

func (mr *MarkdownReporter) generateBody(results []*types.TestCaseResult) string {
	var b strings.Builder

	b.WriteString("# LLM API 压测报告\n\n")
	b.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

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

	b.WriteString(fmt.Sprintf("## 测试用例: %s\n\n", tc.Name))
	b.WriteString("### 测试参数\n\n")
	b.WriteString("| 参数 | 值 |\n|------|----|\n")
	b.WriteString(fmt.Sprintf("| Prompt | `%s` |\n", truncateStr(tc.Prompt, 60)))
	b.WriteString(fmt.Sprintf("| Max Tokens | %d |\n", tc.MaxTokens))
	b.WriteString(fmt.Sprintf("| Num Words | %d |\n", tc.NumWords))
	b.WriteString(fmt.Sprintf("| 并发级别 | %s |\n\n", joinIntList(tc.Concurrency, ", ")))

	b.WriteString("### 结果\n\n")
	b.WriteString("| 并发数 | 生成吞吐量(tokens/s) | 提示吞吐量(tokens/s) | 最小TTFT(s) | 最大TTFT(s) | 平均TTFT(s) | 成功/总数 |\n")
	b.WriteString("|--------|----------------------|----------------------|-------------|-------------|-------------|-----------|\n")

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

// --- 辅助函数 ---

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func joinIntList(ints []int, sep string) string {
	parts := make([]string, len(ints))
	for i, v := range ints {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, sep)
}

func fmtFloat(f float64) string {
	if f == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", f)
}

func fmtDur(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.4f", d.Seconds())
}
