# Basic Examples

This directory demonstrates the fundamental usage patterns of Lethe log rotation library. These examples cover all available constructor functions and basic configuration options.

## Quick Start

```bash
cd basic/
go run .
```

The examples will create log files in the `./logs/` directory with different rotation configurations.

## Constructor Functions Covered

### 1. `NewWithDefaults(filename)` - **Recommended for Production**
- **Purpose**: Production-ready configuration with sensible defaults
- **Configuration**: 100MB size limit, 7-day rotation, 10 backups, compressed
- **Features**: Async mode, adaptive backpressure, optimal performance
- **Use Case**: General production applications requiring reliable log rotation

### 2. `NewSimple(filename, maxSize, maxBackups)` - **Modern String-Based**
- **Purpose**: Easy configuration with string-based size limits
- **Configuration**: Custom size (e.g., "50MB"), custom backup count
- **Features**: Performance optimizations enabled by default
- **Use Case**: Applications needing custom size limits with minimal configuration

### 3. `New(filename, maxSizeMB, maxBackups)` - **Legacy Compatibility**
- **Purpose**: Backward compatibility with older APIs
- **Configuration**: Integer-based size limits (in MB)
- **Features**: Basic functionality for migration scenarios
- **Use Case**: Migrating from other rotation libraries or legacy systems

### 4. `NewDaily(filename)` - **Time-Based Daily Rotation**
- **Purpose**: Rotate logs every 24 hours regardless of size
- **Configuration**: Daily rotation, 7 backups, 50MB size limit
- **Features**: Time-based rotation ideal for operational logs
- **Use Case**: Applications requiring daily log separation (audit logs, reports)

### 5. `NewWeekly(filename)` - **Time-Based Weekly Rotation**
- **Purpose**: Rotate logs every 7 days for longer retention
- **Configuration**: Weekly rotation, 4 backups, 200MB size limit
- **Features**: Optimized for lower-frequency, larger log files
- **Use Case**: Summary logs, weekly reports, low-volume applications

### 6. `NewDevelopment(filename)` - **Development Optimized**
- **Purpose**: Fast feedback and easy debugging during development
- **Configuration**: 10MB size, hourly rotation, no compression, sync writes
- **Features**: Immediate file visibility, frequent rotation for testing
- **Use Case**: Local development, debugging, testing rotation behavior

### 7. `NewWithConfig(config)` - **Full Configuration Control**
- **Purpose**: Complete control over all configuration options
- **Configuration**: Custom LoggerConfig with all available settings
- **Features**: Advanced options like checksums, custom error callbacks
- **Use Case**: Applications requiring fine-grained control over rotation behavior

## Configuration Options Explained

### Size-Based Rotation
- **MaxSizeStr**: Human-readable size limits ("100MB", "1GB", "500KB")
- **MaxSize** (legacy): Integer size limits in MB

### Time-Based Rotation
- **MaxAgeStr**: Time-based rotation ("24h", "7d", "30d")
- Automatic rotation regardless of file size

### Backup Management
- **MaxBackups**: Number of rotated files to retain
- Older files are automatically deleted

### Performance Options
- **Async**: Enable asynchronous writes for better performance
- **BufferSize**: Internal buffer size for write batching
- **BackpressurePolicy**: Overflow handling strategy ("adaptive", "drop", "block")

### Data Integrity
- **Compress**: Gzip compression for rotated files
- **Checksum**: Data integrity verification
- **LocalTime**: Use local timezone for rotation timestamps

## File Output

After running the examples, you'll find these log files in `./logs/`:

```
logs/
├── production.log     # NewWithDefaults example
├── simple.log         # NewSimple example  
├── legacy.log         # New (legacy) example
├── daily.log          # NewDaily example
├── weekly.log         # NewWeekly example
├── debug.log          # NewDevelopment example
└── custom.log         # NewWithConfig example
```

## Next Steps

1. **Advanced Examples**: Check `../advanced/` for MPSC buffers, zero-copy operations
2. **Iris Integration**: See `../iris-integration/` for high-performance logging integration
3. **Production Setup**: Use `../production/` for deployment-ready configurations

## Common Patterns

### Production Application
```go
logger, err := lethe.NewWithDefaults("app.log")
```

### Custom Size Limit
```go
logger, err := lethe.NewSimple("app.log", "200MB", 20)
```

### Development/Testing
```go
logger, err := lethe.NewDevelopment("debug.log")
```

### Full Control
```go
config := &lethe.LoggerConfig{
    Filename:           "app.log",
    MaxSizeStr:         "100MB",
    MaxAgeStr:          "7d",
    MaxBackups:         10,
    Compress:           true,
    Async:              true,
    BackpressurePolicy: "adaptive",
}
logger, err := lethe.NewWithConfig(config)
```
