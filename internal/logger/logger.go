// Package logger provides logging that writes to both a file and stderr.
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

// New creates a logger that writes to a timestamped file in logDir and to stderr.
func New(logDir string) (*log.Logger, error) {
	if logDir == "" {
		logDir = defaultLogDir
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log dir %s: %w", logDir, err)
	}

	timestamp := time.Now().Format("20060102_150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("benchmark_%s.log", timestamp))

	f, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file %s: %w", logPath, err)
	}

	multiWriter := io.MultiWriter(f, os.Stderr)
	l := log.New(multiWriter, "", log.LstdFlags|log.Lmicroseconds)

	fmt.Fprintf(os.Stderr, "Log file: %s\n", logPath)
	l.Printf("========== Benchmark Started ==========")
	return l, nil
}
