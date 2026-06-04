// Package runner 实现并发压测引擎。
//
// 对每个并发级别，启动对应数量的 goroutine 同时发送请求，
// 采集 TTFT、完成时间等指标，最后聚合生成吞吐量数据。
package runner

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"go-llm-api-benchmark/internal/provider"
	"go-llm-api-benchmark/internal/types"
)

// Runner 并发压测引擎
type Runner struct {
	provider provider.Provider
	logger   *log.Logger
}

// NewRunner 创建压测引擎
func NewRunner(p provider.Provider, l *log.Logger) *Runner {
	return &Runner{provider: p, logger: l}
}

// RunTestCase 执行一个测试用例在所有并发级别下的压测
func (r *Runner) RunTestCase(ctx context.Context, tc *types.TestCase) *types.TestCaseResult {
	result := &types.TestCaseResult{
		TestCase: *tc,
	}

	for _, concurrency := range tc.Concurrency {
		cr := r.runConcurrencyLevel(ctx, tc, concurrency)
		result.Concurrency = append(result.Concurrency, *cr)
	}

	return result
}

// runConcurrencyLevel 在指定并发级别下执行压测
//
// 策略：启动 concurrency 个 worker，每个 worker 发送 1 次请求，
// 记录所有请求的指标后聚合。
func (r *Runner) runConcurrencyLevel(ctx context.Context, tc *types.TestCase, concurrency int) *types.ConcurrencyResult {
	results := make([]*types.ChatResult, concurrency)
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := &types.ChatRequest{
				Model:     tc.Model,
				Prompt:    tc.Prompt,
				MaxTokens: tc.MaxTokens,
				User:      fmt.Sprintf("benchmark-%d-%d", concurrency, idx),
			}

			chatResult, err := r.provider.Chat(ctx, req)
			if err != nil {
				r.logger.Printf("[ERROR] 请求失败 (并发=%d): %v", concurrency, err)
				results[idx] = &types.ChatResult{Error: fmt.Errorf("请求失败: %v", err)}
				return
			}
			results[idx] = chatResult
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	return aggregateResults(concurrency, results, totalDuration)
}

// aggregateResults 聚合原始结果生成 ConcurrencyResult
func aggregateResults(concurrency int, results []*types.ChatResult, duration time.Duration) *types.ConcurrencyResult {
	cr := &types.ConcurrencyResult{
		Concurrency:   concurrency,
		TotalDuration: duration,
		MinTTFT:       time.Duration(math.MaxInt64),
	}

	var (
		totalOutputTokens int
		totalInputTokens  int
		ttftSum           time.Duration
		ttftCount         int
	)

	for _, r := range results {
		if r == nil {
			cr.FailedRequests++
			continue
		}
		if r.Error != nil {
			cr.FailedRequests++
			continue
		}

		cr.TotalRequests++

		totalOutputTokens += r.CompletionTokens
		totalInputTokens += r.PromptTokens

		if r.TTFT > 0 {
			if r.TTFT < cr.MinTTFT {
				cr.MinTTFT = r.TTFT
			}
			if r.TTFT > cr.MaxTTFT {
				cr.MaxTTFT = r.TTFT
			}
			ttftSum += r.TTFT
			ttftCount++
		}
	}

	// 处理无成功请求的情况
	if cr.TotalRequests == 0 {
		cr.MinTTFT = 0
		return cr
	}

	// 计算吞吐量
	durationSec := duration.Seconds()
	if durationSec > 0 {
		cr.GenerationThroughput = float64(totalOutputTokens) / durationSec
		cr.PromptThroughput = float64(totalInputTokens) / durationSec
	}

	// 计算平均 TTFT
	if ttftCount > 0 {
		cr.AvgTTFT = time.Duration(int64(ttftSum) / int64(ttftCount))
	} else {
		cr.MinTTFT = 0
		cr.MaxTTFT = 0
	}

	return cr
}
