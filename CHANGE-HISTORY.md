# Change History

## 2026-06-04

### All-Chinese text migrated to English
- Code comments, console output, log messages, Markdown reports, config comments all changed to English
- Removed all emoji characters from output and reports
- Created `CHANGE-HISTORY.md` and linked from `README.md`

### Detailed per-request markdown report
- Added `DetailedMarkdownReporter` that generates one detailed report per test case
- Each report lists every individual request with its TTFT, latency, prompt (truncated) and response (truncated)
- Output goes to `reports/detailed/`

### Interactive menu introduced
- Menu with 3 options: Manual Test, Auto Benchmark, Exit
- Manual Test mode: user enters a question, selects concurrency (1-8), views streaming (single) or token-count progress (multi)
- Manual test results saved to `reports/manual/manual_*.md` with per-request metrics and full response
- Single-concurrency mode streams tokens in real time; multi-concurrency buffers per request and shows live token counts

### Provider-specific configuration files
- `config.yaml` simplified to only `provider` + `config_file`
- Each provider has its own dedicated YAML file (e.g. `config.huzhouai.yaml`)
- New provider types can be added without touching shared config structs
- Added `Model()` method to Provider interface for display/report use

### Runner error visibility improved
- Errors from API calls are now printed to stderr in real time
- Log file entries written for every request failure
- MinTTFT sentinel value fixed (no longer shows `9223372036` when all requests fail)

### Concurrent output interleaving fixed
- Multi-concurrency manual test no longer mixes output across requests
- Each request buffers independently; progress shown via single-line `\r` updates
- Full response written only to Markdown report, never to console

### HuzhouAI SSE parsing implemented
- Dify-style SSE streaming parser with TTFT measurement
- Real-time chunk callback support for streaming display
- Token usage extracted from `message_end` metadata

### Config example files updated
- `config.huzhouai.yaml.example`: removed `model` (API does not need it)
- `config.openai.yaml.example`: added as skeleton for future use
- Both provider config files now gitignored via `config.*.yaml` pattern

### Initial project scaffold
- Go module initialized with `go-llm-api-benchmark`
- Provider interface: `Name()`, `Model()`, `Chat()`, `ChatStream()`, `ListModels()`
- HuzhouAIProvider implementation for Dify-style `POST /chat-messages`
- OpenAIProvider skeleton
- `Runner` with concurrent concurrency-level execution and metric aggregation
- ConsoleReporter for real-time table output (tabwriter)
- MarkdownReporter for summary reports per test file
- Logger with timestamped file + stderr multi-writer
- Config loader for both main and provider-specific YAML files
- Fully typed data structures in `types` package
- Cross-compile `build.bat` producing Windows and Linux x64 binaries
- Test cases: `quick-smoke.yaml`, `full-benchmark.yaml`, `stress-test.yaml`
- All concurrency levels limited to `[1, 2, 4, 8]`

## Technical Details

- Language: Go 1.21+
- Dependencies: `gopkg.in/yaml.v3`
- All text (code, comments, logs, reports, configs) in English
- Zero emoji in any output
