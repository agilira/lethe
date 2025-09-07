# Hot Reload Configuration Management

Lethe provides dynamic configuration hot reload functionality using Argus. This enables runtime configuration changes without requiring application restarts.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration Format](#configuration-format)
- [API Reference](#api-reference)
- [Supported Parameters](#supported-parameters)
- [Audit Trail](#audit-trail)
- [Error Handling](#error-handling)
- [Performance](#performance)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

Hot reload allows Lethe loggers to automatically detect and apply configuration changes from JSON files without interrupting logging operations. The system monitors configuration files using efficient file system events and applies changes atomically.

### Key Features

- **Zero-downtime updates**: Configuration changes apply without stopping the logger
- **Atomic operations**: Configuration updates are applied as complete transactions
- **Audit trail**: All configuration changes are logged for compliance and debugging
- **Error resilience**: Invalid configurations are rejected while maintaining current settings
- **Performance optimized**: Minimal overhead using efficient file system monitoring

### Architecture

The hot reload system consists of three main components:

1. **DynamicConfigWatcher**: Monitors configuration files and coordinates updates
2. **Argus Integration**: Provides efficient file system monitoring with cross-platform support
3. **Configuration Applier**: Safely applies runtime-changeable settings to active loggers

## Quick Start

### Basic Setup

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    // Create logger
    logger, err := lethe.NewWithDefaults("app.log")
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    // Enable hot reload
    watcher, err := lethe.EnableDynamicConfig(logger, "config.json")
    if err != nil {
        log.Printf("Hot reload disabled: %v", err)
    } else {
        defer watcher.Stop()
        log.Println("Hot reload enabled")
    }

    // Your application logic here
    logger.Write([]byte("Application started\n"))
    
    // Configuration changes in config.json will now apply automatically
}
```

### Configuration File

Create a `config.json` file:

```json
{
  "max_size_str": "100MB",
  "max_age_str": "7d",
  "max_backups": 10,
  "compress": true,
  "local_time": true,
  "backpressure_policy": "adaptive"
}
```

## Configuration Format

Configuration files must be valid JSON. The following format is supported:

```json
{
  "max_size_str": "string",
  "max_age_str": "string", 
  "max_backups": 0,
  "compress": false,
  "checksum": false,
  "local_time": false,
  "backpressure_policy": "string",
  "adaptive_flush": false
}
```

### Size Format

The `max_size_str` field accepts human-readable size specifications:

- **Bytes**: `"1024"`, `"2048"`
- **Kilobytes**: `"10KB"`, `"10K"`
- **Megabytes**: `"100MB"`, `"100M"`
- **Gigabytes**: `"1GB"`, `"1G"`
- **Terabytes**: `"1TB"`, `"1T"`

### Time Format

The `max_age_str` field accepts duration specifications:

- **Minutes**: `"30m"`
- **Hours**: `"24h"`
- **Days**: `"7d"`
- **Weeks**: `"2w"`
- **Standard Go durations**: `"1h30m"`, `"45m30s"`

## API Reference

### DynamicConfigWatcher

The main interface for hot reload functionality.

#### Constructor

```go
func NewDynamicConfigWatcher(configPath string, logger *Logger) (*DynamicConfigWatcher, error)
```

Creates a new configuration watcher for the specified logger and config file.

**Parameters:**
- `configPath`: Path to the JSON configuration file
- `logger`: Target logger instance to update

**Returns:**
- `*DynamicConfigWatcher`: Watcher instance
- `error`: Configuration or setup errors

#### Methods

```go
// Start begins monitoring the configuration file
func (w *DynamicConfigWatcher) Start() error

// Stop stops monitoring the configuration file  
func (w *DynamicConfigWatcher) Stop() error

// IsRunning returns true if the watcher is active
func (w *DynamicConfigWatcher) IsRunning() bool

// GetLastConfig returns the last successfully applied configuration
func (w *DynamicConfigWatcher) GetLastConfig() *LoggerConfig
```

### Convenience Functions

#### EnableDynamicConfig

```go
func EnableDynamicConfig(logger *Logger, configPath string) (*DynamicConfigWatcher, error)
```

Combines watcher creation and startup in a single operation.

#### CreateSampleConfig

```go
func CreateSampleConfig(filename string) error
```

Generates a sample configuration file with common settings.

## Supported Parameters

The following configuration parameters can be changed at runtime:

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `max_size_str` | string | File size rotation limit | `"100MB"` |
| `max_age_str` | string | Time-based rotation interval | `"7d"` |
| `max_backups` | integer | Number of backup files to retain | `10` |
| `compress` | boolean | Enable gzip compression of rotated files | `true` |
| `checksum` | boolean | Enable SHA-256 checksums for integrity | `false` |
| `local_time` | boolean | Use local time in backup filenames | `true` |
| `backpressure_policy` | string | Buffer overflow handling policy | `"adaptive"` |
| `adaptive_flush` | boolean | Enable adaptive flush timing | `true` |

### Backpressure Policies

- **`"fallback"`**: Switch to synchronous mode when buffer is full
- **`"drop"`**: Discard messages when buffer is full  
- **`"adaptive"`**: Dynamically resize buffers under pressure

### Non-Runtime Parameters

The following parameters require logger restart and cannot be changed via hot reload:

- `filename`: Log file path
- `async`: Asynchronous mode setting
- `buffer_size`: MPSC buffer size
- `file_mode`: File permissions

## Audit Trail

All configuration changes are automatically logged to `lethe-config-audit.jsonl` in JSON Lines format.

### Audit Record Format

```json
{
  "timestamp": "2025-09-07T16:48:21.830+02:00",
  "level": 0,
  "event": "file_changed",
  "component": "lethe",
  "file_path": "/app/config.json",
  "process_id": 1234,
  "process_name": "myapp",
  "checksum": "45046fe016ec04459b742a63cfe4b9f10ca0274a18f5d100f2aab2af0738352f"
}
```

### Audit Configuration

Audit settings are configured automatically with production-ready defaults:

- **Output file**: `lethe-config-audit.jsonl`
- **Buffer size**: 1000 entries
- **Flush interval**: 5 seconds
- **Minimum level**: Info (captures all changes)

## Error Handling

The hot reload system implements robust error handling:

### Graceful Degradation

- **Invalid JSON**: Syntax errors are logged, current configuration maintained
- **Invalid values**: Type or range errors are logged, invalid fields ignored
- **File access errors**: Permission or I/O errors logged, monitoring continues
- **Parse errors**: Malformed configurations rejected, previous settings preserved

### Error Reporting

Errors are reported through the logger's `ErrorCallback` mechanism:

```go
logger.ErrorCallback = func(operation string, err error) {
    log.Printf("Logger error (%s): %v", operation, err)
}
```

### Error Categories

- **`config_watcher`**: File monitoring errors
- **`config_reload`**: Configuration loading errors  
- **`config_apply`**: Configuration application errors

## Performance

### Monitoring Overhead

- **Polling interval**: 2 seconds (configurable via Argus)
- **CPU overhead**: Minimal, event-driven file monitoring
- **Memory footprint**: <1KB per watched file
- **I/O operations**: Optimized with file modification time checks

### Configuration Application

- **Atomic updates**: All settings applied as a single transaction
- **Non-blocking**: Configuration updates don't block logging operations
- **Validation**: Settings validated before application to prevent corruption

## Examples

### Web Service Integration

```go
func setupLogging() (*lethe.Logger, *lethe.DynamicConfigWatcher, error) {
    logger, err := lethe.NewWithDefaults("app.log")
    if err != nil {
        return nil, nil, err
    }

    watcher, err := lethe.EnableDynamicConfig(logger, "/etc/app/config.json")
    if err != nil {
        log.Printf("Hot reload disabled: %v", err)
        return logger, nil, nil
    }

    return logger, watcher, nil
}
```

### Microservice with Multiple Loggers

```go
func setupComponentLogger(name, configFile string) (*lethe.Logger, error) {
    logger, err := lethe.NewWithDefaults(fmt.Sprintf("%s.log", name))
    if err != nil {
        return nil, err
    }

    _, err = lethe.EnableDynamicConfig(logger, configFile)
    if err != nil {
        log.Printf("Hot reload disabled for %s: %v", name, err)
    }

    return logger, nil
}
```

### Configuration Testing

```go
func TestConfigurationChanges(t *testing.T) {
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "test-config.json")
    
    // Create sample config
    if err := lethe.CreateSampleConfig(configFile); err != nil {
        t.Fatal(err)
    }

    logger, err := lethe.NewWithDefaults("test.log")
    if err != nil {
        t.Fatal(err)
    }
    defer logger.Close()

    watcher, err := lethe.EnableDynamicConfig(logger, configFile)
    if err != nil {
        t.Fatal(err)
    }
    defer watcher.Stop()

    // Test configuration changes...
}
```

## Best Practices

### Production Deployment

1. **File Permissions**: Restrict config file access to application user
2. **Backup Configurations**: Maintain versioned configuration backups
3. **Validation**: Test configuration changes in staging environments
4. **Monitoring**: Monitor audit logs for unauthorized changes
5. **Rollback Plan**: Maintain known-good configuration for emergency rollback

### Configuration Management

1. **Atomic Updates**: Make complete configuration changes, not partial updates
2. **Validation**: Validate JSON syntax before deploying configuration changes
3. **Documentation**: Document configuration changes with rationale
4. **Testing**: Test configuration changes with realistic workloads
5. **Gradual Rollout**: Deploy configuration changes gradually across instances

### Development Workflow

1. **Local Testing**: Test configuration changes locally before deployment
2. **Version Control**: Store configuration files in version control
3. **Change Tracking**: Use commit messages to document configuration rationale
4. **Peer Review**: Review configuration changes through pull requests
5. **Automated Testing**: Include configuration testing in CI/CD pipelines

## Troubleshooting

### Hot Reload Not Working

**Symptoms**: Configuration changes not applied

**Diagnosis**:
```go
if !watcher.IsRunning() {
    log.Error("Config watcher is not running")
}

// Check file permissions
if _, err := os.Stat("config.json"); err != nil {
    log.Error("Config file access error: %v", err)
}
```

**Solutions**:
1. Verify file exists and is readable
2. Check file permissions (644 or 755)
3. Ensure watcher was started successfully
4. Review audit logs for error messages

### Invalid Configuration

**Symptoms**: Configuration changes ignored or error messages

**Diagnosis**:
- Check JSON syntax with `json.Valid()`
- Verify field names match expected schema
- Validate value ranges and types

**Solutions**:
1. Validate JSON syntax before deployment
2. Use `CreateSampleConfig()` as a template
3. Check error callback messages for specific issues
4. Refer to [Configuration Format](#configuration-format) section

### Permission Errors

**Symptoms**: File access denied errors in audit logs

**Solutions**:
1. Ensure application has read access to configuration file
2. Verify parent directory permissions
3. Check SELinux or AppArmor restrictions
4. Validate file ownership matches application user

### Performance Issues

**Symptoms**: High CPU usage or slow configuration updates

**Diagnosis**:
- Monitor file system I/O patterns
- Check for excessive file modifications
- Review audit log frequency

**Solutions**:
1. Increase Argus polling interval if needed
2. Avoid rapid configuration changes
3. Use atomic file updates (write to temp file, then rename)
4. Monitor system resources during configuration changes

---

*For additional support, please refer to the [CONTRIBUTING.md](../CONTRIBUTING.md) guidelines or open an issue in the project repository.*
