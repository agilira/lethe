# Low-Level API Examples

**IMPORTANT PERFORMANCE NOTE**

**These examples use Lethe's direct API for MAXIMUM raw performance.**

**DO NOT compare these numbers with framework integration examples!**

This is like comparing:
- Raw file.Write() operations (this directory)
- vs Full database ORM with SQL, transactions, validations (iris-integration/)

The performance difference is **by design** - you trade features for speed.

---

This directory demonstrates direct Lethe API usage for applications that need absolute maximum performance. These examples show raw buffer operations, zero-copy writes, and direct WriteOwned calls without any framework overhead.

## Quick Start

```bash
cd advanced/
go run .
```

The examples will create various log files in the `./logs/` directory demonstrating different advanced capabilities.

## Advanced Features Covered

### 1. MPSC Buffer Performance Test
- **Feature**: Multi-Producer Single-Consumer buffer architecture
- **Demonstration**: 10 concurrent producers writing 1000 entries each
- **Benefits**: Lock-free concurrent access, optimal CPU cache utilization
- **Use Case**: High-throughput applications with multiple logging sources

### 2. Zero-Copy WriteOwned Operations
- **Feature**: Zero-copy buffer transfer using `WriteOwned()` method
- **Demonstration**: 1000 pre-allocated buffers transferred without copying
- **Benefits**: Reduced memory allocations, lower GC pressure, higher throughput
- **Use Case**: Performance-critical applications requiring sub-microsecond latencies

### 3. Backpressure Policy Testing
- **Feature**: Configurable overflow handling strategies
- **Policies Tested**:
  - `adaptive`: Intelligent buffering with dynamic scaling
  - `drop`: Drop new entries when buffer is full
  - `block`: Block writers until buffer space available
- **Use Case**: Managing load spikes and preventing memory exhaustion

### 4. High-Throughput Concurrent Logging
- **Feature**: Realistic multi-goroutine logging simulation
- **Demonstration**: CPU-count × 2 workers simulating different service types
- **Worker Types**: API requests, database queries, cache operations, authentication
- **Benefits**: Real-world performance testing with mixed workloads

### 5. Advanced Configuration Features
- **Features Demonstrated**:
  - **Checksums**: Data integrity verification for log entries
  - **Error Callbacks**: Real-time monitoring of rotation events
  - **Time-based Rotation**: Automatic rotation based on time intervals
  - **Compression**: Automatic gzip compression of rotated files
- **Use Case**: Production environments requiring data integrity and monitoring

### 6. Error Handling and Recovery
- **Feature**: Robust error handling and graceful degradation
- **Scenarios Tested**:
  - Normal operation logging
  - Error event collection and reporting
  - Graceful shutdown handling
  - Write-after-close error management
- **Benefits**: Production reliability and debugging capabilities

### 7. Performance Benchmarking
- **Feature**: Comprehensive performance measurement across configurations
- **Benchmarks**:
  - Synchronous vs Asynchronous writing
  - Small vs Large buffer sizes
  - Compression impact on throughput
- **Metrics**: Operations per second, total execution time
- **Use Case**: Performance optimization and capacity planning

## Configuration Options in Detail

### MPSC Buffer Configuration
```go
config := &lethe.LoggerConfig{
    BufferSize:         16384,           // 16KB MPSC buffer
    Async:              true,            // Enable async processing
    BackpressurePolicy: "adaptive",      // Intelligent overflow handling
}
```

### Zero-Copy Operations
```go
// Pre-allocate buffer
buffer := []byte("log entry data\n")

// Transfer ownership to logger (zero-copy)
n, err := logger.WriteOwned(buffer)

// buffer is now owned by logger, don't modify it
```

### Backpressure Policies
- **`adaptive`**: Dynamically adjusts buffer size and batching based on load
- **`drop`**: Discards new entries when buffer capacity is reached
- **`block`**: Blocks writing goroutines until buffer space becomes available

### Advanced Error Handling
```go
config := &lethe.LoggerConfig{
    ErrorCallback: func(eventType string, err error) {
        switch eventType {
        case "checksum":
            // Handle data integrity errors
        case "rotation":
            // Monitor rotation events
        case "backpressure":
            // Handle overflow conditions
        }
    },
}
```

## Performance Characteristics

### Typical Throughput (varies by hardware)
- **Async + Large Buffer**: ~800,000+ ops/sec
- **Async + Small Buffer**: ~500,000+ ops/sec  
- **Sync + Small Buffer**: ~200,000+ ops/sec
- **Zero-Copy Operations**: ~1,000,000+ ops/sec

### Memory Efficiency
- **MPSC Buffers**: Lock-free, cache-friendly access patterns
- **Zero-Copy**: Eliminates buffer copying overhead
- **Adaptive Backpressure**: Prevents unbounded memory growth

### Latency Characteristics
- **Async Mode**: Sub-microsecond write latencies
- **Zero-Copy**: Minimal allocation overhead
- **Compression**: Balanced I/O vs CPU trade-off

## Production Configuration Examples

### High-Throughput Service
```go
config := &lethe.LoggerConfig{
    Filename:           "service.log",
    MaxSizeStr:         "200MB",
    MaxBackups:         20,
    Async:              true,
    BufferSize:         32768,           // 32KB buffer
    BackpressurePolicy: "adaptive",
    Compress:           true,
    Checksum:           true,
}
```

### Memory-Constrained Environment
```go
config := &lethe.LoggerConfig{
    Filename:           "constrained.log", 
    MaxSizeStr:         "50MB",
    MaxBackups:         5,
    Async:              true,
    BufferSize:         4096,            // 4KB buffer
    BackpressurePolicy: "drop",          // Prefer dropping over blocking
    Compress:           true,
}
```

### Data-Critical Application
```go
config := &lethe.LoggerConfig{
    Filename:           "critical.log",
    MaxSizeStr:         "100MB", 
    MaxBackups:         50,              // High retention
    Async:              false,           // Synchronous for durability
    Checksum:           true,            // Data integrity verification
    BackpressurePolicy: "block",         // Never drop entries
}
```

## File Output

After running the examples, you'll find these log files in `./logs/`:

```
logs/
├── mpsc-test.log                    # MPSC buffer test output
├── zero-copy.log                    # Zero-copy operations test
├── backpressure-adaptive.log        # Adaptive backpressure test
├── backpressure-drop.log           # Drop backpressure test  
├── backpressure-block.log          # Block backpressure test
├── concurrent.log                   # High-throughput concurrent test
├── advanced-config.log              # Advanced configuration features
├── error-handling.log               # Error handling test
└── bench-*.log                     # Performance benchmark outputs
```

## Integration with Other Examples

1. **Basic Examples**: Start with `../basic/` to understand fundamental concepts
2. **Iris Integration**: See `../iris-integration/` for zero-copy adapter usage
3. **Production Setup**: Check `../production/` for deployment configurations

## Common Advanced Patterns

### High-Performance Logging
```go
// Use WriteOwned for zero-copy when possible
buffer := make([]byte, 0, 1024)
buffer = append(buffer, logData...)
logger.WriteOwned(buffer)
```

### Monitoring and Alerting
```go
config.ErrorCallback = func(eventType string, err error) {
    if eventType == "backpressure" {
        // Alert on backpressure events
        alerting.SendAlert("Log backpressure detected", err)
    }
}
```

### Load Testing
```go
// Measure throughput under load
start := time.Now()
for i := 0; i < testEntries; i++ {
    logger.Write(logEntry)
}
throughput := float64(testEntries) / time.Since(start).Seconds()
```

## Best Practices

1. **Buffer Sizing**: Start with 8KB-16KB buffers, scale based on load testing
2. **Backpressure Policy**: Use "adaptive" for most cases, "drop" for memory-constrained environments
3. **Zero-Copy**: Use `WriteOwned()` for high-frequency logging in hot paths
4. **Error Monitoring**: Always implement error callbacks for production environments
5. **Performance Testing**: Benchmark your specific workload patterns before deployment
