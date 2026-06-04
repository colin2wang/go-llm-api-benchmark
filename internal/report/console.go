package report

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"go-llm-api-benchmark/internal/types"
)

// ConsoleReporter prints benchmark result tables to the terminal in real time.
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
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("  Test Case: %s\n", name)
	fmt.Println(strings.Repeat("-", 70))
}

func (cr *ConsoleReporter) printTable(rows []types.ConcurrencyResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	fmt.Fprintln(w, "Concurrency\tGen Throughput(tok/s)\tPrompt Throughput(tok/s)\tMin TTFT(s)\tMax TTFT(s)\tAvg TTFT(s)\tSuccess/Total")
	fmt.Fprintln(w, "-----------\t---------------------\t-----------------------\t------------\t------------\t------------\t-------------")

	for _, r := range rows {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%d/%d\n",
			r.Concurrency,
			fmtFloat(r.GenerationThroughput),
			fmtFloat(r.PromptThroughput),
			fmtDur(r.MinTTFT),
			fmtDur(r.MaxTTFT),
			fmtDur(r.AvgTTFT),
			r.TotalRequests, r.TotalRequests+r.FailedRequests,
		)
	}

	w.Flush()
}
