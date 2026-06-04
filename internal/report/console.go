package report

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"go-llm-api-benchmark/internal/types"
)

// ConsoleReporter 在终端实时输出压测结果表格
type ConsoleReporter struct{}

func NewConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{}
}

func (cr *ConsoleReporter) Report(result *types.TestCaseResult) {
	cr.printHeader(result.TestCase.Name)
	cr.printTable(result.Concurrency)
	fmt.Fprintln(os.Stdout)
}

func (cr *ConsoleReporter) printHeader(name string) {
	fmt.Println(strings.Repeat("─", 70))
	fmt.Printf("  测试用例: %s\n", name)
	fmt.Println(strings.Repeat("─", 70))
}

func (cr *ConsoleReporter) printTable(rows []types.ConcurrencyResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	// 表头
	fmt.Fprintln(w, "并发数\t生成吞吐量(tokens/s)\t提示吞吐量(tokens/s)\t最小TTFT(s)\t最大TTFT(s)\t平均TTFT(s)\t成功/总请求")
	fmt.Fprintln(w, "------\t---------------------\t---------------------\t-----------\t-----------\t-----------\t-----------")

	for _, r := range rows {
		genTP := formatFloat(r.GenerationThroughput)
		promptTP := formatFloat(r.PromptThroughput)
		minTTFT := formatDur(r.MinTTFT)
		maxTTFT := formatDur(r.MaxTTFT)
		avgTTFT := formatDur(r.AvgTTFT)

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%d/%d\n",
			r.Concurrency, genTP, promptTP, minTTFT, maxTTFT, avgTTFT,
			r.TotalRequests, r.TotalRequests+r.FailedRequests,
		)
	}

	w.Flush()
}

func formatFloat(f float64) string {
	if f == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", f)
}

func formatDur(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.4f", d.Seconds())
}
