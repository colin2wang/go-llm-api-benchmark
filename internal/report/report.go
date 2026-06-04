// Package report 提供压测结果的多种输出格式。
package report

import "go-llm-api-benchmark/internal/types"

// Reporter 报告输出接口（终端实时输出适用）
type Reporter interface {
	// Report 输出单个测试用例的结果
	Report(result *types.TestCaseResult)
}
