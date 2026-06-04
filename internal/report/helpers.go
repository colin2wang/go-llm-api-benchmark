package report

import (
	"fmt"
	"strings"
	"time"
)

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func joinIntList(ints []int, sep string) string {
	parts := make([]string, len(ints))
	for i, v := range ints {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, sep)
}

func fmtFloat(f float64) string {
	if f == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", f)
}

func fmtDur(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.4f", d.Seconds())
}
