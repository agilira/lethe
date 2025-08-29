# Lethe: Super High-Performance Log Rotation for Go
### an AGILira fragment

Lethe is a lock-free log rotation library for Go, built for applications that demand maximum performance, automatic file management, and production-grade reliability — with intelligent auto-scaling between synchronous and asynchronous modes.

[![CI/CD Pipeline](https://github.com/agilira/lethe/actions/workflows/ci.yml/badge.svg)](https://github.com/agilira/lethe/actions/workflows/ci.yml)
[![Security](https://img.shields.io/badge/security-gosec%20scanned-brightgreen.svg)](https://github.com/agilira/lethe/actions/workflows/ci.yml)
[![Go Report Card](https://img.shields.io/badge/go%20report-A%2B-brightgreen.svg)](https://goreportcard.com/report/github.com/agilira/lethe)
[![Test Coverage](https://img.shields.io/badge/coverage-87%25-brightgreen.svg)](.)

### Features
- **Zero-Lock Architecture**: Atomic operations and CAS-based coordination eliminate mutex overhead
- **Auto-Scaling Performance**: Intelligent switching between sync and MPSC modes based on contention
- **Zero-Allocation Hot Paths**: Pre-allocated buffers and time caching reduce GC pressure
- **Universal Compatibility**: Direct `io.Writer` implementation works with any logging framework
- **Native Iris Integration**: Specifically designed for Iris ultra-high performance logging
- **Built to Scale** - handle millions of log entries with minimal latency

## Architecture

Lethe provides intelligent log rotation through lock-free algorithms and adaptive performance scaling:

```mermaid
graph TB
    %% Application Layer
    App[Application Code] --> LogFw[Logging Frameworks]
    LogFw --> FwInt[Framework Integration<br/>Iris, Logrus, Zap]
    
    %% Lethe Logger Interface
    FwInt --> WriteInt[Write Interface]
    FwInt --> WriteOwned[WriteOwned Interface<br/>Zero-Copy]
    FwInt --> AutoScale[Auto-Scaling Logic<br/>Contention Detection]
    
    %% Execution Modes
    WriteInt --> SyncMode[SYNC MODE<br/>Default]
    WriteOwned --> SyncMode
    AutoScale --> SyncMode
    AutoScale --> MPSCMode[MPSC MODE<br/>High Load]
    
    %% Sync Mode Flow
    SyncMode --> SyncWrite[File.Write]
    SyncWrite --> SyncAtomic[Atomic Update]
    SyncAtomic --> SyncRot[Rotation Check]
    
    %% MPSC Mode Flow
    MPSCMode --> RingBuf[Ring Buffer<br/>Lock-Free]
    RingBuf --> Consumer[Consumer Goroutine]
    Consumer --> MPSCWrite[File.Write]
    MPSCWrite --> MPSCAtomic[Atomic Update]
    MPSCAtomic --> MPSCRot[Rotation Check]
    
    %% Core Components
    RingBuf --> RingBufDetails[Ring Buffer Details<br/>• CAS Operations<br/>• MPSC Pattern<br/>• Buffer Pool]
    SyncWrite --> FileMgr[File Manager<br/>Atomic]
    MPSCWrite --> FileMgr
    FileMgr --> FileMgrDetails[File Manager Details<br/>• File Creation<br/>• Rotation<br/>• Size Tracking]
    
    %% Background Workers
    SyncRot --> BgWorkers[Background Workers<br/>Async Tasks]
    MPSCRot --> BgWorkers
    BgWorkers --> BgDetails[Background Tasks<br/>• Compression<br/>• Cleanup<br/>• Checksums]
    
    %% File System
    FileMgr --> CurrentLog[Current Log<br/>app.log<br/>• Active Writes<br/>• Size Tracking]
    BgWorkers --> RotatedLogs[Rotated Logs<br/>timestamped<br/>• Backup Files<br/>• Age Management]
    BgWorkers --> CompressedFiles[Compressed Files<br/>.gz + .sha256<br/>• Space Optimization<br/>• Integrity Verification]
    
    %% Styling
    classDef applicationLayer fill:#e1f5fe,stroke:#01579b,stroke-width:2px
    classDef letheLayer fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    classDef executionLayer fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    classDef coreLayer fill:#fff3e0,stroke:#e65100,stroke-width:2px
    classDef fileLayer fill:#fce4ec,stroke:#880e4f,stroke-width:2px
    
    class App,LogFw,FwInt applicationLayer
    class WriteInt,WriteOwned,AutoScale letheLayer
    class SyncMode,MPSCMode,SyncWrite,SyncAtomic,SyncRot,MPSCWrite,MPSCAtomic,MPSCRot executionLayer
    class RingBuf,Consumer,RingBufDetails,FileMgr,FileMgrDetails,BgWorkers,BgDetails coreLayer
    class CurrentLog,RotatedLogs,CompressedFiles fileLayer
```

- Zero-lock operations with atomic coordination
- Auto-scaling between sync/async modes
- Background compression and integrity checks

## Performance

Lethe is engineered for ultra-high performance logging. The following benchmarks demonstrate sustained throughput with minimal overhead and intelligent auto-scaling.

```
Write Performance (Sync Mode):            ~3.3 μs/op     (zero-lock operations)
Write Performance (MPSC Mode):            ~3.3 μs/op     (multi-producer scaling)
High Contention (Sync):                   ~109 ns/op     (atomic coordination)
High Contention (MPSC):                   ~105 ns/op     (lock-free scaling)
Zero-Allocation Hot Paths:                0 B/op         (pre-allocated buffers)
Throughput Scaling:                       1-1000+ goroutines (adaptive buffering)
```

**Native Integration:**
- **Iris** - Native integration with zero-copy `WriteOwned()` API
- **Standard Library** - Direct `io.Writer` implementation
- **Universal Compatibility** - Works with any logging framework

**Framework Support:**
- **Logrus** - Drop-in replacement via `logrus.SetOutput(rotator)`
- **Zap** - Core integration via `zapcore.AddSync(rotator)`
- **Zerolog** - Direct integration via `zerolog.New(rotator)`

> **For Maximum Performance**: Use Iris integration with `WriteOwned()` for zero-copy transfers.
> This achieves the highest throughput with minimal memory allocations.
> See [docs/QUICK_START.md](docs/QUICK_START.md) for complete integration examples.

## Installation

```bash
go get github.com/agilira/lethe
```

## Quick Start

```go
import "github.com/agilira/lethe"

// Create logger with sensible defaults
logger, err := lethe.NewWithDefaults("app.log")
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Use as io.Writer - works with any logging framework
logger.Write([]byte("Hello, Lethe!\n"))
```

## Use Cases

- **Ultra-High Performance Logging**: Iris integration with zero-copy transfers
- **Microservices Logging**: Automatic rotation with compression and cleanup
- **High-Throughput Applications**: Auto-scaling between sync/async modes
- **Production Systems**: Crash-safe operations with integrity checks
- **Container Environments**: Automatic file management and rotation

## The Philosophy Behind Lethe

In Greek mythology, Lethe was one of the Oceanids, the daughters of Oceanus and Tethys. As the personification of forgetfulness, Lethe possessed the power to grant oblivion and renewal—the ability to cleanse the past and provide fresh beginnings without the burden of accumulated history.

This embodies Lethe's design philosophy: intelligent log management that gracefully handles the past (old log files) while ensuring the present (current logging) operates at maximum efficiency. Like the Oceanid who could wash away memories, Lethe automatically manages log rotation and cleanup, allowing applications to focus on their core functionality without being weighed down by log file management.

Lethe doesn't just rotate logs—it intelligently manages the entire logging lifecycle, adapting to your application's performance needs while maintaining the reliability and efficiency that production systems demand, just as the Oceanid Lethe provided renewal and fresh starts in the ancient myths.

### File System Security

Lethe includes production-grade file system security with path validation and permission management:

```go
// Default security configuration
config := &lethe.LoggerConfig{
    Filename:           "app.log",
    MaxSizeStr:         "100MB",
    MaxBackups:         10,
    Compress:           true,
    Checksum:           true, // SHA-256 integrity checks
    LocalTime:          true,
    ErrorCallback: func(op string, err error) {
        // Custom error handling for security events
        log.Printf("Lethe security event [%s]: %v", op, err)
    },
}
```

> **Security Notes**: Lethe automatically validates file paths and prevents directory traversal attacks.
> For production deployments, ensure proper file permissions and consider using dedicated log directories.
> Enable checksums for log integrity verification in security-critical environments.

## Documentation

**Quick Links:**
- **[Quick Start Guide](./docs/QUICK_START.md)** - Get running in 2 minutes
- **[API Reference](./docs/API.md)** - Complete API documentation  
- **[Architecture Guide](./docs/ARCHITECTURE.md)** - Deep dive into zero-lock log rotation design
- **[Examples](./examples/)** - Production-ready integration patterns

## License

Lethe is licensed under the [Mozilla Public License 2.0](./LICENSE.md).

---

Lethe • an AGILira fragment
