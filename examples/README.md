# Lethe Examples

This directory contains comprehensive examples demonstrating Lethe's log rotation capabilities, from basic usage patterns to advanced production deployments and zero-copy integrations.

## Example Categories

###  [Basic Examples](./basic/)
**Start here for fundamental Lethe usage**
- All constructor functions (NewWithDefaults, NewSimple, NewDaily, etc.)
- Basic configuration options and rotation policies
- Standard library integration patterns
- **Recommended for**: New users, getting started, understanding core concepts

###  [Low-Level API Examples](./low-level-api/)
**Direct Lethe API for maximum raw performance**
- MPSC buffer operations and zero-copy writes
- Direct WriteOwned calls without framework overhead
- Raw buffer management and optimization
- **Recommended for**: Performance-critical applications requiring maximum speed

###  [Iris Integration](./iris-integration/)
**Production-ready integration with Iris**
- Zero-copy adapter with WriteSyncer interface
- Iris + Lethe integration with professional features
- Structured logging, levels, and JSON formatting
- **Recommended for**: Iris users, production applications, professional logging

## Quick Start Guide

### For New Users
```bash
# Start with basic examples to understand fundamentals
cd basic/
go run .

# Explore constructor functions and basic rotation
ls logs/  # See generated log files
```

### For High-Performance Raw Operations
```bash
# Test direct Lethe API without framework overhead
cd low-level-api/
go run .

# Review raw buffer operations and WriteOwned performance
# These are NOT comparable to framework integration!
```

### For Production Iris Integration
```bash
# Experience Iris-Lethe integration with full logging features
cd iris-integration/
go run .

# Observe professional logging with structured data, levels, JSON
# This includes framework overhead but provides full logging capabilities
```

## Performance Overview

| Example Category | Typical Throughput | Key Features |
|------------------|-------------------|--------------|
| **Basic** | 500K+ ops/sec | Constructor functions, basic rotation |
| **Low-Level API** | 4-6M ops/sec | Direct Lethe calls, raw buffers, zero allocations |
| **Iris Integration** | 3-4M ops/sec | Full logging framework, JSON, levels, professional features |

## Example Roadmap

### Learning Path
1. **Start**: `basic/` - Master constructor functions and rotation policies
2. **Advance**: `advanced/` - Explore high-performance features and optimization
3. **Integrate**: `iris-integration/` - Implement production-ready zero-copy logging

### Use Case Selection
- **Simple Applications**: Use `basic/NewWithDefaults()` patterns
- **High-Throughput Services**: Implement `advanced/` MPSC and backpressure strategies  
- **Iris Applications**: Deploy `iris-integration/` zero-copy adapters
- **professional Systems**: Combine advanced features with monitoring and error callbacks

## Configuration Quick Reference

### Production Defaults (Recommended)
```go
logger, err := lethe.NewWithDefaults("app.log")
// 100MB size, 7d rotation, 10 backups, compressed, async
```

### High-Performance Configuration
```go
config := &lethe.LoggerConfig{
    Filename:           "app.log",
    MaxSizeStr:         "200MB",
    MaxBackups:         20,
    Async:              true,
    BufferSize:         32768,  // 32KB buffer
    BackpressurePolicy: "adaptive",
    Compress:           true,
}
```

### Development Configuration
```go
logger, err := lethe.NewDevelopment("debug.log")
// 10MB size, 1h rotation, 5 backups, uncompressed, sync
```

## Integration Examples

### Standard Library
```go
logger, err := lethe.NewWithDefaults("app.log")
log.SetOutput(logger)  // Replace default output
```

### With Iris (Zero-Copy)
```go
letheLogger, _ := lethe.NewWithDefaults("iris.log")
adapter := NewLetheIrisAdapter(letheLogger)
irisLogger, _ := iris.New(iris.Config{Output: adapter})
```

### Custom Applications
```go
type MyLogger struct {
    lethe *lethe.Logger
}

func (m *MyLogger) Log(message string) {
    m.lethe.Write([]byte(message + "\n"))
}
```

## Best Practices Summary

### Constructor Selection
- **Production**: `NewWithDefaults()` - Battle-tested defaults
- **Custom Size**: `NewSimple()` - String-based configuration  
- **Time-Based**: `NewDaily()` or `NewWeekly()` - Predictable rotation
- **Full Control**: `NewWithConfig()` - Advanced features

### Performance Optimization
- **Enable Async**: Set `Async: true` for high-throughput scenarios
- **Size Buffers**: Use 16KB-32KB buffers for optimal performance
- **Choose Policy**: `adaptive` for most cases, `drop` for memory-constrained
- **Monitor Events**: Implement ErrorCallback for production monitoring

### Error Handling
```go
config.ErrorCallback = func(eventType string, err error) {
    switch eventType {
    case "rotation":
        log.Printf("Log rotated: %v", err)
    case "compression":
        log.Printf("Compression event: %v", err)
    case "checksum":
        log.Printf("Integrity check: %v", err)
    }
}
```

## File Organization

```
examples/
├── README.md                    # This overview file
├── basic/                       # Fundamental usage patterns
│   ├── main.go                 # All constructor examples
│   ├── README.md               # Detailed basic documentation
│   └── logs/                   # Generated log files
├── advanced/                    # High-performance features
│   ├── main.go                 # MPSC, zero-copy, backpressure
│   ├── README.md               # Advanced feature documentation
│   └── logs/                   # Performance test outputs
└── iris-integration/            # Iris logging integration
    ├── main.go                 # Zero-copy adapter examples
    ├── README.md               # Integration documentation
    └── logs/                   # Integration test outputs
```

## Next Steps

### For Library Authors
Explore the adapter patterns in `iris-integration/` to integrate your logging framework with Lethe's rotation capabilities.

### For Application Developers
1. Start with `basic/` examples to understand core concepts
2. Review `advanced/` for performance optimization techniques
3. Implement production patterns based on your requirements

### For Performance Engineers
Focus on `advanced/` examples for:
- MPSC buffer tuning and scaling
- Backpressure policy selection and testing
- Zero-copy optimization techniques
- Concurrent access pattern validation

## Contributing

When adding new examples:
1. Follow the existing directory structure
2. Include comprehensive README.md documentation
3. Provide both basic and advanced usage patterns
4. Add performance benchmarks where relevant
5. Include error handling and monitoring examples

## Support

- **Documentation**: Each subdirectory contains detailed README files
- **Performance**: All examples include throughput measurements
- **Integration**: See `iris-integration/` for framework integration patterns
- **Best Practices**: Comprehensive configuration guides in each category
