// Package lethe provides universal log rotation, designed, originally, for Iris.
//
// Lethe offers superior performance through zero locks, zero allocations, and professional features
// like compression, checksums, and time-based rotation. While compatible with standard logging libraries,
// Lethe was specifically created to provide the native log rotation for Iris but works perfectly with any logging library.
//
// # Quick Start
//
// Basic usage with production defaults:
//
//	logger, err := lethe.NewWithDefaults("app.log")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer logger.Close()
//
//	logger.Write([]byte("Hello, World!"))
//
// # Quick Start with Iris (Recommended)
//
// Advanced log rotation integration with Iris ultra-high performance logging library:
//
//	import "github.com/agilira/lethe/examples"
//
//	// Create Iris high-performance logger
//	irisLogger, _ := iris.New(iris.Config{
//		Level:    iris.Info,
//		Output:   letheAdapter, // Use Lethe for log rotation
//		Encoder:  iris.NewJSONEncoder(),
//		Capacity: 8192,
//	})
//
//	// Create Lethe rotator and adapter
//	rotator, _ := lethe.NewWithDefaults("iris-app.log")
//	adapter := examples.NewLetheIrisAdapter(rotator)
//
//	// Now Iris uses Lethe for ultra-high performance log rotation
//
// # Constructor Functions
//
// Lethe provides multiple constructor functions for different use cases:
//
//	// Simple constructor with legacy parameters (backward compatible)
//	logger, err := lethe.New("app.log", 100, 5) // 100MB, 5 backups
//
//	// Modern string-based constructor (recommended)
//	logger, err := lethe.NewSimple("app.log", "100MB", 5)
//
//	// Production defaults: 100MB, 7d rotation, 10 backups, compression
//	logger, err := lethe.NewWithDefaults("app.log")
//
//	// Daily rotation: 50MB, 24h rotation, 7 backups
//	logger, err := lethe.NewDaily("daily.log")
//
//	// Weekly rotation: 200MB, 7d rotation, 4 backups
//	logger, err := lethe.NewWeekly("weekly.log")
//
//	// Development: 10MB, 1h rotation, no compression, sync writes
//	logger, err := lethe.NewDevelopment("debug.log")
//
// # Advanced Configuration
//
// Full control with detailed configuration:
//
//	config := &lethe.LoggerConfig{
//		Filename:           "app.log",
//		MaxSizeStr:         "500MB",
//		MaxAgeStr:          "30d",
//		MaxBackups:         20,
//		MaxFileAge:         180 * 24 * time.Hour, // 6 months
//		Compress:           true,
//		Checksum:           true,
//		Async:              true,
//		BackpressurePolicy: "adaptive",
//		LocalTime:          true,
//		ErrorCallback: func(eventType string, err error) {
//			log.Printf("Log rotation error (%s): %v", eventType, err)
//		},
//	}
//	logger, err := lethe.NewWithConfig(config)
//
// # String-Based Configuration
//
// Size formats (MaxSizeStr):
//   - "100MB", "1GB", "500KB", "2TB"
//   - Single letters: "100M", "1G", "500K", "2T"
//   - Case insensitive: "100mb", "1gb"
//
// Duration formats (MaxAgeStr):
//   - Hours: "24h", "72h"
//   - Days: "7d", "30d"
//   - Weeks: "2w"
//   - Years: "1y"
//   - Standard Go durations: "30m", "45s", "2h30m"
//
// # High-Performance Async Mode
//
// For high-throughput applications, enable async mode with MPSC buffer:
//
//	logger := &lethe.Logger{
//		Filename:           "high-throughput.log",
//		MaxSizeStr:         "1GB",
//		Async:              true,
//		BufferSize:         4096,               // Larger buffer for high throughput
//		BackpressurePolicy: "adaptive",        // Intelligent backpressure handling
//		AdaptiveFlush:      true,               // Dynamic flush timing
//	}
//
// # Logging Library Integration
//
// Lethe provides exceptional integration with Iris ultra-high performance logging library and supports standard logging frameworks:
//
//	// Iris integration (recommended) - Advanced adapter with WriteSyncer support
//	import "github.com/agilira/lethe/examples"
//
//	app := iris.New()
//	logger, _ := lethe.NewWithDefaults("iris-app.log")
//
//	// Create advanced Iris adapter with WriteSyncer interface
//	adapter := examples.NewLetheIrisAdapter(logger)
//	app.Logger().SetOutput(adapter)
//
//	// Zero-copy performance for high-throughput scenarios
//	adapter.WriteOwned(data) // Transfer buffer ownership for maximum performance
//
//	// Standard library
//	log.SetOutput(logger)
//
//	// Other logging frameworks
//	logrus.SetOutput(logger)
//
//	// Zap integration
//	core := zapcore.NewCore(encoder, zapcore.AddSync(logger), level)
//
//	// Zerolog integration
//	log := zerolog.New(logger).With().Timestamp().Logger()
//
// # Standard Library Compatibility
//
// Lethe provides seamless integration with existing logging patterns:
//
//	// Traditional approach with standard log rotation
//	logger := &SomeLogger{
//		Filename:   "app.log",
//		MaxSize:    100,
//		MaxBackups: 3,
//		MaxAge:     28,
//		Compress:   true,
//	}
//
//	// Modern Lethe approach with enhanced features
//	logger := &lethe.Logger{
//		Filename:    "app.log",
//		MaxSizeStr:  "100MB",        // String-based configuration
//		MaxBackups:  3,
//		MaxAgeStr:   "28d",          // Flexible age configuration
//		MaxFileAge:  90 * 24 * time.Hour, // Advanced backup retention
//		Compress:    true,
//		Checksum:    true,           // Data integrity verification
//		Async:       true,           // Zero-lock performance
//		LocalTime:   true,           // Timezone-aware timestamps
//	}
//
// # Iris High-Performance Logging Integration
//
// Lethe was specifically designed for Iris ultra-high performance logging library and provides exceptional integration:
//
//	app := iris.New()
//
//	// Create high-performance logger for Iris
//	logger, err := lethe.NewWithConfig(&lethe.LoggerConfig{
//		Filename:           "iris-app.log",
//		MaxSizeStr:         "100MB",
//		MaxAgeStr:          "24h",         // Daily rotation for web apps
//		Compress:           true,          // Save disk space
//		Async:              true,          // Handle high request volume
//		BufferSize:         4096,          // Optimized for web traffic
//		BackpressurePolicy: "adaptive",   // Handle traffic spikes
//	})
//
//	// Create advanced Iris adapter with full feature support
//	adapter := examples.NewLetheIrisAdapter(logger)
//	app.Logger().SetOutput(adapter)
//
//	// Advanced features available with Iris integration:
//	// - WriteSyncer interface support for data durability
//	// - WriteOwned() for zero-copy performance
//	// - Automatic buffer management for HTTP request patterns
//	// - Graceful shutdown coordination with Iris lifecycle
//
// This integration provides professional performance for high-throughput logging applications
// with specialized Iris adapter, zero-copy writes, and intelligent buffering optimized
// specifically for ultra-high performance structured logging patterns.
//
// # Performance Features
//
// - Zero locks: Full atomic operations for thread safety
// - Zero allocations: No heap pressure in hot paths
// - MPSC buffer: Lock-free ring buffer for async mode
// - Time caching: 10x faster time operations
// - Auto-scaling: Automatic sync to async under load
// - Iris-optimized: Designed specifically for ultra-high performance logging patterns
// - WriteSyncer support: Advanced synchronization interface for Iris logging
// - Adapter pattern: Specialized bridge for seamless high-performance logging integration
//
// # Professional Features
//
// - SHA-256 checksums: Data integrity verification
// - Crash-safe compression: Atomic file operations
// - Adaptive backpressure: Intelligent overflow handling
// - Cross-platform: Windows/Linux/macOS with retry logic
// - Telemetry: Prometheus-ready metrics
//
// # Error Handling
//
// Lethe provides comprehensive error handling with optional callbacks:
//
//	logger := &lethe.Logger{
//		Filename: "app.log",
//		ErrorCallback: func(eventType string, err error) {
//			// Handle rotation errors, file I/O issues, etc.
//			metrics.Counter("log_errors").WithTag("type", eventType).Inc()
//			if eventType == "file_open" {
//				// Critical error - consider alerting
//				alerting.Send("Log file access failed: " + err.Error())
//			}
//		},
//	}
//
// # Thread Safety
//
// All Logger methods are thread-safe and can be called concurrently from multiple goroutines.
// Lethe uses atomic operations and lock-free algorithms for maximum performance.
//
// # Performance Tips
//
// 1. Use NewWithDefaults() for most applications
// 2. Enable Async for high-throughput scenarios (>1000 writes/sec)
// 3. Tune BufferSize based on your write patterns
// 4. Use WriteOwned() for zero-copy scenarios when possible
// 5. Enable compression to save disk space
// 6. Set appropriate MaxBackups and MaxFileAge for retention policies
//
// # Best Practices
//
// 1. Always call Close() when shutting down (use defer)
// 2. Handle errors from constructor functions
// 3. Use string-based configuration (MaxSizeStr, MaxAgeStr) for clarity
// 4. Set ErrorCallback for production monitoring
// 5. Test rotation behavior with your actual log volume
// 6. Monitor telemetry in production via Stats()
package lethe
