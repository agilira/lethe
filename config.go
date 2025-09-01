// config.go: Configuration parsing utilities
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"encoding/json"
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
	// Common invalid characters across platforms: < > : " | ? * and control characters
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	result := filename

	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Remove control characters (0-31) and null characters
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

// ConfigSource defines how to load LoggerConfig from multiple sources
// Supports JSON files, environment variables, and programmatic defaults
type ConfigSource struct {
	// JSONFile path to JSON configuration file (optional)
	JSONFile string

	// EnvPrefix for environment variable names (optional)
	// Example: "LETHE" will load LETHE_FILENAME, LETHE_MAX_SIZE, etc.
	EnvPrefix string

	// Defaults to use when values are not provided by other sources (optional)
	Defaults *LoggerConfig
}

// LoadFromJSON parses LoggerConfig from JSON data
// Uses Go's standard encoding/json package for zero-dependency parsing
//
// Example JSON:
//
//	{
//	  "filename": "app.log",
//	  "max_size_str": "100MB",
//	  "max_age_str": "7d",
//	  "max_backups": 10,
//	  "compress": true,
//	  "async": true,
//	  "local_time": true
//	}
//
// Returns parsed config or error if JSON is invalid
func LoadFromJSON(jsonData []byte) (*LoggerConfig, error) {
	config := &LoggerConfig{}
	if err := json.Unmarshal(jsonData, config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	// Validate required fields
	if config.Filename == "" {
		return nil, fmt.Errorf("filename is required in JSON config")
	}

	return config, nil
}

// LoadFromJSONFile loads LoggerConfig from a JSON file
// Handles file reading and JSON parsing with proper error messages
//
// Parameters:
//   - filepath: Path to JSON configuration file
//
// Returns parsed config or error if file cannot be read or JSON is invalid
func LoadFromJSONFile(filepath string) (*LoggerConfig, error) {
	data, err := os.ReadFile(filepath) // #nosec G304 -- filepath is controlled by application, not user input
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", filepath, err)
	}

	return LoadFromJSON(data)
}

// LoadFromEnv loads LoggerConfig from environment variables
// Supports flexible naming with configurable prefix
//
// Environment variable mapping:
//   - {PREFIX}_FILENAME -> Filename
//   - {PREFIX}_MAX_SIZE -> MaxSizeStr
//   - {PREFIX}_MAX_AGE -> MaxAgeStr
//   - {PREFIX}_MAX_BACKUPS -> MaxBackups
//   - {PREFIX}_COMPRESS -> Compress
//   - {PREFIX}_CHECKSUM -> Checksum
//   - {PREFIX}_ASYNC -> Async
//   - {PREFIX}_LOCAL_TIME -> LocalTime
//   - {PREFIX}_BACKPRESSURE_POLICY -> BackpressurePolicy
//   - {PREFIX}_BUFFER_SIZE -> BufferSize
//   - {PREFIX}_FLUSH_INTERVAL -> FlushInterval
//   - {PREFIX}_ADAPTIVE_FLUSH -> AdaptiveFlush
//   - {PREFIX}_FILE_MODE -> FileMode
//   - {PREFIX}_RETRY_COUNT -> RetryCount
//   - {PREFIX}_RETRY_DELAY -> RetryDelay
//
// Parameters:
//   - prefix: Environment variable prefix (e.g., "LETHE", "LOG")
//
// Returns config with values loaded from environment, or error if parsing fails
func LoadFromEnv(prefix string) (*LoggerConfig, error) {
	if prefix == "" {
		return nil, fmt.Errorf("env prefix cannot be empty")
	}

	config := &LoggerConfig{}

	// Helper function to get env value with prefix
	getEnv := func(key string) string {
		return os.Getenv(prefix + "_" + key)
	}

	// Parse string values
	if val := getEnv("FILENAME"); val != "" {
		config.Filename = val
	}
	if val := getEnv("MAX_SIZE"); val != "" {
		config.MaxSizeStr = val
	}
	if val := getEnv("MAX_AGE"); val != "" {
		config.MaxAgeStr = val
	}
	if val := getEnv("BACKPRESSURE_POLICY"); val != "" {
		config.BackpressurePolicy = val
	}

	// Parse boolean values
	if val := getEnv("COMPRESS"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.Compress = b
		} else {
			return nil, fmt.Errorf("invalid boolean value for %s_COMPRESS: %q", prefix, val)
		}
	}
	if val := getEnv("CHECKSUM"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.Checksum = b
		} else {
			return nil, fmt.Errorf("invalid boolean value for %s_CHECKSUM: %q", prefix, val)
		}
	}
	if val := getEnv("ASYNC"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.Async = b
		} else {
			return nil, fmt.Errorf("invalid boolean value for %s_ASYNC: %q", prefix, val)
		}
	}
	if val := getEnv("LOCAL_TIME"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.LocalTime = b
		} else {
			return nil, fmt.Errorf("invalid boolean value for %s_LOCAL_TIME: %q", prefix, val)
		}
	}
	if val := getEnv("ADAPTIVE_FLUSH"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.AdaptiveFlush = b
		} else {
			return nil, fmt.Errorf("invalid boolean value for %s_ADAPTIVE_FLUSH: %q", prefix, val)
		}
	}

	// Parse integer values
	if val := getEnv("MAX_BACKUPS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.MaxBackups = i
		} else {
			return nil, fmt.Errorf("invalid integer value for %s_MAX_BACKUPS: %q", prefix, val)
		}
	}
	if val := getEnv("BUFFER_SIZE"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.BufferSize = i
		} else {
			return nil, fmt.Errorf("invalid integer value for %s_BUFFER_SIZE: %q", prefix, val)
		}
	}
	if val := getEnv("RETRY_COUNT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.RetryCount = i
		} else {
			return nil, fmt.Errorf("invalid integer value for %s_RETRY_COUNT: %q", prefix, val)
		}
	}

	// Parse duration values
	if val := getEnv("FLUSH_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.FlushInterval = d
		} else {
			return nil, fmt.Errorf("invalid duration value for %s_FLUSH_INTERVAL: %q", prefix, val)
		}
	}
	if val := getEnv("RETRY_DELAY"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.RetryDelay = d
		} else {
			return nil, fmt.Errorf("invalid duration value for %s_RETRY_DELAY: %q", prefix, val)
		}
	}

	// Parse file mode
	if val := getEnv("FILE_MODE"); val != "" {
		if mode, err := strconv.ParseUint(val, 8, 32); err == nil {
			config.FileMode = os.FileMode(mode)
		} else {
			return nil, fmt.Errorf("invalid file mode value for %s_FILE_MODE: %q", prefix, val)
		}
	}

	return config, nil
}

// LoadFromSources loads LoggerConfig from multiple sources with precedence
// Sources are applied in order: Defaults -> JSON -> Environment
// Later sources override earlier ones for the same field
//
// Design rationale:
// - Defaults provide safe fallbacks
// - JSON files for structured configuration
// - Environment variables for runtime overrides (Docker, Kubernetes, etc.)
// - Precedence allows flexible deployment scenarios
//
// Parameters:
//   - source: ConfigSource specifying how to load configuration
//
// Returns merged config or error if loading/parsing fails
func LoadFromSources(source ConfigSource) (*LoggerConfig, error) {
	// Start with defaults if provided
	config := &LoggerConfig{}
	if source.Defaults != nil {
		// Deep copy defaults to avoid modifying the original
		defaultsJSON, err := json.Marshal(source.Defaults)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal defaults: %w", err)
		}
		if err := json.Unmarshal(defaultsJSON, config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal defaults: %w", err)
		}
	}

	// Load from JSON file if specified
	if source.JSONFile != "" {
		jsonConfig, err := LoadFromJSONFile(source.JSONFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load JSON config: %w", err)
		}

		// Merge JSON config (JSON takes precedence over defaults)
		if jsonConfig.Filename != "" {
			config.Filename = jsonConfig.Filename
		}
		if jsonConfig.MaxSizeStr != "" {
			config.MaxSizeStr = jsonConfig.MaxSizeStr
		}
		if jsonConfig.MaxAgeStr != "" {
			config.MaxAgeStr = jsonConfig.MaxAgeStr
		}
		if jsonConfig.BackpressurePolicy != "" {
			config.BackpressurePolicy = jsonConfig.BackpressurePolicy
		}
		// Apply non-zero values for other fields
		if jsonConfig.MaxSize > 0 {
			config.MaxSize = jsonConfig.MaxSize
		}
		if jsonConfig.MaxBackups > 0 {
			config.MaxBackups = jsonConfig.MaxBackups
		}
		if jsonConfig.MaxAge > 0 {
			config.MaxAge = jsonConfig.MaxAge
		}
		if jsonConfig.MaxFileAge > 0 {
			config.MaxFileAge = jsonConfig.MaxFileAge
		}
		if jsonConfig.BufferSize > 0 {
			config.BufferSize = jsonConfig.BufferSize
		}
		if jsonConfig.RetryCount > 0 {
			config.RetryCount = jsonConfig.RetryCount
		}
		if jsonConfig.FlushInterval > 0 {
			config.FlushInterval = jsonConfig.FlushInterval
		}
		if jsonConfig.RetryDelay > 0 {
			config.RetryDelay = jsonConfig.RetryDelay
		}
		if jsonConfig.FileMode > 0 {
			config.FileMode = jsonConfig.FileMode
		}

		// Apply boolean values
		config.Compress = jsonConfig.Compress
		config.Checksum = jsonConfig.Checksum
		config.Async = jsonConfig.Async
		config.LocalTime = jsonConfig.LocalTime
		config.AdaptiveFlush = jsonConfig.AdaptiveFlush

		// Apply function if provided
		if jsonConfig.ErrorCallback != nil {
			config.ErrorCallback = jsonConfig.ErrorCallback
		}
	}

	// Load from environment variables if prefix specified
	if source.EnvPrefix != "" {
		envConfig, err := LoadFromEnv(source.EnvPrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to load env config: %w", err)
		}

		// Merge env config (env takes precedence over JSON and defaults)
		// Only override with non-empty/non-zero values from env
		if envConfig.Filename != "" {
			config.Filename = envConfig.Filename
		}
		if envConfig.MaxSizeStr != "" {
			config.MaxSizeStr = envConfig.MaxSizeStr
		}
		if envConfig.MaxAgeStr != "" {
			config.MaxAgeStr = envConfig.MaxAgeStr
		}
		if envConfig.BackpressurePolicy != "" {
			config.BackpressurePolicy = envConfig.BackpressurePolicy
		}

		// Apply non-zero values for numeric fields (only if explicitly set in env)
		// Note: We can't distinguish between "not set" and "set to 0" for numeric fields
		// So we apply all numeric values from env (this is expected behavior)
		config.MaxSize = envConfig.MaxSize
		config.MaxBackups = envConfig.MaxBackups
		config.MaxAge = envConfig.MaxAge
		config.MaxFileAge = envConfig.MaxFileAge
		config.BufferSize = envConfig.BufferSize
		config.RetryCount = envConfig.RetryCount
		config.FlushInterval = envConfig.FlushInterval
		config.RetryDelay = envConfig.RetryDelay
		config.FileMode = envConfig.FileMode

		// Apply boolean values (only if they were explicitly set in env)
		// We need to check if the env var was actually set, not just the parsed value
		envVarMap := make(map[string]bool)
		prefix := source.EnvPrefix + "_"
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, prefix) {
				key := strings.SplitN(env, "=", 2)[0]
				envVarMap[key] = true
			}
		}

		if envVarMap[prefix+"COMPRESS"] {
			config.Compress = envConfig.Compress
		}
		if envVarMap[prefix+"CHECKSUM"] {
			config.Checksum = envConfig.Checksum
		}
		if envVarMap[prefix+"ASYNC"] {
			config.Async = envConfig.Async
		}
		if envVarMap[prefix+"LOCAL_TIME"] {
			config.LocalTime = envConfig.LocalTime
		}
		if envVarMap[prefix+"ADAPTIVE_FLUSH"] {
			config.AdaptiveFlush = envConfig.AdaptiveFlush
		}

		// Apply function if provided
		if envConfig.ErrorCallback != nil {
			config.ErrorCallback = envConfig.ErrorCallback
		}
	}

	// Validate final configuration
	if config.Filename == "" {
		return nil, fmt.Errorf("filename is required (not provided by any source)")
	}

	return config, nil
}
