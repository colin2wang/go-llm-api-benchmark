# LLM API Benchmark Tool

A Go-based performance benchmarking tool for OpenAI-compatible LLM APIs. This tool measures and analyzes API throughput, generation speed, and token processing capabilities across different concurrency levels.

## Features

- Dynamic Concurrency Testing - Test across multiple concurrency levels
- Comprehensive Metrics - Measure Generation Throughput, Prompt Throughput, and Time-to-First-Token (TTFT)
- Flexible Configuration - YAML-based test cases and configuration
- Markdown Reporting - Generate detailed performance reports
- Multi-Provider Support - Works with OpenAI and HuzhouAI APIs
- Pre-built Binaries - Ready-to-use binaries for Linux and Windows

## Quick Start

### Prerequisites

- Go 1.21+ (for building from source)
- An API key from your LLM provider

### Installation

Option 1: Use pre-built binaries

```bash
# Linux
./bin/llm-api-benchmark_linux_x64

# Windows
./bin/llm-api-benchmark_win_x64.exe
```

Option 2: Build from source

```bash
go build -o llm-api-benchmark main.go
```

### Configuration

1. Copy the example config file:

```bash
cp config.yaml.example config.yaml
```

2. Edit config.yaml with your API credentials:

```yaml
provider: openai                    # or huzhouai
base_url: https://api.openai.com/v1
api_key: your-api-key-here
model: gpt-3.5-turbo
```

### Running Benchmarks

```bash
# Run all test cases
./llm-api-benchmark

# Quick smoke test
./llm-api-benchmark --test-case test-cases/quick-smoke.yaml
```

## Test Cases

Test cases are defined in YAML files under test-cases/:

```yaml
test_cases:
  - name: Short Text - Low Concurrency
    prompt: Explain quantum computing in simple terms.
    max_tokens: 256
    concurrency: [1, 2, 4]

  - name: Long Text - High Concurrency
    num_words: 512                  # Generate random prompt with 512 words
    max_tokens: 1024
    concurrency: [8, 16, 32]
```

## Performance Metrics

| Metric | Description |
|--------|-------------|
| Generation Throughput | Tokens generated per second |
| Prompt Throughput | Input token processing speed |
| Min TTFT | Minimum Time-to-First-Token |
| Max TTFT | Maximum Time-to-First-Token |

## Project Structure

```
├── bin/                    # Pre-built binaries
├── docs/                   # Documentation
├── internal/               # Source code
│   ├── app/               # Application logic
│   ├── config/            # Configuration handling
│   ├── logger/            # Logging utilities
│   ├── provider/          # API providers (OpenAI, HuzhouAI)
│   ├── report/            # Report generators
│   ├── runner/            # Benchmark runner
│   └── types/             # Type definitions
├── test-cases/            # Test case definitions
├── config.yaml.example    # Configuration template
└── main.go               # Entry point
```

## Supported Providers

- OpenAI - Standard OpenAI API
- HuzhouAI - Huzhou Government LLM API

## Output

The tool generates:
1. Real-time console output with progress
2. Markdown reports in reports/ directory

## License

MIT License