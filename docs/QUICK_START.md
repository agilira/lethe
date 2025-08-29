# Lethe Quick Start Guide

Get up and running with Lethe in minutes. This guide covers the most common use cases and integration patterns.

## Installation

```bash
go get github.com/agilira/lethe
```

## Basic Usage

### 1. Simple Logger (Recommended for Most Cases)

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    // Create a logger with sensible defaults
    logger, err := lethe.NewWithDefaults("app.log")
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    // Use as io.Writer
    logger.Write([]byte("Hello, Lethe!\n"))
}
```

### 2. Custom Configuration

```go
// String-based configuration (human-readable)
logger, err := lethe.NewSimple("app.log", "100MB", 5)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### 3. Time-Based Rotation

```go
// Daily rotation
dailyLogger, err := lethe.NewDaily("daily.log")
if err != nil {
    log.Fatal(err)
}
defer dailyLogger.Close()

// Weekly rotation
weeklyLogger, err := lethe.NewWeekly("weekly.log")
if err != nil {
    log.Fatal(err)
}
defer weeklyLogger.Close()
```

## Integration Patterns

### Standard Library Integration

Replace your existing log output with Lethe:

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    // Create Lethe logger
    rotator, err := lethe.NewWithDefaults("app.log")
    if err != nil {
        log.Fatal(err)
    }
    defer rotator.Close()

    // Replace standard library output
    log.SetOutput(rotator)

    // Use standard logging as usual
    log.Println("This goes to Lethe with rotation!")
    log.Printf("User %s logged in", "john_doe")
}
```

### High-Performance Async Logging

For high-throughput applications:

```go
package main

import (
    "fmt"
    "time"
    "github.com/agilira/lethe"
)

func main() {
    // High-performance configuration
    logger := &lethe.Logger{
        Filename:           "high_perf.log",
        MaxSizeStr:         "50MB",
        MaxBackups:         10,
        Compress:           true,
        Async:              true,                   // Enable MPSC mode
        BufferSize:         4096,                   // Large buffer
        BackpressurePolicy: "adaptive",             // Adaptive buffer resizing
        FlushInterval:      500 * time.Microsecond, // Fast flush
    }
    defer logger.Close()

    // High-throughput logging
    start := time.Now()
    for i := 0; i < 10000; i++ {
        entry := fmt.Sprintf("Log entry %d\n", i)
        logger.Write([]byte(entry))
    }
    
    // Allow async processing to complete
    time.Sleep(100 * time.Millisecond)
    
    duration := time.Since(start)
    fmt.Printf("Logged 10,000 entries in %v\n", duration)
}
```

### Development Logger

For development and debugging:

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    // Development-optimized logger
    logger, err := lethe.NewDevelopment("debug.log")
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    // Features:
    // - 10MB size limit (smaller files for debugging)
    // - 1 hour rotation (frequent rotation)
    // - No compression (easier to read)
    // - Synchronous writes (immediate feedback)
    
    logger.Write([]byte("Debug message\n"))
}
```

## Integration

### With Iris (Native)

Lethe was originally designed for Iris, the ultra-high performance logging library:

```go
package main

import (
    "github.com/agilira/lethe"
    "github.com/agilira/iris"
)

func main() {
    // Create Lethe rotator optimized for Iris
    rotator, err := lethe.NewWithDefaults("iris.log")
    if err != nil {
        panic(err)
    }
    defer rotator.Close()

    // Create Iris logger with Lethe as output
    logger, err := iris.New(iris.Config{
        Level:    iris.Info,
        Output:   rotator,
        Encoder:  iris.NewJSONEncoder(),
        Capacity: 8192,
    })
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    logger.Start()

    // Ultra-high performance structured logging with rotation
    logger.Info("Iris with Lethe rotation")
    logger.Info("User action", iris.Str("user", "john"), iris.Int("id", 123))
}
```

### With Logrus

```go
package main

import (
    "github.com/sirupsen/logrus"
    "github.com/agilira/lethe"
)

func main() {
    // Create Lethe rotator
    rotator, err := lethe.NewWithDefaults("logrus.log")
    if err != nil {
        panic(err)
    }
    defer rotator.Close()

    // Setup Logrus with Lethe
    logrus.SetOutput(rotator)
    logrus.SetFormatter(&logrus.JSONFormatter{})

    // Use Logrus as usual
    logrus.Info("Logrus with Lethe rotation")
    logrus.WithField("user", "john").Info("User action")
}
```

### With Zap

```go
package main

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "github.com/agilira/lethe"
)

func main() {
    // Create Lethe rotator
    rotator, err := lethe.NewWithDefaults("zap.log")
    if err != nil {
        panic(err)
    }
    defer rotator.Close()

    // Create Zap logger with Lethe
    core := zapcore.NewCore(
        zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
        zapcore.AddSync(rotator),
        zapcore.InfoLevel,
    )
    logger := zap.New(core)
    defer logger.Sync()

    // Use Zap as usual
    logger.Info("Zap with Lethe rotation")
    logger.Info("User action", zap.String("user", "john"))
}
```

### With Zerolog

```go
package main

import (
    "github.com/rs/zerolog"
    "github.com/agilira/lethe"
)

func main() {
    // Create Lethe rotator
    rotator, err := lethe.NewWithDefaults("zerolog.log")
    if err != nil {
        panic(err)
    }
    defer rotator.Close()

    // Setup Zerolog with Lethe
    logger := zerolog.New(rotator).With().Timestamp().Logger()

    // Use Zerolog as usual
    logger.Info().Msg("Zerolog with Lethe rotation")
    logger.Info().Str("user", "john").Msg("User action")
}
```

## Advanced Features

### Zero-Copy Integration with Iris

Lethe was specifically designed for Iris, our super fast logging library. This integration provides maximum performance:

```go
package main

import (
    "github.com/agilira/lethe"
    "github.com/agilira/iris"
)

func main() {
    // Create Lethe rotator for Iris
    rotator, err := lethe.NewWithDefaults("iris.log")
    if err != nil {
        panic(err)
    }
    defer rotator.Close()

    // Create Iris logger with Lethe as output
    logger, err := iris.New(iris.Config{
        Level:  iris.Info,
        Output: rotator,
    })
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    logger.Start()

    // Zero-allocation structured logging with automatic rotation
    logger.Info("User authenticated",
        iris.Str("username", "john_doe"),
        iris.Int("user_id", 12345),
    )
}
```

### Custom Configuration

For full control over all settings:

```go
package main

import (
    "time"
    "github.com/agilira/lethe"
)

func main() {
    config := &lethe.LoggerConfig{
        Filename:           "custom.log",
        MaxSizeStr:         "500MB",
        MaxAgeStr:          "30d",
        MaxBackups:         20,
        Compress:           true,
        Checksum:           true, // Enable SHA-256 checksums
        Async:              true,
        BackpressurePolicy: "adaptive",
        LocalTime:          true,
        ErrorCallback: func(op string, err error) {
            // Custom error handling
            log.Printf("Lethe error [%s]: %v", op, err)
        },
    }
    
    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        panic(err)
    }
    defer logger.Close()
}
```

### Performance Monitoring

Monitor your logger's performance:

```go
package main

import (
    "fmt"
    "github.com/agilira/lethe"
)

func main() {
    logger, err := lethe.NewWithDefaults("monitored.log")
    if err != nil {
        panic(err)
    }
    defer logger.Close()

    // Log some data
    for i := 0; i < 1000; i++ {
        logger.Write([]byte(fmt.Sprintf("Entry %d\n", i)))
    }

    // Get performance statistics
    stats := logger.Stats()
    fmt.Printf("Write Count: %d\n", stats.WriteCount)
    fmt.Printf("Total Bytes: %d\n", stats.TotalBytes)
    fmt.Printf("Average Latency: %d ns\n", stats.AvgLatencyNs)
    fmt.Printf("Buffer Fill: %d\n", stats.BufferFill)
}
```

## Constructor Reference

| Constructor | Use Case | Description |
|-------------|----------|-------------|
| `NewWithDefaults()` | **Most applications** | Production-ready defaults (100MB, 7d rotation, 10 backups, compression) |
| `NewSimple()` | **Custom size** | String-based size configuration ("100MB", "1GB") |
| `NewDaily()` | **Daily rotation** | 24-hour rotation with compression |
| `NewWeekly()` | **Weekly rotation** | 7-day rotation with compression |
| `NewDevelopment()` | **Development** | Small files, frequent rotation, no compression |
| `NewWithConfig()` | **Full control** | Complete configuration control |

## Configuration Options

### Size Formats
- `"10MB"`, `"1GB"`, `"500KB"` - Case insensitive
- `"10m"`, `"1g"`, `"500k"` - Short format

### Duration Formats
- `"7d"`, `"24h"`, `"30m"` - Standard Go duration
- `"7 days"`, `"24 hours"`, `"30 minutes"` - Human readable

### Backpressure Policies
- `"block"` - Block until space available (default)
- `"drop"` - Drop messages when buffer full
- `"adaptive"` - Dynamically resize buffer

## Best Practices

### 1. Always Close Your Logger
```go
logger, err := lethe.NewWithDefaults("app.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close() // Essential for proper cleanup
```

### 2. Choose the Right Constructor
- **Production**: `NewWithDefaults()`
- **Development**: `NewDevelopment()`
- **Time-based**: `NewDaily()` or `NewWeekly()`
- **Custom needs**: `NewWithConfig()`

### 3. Handle Errors
```go
logger, err := lethe.NewWithDefaults("app.log")
if err != nil {
    log.Fatal("Failed to create logger:", err)
}
```

### 4. Use Async for High Throughput
```go
logger := &lethe.Logger{
    Filename: "app.log",
    Async:    true, // Enable for high-throughput applications
}
```

### 5. Monitor Performance
```go
stats := logger.Stats()
if stats.BufferFill > 80 {
    // Consider increasing buffer size or using adaptive policy
}
```

## Common Patterns

### Application Logger
```go
var AppLogger *lethe.Logger

func init() {
    var err error
    AppLogger, err = lethe.NewWithDefaults("app.log")
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    defer AppLogger.Close()
    // Your application code
}
```

### Service Logger
```go
type Service struct {
    logger *lethe.Logger
}

func NewService() (*Service, error) {
    logger, err := lethe.NewWithDefaults("service.log")
    if err != nil {
        return nil, err
    }
    
    return &Service{logger: logger}, nil
}

func (s *Service) Close() error {
    return s.logger.Close()
}
```

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure write permissions to log directory
2. **File Locked**: Make sure to call `Close()` on your logger
3. **High Memory Usage**: Reduce `BufferSize` or disable `Async`
4. **Slow Performance**: Enable `Async` and increase `BufferSize`

### Getting Help

- Check the [API Documentation](API.md) for detailed function reference
- Review the [Architecture Guide](ARCHITECTURE.md) for technical details
- Run the examples in the `examples/` directory
- Check the test files for usage patterns

## Next Steps

- Explore the [API Documentation](API.md) for advanced features
- Read the [Architecture Guide](ARCHITECTURE.md) to understand the internals
- Check out the examples in the `examples/` directory
- Run benchmarks to measure performance in your environment

---

Lethe • an AGILira fragment