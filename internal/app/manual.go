// Package app contains the manual test mode:
// user enters a question -> selects concurrency -> views streaming results -> report saved.
package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-llm-api-benchmark/internal/provider"
	"go-llm-api-benchmark/internal/types"
)

const manualReportDir = "reports" + string(os.PathSeparator) + "manual"

// runManualTest runs the interactive manual test flow.
func runManualTest(p provider.Provider, cfgModel string) {
	reader := bufio.NewReader(os.Stdin)

	// Display conversation_id if the provider supports it
	printSessionInfo(p)

	for {
		fmt.Println("\n================================================")
		fmt.Println("  Manual Test Mode")
		fmt.Println("================================================")

		fmt.Print("\nEnter your question (type /back to return to menu, /session new to reset conversation): ")
		question, _ := reader.ReadString('\n')
		question = strings.TrimSpace(question)
		if question == "" {
			continue
		}
		if question == "/back" {
			return
		}

		// Handle /session new command
		if handleSessionCommand(p, question) {
			printSessionInfo(p)
			continue
		}

		concurrency := 1
		for {
			fmt.Print("Concurrency (1-8, default 1): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				concurrency = 1
				break
			}
			n, err := fmt.Sscanf(input, "%d", &concurrency)
			if err != nil || n != 1 || concurrency < 1 || concurrency > 8 {
				fmt.Println("  Please enter a number between 1 and 8.")
				continue
			}
			break
		}

		fmt.Println()
		runSingleManualTest(p, cfgModel, question, concurrency)
		printSessionInfo(p)
	}
}

// printSessionInfo displays the current conversation_id if the provider supports it.
func printSessionInfo(p provider.Provider) {
	if hz, ok := p.(*provider.HuzhouAIProvider); ok {
		if cid := hz.ConversationID(); cid != "" {
			fmt.Printf("  [session] conversation_id: %s\n", cid)
		} else {
			fmt.Println("  [session] no active session (send a question to start one)")
		}
	}
}

// handleSessionCommand checks if the input is a session command and acts on it.
// Returns true if the input was a session command (caller should skip normal processing).
func handleSessionCommand(p provider.Provider, input string) bool {
	if strings.HasPrefix(input, "/session new") {
		if hz, ok := p.(*provider.HuzhouAIProvider); ok {
			hz.ResetSession()
			fmt.Println("  [session] conversation reset (next request will start a new conversation)")
		} else {
			fmt.Println("  [session] session management not supported by this provider")
		}
		return true
	}
	return false
}

// runSingleManualTest executes a single manual test.
//
// concurrency = 1 : streams output to console in real time.
// concurrency > 1 : buffers per-request responses, shows token progress only,
//
//	full responses go directly to the Markdown report.
func runSingleManualTest(p provider.Provider, cfgModel, question string, concurrency int) {
	fmt.Printf("Sending %d concurrent request(s)...\n\n", concurrency)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		results  = make([]*types.ChatResult, concurrency)
		progress = make([]int, concurrency)
		buffers  = make([]*strings.Builder, concurrency)
	)

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		buffers[i] = &strings.Builder{}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := &types.ChatRequest{
				Model:     cfgModel,
				Prompt:    question,
				MaxTokens: 2048,
				User:      fmt.Sprintf("manual-%d", idx),
			}

			var result *types.ChatResult
			var err error

			if concurrency == 1 {
				result, err = p.ChatStream(context.Background(), req, func(chunk string) {
					fmt.Print(chunk)
				})
			} else {
				result, err = p.ChatStream(context.Background(), req, func(chunk string) {
					mu.Lock()
					buffers[idx].WriteString(chunk)
					progress[idx] += len(strings.Fields(chunk))
					fmt.Print("\r")
					for j := 0; j < concurrency; j++ {
						fmt.Printf("[#%d] %d tokens  ", j+1, progress[j])
					}
					mu.Unlock()
				})
			}

			mu.Lock()
			if err != nil {
				errMsg := fmt.Sprintf("request #%d failed: %v", idx+1, err)
				fmt.Printf("\n[#%d] FAIL %s\n", idx+1, errMsg)
				results[idx] = &types.ChatResult{
					RequestIndex: idx,
					Prompt:       question,
					Error:        fmt.Errorf("%s", errMsg),
				}
				mu.Unlock()
				return
			}
			result.RequestIndex = idx
			result.Prompt = question
			result.Response = buffers[idx].String()
			results[idx] = result

			fmt.Printf("\n[#%d] OK %d tokens (TTFT: %.2fs, latency: %.2fs)\n",
				idx+1, result.CompletionTokens, result.TTFT.Seconds(), result.TotalLatency.Seconds())
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)
	fmt.Printf("\nTotal duration: %.2fs\n", totalDuration.Seconds())

	saveManualReport(question, concurrency, results, totalDuration)
}

// saveManualReport saves manual test results to a timestamped Markdown file.
func saveManualReport(question string, concurrency int, results []*types.ChatResult, totalDuration time.Duration) {
	if err := os.MkdirAll(manualReportDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create report directory: %v\n", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	sanitized := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_").Replace(question)
	if len(sanitized) > 40 {
		sanitized = sanitized[:40]
	}
	filename := fmt.Sprintf("manual_%s_%s.md", timestamp, sanitized)
	path := filepath.Join(manualReportDir, filename)

	var b strings.Builder
	b.WriteString("# Manual Test Report\n\n")
	b.WriteString(fmt.Sprintf("**Time**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("## Parameters\n\n")
	b.WriteString("| Param | Value |\n|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| Question | `%s` |\n", question))
	b.WriteString(fmt.Sprintf("| Concurrency | %d |\n", concurrency))
	b.WriteString(fmt.Sprintf("| Total Duration | %.2fs |\n\n", totalDuration.Seconds()))

	b.WriteString("## Per-Request Results\n\n")
	for i, r := range results {
		status := "OK"
		errMsg := ""
		if r == nil || r.Error != nil {
			status = "FAIL"
			if r != nil && r.Error != nil {
				errMsg = r.Error.Error()
			} else {
				errMsg = "no result returned"
			}
		}

		b.WriteString(fmt.Sprintf("### Request #%d [%s]\n\n", i+1, status))
		b.WriteString("| Metric | Value |\n|--------|-------|\n")

		if errMsg != "" {
			b.WriteString(fmt.Sprintf("| Status | FAIL |\n"))
			b.WriteString(fmt.Sprintf("| Error | `%s` |\n", errMsg))
		} else {
			b.WriteString(fmt.Sprintf("| TTFT | %.4fs |\n", r.TTFT.Seconds()))
			b.WriteString(fmt.Sprintf("| Latency | %.4fs |\n", r.TotalLatency.Seconds()))
			b.WriteString(fmt.Sprintf("| Prompt Tokens | %d |\n", r.PromptTokens))
			b.WriteString(fmt.Sprintf("| Completion Tokens | %d |\n", r.CompletionTokens))
			b.WriteString(fmt.Sprintf("| Total Tokens | %d |\n", r.TotalTokens))
		}

		b.WriteString("\n**Response**:\n\n")
		if r != nil && r.Response != "" {
			b.WriteString(fmt.Sprintf("```\n%s\n```\n", r.Response))
		} else {
			b.WriteString("*(empty)*\n")
		}
		b.WriteString("\n")
	}

	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write report: %v\n", err)
		return
	}

	fmt.Printf("\nReport saved: %s\n", path)
}
