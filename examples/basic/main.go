// Package main demonstrates basic Lethe log rotation usage patterns.
//
// This example covers the fundamental constructor functions and configuration
// options that form the foundation of Lethe's log rotation capabilities.
//
// Run with: go run .

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/agilira/lethe"
)

func main() {
	fmt.Println("Lethe Basic Examples")
	fmt.Println("===================")

	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0750); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Example 1: Production defaults (recommended for most applications)
	fmt.Println("\n1. Production Defaults (NewWithDefaults)")
	runProductionDefaults()

	// Example 2: Simple constructor with string-based configuration
	fmt.Println("\n2. Simple Constructor (NewSimple)")
	runSimpleConstructor()

	// Example 3: Legacy constructor for backward compatibility
	fmt.Println("\n3. Legacy Constructor (New)")
	runLegacyConstructor()

	// Example 4: Daily rotation pattern
	fmt.Println("\n4. Daily Rotation (NewDaily)")
	runDailyRotation()

	// Example 5: Weekly rotation pattern
	fmt.Println("\n5. Weekly Rotation (NewWeekly)")
	runWeeklyRotation()

	// Example 6: Development-optimized configuration
	fmt.Println("\n6. Development Configuration (NewDevelopment)")
	runDevelopmentConfig()

	// Example 7: Custom configuration with full control
	fmt.Println("\n7. Custom Configuration (NewWithConfig)")
	runCustomConfig()

	fmt.Println("\nAll basic examples completed successfully.")
	fmt.Println("Check the ./logs/ directory for generated log files.")
}

// runProductionDefaults demonstrates the recommended constructor for production use.
// This provides sensible defaults suitable for most applications.
func runProductionDefaults() {
	// NewWithDefaults provides production-ready configuration:
	// - MaxSizeStr: "100MB" (rotates at 100MB)
	// - MaxAgeStr: "7d" (rotates weekly)
	// - MaxBackups: 10 (keeps 10 backup files)
	// - Compress: true (saves disk space)
	// - Async: true (better performance)
	// - BackpressurePolicy: "adaptive" (intelligent overflow handling)
	logger, err := lethe.NewWithDefaults("logs/production.log")
	if err != nil {
		log.Fatalf("Failed to create production logger: %v", err)
	}
	defer logger.Close()

	// Write sample log entries
	entries := []string{
		"Application started with production configuration",
		"Database connection established",
		"HTTP server listening on port 8080",
		"Background worker initialized",
	}

	for _, entry := range entries {
		if _, err := logger.Write([]byte(entry + "\n")); err != nil {
			log.Printf("Failed to write log entry: %v", err)
		}
	}

	fmt.Println("   Production logger created with defaults")
	fmt.Println("   Configuration: 100MB size, 7d rotation, 10 backups, compressed")
}

// runSimpleConstructor demonstrates modern string-based configuration.
// This is recommended when you need custom size limits with enhanced defaults.
func runSimpleConstructor() {
	// NewSimple enables modern string-based size configuration
	// with performance optimizations enabled by default
	logger, err := lethe.NewSimple("logs/simple.log", "50MB", 5)
	if err != nil {
		log.Fatalf("Failed to create simple logger: %v", err)
	}
	defer logger.Close()

	// Demonstrate different log entry formats
	logEntries := []string{
		"INFO: Service initialization completed",
		"WARN: High memory usage detected (85%)",
		"ERROR: Failed to connect to external API",
		"DEBUG: Processing user request ID=12345",
	}

	for _, entry := range logEntries {
		if _, err := logger.Write([]byte(entry + "\n")); err != nil {
			log.Printf("Failed to write log entry: %v", err)
		}
	}

	fmt.Println("   Simple logger created with 50MB limit and 5 backups")
	fmt.Println("   Features: async mode, adaptive backpressure, 4KB buffer")
}

// runLegacyConstructor demonstrates backward compatibility with older APIs.
// Use this when migrating from other rotation libraries.
func runLegacyConstructor() {
	// New() provides backward compatibility with integer-based configuration
	logger, err := lethe.New("logs/legacy.log", 25, 3) // 25MB, 3 backups
	if err != nil {
		log.Fatalf("Failed to create legacy logger: %v", err)
	}
	defer logger.Close()

	// Write legacy-style log entries
	legacyEntries := []string{
		"[2025-01-15 10:30:00] INFO: Legacy system startup",
		"[2025-01-15 10:30:01] INFO: Configuration loaded from config.xml",
		"[2025-01-15 10:30:02] WARN: Legacy API endpoint deprecated",
	}

	for _, entry := range legacyEntries {
		if _, err := logger.Write([]byte(entry + "\n")); err != nil {
			log.Printf("Failed to write legacy log entry: %v", err)
		}
	}

	fmt.Println("   Legacy logger created for backward compatibility")
	fmt.Println("   Configuration: 25MB size limit, 3 backup files")
}

// runDailyRotation demonstrates time-based rotation for daily log archives.
// Ideal for applications requiring daily log separation.
func runDailyRotation() {
	// NewDaily configures optimal settings for daily rotation
	logger, err := lethe.NewDaily("logs/daily.log")
	if err != nil {
		log.Fatalf("Failed to create daily logger: %v", err)
	}
	defer logger.Close()

	// Simulate daily operational logs
	dailyEntries := []string{
		"Daily backup process started",
		"Processing 1,247 user transactions",
		"System health check: all services operational",
		"Daily report generation completed",
	}

	for _, entry := range dailyEntries {
		if _, err := logger.Write([]byte(entry + "\n")); err != nil {
			log.Printf("Failed to write daily log entry: %v", err)
		}
	}

	fmt.Println("   Daily rotation logger configured")
	fmt.Println("   Rotation: every 24 hours, 7 backups, 50MB size limit")
}

// runWeeklyRotation demonstrates weekly rotation for lower-frequency logging.
// Perfect for summary logs or weekly reports.
func runWeeklyRotation() {
	// NewWeekly optimizes for weekly rotation with larger file sizes
	logger, err := lethe.NewWeekly("logs/weekly.log")
	if err != nil {
		log.Fatalf("Failed to create weekly logger: %v", err)
	}
	defer logger.Close()

	// Simulate weekly summary logs
	weeklyEntries := []string{
		"Weekly summary: 10,247 requests processed",
		"Performance metrics: avg response time 125ms",
		"Error rate: 0.02% (within acceptable range)",
		"Resource utilization: CPU 65%, Memory 78%",
	}

	for _, entry := range weeklyEntries {
		if _, err := logger.Write([]byte(entry + "\n")); err != nil {
			log.Printf("Failed to write weekly log entry: %v", err)
		}
	}

	fmt.Println("   Weekly rotation logger configured")
	fmt.Println("   Rotation: every 7 days, 4 backups, 200MB size limit")
}

// runDevelopmentConfig demonstrates development-optimized settings.
// Configured for immediate visibility and easy debugging.
func runDevelopmentConfig() {
	// NewDevelopment optimizes for development workflow
	logger, err := lethe.NewDevelopment("logs/debug.log")
	if err != nil {
		log.Fatalf("Failed to create development logger: %v", err)
	}
	defer logger.Close()

	// Simulate development debugging logs
	debugEntries := []string{
		"DEBUG: Function parseUserInput() called with args=['test']",
		"DEBUG: Database query executed in 12ms",
		"DEBUG: Cache hit for key 'user:session:abc123'",
		"DEBUG: Memory allocation: 2.3MB heap, 1.1MB stack",
	}

	for _, entry := range debugEntries {
		if _, err := logger.Write([]byte(entry + "\n")); err != nil {
			log.Printf("Failed to write debug log entry: %v", err)
		}
	}

	fmt.Println("   Development logger configured")
	fmt.Println("   Features: 10MB size, hourly rotation, no compression, sync writes")
}

// runCustomConfig demonstrates full configuration control using LoggerConfig.
// Use this when you need fine-grained control over all settings.
func runCustomConfig() {
	// NewWithConfig provides complete control over all options
	config := &lethe.LoggerConfig{
		Filename:           "logs/custom.log",
		MaxSizeStr:         "75MB",
		MaxAgeStr:          "3d",       // Rotate every 3 days
		MaxBackups:         15,         // Keep 15 backup files
		Compress:           true,       // Enable compression
		Checksum:           true,       // Enable data integrity verification
		Async:              true,       // Enable async mode
		BufferSize:         8192,       // 8KB buffer for performance
		BackpressurePolicy: "adaptive", // Intelligent overflow handling
		LocalTime:          true,       // Use local timezone for timestamps
		ErrorCallback: func(eventType string, err error) {
			// Custom error handling
			log.Printf("Log rotation event [%s]: %v", eventType, err)
		},
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Failed to create custom logger: %v", err)
	}
	defer logger.Close()

	// Demonstrate custom configuration with structured logging
	customEntries := []string{
		`{"level":"info","timestamp":"2025-01-15T10:30:00Z","message":"Custom logger initialized"}`,
		`{"level":"info","timestamp":"2025-01-15T10:30:01Z","message":"Processing with custom configuration"}`,
		`{"level":"warn","timestamp":"2025-01-15T10:30:02Z","message":"Custom checksum verification enabled"}`,
	}

	for _, entry := range customEntries {
		if _, err := logger.Write([]byte(entry + "\n")); err != nil {
			log.Printf("Failed to write custom log entry: %v", err)
		}
	}

	fmt.Println("   Custom logger created with full configuration control")
	fmt.Println("   Features: 75MB size, 3d rotation, checksums, 8KB buffer")
}
