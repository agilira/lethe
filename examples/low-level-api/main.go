// Package main demonstrates advanced Lethe log rotation features.
//
// This example covers high-performance capabilities including MPSC buffers,
// zero-copy operations, backpressure policies, and advanced configuration
// options for demanding production environments.
//
// Run with: go run .

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/agilira/lethe"
)

func main() {
	fmt.Println("Lethe Advanced Examples")
	fmt.Println("======================")

	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0750); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Example 1: MPSC (Multi-Producer Single-Consumer) Buffer Operations
	fmt.Println("\n1. MPSC Buffer Performance Test")
	runMPSCBufferTest()

	// Example 2: Zero-Copy Operations with WriteOwned
	fmt.Println("\n2. Zero-Copy WriteOwned Operations")
	runZeroCopyOperations()

	// Example 3: Backpressure Policy Testing
	fmt.Println("\n3. Backpressure Policy Demonstration")
	runBackpressurePolicyTest()

	// Example 4: High-Throughput Concurrent Logging
	fmt.Println("\n4. High-Throughput Concurrent Logging")
	runConcurrentLoggingTest()

	// Example 5: Advanced Configuration Features
	fmt.Println("\n5. Advanced Configuration Features")
	runAdvancedConfigFeatures()

	// Example 6: Error Handling and Recovery
	fmt.Println("\n6. Error Handling and Recovery")
	runErrorHandlingTest()

	// Example 7: Performance Benchmarking
	fmt.Println("\n7. Performance Benchmarking")
	runPerformanceBenchmark()

	fmt.Println("\nAll advanced examples completed successfully.")
	fmt.Println("Check the ./logs/ directory for generated log files.")
}

// runMPSCBufferTest demonstrates the Multi-Producer Single-Consumer buffer
// capabilities that enable high-performance concurrent logging.
func runMPSCBufferTest() {
	config := &lethe.LoggerConfig{
		Filename:           "logs/mpsc-test.log",
		MaxSizeStr:         "50MB",
		MaxBackups:         5,
		Async:              true,
		BufferSize:         16384, // 16KB buffer for high throughput
		BackpressurePolicy: "adaptive",
		ErrorCallback: func(eventType string, err error) {
			log.Printf("MPSC event [%s]: %v", eventType, err)
		},
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create MPSC logger: %v", err)
	}
	defer logger.Close()

	// Simulate multiple producers writing concurrently
	const numProducers = 10
	const entriesPerProducer = 1000

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()

			for j := 0; j < entriesPerProducer; j++ {
				entry := fmt.Sprintf("Producer-%d Entry-%d: Processing batch operation with data payload\n",
					producerID, j)

				if _, err := logger.Write([]byte(entry)); err != nil {
					log.Printf("Producer %d failed to write entry %d: %v", producerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)
	totalEntries := numProducers * entriesPerProducer

	fmt.Printf("   MPSC Test: %d entries from %d producers in %v\n",
		totalEntries, numProducers, duration)
	fmt.Printf("   Throughput: %.0f entries/sec\n",
		float64(totalEntries)/duration.Seconds())
	fmt.Printf("   Configuration: 16KB buffer, adaptive backpressure\n")
}

// runZeroCopyOperations demonstrates WriteOwned for zero-copy performance.
// This is critical for high-frequency logging where allocation overhead matters.
func runZeroCopyOperations() {
	config := &lethe.LoggerConfig{
		Filename:           "logs/zero-copy.log",
		MaxSizeStr:         "25MB",
		MaxBackups:         3,
		Async:              true,
		BufferSize:         8192,
		BackpressurePolicy: "adaptive",
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create zero-copy logger: %v", err)
	}
	defer logger.Close()

	// Pre-allocate buffers for zero-copy operations
	const numBuffers = 1000
	buffers := make([][]byte, numBuffers)

	// Initialize buffers with structured log data
	for i := range buffers {
		logData := fmt.Sprintf(`{"timestamp":"%s","level":"info","producer_id":%d,"message":"Zero-copy operation test","data_size":256}%c`,
			time.Now().Format(time.RFC3339), i, '\n')
		buffers[i] = []byte(logData)
	}

	startTime := time.Now()

	// Use WriteOwned for zero-copy transfer
	for i, buffer := range buffers {
		// WriteOwned takes ownership of the buffer, enabling zero-copy operation
		if _, err := logger.WriteOwned(buffer); err != nil {
			log.Printf("Failed zero-copy write %d: %v", i, err)
		}
	}

	duration := time.Since(startTime)

	fmt.Printf("   Zero-Copy Test: %d WriteOwned operations in %v\n",
		numBuffers, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n",
		float64(numBuffers)/duration.Seconds())
	fmt.Printf("   Benefits: No buffer copying, reduced GC pressure\n")
}

// runBackpressurePolicyTest demonstrates different backpressure handling strategies
// for managing overflow conditions in high-load scenarios.
func runBackpressurePolicyTest() {
	policies := []string{"adaptive", "drop", "block"}

	for _, policy := range policies {
		fmt.Printf("   Testing backpressure policy: %s\n", policy)

		config := &lethe.LoggerConfig{
			Filename:           fmt.Sprintf("logs/backpressure-%s.log", policy),
			MaxSizeStr:         "10MB",
			MaxBackups:         2,
			Async:              true,
			BufferSize:         1024, // Small buffer to trigger backpressure
			BackpressurePolicy: policy,
			ErrorCallback: func(eventType string, err error) {
				if eventType == "backpressure" {
					fmt.Printf("     Backpressure triggered: %v\n", err)
				}
			},
		}

		logger, err := lethe.NewWithConfig(config)
		if err != nil {
			log.Printf("Failed to create %s logger: %v", policy, err)
			continue
		}

		// Generate high-volume traffic to trigger backpressure
		const burstSize = 5000
		startTime := time.Now()
		successCount := 0

		for i := 0; i < burstSize; i++ {
			entry := fmt.Sprintf("Backpressure test entry %d with policy %s - some payload data\n",
				i, policy)

			if _, err := logger.Write([]byte(entry)); err != nil {
				// Count failures for analysis
				continue
			}
			successCount++
		}

		duration := time.Since(startTime)
		if err := logger.Close(); err != nil {
			log.Printf("Warning: Failed to close logger: %v", err)
		}

		fmt.Printf("     Policy %s: %d/%d successful writes in %v\n",
			policy, successCount, burstSize, duration)
		fmt.Printf("     Behavior: %s overflow handling\n", policy)
	}
}

// runConcurrentLoggingTest simulates realistic concurrent logging scenarios
// with multiple goroutines writing different types of log data.
func runConcurrentLoggingTest() {
	config := &lethe.LoggerConfig{
		Filename:           "logs/concurrent.log",
		MaxSizeStr:         "100MB",
		MaxBackups:         10,
		Async:              true,
		BufferSize:         32768, // 32KB buffer for high concurrency
		BackpressurePolicy: "adaptive",
		Compress:           true,
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create concurrent logger: %v", err)
	}
	defer logger.Close()

	numWorkers := runtime.NumCPU() * 2
	const operationsPerWorker = 2000

	var wg sync.WaitGroup
	startTime := time.Now()

	// Start multiple worker types
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerType := []string{"API", "DB", "CACHE", "AUTH"}[workerID%4]

			for j := 0; j < operationsPerWorker; j++ {
				var entry string
				switch workerType {
				case "API":
					entry = fmt.Sprintf(`{"type":"api","worker":%d,"op":%d,"method":"POST","path":"/api/users","status":200,"duration_ms":45}%c`,
						workerID, j, '\n')
				case "DB":
					entry = fmt.Sprintf(`{"type":"database","worker":%d,"op":%d,"query":"SELECT * FROM users","rows":150,"duration_ms":12}%c`,
						workerID, j, '\n')
				case "CACHE":
					entry = fmt.Sprintf(`{"type":"cache","worker":%d,"op":%d,"key":"user:session:xyz","action":"hit","duration_ms":1}%c`,
						workerID, j, '\n')
				case "AUTH":
					entry = fmt.Sprintf(`{"type":"auth","worker":%d,"op":%d,"user_id":12345,"action":"login","success":true}%c`,
						workerID, j, '\n')
				}

				if _, err := logger.Write([]byte(entry)); err != nil {
					log.Printf("Worker %d failed operation %d: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)
	totalOps := numWorkers * operationsPerWorker

	fmt.Printf("   Concurrent Test: %d operations from %d workers in %v\n",
		totalOps, numWorkers, duration)
	fmt.Printf("   Throughput: %.0f ops/sec\n",
		float64(totalOps)/duration.Seconds())
	fmt.Printf("   Configuration: %d workers, 32KB buffer, compression enabled\n", numWorkers)
}

// runAdvancedConfigFeatures demonstrates sophisticated configuration options
// including checksums, error callbacks, and rotation policies.
func runAdvancedConfigFeatures() {
	var checksumErrors int
	var rotationEvents int

	config := &lethe.LoggerConfig{
		Filename:           "logs/advanced-config.log",
		MaxSizeStr:         "5MB", // Small size to trigger rotation
		MaxAgeStr:          "1h",  // Time-based rotation
		MaxBackups:         3,
		Compress:           true,
		Checksum:           true, // Enable data integrity verification
		Async:              true,
		BufferSize:         4096,
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
		ErrorCallback: func(eventType string, err error) {
			switch eventType {
			case "checksum":
				checksumErrors++
				fmt.Printf("     Checksum verification failed: %v\n", err)
			case "rotation":
				rotationEvents++
				fmt.Printf("     Log rotation event: %v\n", err)
			case "compression":
				fmt.Printf("     Compression event: %v\n", err)
			default:
				fmt.Printf("     Advanced config event [%s]: %v\n", eventType, err)
			}
		},
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create advanced config logger: %v", err)
	}
	defer logger.Close()

	// Generate data to demonstrate advanced features
	const numEntries = 10000
	for i := 0; i < numEntries; i++ {
		entry := fmt.Sprintf(`{"id":%d,"timestamp":"%s","level":"info","module":"advanced","message":"Testing advanced configuration features with checksum verification and compression","data":{"counter":%d,"checksum_enabled":true,"compression_enabled":true}}%c`,
			i, time.Now().Format(time.RFC3339), i, '\n')

		if _, err := logger.Write([]byte(entry)); err != nil {
			log.Printf("Failed to write advanced entry %d: %v", i, err)
		}

		// Add small delay to demonstrate time-based rotation
		if i%1000 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	fmt.Printf("   Advanced Config: %d entries written\n", numEntries)
	fmt.Printf("   Features: checksums (%d errors), compression, time-based rotation (%d events)\n",
		checksumErrors, rotationEvents)
	fmt.Printf("   Error callback: real-time monitoring of rotation and integrity events\n")
}

// runErrorHandlingTest demonstrates robust error handling and recovery mechanisms.
func runErrorHandlingTest() {
	var errorEvents []string

	config := &lethe.LoggerConfig{
		Filename:           "logs/error-handling.log",
		MaxSizeStr:         "20MB",
		MaxBackups:         5,
		Async:              true,
		BufferSize:         2048,
		BackpressurePolicy: "adaptive",
		ErrorCallback: func(eventType string, err error) {
			errorEvents = append(errorEvents, fmt.Sprintf("%s: %v", eventType, err))
		},
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create error handling logger: %v", err)
	}

	// Test normal operations
	normalEntries := 1000
	for i := 0; i < normalEntries; i++ {
		entry := fmt.Sprintf("Normal operation %d: All systems operational\n", i)
		if _, err := logger.Write([]byte(entry)); err != nil {
			log.Printf("Warning: Write failed: %v", err)
		}
	}

	// Simulate error conditions and recovery
	fmt.Printf("   Testing error handling and recovery mechanisms\n")

	// Test graceful close
	if err := logger.Close(); err != nil {
		log.Printf("Error during close: %v", err)
	}

	// Test write after close (should handle gracefully)
	_, err = logger.Write([]byte("Write after close test\n"))
	if err != nil {
		fmt.Printf("     Expected error after close: %v\n", err)
	}

	fmt.Printf("   Error Handling Test: %d normal operations completed\n", normalEntries)
	fmt.Printf("   Error events captured: %d\n", len(errorEvents))
	for i, event := range errorEvents {
		if i < 3 { // Show first few events
			fmt.Printf("     Event %d: %s\n", i+1, event)
		}
	}
}

// runPerformanceBenchmark provides comprehensive performance metrics
// for evaluating Lethe's capabilities under various conditions.
func runPerformanceBenchmark() {
	benchmarks := []struct {
		name       string
		bufferSize int
		async      bool
		compress   bool
	}{
		{"Sync Small Buffer", 1024, false, false},
		{"Async Small Buffer", 1024, true, false},
		{"Async Large Buffer", 32768, true, false},
		{"Async Large Buffer + Compression", 32768, true, true},
	}

	for _, benchmark := range benchmarks {
		fmt.Printf("   Running benchmark: %s\n", benchmark.name)

		config := &lethe.LoggerConfig{
			Filename: fmt.Sprintf("logs/bench-%s.log",
				fmt.Sprintf("%s", benchmark.name)),
			MaxSizeStr:         "50MB",
			MaxBackups:         3,
			Async:              benchmark.async,
			BufferSize:         benchmark.bufferSize,
			Compress:           benchmark.compress,
			BackpressurePolicy: "adaptive",
		}

		logger, err := lethe.NewWithConfig(config)
		if err != nil {
			log.Printf("Failed to create benchmark logger: %v", err)
			continue
		}

		const benchmarkEntries = 50000
		entry := []byte("Benchmark entry with standard log data payload for performance testing\n")

		startTime := time.Now()
		for i := 0; i < benchmarkEntries; i++ {
			if _, err := logger.Write(entry); err != nil {
				log.Printf("Warning: Write failed: %v", err)
			}
		}
		duration := time.Since(startTime)

		if err := logger.Close(); err != nil {
			log.Printf("Warning: Failed to close logger: %v", err)
		}

		throughput := float64(benchmarkEntries) / duration.Seconds()
		fmt.Printf("     %s: %.0f entries/sec (%v total)\n",
			benchmark.name, throughput, duration)
		fmt.Printf("     Config: %dKB buffer, async=%t, compress=%t\n",
			benchmark.bufferSize/1024, benchmark.async, benchmark.compress)
	}
}
