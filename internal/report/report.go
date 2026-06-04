// Package report provides multiple output formats for benchmark results.
package report

import "go-llm-api-benchmark/internal/types"

// Reporter is the interface for console-oriented result output.
type Reporter interface {
	// Report outputs a single test case result.
	Report(result *types.TestCaseResult)
}
