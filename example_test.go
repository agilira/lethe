// example_test.go: Executable examples for godoc
//
// These examples appear in the generated documentation and are executable.
// Run with: go test -run Example

package lethe_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/agilira/lethe"
)

// ExampleNewWithDefaults demonstrates the recommended way to create a production logger.
func ExampleNewWithDefaults() {
	// Create logger with production defaults
	logger, err := lethe.NewWithDefaults("app.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write to the logger
	if _, err := logger.Write([]byte("Application started\n")); err != nil {
		log.Printf("Warning: failed to write application started: %v", err)
	}
	if _, err := logger.Write([]byte("Processing request\n")); err != nil {
		log.Printf("Warning: failed to write processing request: %v", err)
	}

	fmt.Println("Logger created with production defaults")
	// Output: Logger created with production defaults
}

// ExampleNewSimple demonstrates modern string-based configuration.
func ExampleNewSimple() {
	// Create logger with string-based size configuration
	logger, err := lethe.NewSimple("simple.log", "50MB", 3)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	if _, err := logger.Write([]byte("Using string-based configuration\n")); err != nil {
		log.Printf("Warning: failed to write string-based config: %v", err)
	}

	fmt.Println("Logger created with string-based configuration")
	// Output: Logger created with string-based configuration
}

// ExampleNewDaily demonstrates daily log rotation.
func ExampleNewDaily() {
	// Create logger that rotates daily
	logger, err := lethe.NewDaily("daily.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	if _, err := logger.Write([]byte("Daily rotation enabled\n")); err != nil {
		log.Printf("Warning: failed to write daily rotation: %v", err)
	}

	fmt.Println("Logger created with daily rotation")
	// Output: Logger created with daily rotation
}

// ExampleNewDevelopment demonstrates development-optimized logging.
func ExampleNewDevelopment() {
	// Create logger optimized for development
	logger, err := lethe.NewDevelopment("debug.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	if _, err := logger.Write([]byte("Debug message\n")); err != nil {
		log.Printf("Warning: failed to write debug message: %v", err)
	}

	fmt.Println("Logger created for development")
	// Output: Logger created for development
}

// ExampleNewWithConfig demonstrates full configuration control.
func ExampleNewWithConfig() {
	// Create logger with custom configuration
	config := &lethe.LoggerConfig{
		Filename:           "custom.log",
		MaxSizeStr:         "100MB",
		MaxAgeStr:          "7d",
		MaxBackups:         5,
		Compress:           true,
		Async:              true,
		BackpressurePolicy: "adaptive",
		ErrorCallback: func(eventType string, err error) {
			fmt.Printf("Error in %s: %v\n", eventType, err)
		},
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	if _, err := logger.Write([]byte("Custom configuration\n")); err != nil {
		log.Printf("Warning: failed to write custom configuration: %v", err)
	}

	fmt.Println("Logger created with custom configuration")
	// Output: Logger created with custom configuration
}

// ExampleLogger_Write demonstrates basic writing to the logger.
func ExampleLogger_Write() {
	logger, err := lethe.NewWithDefaults("write_example.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write various types of data
	n, err := logger.Write([]byte("Hello, World!\n"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Wrote %d bytes\n", n)
	// Output: Wrote 14 bytes
}

// ExampleLogger_WriteOwned demonstrates zero-copy writing.
func ExampleLogger_WriteOwned() {
	logger, err := lethe.NewWithDefaults("owned_example.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Create buffer for zero-copy write
	message := "Zero-copy message\n"
	buf := make([]byte, len(message))
	copy(buf, message)

	// Transfer ownership to logger
	n, err := logger.WriteOwned(buf)
	if err != nil {
		log.Fatal(err)
	}
	// buf must not be used after this point

	fmt.Printf("Wrote %d bytes with zero-copy\n", n)
	// Output: Wrote 18 bytes with zero-copy
}

// ExampleLogger_Stats demonstrates performance monitoring.
func ExampleLogger_Stats() {
	// Use synchronous mode for predictable byte counting
	config := &lethe.LoggerConfig{
		Filename: "stats_example.log",
		Async:    false, // Synchronous for predictable stats
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write some data
	for i := 0; i < 10; i++ {
		if _, err := logger.Write([]byte(fmt.Sprintf("Message %d\n", i))); err != nil {
			log.Printf("Warning: failed to write message %d: %v", i, err)
		}
	}

	// Get performance statistics
	stats := logger.Stats()
	fmt.Printf("Write count: %d\n", stats.WriteCount)
	fmt.Printf("Async mode: %t\n", stats.IsMPSCActive)
	if stats.WriteCount > 0 {
		fmt.Println("Statistics collected")
	}
	// Output: Write count: 10
	// Async mode: false
	// Statistics collected
}

// ExampleLogger_Rotate demonstrates manual rotation.
func ExampleLogger_Rotate() {
	logger, err := lethe.NewWithDefaults("rotate_example.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write some data
	if _, err := logger.Write([]byte("Before rotation\n")); err != nil {
		log.Printf("Warning: failed to write before rotation: %v", err)
	}

	// Force rotation
	err = logger.Rotate()
	if err != nil {
		log.Fatal(err)
	}

	// Write to new file
	if _, err := logger.Write([]byte("After rotation\n")); err != nil {
		log.Printf("Warning: failed to write after rotation: %v", err)
	}

	fmt.Println("Manual rotation completed")
	// Output: Manual rotation completed
}

// Example_standardLibrary demonstrates integration with Go's standard library.
func Example_standardLibrary() {
	logger, err := lethe.NewWithDefaults("stdlib_example.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Redirect standard library logging
	originalOutput := log.Writer()
	log.SetOutput(logger)
	defer log.SetOutput(originalOutput)

	// Use standard library logging
	log.Println("This goes through lethe")
	log.Printf("Formatted message: %d", 42)

	fmt.Println("Standard library integration")
	// Output: Standard library integration
}

// Example_errorHandling demonstrates error callback usage.
func Example_errorHandling() {
	errorCount := 0

	config := &lethe.LoggerConfig{
		Filename: "error_example.log",
		ErrorCallback: func(eventType string, err error) {
			errorCount++
			fmt.Printf("Error type: %s\n", eventType)
		},
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	if _, err := logger.Write([]byte("Normal operation\n")); err != nil {
		log.Printf("Warning: failed to write normal operation: %v", err)
	}

	fmt.Printf("Errors handled: %d\n", errorCount)
	// Output: Errors handled: 0
}

// Example_asyncMode demonstrates high-performance async mode.
func Example_asyncMode() {
	config := &lethe.LoggerConfig{
		Filename:           "async_example.log",
		MaxSizeStr:         "10MB",
		Async:              true,
		BufferSize:         4096,
		BackpressurePolicy: "adaptive",
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// High-throughput writes
	for i := 0; i < 1000; i++ {
		if _, err := logger.Write([]byte(fmt.Sprintf("High-throughput message %d\n", i))); err != nil {
			log.Printf("Warning: failed to write high-throughput message %d: %v", i, err)
		}
	}

	stats := logger.Stats()
	fmt.Printf("Buffer active: %t\n", stats.IsMPSCActive)
	// Output: Buffer active: true
}

// Example_compression demonstrates log compression.
func Example_compression() {
	config := &lethe.LoggerConfig{
		Filename:   "compressed_example.log",
		MaxSizeStr: "1KB", // Small size to trigger rotation
		MaxBackups: 2,
		Compress:   true,
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write enough data to trigger rotation
	for i := 0; i < 100; i++ {
		if _, err := logger.Write([]byte(fmt.Sprintf("Log entry %d with enough content to trigger rotation\n", i))); err != nil {
			log.Printf("Warning: failed to write log entry %d: %v", i, err)
		}
	}

	// Wait for background compression
	logger.WaitForBackgroundTasks()

	fmt.Println("Compression enabled")
	// Output: Compression enabled
}

// Example_cleanup demonstrates cleanup of this example's files.
func Example_cleanup() {
	// Clean up example files (in real usage, don't delete your logs!)
	files := []string{
		"app.log", "simple.log", "daily.log", "debug.log", "custom.log",
		"write_example.log", "owned_example.log", "stats_example.log",
		"rotate_example.log", "stdlib_example.log", "error_example.log",
		"async_example.log", "compressed_example.log",
	}

	for _, file := range files {
		os.Remove(file)
		// Also remove potential backup files
		matches, _ := filepath.Glob(file + ".*")
		for _, match := range matches {
			os.Remove(match)
		}
	}

	fmt.Println("Example files cleaned up")
	// Output: Example files cleaned up
}

// Example showing time-based rotation with MaxAgeStr
func Example_timeBasedRotation() {
	config := &lethe.LoggerConfig{
		Filename:   "time_rotation.log",
		MaxSizeStr: "10MB", // Large enough to not trigger size rotation
		MaxAgeStr:  "1s",   // Rotate every second for demo
		MaxBackups: 3,
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write initial message
	if _, err := logger.Write([]byte("Message before rotation\n")); err != nil {
		log.Printf("Warning: failed to write message before rotation: %v", err)
	}

	// Wait for time-based rotation
	time.Sleep(1100 * time.Millisecond)

	// This should trigger rotation due to age
	if _, err := logger.Write([]byte("Message after time rotation\n")); err != nil {
		log.Printf("Warning: failed to write message after time rotation: %v", err)
	}

	fmt.Println("Time-based rotation configured")
	// Output: Time-based rotation configured
}
