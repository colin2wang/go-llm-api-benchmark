// Package runner implements the concurrent benchmark engine.
//
// For each concurrency level, it spawns concurrent goroutines,
// collects TTFT and latency metrics, and aggregates throughput data.
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

// Runner drives concurrent benchmark execution.
type Runner struct {
	provider provider.Provider
	logger   *log.Logger
}

// NewRunner creates a new Runner.
func NewRunner(p provider.Provider, l *log.Logger) *Runner {
	return &Runner{provider: p, logger: l}
}

// RunTestCase runs a test case at all configured concurrency levels.
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

// runConcurrencyLevel runs requests at a given concurrency level.
//
// Strategy: spawn N goroutines (N = concurrency), each sends 1 request.
// All metrics are collected and then aggregated.
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
				r.logger.Printf("[ERROR] request failed (concurrency=%d): %v", concurrency, err)
				results[idx] = &types.ChatResult{
					RequestIndex: idx,
					Prompt:       req.Prompt,
					Error:        fmt.Errorf("request failed: %v", err),
				}
				return
			}
			chatResult.RequestIndex = idx
			chatResult.Prompt = req.Prompt
			results[idx] = chatResult
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	return aggregateResults(concurrency, results, totalDuration)
}

// aggregateResults aggregates raw chat results into a ConcurrencyResult.
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
			cr.Requests = append(cr.Requests, types.ChatResult{
				RequestIndex: len(cr.Requests),
				Error:        fmt.Errorf("request did not complete"),
			})
			continue
		}
		if r.Error != nil {
			cr.FailedRequests++
			cr.Requests = append(cr.Requests, *r)
			continue
		}

		cr.TotalRequests++
		cr.Requests = append(cr.Requests, *r)

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

	// No successful requests
	if cr.TotalRequests == 0 {
		cr.MinTTFT = 0
		return cr
	}

	// Throughput calculation
	durationSec := duration.Seconds()
	if durationSec > 0 {
		cr.GenerationThroughput = float64(totalOutputTokens) / durationSec
		cr.PromptThroughput = float64(totalInputTokens) / durationSec
	}

	// Average TTFT
	if ttftCount > 0 {
		cr.AvgTTFT = time.Duration(int64(ttftSum) / int64(ttftCount))
	} else {
		cr.MinTTFT = 0
		cr.MaxTTFT = 0
	}

	return cr
}
