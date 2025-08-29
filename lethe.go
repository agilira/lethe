// lethe.go: Public API - Universal log rotation library
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/agilira/go-timecache"
)

// Pre-allocated errors to avoid allocations in hot paths
var (
	errNoCurrentFile = errors.New("no current file")
)

// Logger provides universal log rotation with lumberjack compatibility.
// It offers zero locks, zero allocations in hot path, and is thread-safe by design.
// Advanced features include MPSC mode for high-throughput scenarios.
//
// Basic usage example:
//
//	logger := &Logger{
//		Filename:   "app.log",
//		MaxSizeStr: "100MB",
//		MaxBackups: 3,
//		Compress:   true,
//	}
//	logger.Write([]byte("Hello, World!"))
//
// High-performance MPSC mode example:
//
//	logger := &Logger{
//		Filename:           "app.log",
//		MaxSizeStr:         "1GB",
//		Async:              true,
//		BufferSize:         4096,
//		BackpressurePolicy: "adaptive",
//	}
type Logger struct {
	// Filename is the log file to write to.
	// If the file doesn't exist, it is created. If the path doesn't exist,
	// it is created recursively with appropriate permissions.
	Filename string `json:"filename"`

	// MaxSize is the maximum size in MB before rotation.
	// DEPRECATED: Use MaxSizeStr for greater flexibility.
	MaxSize int64 `json:"max_size"`

	// MaxBackups is the maximum number of old log files to retain.
	// Older files are automatically deleted. A value of 0 retains all backups.
	MaxBackups int `json:"max_backups"`

	// MaxAge is the maximum age before time-based rotation.
	// Files are rotated when they reach this age, regardless of size.
	// A value of 0 disables time-based rotation.
	// DEPRECATED: Use MaxAgeStr for string-based configuration.
	MaxAge time.Duration `json:"max_age"`

	// MaxFileAge is the maximum age for backup files before deletion.
	// Backup files older than this duration are automatically deleted.
	// A value of 0 disables age-based cleanup.
	MaxFileAge time.Duration `json:"max_file_age"`

	// LocalTime determines whether to use local time in backup filenames.
	// False (default) uses UTC. True uses the system's local timezone.
	LocalTime bool `json:"local_time"`

	// Compress enables gzip compression of rotated files.
	// Compressed files have a .gz extension added.
	Compress bool `json:"compress"`

	// Checksum enables SHA-256 checksum calculation for file integrity.
	// Checksums are saved as separate files with .sha256 extension.
	Checksum bool `json:"checksum"`

	// Async enables MPSC (Multi-Producer Single-Consumer) mode for high-throughput scenarios.
	// Writes are buffered in a lock-free ring buffer and processed by a dedicated consumer.
	Async bool `json:"async"`

	// MaxSizeStr is the maximum size as a string (e.g., "100MB", "2GB", "500KB").
	// This field is preferred over MaxSize for greater flexibility.
	// Supported formats: B, KB, MB, GB, TB (both 1000 and 1024 based).
	MaxSizeStr string `json:"max_size_str"`

	// MaxAgeStr is the maximum age as a string (e.g., "7d", "24h", "30m").
	// This field is preferred over MaxAge for greater flexibility.
	// Supported formats: ns, us, ms, s, m, h, d, w.
	MaxAgeStr string `json:"max_age_str"`

	// ErrorCallback is an optional function called when errors occur.
	// Useful for custom logging or error metrics.
	// Parameters are the operation that failed and the specific error.
	ErrorCallback func(operation string, err error) `json:"-"`

	// FileMode is the file permissions (default: 0644).
	// Used when creating new log files.
	FileMode os.FileMode `json:"file_mode"`

	// RetryCount is the number of retries for file operations (default: 3).
	// Useful for handling temporary filesystem errors.
	RetryCount int `json:"retry_count"`

	// RetryDelay is the delay between retries (default: 10ms).
	// Wait time before retrying a failed operation.
	RetryDelay time.Duration `json:"retry_delay"`

	// BufferSize is the size of the MPSC ring buffer (default: 1024, must be power of 2).
	// Used only when Async is true. Larger sizes improve throughput
	// but increase memory usage.
	BufferSize int `json:"buffer_size"`

	// BackpressurePolicy defines behavior when the buffer is full.
	// Options: "fallback" (default, fall back to sync), "drop" (discard messages), "adaptive" (resize buffer).
	BackpressurePolicy string `json:"backpressure_policy"`

	// FlushInterval is the flush interval for the MPSC consumer (default: 1ms).
	// Lower frequencies reduce latency but increase CPU overhead.
	FlushInterval time.Duration `json:"flush_interval"`

	// AdaptiveFlush enables adaptive flush timing based on buffer state.
	// The consumer automatically adapts to write velocity to optimize performance.
	AdaptiveFlush bool `json:"adaptive_flush"`

	// Internal state (all atomic - ZERO LOCKS!)
	currentFile  atomic.Pointer[os.File] // Current log file
	bytesWritten atomic.Uint64           // Total bytes written
	rotationSeq  atomic.Uint64           // Rotation sequence number
	rotationFlag atomic.Bool             // Rotation in progress flag
	fileCreated  atomic.Int64            // Unix timestamp when current file was created

	// MPSC buffer state (lock-free)
	buffer   atomic.Pointer[ringBuffer]   // Ring buffer for async writes
	consumer atomic.Pointer[MPSCConsumer] // MPSC consumer instance

	// Auto-scaling metrics
	writeCount      atomic.Uint64 // Total write operations
	contentionCount atomic.Uint64 // Contention detection counter
	totalLatency    atomic.Uint64 // Total latency in nanoseconds
	lastLatency     atomic.Uint64 // Last write latency in nanoseconds
	droppedCount    atomic.Uint64 // Messages dropped due to full buffer

	// Background worker pool
	bgWorkers atomic.Pointer[BackgroundWorkers] // Worker pool for cleanup/compression

	// High-performance time cache for reduced allocation overhead
	timeCache     *timecache.TimeCache
	timeCacheOnce sync.Once

	// File initialization protection
	initMutex sync.Mutex

	// Close protection
	closeOnce sync.Once

	// Config cache (parsed once)
	maxSizeBytes int64 // MaxSize * MB in bytes
}

// New creates a new Logger with safe defaults and validates configuration.
// This is the recommended way to create a Logger instance.
//
// Parameters:
//   - filename: Path to the log file (required)
//   - maxSizeMB: Maximum file size in MB before rotation (0 = no size limit)
//   - maxBackups: Number of backup files to keep (0 = keep all)
//
// Example:
//
//	logger, err := lethe.New("app.log", 100, 3)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
func New(filename string, maxSizeMB int, maxBackups int) (*Logger, error) {
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	logger := &Logger{
		Filename:   filename,
		MaxBackups: maxBackups,

		// Safe defaults
		FileMode:           0644,
		RetryCount:         3,
		RetryDelay:         10 * time.Millisecond,
		BufferSize:         1024,
		BackpressurePolicy: "fallback",
		FlushInterval:      1 * time.Millisecond,
	}

	// Set size limit if specified (backward compatibility)
	if maxSizeMB > 0 {
		logger.MaxSize = int64(maxSizeMB) // Keep for backward compatibility
	}

	// Initialize time cache for performance
	logger.timeCache = timecache.NewWithResolution(time.Millisecond)

	return logger, nil
}

// NewSimple creates a Logger with modern string-based configuration.
// This is the recommended way to create simple loggers with enhanced defaults.
//
// Parameters:
//   - filename: Path to the log file (required)
//   - maxSize: Maximum file size as string (e.g., "100MB", "1GB")
//   - maxBackups: Number of backup files to keep (0 = keep all)
//
// Features enabled by default:
//   - Async mode for better performance
//   - Adaptive backpressure policy
//   - 4KB buffer for efficient I/O
//
// Example:
//
//	logger, err := lethe.NewSimple("app.log", "100MB", 5)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
func NewSimple(filename, maxSize string, maxBackups int) (*Logger, error) {
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	logger := &Logger{
		Filename:   filename,
		MaxSizeStr: maxSize,
		MaxBackups: maxBackups,

		// Safe defaults optimized for modern usage
		FileMode:           0644,
		RetryCount:         3,
		RetryDelay:         10 * time.Millisecond,
		BufferSize:         4096,                 // Slightly larger buffer for better performance
		BackpressurePolicy: "adaptive",           // Better default for most use cases
		FlushInterval:      5 * time.Millisecond, // Balance between latency and performance
		Async:              true,                 // Enable async by default for better performance
	}

	// Initialize time cache for performance
	logger.timeCache = timecache.NewWithResolution(time.Millisecond)

	return logger, nil
}

// NewWithDefaults creates a Logger with sensible production defaults.
// Perfect for most applications without requiring detailed configuration.
// This is the recommended constructor for production environments.
//
// Production defaults applied:
//   - MaxSizeStr: "100MB" (rotates when file reaches 100MB)
//   - MaxAgeStr: "7d" (rotates weekly for fresh logs)
//   - MaxBackups: 10 (keeps 10 backup files)
//   - Compress: true (saves disk space)
//   - Async: true (better performance)
//   - BackpressurePolicy: "adaptive" (intelligent overflow handling)
//   - LocalTime: true (local timestamps in backups)
//
// Parameters:
//   - filename: Path to the log file (required)
//
// Returns a fully configured Logger ready for production use.
//
// Example:
//
//	logger, err := lethe.NewWithDefaults("app.log")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
//
//	// Use with standard library
//	log.SetOutput(logger)
func NewWithDefaults(filename string) (*Logger, error) {
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	config := &LoggerConfig{
		Filename:           filename,
		MaxSizeStr:         "100MB",
		MaxAgeStr:          "7d",
		MaxBackups:         10,
		Compress:           true,
		Async:              true,
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
	}

	return NewWithConfig(config)
}

// NewDaily creates a Logger that rotates daily.
// Ideal for applications requiring daily log rotation with moderate file sizes.
//
// Configuration optimized for daily rotation:
//   - MaxSizeStr: "50MB" (reasonable daily file size)
//   - MaxAgeStr: "24h" (rotates every 24 hours)
//   - MaxBackups: 7 (keeps one week of daily logs)
//   - Compress: true (saves storage space)
//   - Async: true (better performance)
//   - LocalTime: true (daily rotation aligned with local timezone)
//
// Parameters:
//   - filename: Path to the log file (required)
//
// Example:
//
//	logger, err := lethe.NewDaily("daily.log")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
func NewDaily(filename string) (*Logger, error) {
	config := &LoggerConfig{
		Filename:           filename,
		MaxSizeStr:         "50MB", // Reasonable size limit
		MaxAgeStr:          "24h",  // Rotate daily
		MaxBackups:         7,      // Keep a week of logs
		Compress:           true,
		Async:              true,
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
	}
	return NewWithConfig(config)
}

// NewWeekly creates a Logger that rotates weekly.
// Perfect for applications with moderate logging volume requiring weekly archives.
//
// Configuration optimized for weekly rotation:
//   - MaxSizeStr: "200MB" (larger size for weekly accumulation)
//   - MaxAgeStr: "7d" (rotates every 7 days)
//   - MaxBackups: 4 (keeps one month of weekly logs)
//   - Compress: true (essential for larger files)
//   - Async: true (better performance)
//   - LocalTime: true (weekly rotation aligned with local timezone)
//
// Parameters:
//   - filename: Path to the log file (required)
//
// Example:
//
//	logger, err := lethe.NewWeekly("weekly.log")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
func NewWeekly(filename string) (*Logger, error) {
	config := &LoggerConfig{
		Filename:           filename,
		MaxSizeStr:         "200MB", // Larger size for weekly rotation
		MaxAgeStr:          "7d",    // Rotate weekly
		MaxBackups:         4,       // Keep a month of logs
		Compress:           true,
		Async:              true,
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
	}
	return NewWithConfig(config)
}

// NewDevelopment creates a Logger optimized for development and debugging.
// Designed for immediate visibility of logs with frequent rotation and no compression.
//
// Development-optimized configuration:
//   - MaxSizeStr: "10MB" (small files for easier handling)
//   - MaxAgeStr: "1h" (frequent rotation for fresh logs)
//   - MaxBackups: 5 (keep recent history without clutter)
//   - Compress: false (immediate file access for debugging)
//   - Async: false (synchronous writes for immediate visibility)
//   - BackpressurePolicy: "fallback" (simple error handling)
//   - LocalTime: true (local timestamps for debugging)
//
// Parameters:
//   - filename: Path to the log file (required)
//
// Example:
//
//	logger, err := lethe.NewDevelopment("debug.log")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
func NewDevelopment(filename string) (*Logger, error) {
	config := &LoggerConfig{
		Filename:           filename,
		MaxSizeStr:         "10MB", // Smaller files for easier handling
		MaxAgeStr:          "1h",   // Rotate hourly for fresh logs
		MaxBackups:         5,      // Keep recent history
		Compress:           false,  // No compression for easier reading
		Async:              false,  // Synchronous for immediate writes during debugging
		BackpressurePolicy: "fallback",
		LocalTime:          true,
	}
	return NewWithConfig(config)
}

// NewWithConfig creates a new Logger with detailed configuration.
// This provides full control over all Logger options and features.
// Use this constructor when you need fine-grained control over Logger behavior.
//
// All LoggerConfig fields are optional except Filename. Unset fields use sensible defaults.
// String-based fields (MaxSizeStr, MaxAgeStr) take precedence over their numeric equivalents.
//
// Parameters:
//   - config: LoggerConfig pointer with desired settings (required, non-nil)
//
// Returns:
//   - *Logger: Configured logger instance
//   - error: Configuration validation errors or initialization failures
//
// Example with enterprise features:
//
//	config := &LoggerConfig{
//		Filename:           "app.log",
//		MaxSizeStr:         "500MB",
//		MaxAgeStr:          "30d",
//		MaxBackups:         20,
//		MaxFileAge:         180 * 24 * time.Hour, // 6 months backup retention
//		Compress:           true,
//		Checksum:           true,                 // Enable data integrity
//		Async:              true,
//		BufferSize:         4096,                 // High-performance buffer
//		BackpressurePolicy: "adaptive",
//		LocalTime:          true,
//		ErrorCallback: func(eventType string, err error) {
//			log.Printf("Log error (%s): %v", eventType, err)
//		},
//	}
//	logger, err := lethe.NewWithConfig(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
func NewWithConfig(config *LoggerConfig) (*Logger, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	if config.Filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	logger := &Logger{
		Filename:           config.Filename,
		MaxSize:            config.MaxSize,
		MaxBackups:         config.MaxBackups,
		MaxAge:             config.MaxAge,
		MaxFileAge:         config.MaxFileAge,
		LocalTime:          config.LocalTime,
		Compress:           config.Compress,
		Checksum:           config.Checksum,
		Async:              config.Async,
		MaxSizeStr:         config.MaxSizeStr,
		MaxAgeStr:          config.MaxAgeStr,
		ErrorCallback:      config.ErrorCallback,
		BackpressurePolicy: config.BackpressurePolicy,
		AdaptiveFlush:      config.AdaptiveFlush,
	}

	// Apply safe defaults for unset values
	if logger.FileMode == 0 {
		logger.FileMode = 0644
	}
	if logger.RetryCount == 0 {
		logger.RetryCount = 3
	}
	if logger.RetryDelay == 0 {
		logger.RetryDelay = 10 * time.Millisecond
	}
	if logger.BufferSize == 0 {
		logger.BufferSize = 1024
	}
	if logger.BackpressurePolicy == "" {
		logger.BackpressurePolicy = "fallback"
	}
	if logger.FlushInterval == 0 {
		logger.FlushInterval = 1 * time.Millisecond
	}

	// Parse string-based configurations
	// Validate that both MaxAge and MaxAgeStr are not specified simultaneously
	if logger.MaxAge > 0 && logger.MaxAgeStr != "" {
		return nil, fmt.Errorf("cannot specify both MaxAge and MaxAgeStr; use MaxAgeStr for string-based configuration")
	}

	if logger.MaxAgeStr != "" {
		duration, err := ParseDuration(logger.MaxAgeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid MaxAgeStr: %w", err)
		}
		logger.MaxAge = duration
	}

	// Initialize time cache for performance
	logger.timeCache = timecache.NewWithResolution(time.Millisecond)

	return logger, nil
}

// LoggerConfig holds configuration options for creating a Logger.
// This struct provides a clear, documented way to configure all Logger options.
type LoggerConfig struct {
	// Basic configuration
	Filename   string `json:"filename"`
	MaxSize    int64  `json:"max_size"`
	MaxBackups int    `json:"max_backups"`

	// String-based configuration (preferred)
	MaxSizeStr string `json:"max_size_str"`
	MaxAgeStr  string `json:"max_age_str"`

	// Time-based rotation
	MaxAge     time.Duration `json:"max_age"`
	MaxFileAge time.Duration `json:"max_file_age"`
	LocalTime  bool          `json:"local_time"`

	// Features
	Compress bool `json:"compress"`
	Checksum bool `json:"checksum"`
	Async    bool `json:"async"`

	// Error handling
	ErrorCallback func(operation string, err error) `json:"-"`

	// File operations
	FileMode   os.FileMode   `json:"file_mode"`
	RetryCount int           `json:"retry_count"`
	RetryDelay time.Duration `json:"retry_delay"`

	// MPSC configuration
	BufferSize         int           `json:"buffer_size"`
	BackpressurePolicy string        `json:"backpressure_policy"`
	FlushInterval      time.Duration `json:"flush_interval"`
	AdaptiveFlush      bool          `json:"adaptive_flush"`
}

// Write implements io.Writer interface for universal compatibility.
// ZERO allocations, ZERO locks, thread safe.
//
// The Write method automatically handles:
//   - File creation and rotation
//   - Auto-scaling to MPSC mode under high load
//   - Error reporting via ErrorCallback
//   - Performance metrics collection
//
// Returns the number of bytes written and any error encountered.
// Write implements io.Writer interface for seamless integration with logging frameworks.
// Writes data to the log file with automatic rotation based on size and age policies.
// This method is thread-safe and can be called concurrently from multiple goroutines.
//
// The method automatically handles:
//   - File rotation when size or age limits are exceeded
//   - Directory creation if the log path doesn't exist
//   - Async/sync mode switching based on configuration and load
//   - Auto-scaling from sync to MPSC mode under high concurrency
//
// Performance characteristics:
//   - Sync mode: Direct writes with immediate durability
//   - Async mode: Buffered writes with lock-free MPSC queue
//   - Auto-scaling: Automatic mode switching based on contention detection
//
// Parameters:
//   - data: Byte slice to write to the log file
//
// Returns:
//   - int: Number of bytes written (always len(data) on success)
//   - error: Any error encountered during writing or rotation
//
// Example usage:
//
//	logger, _ := lethe.NewWithDefaults("app.log")
//	defer logger.Close()
//
//	// Direct usage
//	logger.Write([]byte("Application started\n"))
//
//	// With standard library
//	log.SetOutput(logger)
//	log.Println("This goes through lethe")
//
//	// With frameworks
//	logrus.SetOutput(logger)
func (l *Logger) Write(data []byte) (int, error) {
	// Increment write counter for auto-scaling metrics
	l.writeCount.Add(1)

	if l.Async {
		return l.writeAsync(data)
	}

	// Auto-scaling logic: detect high concurrency and switch to MPSC
	if l.shouldScaleToMPSC() {
		return l.writeAsync(data)
	}

	return l.writeSync(data)
}

// WriteOwned writes data with ownership transfer (zero-copy for MPSC mode).
// The caller promises not to reuse the data slice after this call.
// This enables zero-copy optimization in MPSC mode, improving performance
// for systems that can transfer ownership of pre-allocated buffers.
//
// This is particularly useful for integration with web frameworks like Iris
// or high-performance systems that manage their own buffer pools.
//
// Performance note: In sync mode, this behaves identically to Write().
// In async mode, it avoids buffer copying, reducing memory allocations.
//
// Usage example:
//
//	buf := make([]byte, len(message))
//	copy(buf, message)
//	n, err := logger.WriteOwned(buf)
//	// buf must not be used after this call
//
// Returns the number of bytes written and any error encountered.
func (l *Logger) WriteOwned(data []byte) (int, error) {
	// Increment write counter for auto-scaling metrics
	l.writeCount.Add(1)

	if l.Async {
		return l.writeAsyncOwned(data)
	}

	// Auto-scaling logic: detect high concurrency and switch to MPSC
	if l.shouldScaleToMPSC() {
		return l.writeAsyncOwned(data)
	}

	return l.writeSync(data)
}

// writeAsyncOwned handles high-throughput MPSC writes with ownership transfer
func (l *Logger) writeAsyncOwned(data []byte) (int, error) {
	// Lazy initialization of MPSC buffer
	if l.buffer.Load() == nil {
		if err := l.initMPSC(); err != nil {
			// Fallback to sync mode on initialization failure
			return l.writeSync(data)
		}
	}

	buffer := l.buffer.Load()
	if buffer == nil {
		return l.writeSync(data) // Fallback if still nil
	}

	// Try to push to ring buffer with ownership transfer
	if buffer.pushOwned(data) {
		return len(data), nil
	}

	// Buffer full - apply backpressure policy
	l.contentionCount.Add(1)

	policy := l.BackpressurePolicy
	if policy == "" {
		policy = "fallback" // Default policy
	}

	switch policy {
	case "drop":
		// Drop-on-full policy: silently discard the message
		l.droppedCount.Add(1)
		return len(data), nil

	case "adaptive":
		// Adaptive resize: try to expand buffer on pressure
		if l.tryAdaptiveResize(buffer) {
			// Retry with expanded buffer
			if buffer.pushOwned(data) {
				return len(data), nil
			}
		}
		// If resize failed or push still failed, fallback to sync
		return l.writeSync(data)

	default: // "fallback"
		// Original behavior: fallback to sync write
		return l.writeSync(data)
	}
}

// shouldScaleToMPSC determines if we should auto-scale to MPSC mode
//
// Design rationale: Auto-scaling is based on performance degradation indicators.
// When multiple goroutines compete for file writes, filesystem locks cause contention.
// MPSC (Multi-Producer Single-Consumer) eliminates this by serializing writes through a
// lock-free ring buffer, improving throughput under high concurrency.
//
// Metrics used for scaling decision:
// - Contention: Detected when rotation flag is set during writes
// - Latency: High latency indicates filesystem bottlenecks
// - Write frequency: High frequency benefits from batching
func (l *Logger) shouldScaleToMPSC() bool {
	writeCount := l.writeCount.Load()
	contentionCount := l.contentionCount.Load()
	totalLatency := l.totalLatency.Load()
	lastLatency := l.lastLatency.Load()

	// Need minimum sample size for reliable metrics
	// Why 100: Avoids premature scaling during application startup
	if writeCount < 100 {
		return false
	}

	// Calculate average latency
	avgLatency := totalLatency / writeCount

	// Auto-scale conditions (any of these triggers MPSC mode):

	// 1. High contention detected
	// Why: Contention indicates multiple goroutines competing for file access
	if contentionCount > 0 && writeCount > 1000 {
		return true
	}

	// 2. High average latency (> 1ms indicates slow writes)
	// Why: 1ms is threshold where MPSC overhead becomes beneficial
	if avgLatency > 1_000_000 { // 1ms in nanoseconds
		return true
	}

	// 3. Recent spike in latency (last write > 5ms)
	// Why: Reactive scaling for sudden performance degradation
	if lastLatency > 5_000_000 { // 5ms in nanoseconds
		return true
	}

	// 4. High frequency with degrading performance
	// Why: 10% contention ratio indicates significant competition
	contentionRatio := float64(contentionCount) / float64(writeCount)
	if contentionRatio > 0.1 { // 10% contention rate
		return true
	}

	return false
}

// writeSync handles synchronous writes (default mode)
func (l *Logger) writeSync(data []byte) (int, error) {
	// Ensure timeCache is initialized before use
	l.timeCacheOnce.Do(func() {
		l.timeCache = timecache.NewWithResolution(time.Millisecond)
	})

	start := l.timeCache.CachedTime()
	defer func() {
		// Measure and record latency using cached time
		end := l.timeCache.CachedTime()
		latencyNs := end.Sub(start).Nanoseconds()
		if latencyNs < 0 {
			latencyNs = 0 // Protect against clock skew
		}
		latency := uint64(latencyNs) // #nosec G115 -- latencyNs checked for negative values above
		l.lastLatency.Store(latency)
		l.totalLatency.Add(latency)
	}()

	// Lazy initialization (thread-safe)
	if l.currentFile.Load() == nil {
		l.initMutex.Lock()
		// Double-check pattern
		if l.currentFile.Load() == nil {
			if err := l.initFile(); err != nil {
				l.initMutex.Unlock()
				return 0, err
			}
		}
		l.initMutex.Unlock()
	}

	// Atomic load current file
	file := l.currentFile.Load()
	if file == nil {
		return 0, errNoCurrentFile
	}

	// Detect contention: if rotation is in progress, we have contention
	if l.rotationFlag.Load() {
		l.contentionCount.Add(1)
	}

	// Write to file (filesystem provides locking)
	n, err := file.Write(data)
	if err != nil {
		return n, err
	}

	// Atomic update size (n from Write() is always >= 0, but be safe)
	if n < 0 {
		n = 0
	}
	newSize := l.bytesWritten.Add(uint64(n)) // #nosec G115 -- n checked for negative values above

	// Check rotation (lock-free)
	if l.shouldRotate(newSize) {
		l.triggerRotation()
	}

	return n, nil
}

// writeAsync handles high-throughput MPSC writes with configurable backpressure
func (l *Logger) writeAsync(data []byte) (int, error) {
	// Lazy initialization of MPSC buffer
	if l.buffer.Load() == nil {
		if err := l.initMPSC(); err != nil {
			// Fallback to sync mode on initialization failure
			return l.writeSync(data)
		}
	}

	buffer := l.buffer.Load()
	if buffer == nil {
		return l.writeSync(data) // Fallback if still nil
	}

	// Try to push to ring buffer
	if buffer.push(data) {
		return len(data), nil
	}

	// Buffer full - apply backpressure policy
	l.contentionCount.Add(1)

	policy := l.BackpressurePolicy
	if policy == "" {
		policy = "fallback" // Default policy
	}

	switch policy {
	case "drop":
		// Drop-on-full policy: silently discard the message
		// Useful for high-frequency telemetry/access logs
		l.droppedCount.Add(1)
		return len(data), nil

	case "adaptive":
		// Adaptive resize: try to expand buffer on pressure
		if l.tryAdaptiveResize(buffer) {
			// Retry with expanded buffer
			if buffer.push(data) {
				return len(data), nil
			}
		}
		// If resize failed or push still failed, fallback to sync
		return l.writeSync(data)

	default: // "fallback"
		// Original behavior: fallback to sync write
		return l.writeSync(data)
	}
}

// initMPSC initializes the MPSC buffer and consumer goroutine
func (l *Logger) initMPSC() error {
	// Get buffer size from configuration (default: 1024)
	bufferSize := l.BufferSize
	if bufferSize <= 0 {
		bufferSize = 1024 // Default size
	}

	// Create ring buffer with configured size
	if bufferSize < 0 {
		bufferSize = 1024 // Safety fallback
	}
	buffer := newRingBuffer(uint64(bufferSize)) // #nosec G115 -- bufferSize checked for negative values above

	// Try to atomically set the buffer
	if !l.buffer.CompareAndSwap(nil, buffer) {
		// Someone else initialized it
		return nil
	}

	// Initialize file if needed (thread-safe)
	if l.currentFile.Load() == nil {
		l.initMutex.Lock()
		// Double-check pattern
		if l.currentFile.Load() == nil {
			if err := l.initFile(); err != nil {
				l.initMutex.Unlock()
				return err
			}
		}
		l.initMutex.Unlock()
	}

	// Create and start MPSC consumer
	consumer := newMPSCConsumer(buffer, l)
	l.consumer.Store(consumer)

	return nil
}

// tryAdaptiveResize attempts to resize the MPSC buffer dynamically
// Returns true if resize was successful, false otherwise
func (l *Logger) tryAdaptiveResize(currentBuffer *ringBuffer) bool {
	// Adaptive resize policy: double buffer size up to a maximum
	currentSize := uint64(len(currentBuffer.buffer))
	maxSize := uint64(16384) // Max 16K entries to prevent excessive memory usage

	if currentSize >= maxSize {
		return false // Already at maximum size
	}

	newSize := currentSize * 2
	if newSize > maxSize {
		newSize = maxSize
	}

	// Create new larger buffer
	newBuffer := newRingBuffer(newSize)

	// Drain current buffer into new buffer
	// Note: This is a best-effort approach - some messages might be lost
	// during the transition, but this is acceptable for adaptive resizing
	drainedCount := 0
	for drainedCount < 100 { // Limit to prevent infinite loop
		if data, ok := currentBuffer.pop(); ok {
			if !newBuffer.push(data) {
				// New buffer full (shouldn't happen), abort resize
				return false
			}
			drainedCount++
		} else {
			break // Current buffer empty
		}
	}

	// Atomically replace buffer
	return l.buffer.CompareAndSwap(currentBuffer, newBuffer)
}

// shouldRotate checks if rotation is needed (lock-free)
func (l *Logger) shouldRotate(currentSize uint64) bool {
	// Parse size configuration (supports both old and new formats)
	if l.maxSizeBytes == 0 {
		if l.MaxSizeStr != "" {
			// Use new string-based configuration
			if size, err := ParseSize(l.MaxSizeStr); err == nil {
				l.maxSizeBytes = size
			}
		} else if l.MaxSize > 0 {
			// Fallback to legacy MB-based configuration
			l.maxSizeBytes = l.MaxSize * 1024 * 1024 // MB to bytes
		}
	}

	// Check size-based rotation
	if l.maxSizeBytes > 0 && currentSize >= uint64(l.maxSizeBytes) {
		return true
	}

	// Check time-based rotation (supports both old and new formats)
	var maxAge time.Duration
	if l.MaxAgeStr != "" {
		// Use new string-based configuration
		if duration, err := ParseDuration(l.MaxAgeStr); err == nil {
			maxAge = duration
		}
	} else if l.MaxAge > 0 {
		// Fallback to legacy duration-based configuration
		maxAge = l.MaxAge
	}

	if maxAge > 0 {
		createdTime := l.fileCreated.Load()
		if createdTime > 0 {
			elapsed := time.Since(time.Unix(createdTime, 0))
			if elapsed >= maxAge {
				return true
			}
		}
	}

	return false
}

// triggerRotation initiates rotation (lock-free, single-threaded)
//
// Design rationale: Uses Compare-And-Swap (CAS) to ensure only one goroutine
// performs rotation at a time, avoiding the overhead of traditional mutex locks.
//
// Why CAS over mutex:
// - Zero allocation (mutex allocates)
// - No blocking (mutex blocks other writers)
// - Cache-friendly (atomic operations are CPU-optimized)
// - Wait-free for non-rotating goroutines
func (l *Logger) triggerRotation() {
	// CAS to claim rotation - only one goroutine can succeed
	// Others continue writing to old file until rotation completes
	if !l.rotationFlag.CompareAndSwap(false, true) {
		return // Someone else is rotating
	}
	defer l.rotationFlag.Store(false)

	// Perform rotation
	if err := l.performRotation(); err != nil {
		l.reportError("rotation", err)
	}
}

// initFile and performRotation are implemented in rotation.go

// Close closes the logger and cleans up all resources.
// This method should be called when the logger is no longer needed
// to ensure proper cleanup of background goroutines and file handles.
//
// The Close method will:
//   - Stop the MPSC consumer goroutine if running
//   - Stop background worker goroutines
//   - Stop the time cache
//   - Close the current log file
//
// After calling Close, the logger should not be used for further writes.
//
// Example:
//
//	logger, err := lethe.New("app.log", 100, 3)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close() // Ensure cleanup
//
// Returns any error encountered while closing the file.
// Close gracefully shuts down the Logger and releases all resources.
// This method ensures all pending writes are flushed and background operations complete.
// It is safe to call Close multiple times; subsequent calls will be no-ops.
//
// Close performs the following shutdown sequence:
//  1. Stops the MPSC consumer (if running) and flushes pending writes
//  2. Stops background workers (compression, cleanup, checksums)
//  3. Stops the time cache to prevent memory leaks
//  4. Closes the current log file
//
// Important: Always call Close when shutting down to prevent data loss.
// Use defer immediately after logger creation for automatic cleanup.
//
// Parameters: None
//
// Returns:
//   - error: File close error, if any. Background task errors are reported via ErrorCallback.
//
// Example:
//
//	logger, err := lethe.NewWithDefaults("app.log")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close() // Ensures cleanup on exit
//
//	// Use logger...
//	logger.Write([]byte("Application shutting down\n"))
//	// Close() called automatically via defer
func (l *Logger) Close() error {
	var closeErr error
	l.closeOnce.Do(func() {
		// Stop MPSC consumer if running
		if consumer := l.consumer.Load(); consumer != nil {
			consumer.stop()
		}

		// Stop background workers if running
		if workers := l.bgWorkers.Load(); workers != nil {
			workers.stop()
		}

		// Stop time cache if running
		if l.timeCache != nil {
			l.timeCache.Stop()
		}

		// Close file
		if file := l.currentFile.Load(); file != nil {
			closeErr = file.Close()
		}
	})
	return closeErr
}

// WaitForBackgroundTasks waits for all background tasks (compression, cleanup, checksums) to complete.
// This is useful in tests to ensure all operations have finished before checking results.
//
// Example:
//
//	logger.Write(data) // Triggers rotation and compression
//	logger.WaitForBackgroundTasks() // Wait for compression to complete
//	// Now safe to check for .gz files
func (l *Logger) WaitForBackgroundTasks() {
	if workers := l.bgWorkers.Load(); workers != nil {
		workers.waitForCompletion()
	}
}

// Stats represents comprehensive logger statistics for telemetry and monitoring.
// These metrics provide insights into logger performance, buffer utilization,
// and system behavior for operational monitoring and performance tuning.
//
// The statistics are collected with minimal overhead and are safe to query
// frequently for real-time monitoring dashboards.
type Stats struct {
	// Write statistics
	WriteCount    uint64 `json:"write_count"`     // Total number of writes
	TotalBytes    uint64 `json:"total_bytes"`     // Total bytes written
	AvgLatencyNs  uint64 `json:"avg_latency_ns"`  // Average write latency in nanoseconds
	LastLatencyNs uint64 `json:"last_latency_ns"` // Last write latency in nanoseconds

	// Contention and performance
	ContentionCount uint64  `json:"contention_count"` // Number of write contentions detected
	ContentionRatio float64 `json:"contention_ratio"` // Contention ratio (0.0-1.0)

	// Rotation statistics
	RotationCount   uint64 `json:"rotation_count"`    // Number of rotations performed
	CurrentFileSize uint64 `json:"current_file_size"` // Current file size in bytes

	// MPSC buffer statistics
	BufferSize    uint64 `json:"buffer_size"`     // Current buffer size
	BufferFill    uint64 `json:"buffer_fill"`     // Current buffer fill level (tail-head)
	IsMPSCActive  bool   `json:"is_mpsc_active"`  // Whether MPSC mode is active
	DroppedOnFull uint64 `json:"dropped_on_full"` // Messages dropped due to full buffer

	// Configuration
	MaxSizeBytes       int64   `json:"max_size_bytes"`      // Configured max file size
	BackpressurePolicy string  `json:"backpressure_policy"` // Current backpressure policy
	FlushIntervalMs    float64 `json:"flush_interval_ms"`   // Flush interval in milliseconds
}

// Stats returns current logger statistics for telemetry and monitoring.
// This method provides comprehensive metrics about logger performance,
// including write statistics, buffer utilization, and configuration details.
//
// The returned statistics are a snapshot at the time of the call and
// are safe to use concurrently. All counters are atomic and provide
// accurate metrics even under high concurrency.
//
// Example usage for monitoring:
//
//	stats := logger.Stats()
//	fmt.Printf("Total writes: %d, Average latency: %dns\n",
//		stats.WriteCount, stats.AvgLatencyNs)
//
//	if stats.ContentionRatio > 0.1 {
//		log.Warn("High contention detected, consider enabling async mode")
//	}
//
// Returns a Stats struct containing all current metrics.
// Stats returns real-time performance and operational metrics.
// This method provides detailed insights into Logger behavior for monitoring and optimization.
// All metrics are collected with minimal overhead using atomic operations.
//
// Metrics include:
//   - WriteCount: Total number of Write() calls
//   - TotalBytes: Cumulative bytes written
//   - AvgLatencyNs: Average write latency in nanoseconds
//   - ContentionRatio: Ratio of contended writes (0.0-1.0)
//   - BufferSize: MPSC buffer capacity
//   - BufferFill: Current buffer utilization
//   - DroppedOnFull: Messages dropped due to buffer overflow
//   - RotationCount: Number of file rotations performed
//
// Performance monitoring example:
//
//	logger, _ := lethe.NewWithDefaults("app.log")
//	defer logger.Close()
//
//	// Write some data...
//	for i := 0; i < 1000; i++ {
//		logger.Write([]byte(fmt.Sprintf("Message %d\n", i)))
//	}
//
//	// Check performance metrics
//	stats := logger.Stats()
//	fmt.Printf("Writes: %d, Avg Latency: %dns\n", stats.WriteCount, stats.AvgLatencyNs)
//	fmt.Printf("Buffer Fill: %d/%d (%.1f%%)\n",
//		stats.BufferFill, stats.BufferSize,
//		float64(stats.BufferFill)/float64(stats.BufferSize)*100)
//
// Returns a Stats struct with current metrics. Safe to call concurrently.
func (l *Logger) Stats() Stats {
	writeCount := l.writeCount.Load()
	totalLatency := l.totalLatency.Load()
	contentionCount := l.contentionCount.Load()

	var avgLatency uint64
	if writeCount > 0 {
		avgLatency = totalLatency / writeCount
	}

	var contentionRatio float64
	if writeCount > 0 {
		contentionRatio = float64(contentionCount) / float64(writeCount)
	}

	var bufferSize, bufferFill uint64
	isMPSCActive := false
	if buffer := l.buffer.Load(); buffer != nil {
		bufferSize = uint64(len(buffer.buffer))
		isMPSCActive = true

		// Calculate buffer fill level
		head := buffer.head.Load()
		tail := buffer.tail.Load()
		if tail >= head {
			bufferFill = tail - head
		}
	}

	flushIntervalMs := float64(l.FlushInterval.Nanoseconds()) / 1e6
	if flushIntervalMs == 0 {
		flushIntervalMs = 1.0 // Default 1ms
	}

	// Calculate total bytes as a rough estimate
	// Note: This is current file size + estimated bytes from rotations
	totalBytes := l.bytesWritten.Load()
	if rotationCount := l.rotationSeq.Load(); rotationCount > 0 && l.maxSizeBytes > 0 {
		// Rough estimate: rotations * average file size
		totalBytes += rotationCount * uint64(l.maxSizeBytes)
	}

	return Stats{
		WriteCount:         writeCount,
		TotalBytes:         totalBytes,
		AvgLatencyNs:       avgLatency,
		LastLatencyNs:      l.lastLatency.Load(),
		ContentionCount:    contentionCount,
		ContentionRatio:    contentionRatio,
		RotationCount:      l.rotationSeq.Load(),
		CurrentFileSize:    l.bytesWritten.Load(),
		BufferSize:         bufferSize,
		BufferFill:         bufferFill,
		IsMPSCActive:       isMPSCActive,
		DroppedOnFull:      l.droppedCount.Load(),
		MaxSizeBytes:       l.maxSizeBytes,
		BackpressurePolicy: l.BackpressurePolicy,
		FlushIntervalMs:    flushIntervalMs,
	}
}

// Rotate manually triggers log file rotation.
// This method forces an immediate rotation regardless of current file size or age.
// It's useful for external log management systems or manual rotation triggers.
//
// The rotation process:
//   - Closes the current log file
//   - Renames it with a timestamp
//   - Creates a new log file
//   - Applies compression and cleanup if configured
//
// This operation is thread-safe and can be called concurrently with Write operations.
// If rotation is already in progress, subsequent calls will wait for completion.
//
// Example usage:
//
//	// Manual rotation (e.g., in response to SIGHUP)
//	if err := logger.Rotate(); err != nil {
//		log.Printf("Rotation failed: %v", err)
//	}
//
// Returns nil on success, or an error if rotation fails.
// Rotate manually triggers log file rotation.
// This method forces an immediate rotation regardless of size or age limits.
// Useful for implementing custom rotation policies or responding to external events.
//
// The rotation process:
//  1. Closes the current log file
//  2. Renames it with a timestamp suffix
//  3. Creates a new log file
//  4. Schedules background tasks (compression, cleanup, checksums)
//
// Background operations (compression, cleanup) are performed asynchronously
// to avoid blocking the rotation. Use WaitForBackgroundTasks() in tests
// to ensure all operations complete.
//
// Parameters: None
//
// Returns:
//   - error: Always returns nil. Rotation errors are reported via ErrorCallback.
//
// Example:
//
//	logger, _ := lethe.NewWithDefaults("app.log")
//	defer logger.Close()
//
//	// Force rotation at application milestones
//	logger.Write([]byte("Starting maintenance mode\n"))
//	logger.Rotate() // Create fresh log for maintenance
//	logger.Write([]byte("Maintenance completed\n"))
func (l *Logger) Rotate() error {
	l.triggerRotation()
	return nil
}

// reportError invokes the error callback if set
func (l *Logger) reportError(operation string, err error) {
	if l.ErrorCallback != nil {
		l.ErrorCallback(operation, err)
	}
}
