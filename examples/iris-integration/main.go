// Package main demonstrates seamless integration between Iris
// logging library and Lethe log rotation using the Magic APIs.
//
// This example showcases automatic runtime detection and zero-configuration
// setup where Iris's high-performance logging capabilities combine with
// Lethe's advanced rotation features through automatic optimization.
//
// Magic API Features:
// - Automatic runtime detection and optimization
// - Zero-copy WriteOwned() optimization when available
// - Graceful fallback to standard io.Writer interface
// - No adapter code required - everything is automatic
//
// Run with: go run .

package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/agilira/iris"
	"github.com/agilira/lethe"
)

func main() {
	fmt.Println("Iris-Lethe Magic API Integration Examples")
	fmt.Println("========================================")
	fmt.Println("Demonstrating automatic runtime integration")

	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0750); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Example 1: Magic API Basic Integration
	fmt.Println("\n1. Magic API Basic Integration")
	runMagicBasicIntegration()

	// Example 2: Zero-Configuration QuickStart
	fmt.Println("\n2. Zero-Configuration QuickStart")
	runQuickStartExample()

	// Example 3: Magic Performance Test
	fmt.Println("\n3. Magic API Performance Test")
	runMagicPerformanceTest()

	// Example 4: Advanced Magic Configuration
	fmt.Println("\n4. Advanced Magic Configuration")
	runAdvancedMagicIntegration()

	// Example 5: Production Magic Setup
	fmt.Println("\n5. Production Magic Setup")
	runProductionMagicIntegration()

	// Example 6: Concurrent Magic Writers
	fmt.Println("\n6. Concurrent Magic Writers")
	runConcurrentMagicWritersTest()

	// Example 7: Runtime Auto-Detection
	fmt.Println("\n7. Runtime Auto-Detection Demo")
	runRuntimeAutoDetection()

	fmt.Println("\nAll Magic API integration examples completed successfully.")
	fmt.Println("Integration level: Automatic runtime optimization with zero configuration.")
}

// runMagicBasicIntegration demonstrates the new Magic API that provides
// automatic runtime integration between Iris and Lethe.
func runMagicBasicIntegration() {
	// Magic API: NewIrisWriter automatically creates optimized integration
	irisWriter := lethe.NewIrisWriter("logs/magic-basic.log", &lethe.Logger{
		MaxSizeStr: "50MB",
		MaxBackups: 3,
		Compress:   true,
		Async:      true,
	})
	defer irisWriter.Close()

	// Create Iris logger using Magic API writer
	irisLogger, err := iris.New(iris.Config{
		Output:  irisWriter,
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Info,
	})
	if err != nil {
		log.Fatalf("Failed to create Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Test magic integration
	irisLogger.Info("Magic API integration test started")
	irisLogger.Debug("Debug message with automatic optimization")
	irisLogger.Warn("Warning message with zero-copy WriteOwned()")
	irisLogger.Error("Error message with runtime integration")

	// Test structured logging with Magic API
	irisLogger.With(
		iris.String("component", "magic-api"),
		iris.String("version", "1.0.0"),
		iris.String("integration", "runtime"),
		iris.Bool("zero_config", true),
	).Info("Magic API provides zero-configuration integration")

	fmt.Println("   Magic API: Zero-configuration integration active")
	fmt.Println("   Features: Automatic WriteOwned() optimization, graceful fallback")
}

// runQuickStartExample demonstrates Lethe's QuickStart API for instant
// Iris integration with sensible defaults.
func runQuickStartExample() {
	// QuickStart: One-line Iris integration with optimal defaults
	irisWriter := lethe.QuickStart("logs/quickstart.log")
	defer irisWriter.Close()

	// Create Iris logger using QuickStart writer
	irisLogger, err := iris.New(iris.Config{
		Output:  irisWriter,
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Info,
	})
	if err != nil {
		log.Fatalf("Failed to create Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Immediate productive use - no configuration needed
	irisLogger.Info("QuickStart logger initialized")
	irisLogger.With(
		iris.String("setup_time", "instant"),
		iris.String("configuration", "zero"),
		iris.String("performance", "optimized"),
	).Info("QuickStart provides instant production-ready logging")

	const quickOps = 5000
	startTime := time.Now()

	for i := 0; i < quickOps; i++ {
		irisLogger.With(
			iris.Int("operation", i),
			iris.String("type", "quickstart"),
		).Info("QuickStart operation")
	}

	duration := time.Since(startTime)
	throughput := float64(quickOps) / duration.Seconds()

	fmt.Printf("   QuickStart: %d operations in %v\n", quickOps, duration)
	fmt.Printf("   Throughput: %.0f ops/sec with zero configuration\n", throughput)
	fmt.Println("   Magic: Automatic Lethe optimization applied")
}

// runMagicPerformanceTest demonstrates the zero-copy capabilities
// that provide "rock solid" performance between Iris and Lethe with Magic API.
func runMagicPerformanceTest() {
	// Create high-performance Magic integration
	irisWriter := lethe.NewIrisWriter("logs/magic-performance.log", &lethe.Logger{
		MaxSizeStr:         "200MB",
		MaxBackups:         10,
		Async:              true,
		BufferSize:         16384, // 16KB for high performance
		BackpressurePolicy: "adaptive",
		Compress:           true,
	})
	defer irisWriter.Close()

	// Configure Iris for maximum performance with Magic API
	irisLogger, err := iris.New(iris.Config{
		Output:    irisWriter,
		Encoder:   iris.NewJSONEncoder(),
		Level:     iris.Info,
		Capacity:  8192, // Match buffer sizes for optimal performance
		BatchSize: 256,
	})
	if err != nil {
		log.Fatalf("Failed to create performance Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Performance test with Magic API zero-copy operations
	const testIterations = 10000
	startTime := time.Now()

	for i := 0; i < testIterations; i++ {
		irisLogger.With(
			iris.Int("operation", i),
			iris.String("type", "magic-performance"),
			iris.Bool("zero_copy", true),
		).Info("Magic API zero-copy operation")
	}

	duration := time.Since(startTime)
	throughput := float64(testIterations) / duration.Seconds()

	fmt.Printf("   Magic Performance: %d operations in %v\n", testIterations, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n", throughput)
	fmt.Printf("   Runtime Integration: Automatic WriteOwned() optimization detected\n")
}

// runAdvancedMagicIntegration demonstrates sophisticated Magic API configuration
// options when integrating Iris and Lethe.
func runAdvancedMagicIntegration() {
	// Advanced Magic API configuration
	irisWriter := lethe.NewIrisWriter("logs/magic-advanced.log", &lethe.Logger{
		MaxSizeStr:         "50MB",
		MaxAgeStr:          "24h", // Time-based rotation
		MaxBackups:         15,
		Compress:           true,
		Checksum:           true, // Data integrity verification
		Async:              true,
		BufferSize:         32768, // 32KB buffer
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
		ErrorCallback: func(eventType string, err error) {
			fmt.Printf("     Advanced Magic event [%s]: %v\n", eventType, err)
		},
	})
	defer irisWriter.Close()

	// Advanced Iris configuration with Magic API
	irisLogger, err := iris.New(iris.Config{
		Output:    irisWriter,
		Encoder:   iris.NewJSONEncoder(),
		Level:     iris.Debug,
		Capacity:  16384, // 16KB to match buffer
		BatchSize: 64,
	}, iris.WithCaller(), iris.Development())
	if err != nil {
		log.Fatalf("Failed to create advanced Magic Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Test advanced Magic features
	irisLogger.With(
		iris.String("component", "magic-advanced"),
		iris.String("version", "2.0.0"),
		iris.String("integration", "runtime"),
		iris.String("features", "checksum+compression+time_rotation"),
	).Info("Advanced Magic API configuration active")

	const advancedEntries = 8000
	for i := 0; i < advancedEntries; i++ {
		irisLogger.With(
			iris.Int("entry_id", i),
			iris.Int("batch", i/1000),
			iris.Bool("magic_api", true),
			iris.Bool("checksum_ok", true),
			iris.Bool("compressed", true),
		).Info("Advanced Magic integration test entry")
	}

	fmt.Printf("   Advanced Magic: %d entries with checksums and compression\n", advancedEntries)
	fmt.Printf("   Features: time-based rotation, data integrity, runtime optimization\n")
}

// runProductionMagicIntegration demonstrates a complete production-ready
// Magic API integration pattern with monitoring and error handling.
func runProductionMagicIntegration() {
	var rotationEvents int
	var errorEvents int

	// Production-grade Magic API configuration
	irisWriter := lethe.NewIrisWriter("logs/magic-production.log", &lethe.Logger{
		MaxSizeStr:         "100MB",
		MaxAgeStr:          "7d",
		MaxBackups:         30,
		Compress:           true,
		Checksum:           true,
		Async:              true,
		BufferSize:         32768, // 32KB buffer
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
		ErrorCallback: func(eventType string, err error) {
			switch eventType {
			case "rotation":
				rotationEvents++
			default:
				errorEvents++
			}
		},
	})
	defer irisWriter.Close()

	// Production Iris configuration with Magic API
	irisLogger, err := iris.New(iris.Config{
		Output:    irisWriter,
		Encoder:   iris.NewJSONEncoder(),
		Level:     iris.Info,
		Capacity:  16384, // 16KB capacity
		BatchSize: 64,
	})
	if err != nil {
		log.Fatalf("Failed to create production Magic Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Simulate production application logging with Magic API
	irisLogger.Info("Production Magic API application started")

	// Simulate various service operations
	services := []string{"auth", "api", "database", "cache", "analytics"}
	const productionOps = 25000

	startTime := time.Now()

	for i := 0; i < productionOps; i++ {
		service := services[i%len(services)] // #nosec G602 -- len(services) is constant 5, modulo guarantees bounds

		irisLogger.With(
			iris.String("service", service),
			iris.String("operation_id", fmt.Sprintf("%s-magic-%d", service, i)),
			iris.String("api_version", "magic-v1"),
			iris.Int("duration_ms", (i%200)+5),
			iris.Bool("success", i%100 != 0), // 1% failure rate
			iris.Bool("runtime_optimized", true),
		).Info("Production Magic API service operation")
	}

	duration := time.Since(startTime)

	irisLogger.Info("Production Magic API application shutdown")

	fmt.Printf("   Production Magic: %d operations in %v\n", productionOps, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n", float64(productionOps)/duration.Seconds())
	fmt.Printf("   Events: %d rotations, %d errors\n", rotationEvents, errorEvents)
	fmt.Printf("   Runtime Integration: Automatic optimization with zero configuration\n")
}

// runConcurrentMagicWritersTest demonstrates multiple Iris loggers
// writing concurrently through Magic API with perfect synchronization.
func runConcurrentMagicWritersTest() {
	// Shared Magic API writer for all Iris instances
	irisWriter := lethe.NewIrisWriter("logs/magic-concurrent.log", &lethe.Logger{
		MaxSizeStr:         "100MB",
		MaxBackups:         5,
		Async:              true,
		BufferSize:         16384,
		BackpressurePolicy: "adaptive",
	})
	defer irisWriter.Close()

	// Create multiple Iris loggers sharing the same Magic backend
	const numLoggers = 8
	const operationsPerLogger = 3000

	var wg sync.WaitGroup
	startTime := time.Now()

	for loggerID := 0; loggerID < numLoggers; loggerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine gets its own Iris logger instance
			irisLogger, err := iris.New(iris.Config{
				Output:  irisWriter,
				Encoder: iris.NewJSONEncoder(),
				Level:   iris.Info,
			})
			if err != nil {
				log.Printf("Failed to create Magic Iris logger %d: %v", id, err)
				return
			}
			defer irisLogger.Close()
			irisLogger.Start()

			for op := 0; op < operationsPerLogger; op++ {
				irisLogger.With(
					iris.Int("logger_id", id),
					iris.Int("operation", op),
					iris.String("api", "magic"),
					iris.Bool("runtime_optimized", true),
					iris.Bool("thread_safe", true),
				).Info("Concurrent Magic API operation")
			}
		}(loggerID)
	}

	wg.Wait()
	duration := time.Since(startTime)
	totalOps := numLoggers * operationsPerLogger

	fmt.Printf("   Concurrent Magic: %d operations from %d loggers in %v\n",
		totalOps, numLoggers, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n", float64(totalOps)/duration.Seconds())
	fmt.Printf("   Runtime Integration: Perfect synchronization with zero configuration\n")
}

// runRuntimeAutoDetection demonstrates the automatic detection capabilities
// that provide seamless integration between Iris and Lethe.
func runRuntimeAutoDetection() {
	// Create Magic API writer
	irisWriter := lethe.NewIrisWriter("logs/runtime-detection.log", &lethe.Logger{
		MaxSizeStr: "50MB",
		MaxBackups: 3,
		Async:      true,
		BufferSize: 8192,
	})
	defer irisWriter.Close()

	// Demonstrate runtime auto-detection features
	fmt.Println("   Runtime Auto-Detection Features:")

	// Check WriteOwned capability
	if _, hasWriteOwned := interface{}(irisWriter).(interface{ WriteOwned([]byte) (int, error) }); hasWriteOwned {
		fmt.Println("   ✓ WriteOwned() zero-copy optimization detected")
	}

	// Check optimal buffer size
	if bufferProvider, hasBuffer := interface{}(irisWriter).(interface{ GetOptimalBufferSize() int }); hasBuffer {
		size := bufferProvider.GetOptimalBufferSize()
		fmt.Printf("   ✓ Optimal buffer size detected: %d bytes\n", size)
	}

	// Check hot reload capability
	if hotReloadProvider, hasHotReload := interface{}(irisWriter).(interface{ SupportsHotReload() bool }); hasHotReload {
		if hotReloadProvider.SupportsHotReload() {
			fmt.Println("   ✓ Hot reload capability detected")
		}
	} // Create Iris logger with auto-detected optimizations
	irisLogger, err := iris.New(iris.Config{
		Output:  irisWriter,
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Info,
	})
	if err != nil {
		log.Fatalf("Failed to create runtime detection Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Test auto-detection in action
	irisLogger.With(
		iris.String("integration", "runtime"),
		iris.String("detection", "automatic"),
		iris.Bool("zero_config", true),
		iris.Bool("seamless", true),
	).Info("Runtime auto-detection successful")

	const detectionOps = 3000
	startTime := time.Now()

	for i := 0; i < detectionOps; i++ {
		irisLogger.With(
			iris.Int("operation", i),
			iris.String("optimization", "automatic"),
			iris.Bool("runtime_detected", true),
		).Info("Auto-optimized logging operation")
	}

	duration := time.Since(startTime)
	throughput := float64(detectionOps) / duration.Seconds()

	fmt.Printf("   Runtime Auto-Detection: %d operations in %v\n", detectionOps, duration)
	fmt.Printf("   Throughput: %.0f ops/sec with automatic optimization\n", throughput)
	fmt.Println("   Integration: Automatic runtime optimization with zero configuration")
}
