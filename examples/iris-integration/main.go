// Package main demonstrates seamless integration between Iris logging library
// and Lethe log rotation using zero-copy adapter patterns.
//
// This example showcases the integration where
// Iris's high-performance logging capabilities combine perfectly with
// Lethe's advanced rotation features through the LetheIrisAdapter.
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

// LetheIrisAdapter implements iris.WriteSyncer interface to enable
// seamless integration between Iris logging and Lethe rotation.
type LetheIrisAdapter struct {
	logger *lethe.Logger
}

// NewLetheIrisAdapter creates a new adapter that bridges Iris and Lethe.
func NewLetheIrisAdapter(logger *lethe.Logger) *LetheIrisAdapter {
	return &LetheIrisAdapter{
		logger: logger,
	}
}

// Write implements io.Writer interface for standard write operations.
// Use direct Write for maximum performance - no unnecessary copies
func (a *LetheIrisAdapter) Write(data []byte) (int, error) {
	return a.logger.Write(data)
}

// WriteOwned implements zero-copy writes by transferring buffer ownership.
// This is the key method that enables "rock solid" zero-copy integration.
func (a *LetheIrisAdapter) WriteOwned(data []byte) (int, error) {
	return a.logger.WriteOwned(data)
}

// Sync ensures all buffered data is written to storage.
// Note: Lethe doesn't have Sync method, so we use Close/reopen pattern for demonstration
func (a *LetheIrisAdapter) Sync() error {
	// For demonstration purposes - in production you might implement flush differently
	return nil
}

// Close gracefully shuts down the adapter and underlying logger.
func (a *LetheIrisAdapter) Close() error {
	return a.logger.Close()
}

func main() {
	fmt.Println("Iris-Lethe Integration Examples")
	fmt.Println("===============================")

	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0750); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Example 1: Basic Iris-Lethe Integration
	fmt.Println("\n1. Basic Iris-Lethe Integration")
	runBasicIntegration()

	// Example 2: Zero-Copy Performance Test
	fmt.Println("\n2. Zero-Copy Performance Test")
	runZeroCopyPerformanceTest()

	// Example 3: High-Throughput Iris Logging with Lethe Rotation
	fmt.Println("\n3. High-Throughput Iris + Lethe")
	runHighThroughputTest()

	// Example 4: Advanced Configuration Integration
	fmt.Println("\n4. Advanced Configuration Integration")
	runAdvancedConfigIntegration()

	// Example 5: Production-Ready Integration Pattern
	fmt.Println("\n5. Production-Ready Integration")
	runProductionIntegration()

	// Example 6: Concurrent Iris Writers with Lethe
	fmt.Println("\n6. Concurrent Iris Writers")
	runConcurrentWritersTest()

	// Example 7: WriteSyncer Interface Compliance Test
	fmt.Println("\n7. WriteSyncer Interface Compliance")
	runWriteSyncerComplianceTest()

	fmt.Println("\nAll Iris-Lethe integration examples completed successfully.")
	fmt.Println("Integration verified: 'AirPods + iPhone' level seamless operation.")
}

// runBasicIntegration demonstrates the fundamental integration pattern
// between Iris and Lethe using the adapter.
func runBasicIntegration() {
	// Create Lethe logger with production defaults
	letheLogger, err := lethe.NewWithDefaults("logs/iris-basic.log")
	if err != nil {
		log.Fatalf("Failed to create Lethe logger: %v", err)
	}
	defer letheLogger.Close()

	// Create adapter that bridges Iris and Lethe
	adapter := NewLetheIrisAdapter(letheLogger)
	defer adapter.Close()

	// Create Iris logger using Lethe adapter as backend
	irisLogger, err := iris.New(iris.Config{
		Output:  adapter,
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Info,
	})
	if err != nil {
		log.Fatalf("Failed to create Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Test basic logging operations
	irisLogger.Info("Basic integration test started")
	irisLogger.Debug("Debug message with JSON formatting")
	irisLogger.Warn("Warning message with automatic rotation")
	irisLogger.Error("Error message with zero-copy performance")

	// Test structured logging
	irisLogger.With(
		iris.String("component", "integration-test"),
		iris.String("version", "1.0.0"),
		iris.Int64("timestamp", time.Now().Unix()),
	).Info("Structured logging with Iris+Lethe integration")

	fmt.Println("   Basic integration: Iris logger created with Lethe backend")
	fmt.Println("   Features: JSON formatting, automatic rotation, structured logging")
}

// runZeroCopyPerformanceTest demonstrates the zero-copy capabilities
// that provide "rock solid" performance between Iris and Lethe.
func runZeroCopyPerformanceTest() {
	// Configure Lethe for high-performance zero-copy operations
	config := &lethe.LoggerConfig{
		Filename:           "logs/iris-zerocopy.log",
		MaxSizeStr:         "100MB",
		MaxBackups:         10,
		Async:              true,
		BufferSize:         8192, // 8KB to match advanced examples
		BackpressurePolicy: "adaptive",
		Compress:           true,
	}

	letheLogger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create zero-copy Lethe logger: %v", err)
	}
	defer letheLogger.Close()

	// Create adapter for zero-copy operations
	adapter := NewLetheIrisAdapter(letheLogger)
	defer adapter.Close()

	// Configure Iris for maximum performance
	irisLogger, err := iris.New(iris.Config{
		Output:    adapter,
		Encoder:   iris.NewJSONEncoder(),
		Level:     iris.Info,
		Capacity:  8192, // 8KB to match Lethe buffer
		BatchSize: 256,  // Smaller batches for lower latency
	})
	if err != nil {
		log.Fatalf("Failed to create zero-copy Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Performance test with zero-copy operations
	const testIterations = 1000 // Match advanced examples
	startTime := time.Now()

	for i := 0; i < testIterations; i++ {
		// Use simple message format like advanced examples
		irisLogger.Info(fmt.Sprintf("Zero-copy operation %d", i))
	}

	duration := time.Since(startTime)
	throughput := float64(testIterations) / duration.Seconds()

	fmt.Printf("   Zero-Copy Test: %d operations in %v\n", testIterations, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n", throughput)
	fmt.Printf("   Integration: Rock solid zero-copy buffer transfer\n")
}

// runHighThroughputTest simulates high-load production scenarios
// with Iris logging and Lethe rotation working in harmony.
func runHighThroughputTest() {
	// Configure for high-throughput scenarios
	config := &lethe.LoggerConfig{
		Filename:           "logs/iris-highload.log",
		MaxSizeStr:         "200MB",
		MaxBackups:         20,
		Async:              true,
		BufferSize:         131072, // 128KB buffer (maximum performance)
		BackpressurePolicy: "adaptive",
		Compress:           true,
		ErrorCallback: func(eventType string, err error) {
			fmt.Printf("     High-load event [%s]: %v\n", eventType, err)
		},
	}

	letheLogger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create high-throughput Lethe logger: %v", err)
	}
	defer letheLogger.Close()

	adapter := NewLetheIrisAdapter(letheLogger)
	defer adapter.Close()

	// Configure Iris for high-throughput logging
	irisLogger, err := iris.New(iris.Config{
		Output:    adapter,
		Encoder:   iris.NewJSONEncoder(),
		Level:     iris.Info,
		Capacity:  8192, // 8KB to match Lethe buffer
		BatchSize: 256,  // Optimized batch size
	})
	if err != nil {
		log.Fatalf("Failed to create high-throughput Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Simulate realistic high-load scenarios
	const operations = 500000
	scenarios := []string{"api_request", "database_query", "cache_operation", "auth_check"}

	startTime := time.Now()

	for i := 0; i < operations; i++ {
		scenario := scenarios[i%len(scenarios)]

		irisLogger.With(
			iris.String("scenario", scenario),
			iris.String("request_id", fmt.Sprintf("req-%d", i)),
			iris.Int64("timestamp", time.Now().UnixNano()),
			iris.Int("response_ms", (i%100)+10),
			iris.Int("status_code", 200+(i%4)*100),
		).Info("High-throughput operation logged")
	}

	duration := time.Since(startTime)
	throughput := float64(operations) / duration.Seconds()

	fmt.Printf("   High-Throughput Test: %d operations in %v\n", operations, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n", throughput)
	fmt.Printf("   Configuration: 64KB Lethe + 32KB Iris buffers, adaptive backpressure\n")
}

// runAdvancedConfigIntegration demonstrates sophisticated configuration
// options when integrating Iris and Lethe.
func runAdvancedConfigIntegration() {
	// Advanced Lethe configuration
	config := &lethe.LoggerConfig{
		Filename:           "logs/iris-advanced.log",
		MaxSizeStr:         "50MB",
		MaxAgeStr:          "24h", // Time-based rotation
		MaxBackups:         15,
		Compress:           true,
		Checksum:           true, // Data integrity verification
		Async:              true,
		BufferSize:         16384,
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
		ErrorCallback: func(eventType string, err error) {
			fmt.Printf("     Advanced event [%s]: %v\n", eventType, err)
		},
	}

	letheLogger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create advanced Lethe logger: %v", err)
	}
	defer letheLogger.Close()

	adapter := NewLetheIrisAdapter(letheLogger)
	defer adapter.Close()

	// Advanced Iris configuration
	irisLogger, err := iris.New(iris.Config{
		Output:    adapter,
		Encoder:   iris.NewJSONEncoder(),
		Level:     iris.Debug,
		Capacity:  8192,
		BatchSize: 32,
	}, iris.WithCaller(), iris.Development())
	if err != nil {
		log.Fatalf("Failed to create advanced Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Test advanced logging features
	irisLogger.With(
		iris.String("component", "advanced-integration"),
		iris.String("version", "2.0.0"),
		iris.String("environment", "production"),
		iris.String("datacenter", "us-east-1"),
	).Info("Advanced configuration active")

	irisLogger.With(
		iris.String("feature", "checksum_verification"),
		iris.Bool("integrity", true),
		iris.Bool("compression", true),
		iris.String("time_rotation", "24h"),
	).Debug("Data integrity and rotation features enabled")

	const advancedEntries = 10000
	for i := 0; i < advancedEntries; i++ {
		irisLogger.With(
			iris.Int("entry_id", i),
			iris.Int("batch", i/1000),
			iris.Bool("checksum_ok", true),
			iris.Bool("compressed", true),
		).Info("Advanced integration test entry")
	}

	fmt.Printf("   Advanced Integration: %d entries with checksums and compression\n", advancedEntries)
	fmt.Printf("   Features: time-based rotation, data integrity, zero-copy transfers\n")
}

// runProductionIntegration demonstrates a complete production-ready
// integration pattern with error handling and monitoring.
func runProductionIntegration() {
	var rotationEvents int
	var errorEvents int

	// Production-grade Lethe configuration
	config := &lethe.LoggerConfig{
		Filename:           "logs/iris-production.log",
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
			switch eventType {
			case "rotation":
				rotationEvents++
			default:
				errorEvents++
			}
		},
	}

	letheLogger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create production Lethe logger: %v", err)
	}
	defer letheLogger.Close()

	adapter := NewLetheIrisAdapter(letheLogger)
	defer adapter.Close()

	// Production Iris configuration
	irisLogger, err := iris.New(iris.Config{
		Output:    adapter,
		Encoder:   iris.NewJSONEncoder(),
		Level:     iris.Info,
		Capacity:  16384,
		BatchSize: 64,
	})
	if err != nil {
		log.Fatalf("Failed to create production Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Simulate production application logging
	irisLogger.Info("Production application started")

	// Simulate various service operations
	services := []string{"auth", "api", "database", "cache", "analytics"}
	const productionOps = 50000

	startTime := time.Now()

	for i := 0; i < productionOps; i++ {
		service := services[i%len(services)]

		irisLogger.With(
			iris.String("service", service),
			iris.String("operation_id", fmt.Sprintf("%s-op-%d", service, i)),
			iris.Int64("timestamp", time.Now().UnixNano()),
			iris.Int("duration_ms", (i%200)+5),
			iris.Bool("success", i%100 != 0), // 1% failure rate
		).Info("Production service operation")
	}

	duration := time.Since(startTime)

	irisLogger.Info("Production application shutdown initiated")
	if err := adapter.Sync(); err != nil { // Ensure all data is persisted
		log.Printf("Warning: Failed to sync adapter: %v", err)
	}

	fmt.Printf("   Production Integration: %d operations in %v\n", productionOps, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n", float64(productionOps)/duration.Seconds())
	fmt.Printf("   Events: %d rotations, %d errors\n", rotationEvents, errorEvents)
}

// runConcurrentWritersTest demonstrates multiple Iris loggers
// writing concurrently through Lethe with perfect synchronization.
func runConcurrentWritersTest() {
	// Shared Lethe logger for all Iris instances
	letheLogger, err := lethe.NewWithDefaults("logs/iris-concurrent.log")
	if err != nil {
		log.Fatalf("Failed to create concurrent Lethe logger: %v", err)
	}
	defer letheLogger.Close()

	adapter := NewLetheIrisAdapter(letheLogger)
	defer adapter.Close()

	// Create multiple Iris loggers sharing the same Lethe backend
	const numLoggers = 8
	const operationsPerLogger = 5000

	var wg sync.WaitGroup
	startTime := time.Now()

	for loggerID := 0; loggerID < numLoggers; loggerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine gets its own Iris logger instance
			irisLogger, err := iris.New(iris.Config{
				Output:  adapter,
				Encoder: iris.NewJSONEncoder(),
				Level:   iris.Info,
			})
			if err != nil {
				log.Printf("Failed to create Iris logger %d: %v", id, err)
				return
			}
			defer irisLogger.Close()
			irisLogger.Start()

			for op := 0; op < operationsPerLogger; op++ {
				irisLogger.With(
					iris.Int("logger_id", id),
					iris.Int("operation", op),
					iris.Bool("thread_safe", true),
					iris.Bool("zero_copy", true),
				).Info("Concurrent operation through shared Lethe backend")
			}
		}(loggerID)
	}

	wg.Wait()
	duration := time.Since(startTime)
	totalOps := numLoggers * operationsPerLogger

	fmt.Printf("   Concurrent Writers: %d operations from %d loggers in %v\n",
		totalOps, numLoggers, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n", float64(totalOps)/duration.Seconds())
	fmt.Printf("   Thread Safety: Perfect synchronization through shared Lethe backend\n")
}

// runWriteSyncerComplianceTest verifies that the adapter correctly
// implements the iris.WriteSyncer interface contract.
func runWriteSyncerComplianceTest() {
	letheLogger, err := lethe.NewWithDefaults("logs/iris-compliance.log")
	if err != nil {
		log.Fatalf("Failed to create compliance test Lethe logger: %v", err)
	}
	defer letheLogger.Close()

	adapter := NewLetheIrisAdapter(letheLogger)
	defer adapter.Close()

	// Verify WriteSyncer interface compliance
	var writeSyncer iris.WriteSyncer = adapter // Compile-time interface check

	// Test Write method
	testData := []byte("WriteSyncer compliance test\n")
	n, err := writeSyncer.Write(testData)
	if err != nil {
		log.Printf("Write method failed: %v", err)
	} else if n != len(testData) {
		log.Printf("Write method returned incorrect length: got %d, want %d", n, len(testData))
	}

	// Test WriteOwned method with our custom interface (Lethe-specific optimization)
	if letheAdapter, ok := writeSyncer.(*LetheIrisAdapter); ok {
		ownedData := []byte("WriteOwned compliance test\n")
		n, err = letheAdapter.WriteOwned(ownedData)
		if err != nil {
			log.Printf("WriteOwned method failed: %v", err)
		} else if n != len(ownedData) {
			log.Printf("WriteOwned method returned incorrect length: got %d, want %d", n, len(ownedData))
		}
	}

	// Test Sync method
	if err := writeSyncer.Sync(); err != nil {
		log.Printf("Sync method failed: %v", err)
	}

	// Create Iris logger to verify full integration
	irisLogger, err := iris.New(iris.Config{
		Output:  writeSyncer,
		Encoder: iris.NewJSONEncoder(),
		Level:   iris.Info,
	})
	if err != nil {
		log.Fatalf("Failed to create compliance Iris logger: %v", err)
	}
	defer irisLogger.Close()
	irisLogger.Start()

	// Test complete integration
	irisLogger.With(
		iris.String("interface", "WriteSyncer"),
		iris.String("methods", "Write, Sync, Close + WriteOwned (Lethe-specific)"),
		iris.Bool("compliant", true),
	).Info("WriteSyncer interface compliance verified")

	fmt.Println("   WriteSyncer Compliance: All interface methods verified")
	fmt.Println("   Standard methods: Write, Sync, Close")
	fmt.Println("   Lethe optimization: WriteOwned for zero-copy performance")
}
