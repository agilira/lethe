// config.go: Configuration parsing utilities
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ParseSize converts size strings like "100MB", "1GB" to bytes
// Supports case-insensitive input and single-letter units (K, M, G, T)
// Zero allocations, simple parsing
func ParseSize(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Handle plain numbers (bytes)
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return val, nil
	}

	// Normalize to uppercase for case-insensitive parsing
	s = strings.ToUpper(s)

	// Find suffix and corresponding multiplier
	var multiplier int64
	var numStr string

	switch {
	// Two-letter suffixes (KB, MB, GB, TB)
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "GB"):
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "TB"):
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = s[:len(s)-2]
	// Single-letter suffixes (K, M, G, T)
	case strings.HasSuffix(s, "K"):
		multiplier = 1024
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "M"):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "G"):
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "T"):
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = s[:len(s)-1]
	default:
		return 0, fmt.Errorf("unknown size suffix in %q (supported: KB/K, MB/M, GB/G, TB/T)", s)
	}

	// Parse number part
	val, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number in %q: %v", s, err)
	}

	result := val * multiplier
	if result < 0 { // Overflow check
		return 0, fmt.Errorf("size %q too large", s)
	}

	return result, nil
}

// ParseDuration converts duration strings like "7d", "24h" to time.Duration
// Supports Go durations plus common extensions
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Try standard Go duration first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle custom suffixes
	s = strings.ToLower(s)

	var multiplier time.Duration
	var numStr string

	switch {
	case strings.HasSuffix(s, "d"):
		multiplier = 24 * time.Hour
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "w"):
		multiplier = 7 * 24 * time.Hour
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "y"):
		multiplier = 365 * 24 * time.Hour
		numStr = s[:len(s)-1]
	default:
		return 0, fmt.Errorf("unknown duration suffix in %q", s)
	}

	// Parse number part
	val, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration number in %q: %v", s, err)
	}

	return time.Duration(val) * multiplier, nil
}

// SanitizeFilename removes or replaces invalid characters for cross-platform compatibility
func SanitizeFilename(filename string) string {
	if runtime.GOOS == "windows" {
		// Windows invalid characters: < > : " | ? * and control characters
		invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
		result := filename

		for _, char := range invalidChars {
			result = strings.ReplaceAll(result, char, "_")
		}

		// Remove control characters (0-31)
		var sanitized strings.Builder
		for _, r := range result {
			if r >= 32 {
				sanitized.WriteRune(r)
			} else {
				sanitized.WriteRune('_')
			}
		}

		return sanitized.String()
	}

	// For Unix-like systems, just remove null characters
	return strings.ReplaceAll(filename, "\x00", "_")
}

// ValidatePathLength checks if the path length is within OS limits
func ValidatePathLength(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}

	pathLen := len(absPath)

	switch runtime.GOOS {
	case "windows":
		// Windows has a 260 character limit for paths (historically)
		// Modern versions support longer paths with certain configurations
		if pathLen > 260 {
			return fmt.Errorf("path too long for Windows: %d characters (limit: 260)", pathLen)
		}
	default:
		// Unix-like systems typically have higher limits (4096 on Linux)
		if pathLen > 4096 {
			return fmt.Errorf("path too long: %d characters (limit: 4096)", pathLen)
		}
	}

	return nil
}

// GetDefaultFileMode returns the appropriate default file mode for the OS
func GetDefaultFileMode() os.FileMode {
	if runtime.GOOS == "windows" {
		// On Windows, Go handles ACL conversion
		// 0644 is still appropriate as Go translates it correctly
		return 0644
	}
	return 0644
}

// RetryFileOperation executes a file operation with retry logic for cross-platform reliability
//
// Design rationale: Windows and network filesystems can have transient failures
// due to antivirus scans, indexing services, or file locking. Retry logic
// improves reliability without masking real errors.
//
// Why retry is needed:
// - Windows: Antivirus can temporarily lock files
// - Network shares: Temporary connectivity issues
// - Container environments: Overlay filesystem quirks
// - High load: Temporary resource exhaustion
//
// Conservative approach: Short delays, limited retries to avoid hanging
func RetryFileOperation(operation func() error, retryCount int, retryDelay time.Duration) error {
	if retryCount <= 0 {
		retryCount = 3 // Default retry count - balances reliability vs latency
	}
	if retryDelay <= 0 {
		retryDelay = 10 * time.Millisecond // Default delay - short enough to be unnoticeable
	}

	var lastErr error
	for i := 0; i < retryCount; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// On the last attempt, don't wait - fail fast
		if i < retryCount-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("operation failed after %d retries: %v", retryCount, lastErr)
}
