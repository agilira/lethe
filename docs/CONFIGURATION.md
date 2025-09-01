# Lethe Configuration Guide

Lethe provides flexible configuration options through multiple sources: programmatic defaults, JSON files, and environment variables. This guide covers all configuration methods with practical examples.

## Quick Reference

| Method | Use Case | Best For |
|--------|----------|----------|
| **Programmatic** | Simple configs, library code | Development, simple applications |
| **JSON Files** | Structured configuration | Production applications, config management |
| **Environment** | Runtime overrides | Docker, Kubernetes, CI/CD |
| **Combined** | Multi-source config | Enterprise applications |

## 1. Programmatic Configuration

### Basic Configuration

```go
package main

import "github.com/agilira/lethe"

func main() {
    // Using constructor functions
    logger, err := lethe.NewWithDefaults("app.log")
    if err != nil {
        panic(err)
    }
    defer logger.Close()

    // Or with custom config
    config := &lethe.LoggerConfig{
        Filename:   "custom.log",
        MaxSizeStr: "100MB",
        MaxBackups: 10,
        Compress:   true,
        Async:      true,
    }

    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        panic(err)
    }
    defer logger.Close()
}
```

### Available Constructors

| Constructor | Description | Use Case |
|-------------|-------------|----------|
| `NewWithDefaults(filename)` | Production-ready defaults | Most applications |
| `NewSimple(filename, maxSize, maxBackups)` | Human-readable sizes | Custom size limits |
| `NewDaily(filename)` | Daily rotation | Daily archives |
| `NewWeekly(filename)` | Weekly rotation | Weekly archives |
| `NewDevelopment(filename)` | Development setup | Debugging, testing |
| `NewWithConfig(config)` | Full control | Advanced customization |

## 2. JSON Configuration

Load configuration from JSON files or strings using the standard Go JSON parser.

### JSON File Example

```json
{
  "filename": "app.log",
  "max_size_str": "100MB",
  "max_age_str": "7d",
  "max_backups": 10,
  "compress": true,
  "checksum": false,
  "async": true,
  "buffer_size": 4096,
  "backpressure_policy": "adaptive",
  "flush_interval": "5ms",
  "adaptive_flush": false,
  "local_time": true,
  "file_mode": 420,
  "retry_count": 3,
  "retry_delay": "10ms"
}
```

### Usage Examples

#### From JSON String

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    jsonConfig := `{
        "filename": "app.log",
        "max_size_str": "100MB",
        "max_age_str": "7d",
        "max_backups": 10,
        "compress": true,
        "async": true,
        "local_time": true
    }`

    config, err := lethe.LoadFromJSON([]byte(jsonConfig))
    if err != nil {
        log.Fatal("Failed to parse JSON config:", err)
    }

    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        log.Fatal("Failed to create logger:", err)
    }
    defer logger.Close()

    logger.Write([]byte("Hello from JSON-configured logger!\n"))
}
```

#### From JSON File

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    config, err := lethe.LoadFromJSONFile("config.json")
    if err != nil {
        log.Fatal("Failed to load config file:", err)
    }

    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        log.Fatal("Failed to create logger:", err)
    }
    defer logger.Close()

    logger.Write([]byte("Hello from file-configured logger!\n"))
}
```

## 3. Environment Variables

Configure Lethe using environment variables with customizable prefixes.

### Environment Variable Mapping

| Variable | Description | Example |
|----------|-------------|---------|
| `{PREFIX}_FILENAME` | Log file path | `app.log` |
| `{PREFIX}_MAX_SIZE` | Max file size | `100MB`, `1GB` |
| `{PREFIX}_MAX_AGE` | Max file age | `7d`, `24h` |
| `{PREFIX}_MAX_BACKUPS` | Number of backups | `10` |
| `{PREFIX}_COMPRESS` | Enable compression | `true`, `false` |
| `{PREFIX}_CHECKSUM` | Enable checksums | `true`, `false` |
| `{PREFIX}_ASYNC` | Enable async mode | `true`, `false` |
| `{PREFIX}_LOCAL_TIME` | Use local timezone | `true`, `false` |
| `{PREFIX}_BACKPRESSURE_POLICY` | Buffer policy | `adaptive`, `drop` |
| `{PREFIX}_BUFFER_SIZE` | Buffer size | `4096` |
| `{PREFIX}_FLUSH_INTERVAL` | Flush interval | `5ms` |
| `{PREFIX}_ADAPTIVE_FLUSH` | Adaptive flushing | `false` |
| `{PREFIX}_FILE_MODE` | File permissions | `420` |
| `{PREFIX}_RETRY_COUNT` | Retry attempts | `3` |
| `{PREFIX}_RETRY_DELAY` | Retry delay | `10ms` |

### Usage Examples

#### Docker Environment

```bash
export LETHE_FILENAME="/var/log/app.log"
export LETHE_MAX_SIZE="500MB"
export LETHE_MAX_AGE="30d"
export LETHE_MAX_BACKUPS="20"
export LETHE_COMPRESS="true"
export LETHE_ASYNC="true"
export LETHE_LOCAL_TIME="true"
```

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    config, err := lethe.LoadFromEnv("LETHE")
    if err != nil {
        log.Fatal("Failed to load env config:", err)
    }

    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        log.Fatal("Failed to create logger:", err)
    }
    defer logger.Close()

    logger.Write([]byte("Hello from environment-configured logger!\n"))
}
```

#### Kubernetes ConfigMap + Environment

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: lethe-config
data:
  config.json: |
    {
      "filename": "/var/log/app.log",
      "max_size_str": "100MB",
      "max_backups": 10,
      "compress": true
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
      - name: app
        env:
        - name: LETHE_FILENAME
          value: "/var/log/app.log"
        - name: LETHE_MAX_SIZE
          value: "200MB"  # Override JSON config
        - name: LETHE_COMPRESS
          value: "true"
        volumeMounts:
        - name: config
          mountPath: /etc/lethe/
      volumes:
      - name: config
        configMap:
          name: lethe-config
```

## 4. Combined Configuration

Combine multiple configuration sources with intelligent precedence: **Defaults → JSON → Environment**.

### Basic Combined Configuration

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    // Define defaults (lowest priority)
    defaults := &lethe.LoggerConfig{
        Filename:   "default.log",
        MaxSizeStr: "10MB",
        MaxBackups: 3,
        Compress:   false,
    }

    // Create configuration source
    source := lethe.ConfigSource{
        Defaults:   defaults,           // Fallback values
        JSONFile:   "config.json",      // Structured config
        EnvPrefix:  "LETHE",            // Runtime overrides
    }

    config, err := lethe.LoadFromSources(source)
    if err != nil {
        log.Fatal("Failed to load combined config:", err)
    }

    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        log.Fatal("Failed to create logger:", err)
    }
    defer logger.Close()

    logger.Write([]byte("Hello from combined configuration!\n"))
}
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  app:
    image: myapp:latest
    environment:
      LETHE_FILENAME: "/app/logs/app.log"
      LETHE_MAX_SIZE: "500MB"
      LETHE_COMPRESS: "true"
      LETHE_ASYNC: "true"
    volumes:
      - ./config.json:/app/config.json:ro
    command: ["/app/myapp", "--config", "/app/config.json"]
```

```go
package main

import (
    "flag"
    "log"
    "github.com/agilira/lethe"
)

func main() {
    configFile := flag.String("config", "", "Path to JSON config file")
    flag.Parse()

    source := lethe.ConfigSource{
        Defaults: &lethe.LoggerConfig{
            Filename:   "/tmp/app.log",
            MaxSizeStr: "10MB",
            Compress:   false,
        },
        EnvPrefix: "LETHE",
    }

    // Use config file if provided
    if *configFile != "" {
        source.JSONFile = *configFile
    }

    config, err := lethe.LoadFromSources(source)
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        log.Fatal("Failed to create logger:", err)
    }
    defer logger.Close()

    log.Printf("Logger configured: filename=%s, max_size=%s, compress=%t",
        config.Filename, config.MaxSizeStr, config.Compress)
}
```

## 5. Configuration Templates

### Production Template

```json
{
  "filename": "/var/log/app/app.log",
  "max_size_str": "500MB",
  "max_age_str": "30d",
  "max_backups": 20,
  "compress": true,
  "checksum": true,
  "async": true,
  "buffer_size": 8192,
  "backpressure_policy": "adaptive",
  "flush_interval": "10ms",
  "local_time": true,
  "retry_count": 5,
  "retry_delay": "50ms"
}
```

### Development Template

```json
{
  "filename": "debug.log",
  "max_size_str": "10MB",
  "max_age_str": "1h",
  "max_backups": 5,
  "compress": false,
  "checksum": false,
  "async": false,
  "local_time": true,
  "retry_count": 3,
  "retry_delay": "10ms"
}
```

### High-Performance Template

```json
{
  "filename": "/var/log/highperf.log",
  "max_size_str": "1GB",
  "max_age_str": "24h",
  "max_backups": 30,
  "compress": true,
  "checksum": false,
  "async": true,
  "buffer_size": 16384,
  "backpressure_policy": "adaptive",
  "flush_interval": "1ms",
  "adaptive_flush": true,
  "retry_count": 3,
  "retry_delay": "5ms"
}
```

## 6. Error Handling

### Configuration Validation

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    // This will fail - missing required filename
    badJSON := `{"max_size_str": "100MB"}`

    _, err := lethe.LoadFromJSON([]byte(badJSON))
    if err != nil {
        log.Printf("Expected error: %v", err)
    }

    // This will also fail - empty env prefix
    _, err = lethe.LoadFromEnv("")
    if err != nil {
        log.Printf("Expected error: %v", err)
    }

    // This will fail - no filename provided anywhere
    source := lethe.ConfigSource{
        Defaults: &lethe.LoggerConfig{MaxSizeStr: "100MB"},
    }

    _, err = lethe.LoadFromSources(source)
    if err != nil {
        log.Printf("Expected error: %v", err)
    }
}
```

### Custom Error Handling

```go
package main

import (
    "log"
    "github.com/agilira/lethe"
)

func main() {
    config := &lethe.LoggerConfig{
        Filename:   "app.log",
        MaxSizeStr: "100MB",
        ErrorCallback: func(operation string, err error) {
            log.Printf("Lethe error in %s: %v", operation, err)
            // Send to monitoring system, etc.
        },
    }

    logger, err := lethe.NewWithConfig(config)
    if err != nil {
        log.Fatal("Failed to create logger:", err)
    }
    defer logger.Close()
}
```

## 7. Best Practices

### 1. Use Combined Configuration for Production

```go
// Production-ready configuration
source := lethe.ConfigSource{
    Defaults: &lethe.LoggerConfig{
        Filename:   "/tmp/app.log",  // Safe fallback
        MaxSizeStr: "10MB",
        Compress:   false,
    },
    JSONFile:  "/etc/myapp/config.json",  // Structured config
    EnvPrefix: "LOG",                      // Runtime overrides
}

config, err := lethe.LoadFromSources(source)
```

### 2. Validate Configuration

```go
config, err := lethe.LoadFromSources(source)
if err != nil {
    log.Fatal("Configuration error:", err)
}

// Additional validation
if config.MaxBackups == 0 {
    config.MaxBackups = 5  // Set reasonable default
}
```

### 3. Use Environment Variables for Secrets

```bash
export LOG_API_KEY="your-secret-key"
export LOG_DATABASE_URL="postgres://..."
```

### 4. Document Your Configuration

```go
// config.go
type AppConfig struct {
    Logger *lethe.LoggerConfig
    APIKey string
    DB     DatabaseConfig
}

// LoadAppConfig loads complete application configuration
func LoadAppConfig() (*AppConfig, error) {
    loggerSource := lethe.ConfigSource{
        JSONFile:  "config.json",
        EnvPrefix: "LOG",
        Defaults:  getDefaultLoggerConfig(),
    }

    loggerConfig, err := lethe.LoadFromSources(loggerSource)
    if err != nil {
        return nil, fmt.Errorf("failed to load logger config: %w", err)
    }

    return &AppConfig{
        Logger: loggerConfig,
        APIKey: os.Getenv("API_KEY"),
        DB:     loadDatabaseConfig(),
    }, nil
}
```

### 5. Handle Configuration Reload

```go
type ConfigurableLogger struct {
    logger *lethe.Logger
    config *lethe.LoggerConfig
    mu     sync.RWMutex
}

func (cl *ConfigurableLogger) ReloadConfig(source lethe.ConfigSource) error {
    cl.mu.Lock()
    defer cl.mu.Unlock()

    newConfig, err := lethe.LoadFromSources(source)
    if err != nil {
        return err
    }

    // Close old logger
    if cl.logger != nil {
        cl.logger.Close()
    }

    // Create new logger
    newLogger, err := lethe.NewWithConfig(newConfig)
    if err != nil {
        return err
    }

    cl.logger = newLogger
    cl.config = newConfig
    return nil
}
```

## 8. Migration Guide

### From Programmatic to JSON

**Before:**
```go
logger := &lethe.Logger{
    Filename:   "app.log",
    MaxSizeStr: "100MB",
    MaxBackups: 10,
    Compress:   true,
}
```

**After:**
```json
// config.json
{
  "filename": "app.log",
  "max_size_str": "100MB",
  "max_backups": 10,
  "compress": true
}
```

```go
config, _ := lethe.LoadFromJSONFile("config.json")
logger, _ := lethe.NewWithConfig(config)
```

### From JSON to Environment

**Before:**
```json
{
  "filename": "app.log",
  "max_size_str": "100MB"
}
```

**After:**
```bash
export APP_FILENAME="app.log"
export APP_MAX_SIZE="100MB"
```

```go
config, _ := lethe.LoadFromEnv("APP")
logger, _ := lethe.NewWithConfig(config)
```

## 9. Troubleshooting

### Common Issues

1. **"filename is required" error**
   - Ensure filename is provided in at least one configuration source
   - Check environment variable names match the prefix

2. **Configuration not applied**
   - Remember precedence: Environment > JSON > Defaults
   - Check for typos in environment variable names
   - Verify JSON file is readable and valid

3. **Performance issues with environment loading**
   - Environment loading is done once at startup
   - Use `LoadFromEnv()` result caching if needed

4. **File permission issues**
   - Ensure the process has write permissions to log directory
   - Check `file_mode` setting (default: 0644)

### Debug Configuration

```go
config, err := lethe.LoadFromSources(source)
if err != nil {
    log.Fatal(err)
}

// Debug: Print actual configuration
log.Printf("Filename: %s", config.Filename)
log.Printf("MaxSize: %s", config.MaxSizeStr)
log.Printf("MaxBackups: %d", config.MaxBackups)
log.Printf("Compress: %t", config.Compress)
log.Printf("Async: %t", config.Async)
```

## See Also

- [Quick Start Guide](QUICK_START.md) - Basic usage and integration
- [API Documentation](API.md) - Complete function reference
- [Architecture Guide](ARCHITECTURE.md) - Technical implementation details

---

Lethe • an AGILira fragment
