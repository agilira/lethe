# Lethe API Documentation

## Overview

Lethe is a high-performance, universal log rotation library for Go applications. It provides zero-lock, zero-allocation logging with automatic file rotation, compression, and professional-grade features. The library is designed for maximum performance and reliability in production environments.

## Core Types

### Logger

The main type that provides log rotation functionality. It implements the `io.Writer` interface for seamless integration with Go's standard library and third-party logging frameworks.

```go
type Logger struct {
    // Configuration fields (see LoggerConfig for details)
    Filename           string
    MaxSizeStr         string
    MaxAgeStr          string
    MaxBackups         int
    Compress           bool
    Checksum           bool
    Async              bool
    BufferSize         int
    BackpressurePolicy string
    // ... additional fields
}
```

### LoggerConfig

Configuration structure for creating Logger instances with full control over all options.

```go
type LoggerConfig struct {
    Filename           string
    MaxSizeStr         string
    MaxAgeStr          string
    MaxBackups         int
    MaxFileAge         time.Duration
    LocalTime          bool
    Compress           bool
    Checksum           bool
    Async              bool
    ErrorCallback      func(operation string, err error)
    FileMode           os.FileMode
    RetryCount         int
    RetryDelay         time.Duration
    BufferSize         int
    BackpressurePolicy string
    FlushInterval      time.Duration
    AdaptiveFlush      bool
}
```

### Stats

Performance and operational metrics for monitoring and optimization.

```go
type Stats struct {
    WriteCount         uint64
    TotalBytes         uint64
    AvgLatencyNs       uint64
    LastLatencyNs      uint64
    ContentionCount    uint64
    ContentionRatio    float64
    RotationCount      uint64
    CurrentFileSize    uint64
    BufferSize         uint64
    BufferFill         uint64
    IsMPSCActive       bool
    DroppedOnFull      uint64
    MaxSizeBytes       int64
    BackpressurePolicy string
    FlushIntervalMs    float64
}
```

## Constructor Functions

### New

Creates a new Logger with basic configuration and safe defaults.

```go
func New(filename string, maxSizeMB int, maxBackups int) (*Logger, error)
```

**Parameters:**
- `filename`: Path to the log file (required)
- `maxSizeMB`: Maximum file size in MB before rotation (0 = no size limit)
- `maxBackups`: Number of backup files to keep (0 = keep all)

**Example:**
```go
logger, err := lethe.New("app.log", 100, 3)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### NewSimple

Creates a Logger with modern string-based configuration and performance optimizations.

```go
func NewSimple(filename, maxSize string, maxBackups int) (*Logger, error)
```

**Parameters:**
- `filename`: Path to the log file (required)
- `maxSize`: Maximum file size as string (e.g., "100MB", "1GB")
- `maxBackups`: Number of backup files to keep (0 = keep all)

**Features enabled by default:**
- Async mode for better performance
- Adaptive backpressure policy
- 4KB buffer for efficient I/O

**Example:**
```go
logger, err := lethe.NewSimple("app.log", "100MB", 5)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### NewWithDefaults

Creates a Logger with production-ready defaults. Recommended for most applications.

```go
func NewWithDefaults(filename string) (*Logger, error)
```

**Production defaults:**
- MaxSizeStr: "100MB" (rotates when file reaches 100MB)
- MaxAgeStr: "7d" (rotates weekly for fresh logs)
- MaxBackups: 10 (keeps 10 backup files)
- Compress: true (saves disk space)
- Async: true (better performance)
- BackpressurePolicy: "adaptive" (intelligent overflow handling)
- LocalTime: true (local timestamps in backups)

**Example:**
```go
logger, err := lethe.NewWithDefaults("app.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Use with standard library
log.SetOutput(logger)
```

### NewDaily

Creates a Logger optimized for daily rotation.

```go
func NewDaily(filename string) (*Logger, error)
```

**Configuration:**
- MaxSizeStr: "50MB" (reasonable daily file size)
- MaxAgeStr: "24h" (rotates every 24 hours)
- MaxBackups: 7 (keeps one week of daily logs)
- Compress: true (saves storage space)
- Async: true (better performance)
- LocalTime: true (daily rotation aligned with local timezone)

**Example:**
```go
logger, err := lethe.NewDaily("daily.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### NewWeekly

Creates a Logger optimized for weekly rotation.

```go
func NewWeekly(filename string) (*Logger, error)
```

**Configuration:**
- MaxSizeStr: "200MB" (larger size for weekly accumulation)
- MaxAgeStr: "7d" (rotates every 7 days)
- MaxBackups: 4 (keeps one month of weekly logs)
- Compress: true (essential for larger files)
- Async: true (better performance)
- LocalTime: true (weekly rotation aligned with local timezone)

**Example:**
```go
logger, err := lethe.NewWeekly("weekly.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### NewDevelopment

Creates a Logger optimized for development and debugging.

```go
func NewDevelopment(filename string) (*Logger, error)
```

**Development-optimized configuration:**
- MaxSizeStr: "10MB" (small files for easier handling)
- MaxAgeStr: "1h" (frequent rotation for fresh logs)
- MaxBackups: 5 (keep recent history without clutter)
- Compress: false (immediate file access for debugging)
- Async: false (synchronous writes for immediate visibility)
- BackpressurePolicy: "fallback" (simple error handling)
- LocalTime: true (local timestamps for debugging)

**Example:**
```go
logger, err := lethe.NewDevelopment("debug.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### NewWithConfig

Creates a Logger with full configuration control.

```go
func NewWithConfig(config *LoggerConfig) (*Logger, error)
```

**Parameters:**
- `config`: LoggerConfig pointer with desired settings (required, non-nil)

**Example with professional features:**
```go
config := &lethe.LoggerConfig{
    Filename:           "app.log",
    MaxSizeStr:         "500MB",
    MaxAgeStr:          "30d",
    MaxBackups:         20,
    MaxFileAge:         180 * 24 * time.Hour, // 6 months backup retention
    Compress:           true,
    Checksum:           true,                 // Enable data integrity
    Async:              true,
    BufferSize:         4096,                 // High-performance buffer
    BackpressurePolicy: "adaptive",
    LocalTime:          true,
    ErrorCallback: func(eventType string, err error) {
        log.Printf("Log error (%s): %v", eventType, err)
    },
}
logger, err := lethe.NewWithConfig(config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

## Core Methods

### Write

Implements the `io.Writer` interface for universal compatibility.

```go
func (l *Logger) Write(data []byte) (int, error)
```

**Features:**
- Zero allocations, zero locks, thread-safe
- Automatic file creation and rotation
- Auto-scaling to MPSC mode under high load
- Error reporting via ErrorCallback
- Performance metrics collection

**Returns:**
- `int`: Number of bytes written (always len(data) on success)
- `error`: Any error encountered during writing or rotation

**Example:**
```go
logger, _ := lethe.NewWithDefaults("app.log")
defer logger.Close()

// Direct usage
logger.Write([]byte("Application started\n"))

// With standard library
log.SetOutput(logger)
log.Println("This goes through lethe")

// With frameworks
logrus.SetOutput(logger)
```

### WriteOwned

Writes data with ownership transfer for zero-copy optimization in MPSC mode.

```go
func (l *Logger) WriteOwned(data []byte) (int, error)
```

**Performance characteristics:**
- Sync mode: Behaves identically to Write()
- Async mode: Avoids buffer copying, reducing memory allocations
- Originally designed for Iris

**Usage:**
```go
buf := make([]byte, len(message))
copy(buf, message)
n, err := logger.WriteOwned(buf)
// buf must not be used after this call
```

### Close

Gracefully shuts down the Logger and releases all resources.

```go
func (l *Logger) Close() error
```

**Shutdown sequence:**
1. Stops the MPSC consumer (if running) and flushes pending writes
2. Stops background workers (compression, cleanup, checksums)
3. Stops the time cache to prevent memory leaks
4. Closes the current log file

**Important:** Always call Close when shutting down to prevent data loss.

**Example:**
```go
logger, err := lethe.NewWithDefaults("app.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close() // Ensures cleanup on exit

// Use logger...
logger.Write([]byte("Application shutting down\n"))
// Close() called automatically via defer
```

### Rotate

Manually triggers log file rotation.

```go
func (l *Logger) Rotate() error
```

**Features:**
- Forces immediate rotation regardless of size or age limits
- Thread-safe and can be called concurrently with Write operations
- Useful for implementing custom rotation policies or responding to external events

**Rotation process:**
1. Closes the current log file
2. Renames it with a timestamp suffix
3. Creates a new log file
4. Schedules background tasks (compression, cleanup, checksums)

**Example:**
```go
logger, _ := lethe.NewWithDefaults("app.log")
defer logger.Close()

// Force rotation at application milestones
logger.Write([]byte("Starting maintenance mode\n"))
logger.Rotate() // Create fresh log for maintenance
logger.Write([]byte("Maintenance completed\n"))
```

### Stats

Returns current logger statistics for telemetry and monitoring.

```go
func (l *Logger) Stats() Stats
```

**Metrics include:**
- WriteCount: Total number of Write() calls
- TotalBytes: Cumulative bytes written
- AvgLatencyNs: Average write latency in nanoseconds
- ContentionRatio: Ratio of contended writes (0.0-1.0)
- BufferSize: MPSC buffer capacity
- BufferFill: Current buffer utilization
- DroppedOnFull: Messages dropped due to buffer overflow
- RotationCount: Number of file rotations performed

**Example:**
```go
logger, _ := lethe.NewWithDefaults("app.log")
defer logger.Close()

// Write some data...
for i := 0; i < 1000; i++ {
    logger.Write([]byte(fmt.Sprintf("Message %d\n", i)))
}

// Check performance metrics
stats := logger.Stats()
fmt.Printf("Writes: %d, Avg Latency: %dns\n", stats.WriteCount, stats.AvgLatencyNs)
fmt.Printf("Buffer Fill: %d/%d (%.1f%%)\n",
    stats.BufferFill, stats.BufferSize,
    float64(stats.BufferFill)/float64(stats.BufferSize)*100)
```

### WaitForBackgroundTasks

Waits for all background tasks (compression, cleanup, checksums) to complete.

```go
func (l *Logger) WaitForBackgroundTasks()
```

**Use case:** Useful in tests to ensure all operations have finished before checking results.

**Example:**
```go
logger.Write(data) // Triggers rotation and compression
logger.WaitForBackgroundTasks() // Wait for compression to complete
// Now safe to check for .gz files
```

## Configuration Utilities

### ParseSize

Converts size strings like "100MB", "1GB" to bytes.

```go
func ParseSize(s string) (int64, error)
```

**Supported formats:**
- B, KB, MB, GB, TB (both 1000 and 1024 based)
- Case-insensitive input
- Single-letter units (K, M, G, T)

**Example:**
```go
size, err := lethe.ParseSize("100MB")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Size in bytes: %d\n", size)
```

### ParseDuration

Converts duration strings like "7d", "24h" to time.Duration.

```go
func ParseDuration(s string) (time.Duration, error)
```

**Supported formats:**
- Standard Go durations (ns, us, ms, s, m, h)
- Custom extensions: d (days), w (weeks), y (years)

**Example:**
```go
duration, err := lethe.ParseDuration("7d")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Duration: %v\n", duration)
```

### SanitizeFilename

Removes or replaces invalid characters for cross-platform compatibility.

```go
func SanitizeFilename(filename string) string
```

**Platform-specific handling:**
- Windows: Removes < > : " | ? * and control characters
- Unix-like: Removes null characters

### ValidatePathLength

Checks if the path length is within OS limits.

```go
func ValidatePathLength(path string) error
```

**Limits:**
- Windows: 260 characters
- Unix-like: 4096 characters

### RetryFileOperation

Executes a file operation with retry logic for cross-platform reliability.

```go
func RetryFileOperation(operation func() error, retryCount int, retryDelay time.Duration) error
```

**Use case:** Handles transient failures due to antivirus scans, network issues, or high load.

## Integration Examples

### Standard Library Integration

```go
logger, err := lethe.NewWithDefaults("app.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Redirect standard library logging
log.SetOutput(logger)
log.Println("This goes through lethe")
```

### High-Performance Async Mode

```go
config := &lethe.LoggerConfig{
    Filename:           "app.log",
    MaxSizeStr:         "10MB",
    Async:              true,
    BufferSize:         4096,
    BackpressurePolicy: "adaptive",
}

logger, err := lethe.NewWithConfig(config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// High-throughput writes
for i := 0; i < 1000; i++ {
    logger.Write([]byte(fmt.Sprintf("High-throughput message %d\n", i)))
}
```

### Professional Features

```go
config := &lethe.LoggerConfig{
    Filename:           "app.log",
    MaxSizeStr:         "100MB",
    MaxAgeStr:          "7d",
    MaxBackups:         20,
    Compress:           true,
    Checksum:           true,                 // SHA-256 checksums
    Async:              true,
    ErrorCallback: func(eventType string, err error) {
        log.Printf("Log error (%s): %v", eventType, err)
    },
}

logger, err := lethe.NewWithConfig(config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### Zero-Copy Integration with Iris

```go
// Create buffer for zero-copy write
message := "Zero-copy message\n"
buf := make([]byte, len(message))
copy(buf, message)

// Transfer ownership to logger
n, err := logger.WriteOwned(buf)
// buf must not be used after this point
```

## Performance Characteristics

### Sync Mode
- Direct writes with immediate durability
- Zero allocations in hot path
- Lock-free design using atomic operations

### Async Mode (MPSC)
- Buffered writes with lock-free ring buffer
- Multi-Producer Single-Consumer pattern
- Automatic backpressure handling
- Adaptive flush timing

### Auto-scaling
- Automatic mode switching based on contention detection
- Performance metrics collection
- Intelligent buffer resizing

## Error Handling

All errors are reported through the optional `ErrorCallback` function:

```go
config := &lethe.LoggerConfig{
    Filename: "app.log",
    ErrorCallback: func(operation string, err error) {
        log.Printf("Log error in %s: %v", operation, err)
    },
}
```

**Common operations that trigger callbacks:**
- File creation/opening failures
- Rotation errors
- Compression failures
- Cleanup errors
- Checksum generation errors

## Thread Safety

Lethe is designed for high-concurrency environments:

- **Thread-safe:** All public methods can be called concurrently
- **Lock-free:** Uses atomic operations instead of mutexes
- **Zero allocations:** Hot path operations don't allocate memory
- **MPSC mode:** Optimized for high-throughput scenarios

## Best Practices

1. **Always use defer logger.Close()** to ensure proper cleanup
2. **Use NewWithDefaults()** for production applications
3. **Enable async mode** for high-throughput scenarios
4. **Set appropriate buffer sizes** based on your workload
5. **Monitor statistics** using the Stats() method
6. **Handle errors** through ErrorCallback for production reliability
7. **Use string-based configuration** (MaxSizeStr, MaxAgeStr) for flexibility

## Compatibility

Lethe implements the standard `io.Writer` interface, making it compatible with:

- Go standard library (`log` package)
- Iris (primary integration) - ultra-high performance logging library
- Popular logging frameworks (Logrus, Zap, Zerolog)
- Web frameworks (Gin, Echo)
- Any library that accepts `io.Writer`

---

Lethe • an AGILira fragment