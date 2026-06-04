// Package config 负责加载和解析 YAML 配置文件。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-llm-api-benchmark/internal/types"
	"gopkg.in/yaml.v3"
)

// DefaultConfigPath 默认配置文件路径
const DefaultConfigPath = "config.yaml"

// DefaultTestCasesDir 默认测试用例目录
const DefaultTestCasesDir = "test-cases"

// LoadConfig 解析 config.yaml 文件
func LoadConfig(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}

	// 字段校验
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("配置项 base_url 不能为空")
	}

	return &cfg, nil
}

// LoadTestCases 扫描目录下所有 .yaml / .yml 文件并解析为 TestSuite 列表
func LoadTestCases(dir string) ([]types.TestFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取测试用例目录 %s 失败: %w", dir, err)
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
			return nil, fmt.Errorf("读取测试用例文件 %s 失败: %w", path, err)
		}

		var suite types.TestSuite
		if err := yaml.Unmarshal(data, &suite); err != nil {
			return nil, fmt.Errorf("解析测试用例文件 %s 失败: %w", path, err)
		}

		files = append(files, types.TestFile{
			FileName: entry.Name(),
			Suite:    suite,
		})
	}

	return files, nil
}
