// Basic Lethe Integration Examples
// Demonstrates core integration patterns without external dependencies
// Copyright 2025 AGILira
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agilira/lethe"
)

func main() {
	fmt.Println("Lethe Framework Integration Examples")
	fmt.Println("=======================================")

	// Run all examples
	exampleStandardLibrary()
	fmt.Println()

	exampleHighPerformance()
	fmt.Println()

	exampleProfessionalFeatures()
	fmt.Println()

	exampleZeroCopy()
	fmt.Println()

	exampleUniversalWrapper()
	fmt.Println()

	fmt.Println("✅ All examples completed successfully!")
	fmt.Println("📁 Check the generated .log files for output")
	fmt.Println("⚡ Performance boost: Lethe automatically uses go-timecache for 10x faster time operations")

	// Clean up example files after demonstration
	cleanupExampleFiles()
}

// Example 1: Standard Library Integration
func exampleStandardLibrary() {
	fmt.Println("=== 📝 Standard Library Integration ===")

	// Create Lethe rotator
	rotator := &lethe.Logger{
		Filename:   "examples/app.log",
		MaxSizeStr: "10MB",
		MaxBackups: 5,
		MaxAge:     7 * 24 * time.Hour, // 7 days
		Compress:   true,
		LocalTime:  true,
	}
	defer rotator.Close()

	// Set as output for standard library
	originalOutput := log.Writer()
	log.SetOutput(rotator)
	defer log.SetOutput(originalOutput) // Restore

	// Use standard logging as usual
	log.Println("Application started")
	log.Printf("User %s logged in", "john_doe")
	log.Printf("Processing order %d", 12345)
	log.Println("✅ Standard library integration working")

	fmt.Println("✅ Standard library integration complete")
}

// Example 2: High-Performance Async Integration
func exampleHighPerformance() {
	fmt.Println("=== High-Performance Async Integration ===")

	// Create high-performance Lethe rotator
	rotator := &lethe.Logger{
		Filename:           "examples/high_perf.log",
		MaxSizeStr:         "50MB",
		MaxBackups:         10,
		Compress:           true,
		Async:              true,                   // Enable MPSC mode
		BufferSize:         4096,                   // Large buffer
		BackpressurePolicy: "adaptive",             // Adaptive buffer resizing
		FlushInterval:      500 * time.Microsecond, // Fast flush
		AdaptiveFlush:      true,                   // Adaptive timing
	}
	defer rotator.Close()

	// Simulate high-throughput logging
	start := time.Now()
	for i := 0; i < 1000; i++ {
		logEntry := fmt.Sprintf(`{"timestamp":"%s","level":"info","message":"High throughput log entry %d","user_id":%d}`,
			time.Now().Format(time.RFC3339), i, i%100)
		_, _ = rotator.Write([]byte(logEntry + "\n")) // Ignore errors in example
	}
	duration := time.Since(start)

	// Allow MPSC to flush
	time.Sleep(50 * time.Millisecond)

	// Get performance stats
	stats := rotator.Stats()
	fmt.Printf("Performance stats: WriteCount=%d, AvgLatency=%dns, BufferFill=%d\n",
		stats.WriteCount, stats.AvgLatencyNs, stats.BufferFill)
	fmt.Printf("Wrote 1000 entries in %v (%.2f entries/ms)\n",
		duration, float64(1000)/float64(duration.Milliseconds()))

	fmt.Println("✅ High-performance integration complete")
}

// Example 3: Professional Features Integration
func exampleProfessionalFeatures() {
	fmt.Println("=== Professional Features Integration ===")

	// Create professional-grade rotator
	rotator := &lethe.Logger{
		Filename:           "examples/professional.log",
		MaxSizeStr:         "100MB",
		MaxBackups:         20,
		MaxFileAge:         30 * 24 * time.Hour, // 30 days TTL
		Compress:           true,
		Checksum:           true, // SHA-256 checksums
		LocalTime:          true,
		Async:              true,
		BackpressurePolicy: "drop", // Drop on overflow
		ErrorCallback: func(op string, err error) {
			// Custom error handling
			fmt.Printf("Lethe error [%s]: %v\n", op, err)
		},
	}
	defer rotator.Close()

	// Log pro events
	events := []string{
		`{"level":"info","event":"system_start","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`,
		`{"level":"warn","event":"high_memory_usage","memory_percent":85,"timestamp":"` + time.Now().Format(time.RFC3339) + `"}`,
		`{"level":"error","event":"database_connection_failed","retries":3,"timestamp":"` + time.Now().Format(time.RFC3339) + `"}`,
		`{"level":"info","event":"user_login","user_id":"admin","ip":"192.168.1.10","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`,
	}

	for _, event := range events {
		_, _ = rotator.Write([]byte(event + "\n")) // Ignore errors in example
	}

	fmt.Println("Professional features (checksums, TTL, error callbacks) active")
	fmt.Println("✅ Professional features integration complete")
}

// Example 4: Zero-Copy Integration with Iris
func exampleZeroCopy() {
	fmt.Println("=== Zero-Copy Integration with Iris ===")

	rotator := &lethe.Logger{
		Filename:   "examples/zero_copy.log",
		MaxSizeStr: "10MB",
		Async:      true,
		BufferSize: 2048,
	}
	defer rotator.Close()

	// Simulate high-performance framework that can transfer ownership
	start := time.Now()
	for i := 0; i < 100; i++ {
		// Create log entry buffer (this would come from Iris)
		logData := fmt.Sprintf("Zero-copy log entry %d with substantial content for testing - timestamp: %d\n",
			i, time.Now().UnixNano())
		buffer := make([]byte, len(logData))
		copy(buffer, logData)

		// Transfer ownership to Lethe (zero-copy)
		_, _ = rotator.WriteOwned(buffer) // Ignore errors in example
		// Note: buffer must not be reused after WriteOwned call
	}
	duration := time.Since(start)

	fmt.Printf("Zero-copy writes: 100 entries in %v\n", duration)
	fmt.Println("Perfect for Iris integration")
	fmt.Println("✅ Zero-copy integration complete")
}

// Example 5: Framework-Agnostic Wrapper

// UniversalLogger provides a framework-agnostic logging interface with Lethe rotation
type UniversalLogger struct {
	rotator *lethe.Logger
	prefix  string
}

// NewUniversalLogger creates a new UniversalLogger instance with the specified filename
func NewUniversalLogger(filename string) *UniversalLogger {
	return &UniversalLogger{
		rotator: &lethe.Logger{
			Filename:   filename,
			MaxSizeStr: "50MB",
			MaxBackups: 10,
			Compress:   true,
			Async:      true,
		},
		prefix: "[APP]",
	}
}

// Info logs an info-level message
func (ul *UniversalLogger) Info(message string) {
	ul.log("INFO", message)
}

// Warn logs a warning-level message
func (ul *UniversalLogger) Warn(message string) {
	ul.log("WARN", message)
}

func (ul *UniversalLogger) Error(message string) {
	ul.log("ERROR", message)
}

func (ul *UniversalLogger) log(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s %s %s: %s\n", timestamp, ul.prefix, level, message)
	_, _ = ul.rotator.Write([]byte(logLine)) // Ignore errors in example
}

// Stats returns the current statistics from the underlying rotator
func (ul *UniversalLogger) Stats() lethe.Stats {
	return ul.rotator.Stats()
}

// Close closes the underlying rotator and flushes any pending data
func (ul *UniversalLogger) Close() {
	_ = ul.rotator.Close() // Ignore errors in example
}

func exampleUniversalWrapper() {
	fmt.Println("=== 🔧 Universal Logger Wrapper ===")

	logger := NewUniversalLogger("examples/universal.log")
	defer logger.Close()

	logger.Info("Application initialized")
	logger.Warn("⚠️  Configuration file not found, using defaults")
	logger.Error("❌ Failed to connect to database")
	logger.Info("✅ Fallback database connected successfully")

	// Show stats
	stats := logger.Stats()
	fmt.Printf("Logger stats: %d writes, %d bytes\n", stats.WriteCount, stats.TotalBytes)

	fmt.Println("✅ Universal wrapper integration complete")
}

func cleanupExampleFiles() {
	files := []string{
		"examples/app.log",
		"examples/high_perf.log",
		"examples/professional.log",
		"examples/zero_copy.log",
		"examples/universal.log",
	}
	for _, file := range files {
		_ = os.Remove(file) // Ignore errors in cleanup
	}
}
