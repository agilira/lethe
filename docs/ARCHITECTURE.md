# Lethe Architecture

## Overview

Lethe is a high-performance log rotation library designed for Go applications requiring zero-lock, zero-allocation logging with advanced features like compression, checksums, and time-based rotation. The architecture prioritizes performance through lock-free algorithms, atomic operations, and intelligent auto-scaling between synchronous and asynchronous modes.

## Core Design Principles

### 1. Zero-Lock Architecture
- **Atomic Operations**: All shared state uses `sync/atomic` primitives
- **Compare-And-Swap (CAS)**: Lock-free coordination for rotation and buffer management
- **Memory Ordering**: Proper memory barriers ensure data consistency without mutex overhead

### 2. Zero-Allocation Hot Paths
- **Pre-allocated Buffers**: Ring buffer uses fixed-size slots with atomic pointers
- **Time Caching**: Integration with `go-timecache` reduces `time.Now()` allocations by 10x
- **Buffer Pooling**: Safe buffer pool prevents garbage collection pressure

### 3. Auto-Scaling Performance
- **Contention Detection**: Automatic switching from sync to MPSC mode under load
- **Adaptive Buffering**: Dynamic buffer resizing based on write velocity
- **Intelligent Backpressure**: Multiple policies for handling buffer overflow

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Lethe Log Rotator                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Writer    │  │   Rotator   │  │   Background        │  │
│  │   Interface │  │   Engine    │  │   Workers           │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│         │                │                    │              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   MPSC      │  │   File      │  │   Compression       │  │
│  │   Buffer    │  │   Manager   │  │   & Cleanup         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. Writer Interface (`lethe.go`)
- **Primary API**: Implements `io.Writer` for universal compatibility
- **Auto-Scaling Logic**: Monitors contention and latency to switch modes
- **Performance Metrics**: Collects telemetry for operational monitoring

#### 2. MPSC Buffer System (`buffer.go`)
- **Lock-Free Ring Buffer**: Multi-producer single-consumer pattern
- **Atomic Operations**: CAS-based head/tail management
- **Buffer Pooling**: Safe reuse of byte slices to prevent allocations

#### 3. Rotation Engine (`rotation.go`)
- **File Management**: Atomic file pointer updates and lazy initialization
- **Background Workers**: Asynchronous compression, cleanup, and checksum generation
- **Crash Consistency**: Temporary files ensure atomic operations

#### 4. Configuration System (`config.go`)
- **String-Based Parsing**: Human-readable size and duration formats
- **Cross-Platform Support**: OS-specific path validation and file permissions
- **Retry Logic**: Resilient file operations with exponential backoff

## Data Flow Architecture

### Synchronous Mode (Default)
```
Write() → File.Write() → Atomic Size Update → Rotation Check
```

### Asynchronous Mode (MPSC)
```
Write() → Ring Buffer → Consumer Goroutine → File.Write() → Background Tasks
```

### Auto-Scaling Decision Tree
```
Write() → Contention Check → Latency Analysis → Mode Selection
    ↓
[Sync Mode] ← Low Load ← [High Load] → [MPSC Mode]
```

## Performance Optimizations

### 1. Lock-Free Algorithms
- **Ring Buffer**: CAS-based producer/consumer coordination
- **Rotation Coordination**: Single-writer rotation with atomic flags
- **State Management**: All counters and flags use atomic operations

### 2. Memory Management
- **Buffer Pool**: Pre-allocated byte slices with safe reuse patterns
- **Time Caching**: Reduces system call overhead for timestamp operations
- **Zero-Copy Writes**: `WriteOwned()` API for ownership transfer

### 3. I/O Optimizations
- **Lazy File Creation**: Files created only when first write occurs
- **Background Processing**: Compression and cleanup don't block writes
- **Adaptive Flushing**: Dynamic timing based on buffer utilization

## Concurrency Model

### Thread Safety Guarantees
- **Multiple Writers**: Safe concurrent writes from multiple goroutines
- **Single Rotator**: Only one goroutine performs rotation at a time
- **Background Workers**: Isolated worker pool for non-critical operations

### Memory Consistency
- **Atomic Operations**: All shared state updates are atomic
- **Memory Barriers**: Proper ordering for cross-goroutine visibility
- **Lock-Free Progress**: No goroutine can block another indefinitely

## Error Handling Architecture

### Error Propagation
- **Callback System**: Configurable error reporting via `ErrorCallback`
- **Graceful Degradation**: Fallback mechanisms for non-critical failures
- **Retry Logic**: Automatic retry for transient filesystem errors

### Failure Modes
- **File System Errors**: Retry with exponential backoff
- **Buffer Overflow**: Configurable backpressure policies
- **Resource Exhaustion**: Graceful degradation to sync mode

## Integration Patterns

### Framework Compatibility
- **Iris Integration**: Native - ultra-high performance logging library
- **Standard Library**: Direct `io.Writer` implementation
- **Structured Logging**: Compatible with logrus, zap, zerolog
- **High-Performance**: Custom adapters for high-throughput applications

### Zero-Copy Integration with Iris
```go
// Iris can transfer buffer ownership to Lethe
buffer := iris.GetBuffer()
logger.WriteOwned(buffer) // No copying, ownership transfer
```

## Configuration Architecture

### Constructor Hierarchy
1. **Legacy**: `New()` - Backward compatibility
2. **Modern**: `NewSimple()` - String-based configuration
3. **Presets**: `NewWithDefaults()`, `NewDaily()`, `NewWeekly()`
4. **Custom**: `NewWithConfig()` - Full control

### String-Based Configuration
- **Size Formats**: "100MB", "1GB", "500KB" with case-insensitive parsing
- **Duration Formats**: "7d", "24h", "30m" with standard Go duration support
- **Validation**: Cross-platform path and permission validation

## Monitoring and Telemetry

### Performance Metrics
- **Write Statistics**: Count, latency, throughput measurements
- **Buffer Utilization**: Fill levels and contention ratios
- **Rotation Metrics**: Frequency and success rates

### Operational Insights
- **Auto-Scaling Events**: Mode transitions and trigger conditions
- **Error Rates**: Categorized error reporting for troubleshooting
- **Resource Usage**: Memory and file handle consumption

## Security Considerations

### File System Security
- **Path Validation**: Prevents directory traversal attacks
- **Permission Management**: Configurable file modes with secure defaults
- **Input Sanitization**: Cross-platform filename sanitization

### Data Integrity
- **SHA-256 Checksums**: Optional integrity verification for rotated files
- **Atomic Operations**: Crash-safe file operations with temporary files
- **Error Isolation**: Failures in background tasks don't affect logging

## Testing Architecture

### Test Coverage Strategy
- **Unit Tests**: Individual component testing with edge cases
- **Integration Tests**: Framework compatibility and real-world scenarios
- **Performance Tests**: Benchmarking and stress testing
- **Cross-Platform**: OS-specific behavior validation

### Quality Assurance
- **Race Condition Detection**: Go race detector validation
- **Memory Leak Testing**: Long-running test scenarios
- **Error Injection**: Simulated failure mode testing

## Future Architecture Considerations

### Scalability Enhancements
- **Multi-File Support**: Parallel logging to multiple files
- **Network Logging**: Remote log aggregation capabilities
- **Plugin System**: Extensible compression and transport mechanisms


## Conclusion

Lethe's architecture represents a careful balance between performance, reliability, and usability. The zero-lock design ensures maximum throughput under high concurrency, while the auto-scaling system provides optimal performance across different load patterns. The modular design allows for easy integration with existing logging frameworks while providing advanced features for demanding applications.

The architecture prioritizes operational excellence through comprehensive telemetry, robust error handling, and cross-platform compatibility. This makes Lethe suitable for both simple applications requiring basic log rotation and complex systems demanding high-performance, reliable logging infrastructure.

---

Lethe • an AGILira fragment