// Package config loads and parses YAML configuration files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-llm-api-benchmark/internal/types"
	"gopkg.in/yaml.v3"
)

// DefaultConfigPath is the default config file path.
const DefaultConfigPath = "config.yaml"

// DefaultTestCasesDir is the default directory for test case YAML files.
const DefaultTestCasesDir = "test-cases"

// LoadConfig parses config.yaml (provider type + config file path only).
func LoadConfig(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %s: %w", path, err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", path, err)
	}

	if cfg.Provider == "" {
		return nil, fmt.Errorf("config field 'provider' must not be empty")
	}

	return &cfg, nil
}

// LoadProviderConfig loads a provider-specific YAML config into the given struct.
func LoadProviderConfig(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read provider config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse provider config %s: %w", path, err)
	}

	return nil
}

// LoadTestCases scans a directory for .yaml/.yml files and parses each as a TestSuite.
func LoadTestCases(dir string) ([]types.TestFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read test cases directory %s: %w", dir, err)
	}

	var files []types.TestFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read test case file %s: %w", path, err)
		}

		var suite types.TestSuite
		if err := yaml.Unmarshal(data, &suite); err != nil {
			return nil, fmt.Errorf("failed to parse test case file %s: %w", path, err)
		}

		files = append(files, types.TestFile{
			FileName: entry.Name(),
			Suite:    suite,
		})
	}

	return files, nil
}
