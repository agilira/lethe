# Iris Integration - Magic API

**Automatic Integration between Iris and Lethe**

This document describes the Magic API that provides automatic, zero-configuration integration between Iris logging library and Lethe log rotation with runtime optimization detection.

## Overview

The Magic API eliminates the need for manual adapter code by providing:

- **Zero Configuration**: `lethe.QuickStart()` creates production-ready logging in one line
- **Runtime Integration**: Automatic detection and optimization capabilities
- **Zero-Copy Optimization**: Automatic `WriteOwned()` method utilization when available
- **Graceful Fallback**: Works with any logger, optimizes specifically with Iris
- **Hot Reload**: Built-in support for dynamic configuration updatesion - Magic API

**Integration between Iris and Lethe**

This document describes the 'Magic API' that provides automatic, zero-configuration integration between Iris logging library and Lethe log rotation, delivering "It Just Works" level seamless operation.

## Overview

The Magic API eliminates the need for manual adapter code by providing:

- **Zero Configuration**: `lethe.QuickStart()` creates production-ready logging in one line
- **Zero Config Integration**: Automatic runtime detection and optimization
- **Zero-Copy Optimization**: Automatic `WriteOwned()` method utilization when available
- **Graceful Fallback**: Works with any logger, optimizes specifically with Iris
- **Hot Reload**: Built-in support for dynamic configuration updates

## Quick Start

### Simplest Integration (Recommended)

```go
package main

import (
    "github.com/agilira/iris"
    "github.com/agilira/lethe"
)

func main() {
    // One line - production ready with rotation!
    writer := lethe.QuickStart("app.log")
    defer writer.Close()
    
    // Direct Iris integration with automatic optimization
    logger, err := iris.New(iris.Config{
        Output:  writer,
        Encoder: iris.NewJSONEncoder(),
        Level:   iris.Info,
    })
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    logger.Start()
    
    // Use normally - automatic optimization is applied
    logger.Info("Magic API integration active")
    logger.With(
        iris.String("performance", "optimized"),
        iris.Bool("zero_config", true),
    ).Info("Logging with automatic rotation")
}
```

### Custom Configuration

```go
// Custom configuration with automatic Iris optimization
writer := lethe.NewIrisWriter("app.log", &lethe.Logger{
    MaxSizeStr:         "100MB",
    MaxBackups:         5,
    Async:              true,
    BufferSize:         16384,
    BackpressurePolicy: "adaptive",
    Compress:           true,
})
defer writer.Close()

logger, err := iris.New(iris.Config{Output: writer})
```

## Magic API Reference

### lethe.QuickStart(filename string) *IrisIntegration

Creates a Lethe writer optimized for Iris with sensible production defaults.

**Defaults:**
- Max file size: 100MB
- Max backups: 5
- Compression: Enabled
- Async: Enabled
- Buffer size: 8KB
- Backpressure: Adaptive

**Example:**
```go
writer := lethe.QuickStart("production.log")
defer writer.Close()
```

### lethe.NewIrisWriter(filename string, config *Logger) *IrisIntegration

Creates a Lethe writer with custom configuration optimized for Iris integration.

**Parameters:**
- `filename`: Target log file path
- `config`: Lethe logger configuration (nil for defaults)

**Example:**
```go
writer := lethe.NewIrisWriter("custom.log", &lethe.Logger{
    MaxSizeStr: "200MB",
    MaxBackups: 10,
    Compress:   true,
    Async:      true,
    BufferSize: 32768,
})
```

## Magic API Features

## Runtime Detection Features

### Automatic Detection

The Magic API automatically detects and enables advanced capabilities:

```go
writer := lethe.QuickStart("app.log")

// Check detected capabilities
if provider, ok := writer.(interface{ GetOptimalBufferSize() int }); ok {
    size := provider.GetOptimalBufferSize()
    fmt.Printf("Optimal buffer: %d bytes\n", size)
}

if provider, ok := writer.(interface{ SupportsHotReload() bool }); ok {
    hotReload := provider.SupportsHotReload()
    fmt.Printf("Hot reload: %v\n", hotReload)
}

// WriteOwned zero-copy optimization automatically used by Iris
```

### Detected Capabilities

1. **WriteOwned() Zero-Copy**: Eliminates buffer copying when Iris detects capability
2. **Optimal Buffer Sizing**: Automatic buffer size recommendation for Iris tuning
3. **Hot Reload Support**: Dynamic configuration updates without restart
4. **Thread-Safe Access**: Perfect synchronization for concurrent Iris loggers

## Performance Benchmarks

### QuickStart API Performance
```
Operation Count: 5,000
Duration: 2.005157ms
Throughput: 2,493,570 ops/sec
Configuration: Zero (automatic optimization)
```

### Custom Magic API Performance
```
Operation Count: 10,000
Duration: 6.399778ms  
Throughput: 1,562,554 ops/sec
Features: Advanced configuration + Magic optimization
```

### Magic Auto-Detection Performance
```
Operation Count: 3,000
Duration: 1.069416ms
Throughput: 2,805,269 ops/sec
Optimization: Automatic WriteOwned() + buffer tuning
```

## Configuration Patterns

### Development Setup
```go
// Quick development logging
writer := lethe.QuickStart("dev.log")
defer writer.Close()

logger, err := iris.New(iris.Config{
    Output:  writer,
    Encoder: iris.NewJSONEncoder(),
    Level:   iris.Debug,
}, iris.WithCaller(), iris.Development())
```

### Production Setup
```go
// Production with monitoring
writer := lethe.NewIrisWriter("production.log", &lethe.Logger{
    MaxSizeStr:         "100MB",
    MaxAgeStr:          "7d",
    MaxBackups:         30,
    Compress:           true,
    Checksum:           true,
    Async:              true,
    BufferSize:         32768,
    BackpressurePolicy: "adaptive",
    LocalTime:          true,
    ErrorCallback: func(eventType string, err error) {
        monitoring.RecordEvent(eventType, err)
    },
})
defer writer.Close()

logger, err := iris.New(iris.Config{
    Output:    writer,
    Encoder:   iris.NewJSONEncoder(),
    Level:     iris.Info,
    Capacity:  16384,
    BatchSize: 64,
})
```

### High-Performance Setup
```go
// Maximum performance configuration
writer := lethe.NewIrisWriter("high-perf.log", &lethe.Logger{
    MaxSizeStr:         "200MB",
    MaxBackups:         10,
    Async:              true,
    BufferSize:         65536, // 64KB
    BackpressurePolicy: "adaptive",
    Compress:           true,
})
defer writer.Close()

logger, err := iris.New(iris.Config{
    Output:    writer,
    Encoder:   iris.NewJSONEncoder(),
    Level:     iris.Info,
    Capacity:  32768, // Match buffer sizing
    BatchSize: 256,
})
```

## Integration Patterns

### Shared Writer Pattern
```go
// One writer, multiple loggers (recommended)
writer := lethe.QuickStart("shared.log")
defer writer.Close()

// Application logger
appLogger, _ := iris.New(iris.Config{Output: writer})

// Request logger
requestLogger := appLogger.With(
    iris.String("component", "http"),
    iris.String("version", "1.0.0"),
)

// Database logger  
dbLogger := appLogger.With(
    iris.String("component", "database"),
    iris.String("pool", "primary"),
)
```

### Microservice Pattern
```go
// Service-specific configuration
writer := lethe.NewIrisWriter("user-service.log", &lethe.Logger{
    MaxSizeStr: "50MB",
    MaxBackups: 10,
    Compress:   true,
})

serviceLogger, _ := iris.New(iris.Config{Output: writer})

// Per-request context
requestLogger := serviceLogger.With(
    iris.String("service", "user-service"),
    iris.String("instance", instanceID),
    iris.String("request_id", requestID),
)
```

### Concurrent Access Pattern
```go
// Shared Magic writer for concurrent access
writer := lethe.NewIrisWriter("concurrent.log", &lethe.Logger{
    BufferSize:         16384,
    BackpressurePolicy: "adaptive",
    Async:              true,
})

// Multiple goroutines can safely share the writer
for i := 0; i < numWorkers; i++ {
    go func(workerID int) {
        logger, _ := iris.New(iris.Config{Output: writer})
        defer logger.Close()
        logger.Start()
        
        logger.With(iris.Int("worker", workerID)).Info("Worker started")
        // ... worker operations
    }(i)
}
```

## Interface Specification

### IrisIntegration Type

The Magic API returns an `*IrisIntegration` that implements:

```go
// Standard io.Writer interface
Write(data []byte) (int, error)

// WriteSyncer interface (for Iris compatibility)
Sync() error
Close() error

// Enhanced capabilities (automatically detected by Iris)
WriteOwned(data []byte) (int, error)    // Zero-copy optimization
GetOptimalBufferSize() int              // Buffer size recommendation
SupportsHotReload() bool                // Hot reload capability
GetLogger() *Logger                     // Access to underlying Lethe logger
```

### Iris Detection Logic

Iris automatically detects and utilizes these enhanced methods:

1. **WriteOwned Detection**: Iris checks for `WriteOwned([]byte) (int, error)` method
2. **Buffer Optimization**: Iris queries `GetOptimalBufferSize()` for tuning
3. **Hot Reload**: Iris checks `SupportsHotReload()` for configuration updates
4. **Graceful Fallback**: Falls back to standard `Write()` if optimizations unavailable

## Error Handling

### Error Callback Integration
```go
writer := lethe.NewIrisWriter("app.log", &lethe.Logger{
    ErrorCallback: func(eventType string, err error) {
        switch eventType {
        case "rotation":
            log.Printf("Log rotation event: %v", err)
            metrics.IncCounter("log.rotation")
        case "write_error":
            log.Printf("Write error: %v", err)
            metrics.IncCounter("log.write_error")
        case "compression":
            log.Printf("Compression event: %v", err)
        default:
            log.Printf("Unknown event %s: %v", eventType, err)
        }
    },
})
```

### Graceful Shutdown
```go
// Proper shutdown sequence
logger.Info("Application shutting down")

// Close Iris logger first
if err := logger.Close(); err != nil {
    log.Printf("Error closing Iris logger: %v", err)
}

// Close Magic writer last
if err := writer.Close(); err != nil {
    log.Printf("Error closing Lethe writer: %v", err)
}
```

## Troubleshooting

### Performance Issues

**Problem**: Lower than expected throughput
**Solutions**:
1. Use `QuickStart()` for automatic optimization
2. Enable async mode: `Async: true`
3. Increase buffer size: `BufferSize: 32768`
4. Use adaptive backpressure: `BackpressurePolicy: "adaptive"`

**Problem**: High memory usage
**Solutions**:
1. Reduce buffer size: `BufferSize: 8192`
2. Enable compression: `Compress: true`
3. Use smaller batch sizes in Iris configuration

### Integration Issues

**Problem**: WriteOwned optimization not working
**Solutions**:
1. Verify using Magic API (`QuickStart` or `NewIrisWriter`)
2. Check Iris is detecting capabilities automatically
3. Ensure latest version of both libraries

**Problem**: Hot reload not working
**Solutions**:
1. Verify `SupportsHotReload()` returns true
2. Check configuration file permissions
3. Ensure proper file watching setup

### Configuration Problems

**Problem**: Logs not rotating
**Solutions**:
1. Check `MaxSizeStr` format (e.g., "100MB")
2. Verify write permissions in log directory
3. Check `MaxBackups` > 0

**Problem**: Compression not working
**Solutions**:
1. Ensure `Compress: true` in configuration
2. Check disk space availability
3. Verify gzip support

## Best Practices

### Recommended Setup
1. **Start with QuickStart**: Use `lethe.QuickStart()` for most applications
2. **Custom Config When Needed**: Use `lethe.NewIrisWriter()` for specific requirements
3. **Trust Auto Detection**: Let automatic optimization handle performance
4. **Share Writers**: One Magic writer per log file, multiple Iris loggers
5. **Monitor Events**: Use ErrorCallback for production monitoring

### Performance Optimization
1. **Buffer Sizing**: Start with defaults, tune based on load testing
2. **Async Mode**: Always enable for production (`Async: true`)
3. **Adaptive Backpressure**: Use for varying load patterns
4. **Compression**: Enable for storage efficiency (`Compress: true`)

### Production Deployment
1. **Monitoring**: Implement comprehensive ErrorCallback handling
2. **Graceful Shutdown**: Proper close sequence for data integrity
3. **Hot Reload**: Enable for zero-downtime configuration updates
4. **Data Integrity**: Use checksums for critical applications (`Checksum: true`)

### New Pattern (Magic API)
```go
// New: Zero configuration required
writer := lethe.QuickStart("app.log")
logger, _ := iris.New(iris.Config{Output: writer})
```

**Migration benefits**:
- ~50 lines of adapter code eliminated
- Automatic optimization detection
- Built-in hot reload support
- Better performance through Magic integration

## Examples and Demos

Complete examples are available in:
- **Basic Example**: `/examples/iris-integration/`
- **Advanced Examples**: Various configuration patterns
- **Production Example**: Complete production setup
- **Performance Benchmarks**: Throughput and latency testing

Run examples:
```bash
cd examples/iris-integration/
go run .
```

## Integration Testing

### Verification Tests
```go
func TestMagicAPIIntegration(t *testing.T) {
    writer := lethe.QuickStart("test.log")
    defer writer.Close()
    
    // Verify interface compliance
    var _ io.Writer = writer
    
    // Check Magic capabilities
    if provider, ok := writer.(interface{ GetOptimalBufferSize() int }); ok {
        size := provider.GetOptimalBufferSize()
        assert.Greater(t, size, 0)
    }
    
    // Test with Iris
    logger, err := iris.New(iris.Config{Output: writer})
    assert.NoError(t, err)
    defer logger.Close()
    logger.Start()
    
    logger.Info("Test message")
}
```

## Version Compatibility

- **Lethe**: v1.0.0+
- **Iris**: v1.0.0+
- **Go**: 1.19+

The Magic API maintains backward compatibility while providing enhanced features for supported versions.

## Summary

The Magic API represents a revolutionary approach to logging integration:

1. **Zero Configuration**: Production-ready logging in one line of code
2. **Magic Integration**: Automatic runtime optimization without setup
3. **Maximum Performance**: Zero-copy operations and optimal buffer sizing
4. **Production Ready**: Built-in monitoring, hot reload, and error handling
5. **Developer Friendly**: Simple API with powerful automatic optimization

Choose Magic API for:
- ✅ New applications requiring logging with rotation
- ✅ High-performance applications needing automatic optimization  
- ✅ Production services requiring zero-configuration setup
- ✅ Applications needing hot reload capabilities
- ✅ Any integration between Iris and Lethe
