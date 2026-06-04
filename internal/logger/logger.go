// Package logger 提供日志功能，将日志同时输出到终端和文件。
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const defaultLogDir = "log"

// New 创建一个日志记录器，在 logDir 目录下生成带时间戳的日志文件，
// 日志同时写入文件并输出到 stderr。
func New(logDir string) (*log.Logger, error) {
	if logDir == "" {
		logDir = defaultLogDir
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录 %s 失败: %w", logDir, err)
	}

	timestamp := time.Now().Format("20060102_150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("benchmark_%s.log", timestamp))

	f, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件 %s 失败: %w", logPath, err)
	}

	// MultiWriter: 同时写入文件 + stderr
	multiWriter := io.MultiWriter(f, os.Stderr)
	l := log.New(multiWriter, "", log.Ltime|log.Lmicroseconds)

	fmt.Fprintf(os.Stderr, "📝 日志文件: %s\n", logPath)
	return l, nil
}
