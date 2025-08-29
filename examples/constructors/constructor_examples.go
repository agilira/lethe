// constructor_examples.go: Demonstrates all Lethe constructor functions
//
// This example shows the different ways to create Logger instances,
// from simple constructors to fully customized configurations.
//
// Run: go run constructor_examples.go

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agilira/lethe"
)

func main() {
	fmt.Println("Lethe Constructor Examples")
	fmt.Println("=====================================")

	// Example 1: Legacy constructor (backward compatible)
	fmt.Println("\n1. Legacy Constructor (New)")
	legacyLogger, err := lethe.New("logs/legacy.log", 50, 3) // 50MB, 3 backups
	if err != nil {
		log.Fatalf("Legacy constructor failed: %v", err)
	}
	if _, err := legacyLogger.Write([]byte("Message from legacy constructor\n")); err != nil {
		log.Printf("Warning: failed to write to legacy logger: %v", err)
	}
	if err := legacyLogger.Close(); err != nil {
		log.Printf("Warning: failed to close legacy logger: %v", err)
	}
	fmt.Println("   ✓ Created logger with legacy constructor")

	// Example 2: Modern simple constructor
	fmt.Println("\n2. Modern Simple Constructor (NewSimple)")
	simpleLogger, err := lethe.NewSimple("logs/simple.log", "100MB", 5)
	if err != nil {
		log.Fatalf("Simple constructor failed: %v", err)
	}
	if _, err := simpleLogger.Write([]byte("Message from simple constructor with async enabled\n")); err != nil {
		log.Printf("Warning: failed to write to simple logger: %v", err)
	}
	if err := simpleLogger.Close(); err != nil {
		log.Printf("Warning: failed to close simple logger: %v", err)
	}
	fmt.Println("   ✓ Created logger with modern string-based constructor")

	// Example 3: Production defaults
	fmt.Println("\n3. Production Defaults (NewWithDefaults)")
	defaultLogger, err := lethe.NewWithDefaults("logs/production.log")
	if err != nil {
		log.Fatalf("Default constructor failed: %v", err)
	}
	if _, err := defaultLogger.Write([]byte("Message from production logger with sensible defaults\n")); err != nil {
		log.Printf("Warning: failed to write to default logger: %v", err)
	}
	if err := defaultLogger.Close(); err != nil {
		log.Printf("Warning: failed to close default logger: %v", err)
	}
	fmt.Println("   ✓ Created logger with production defaults:")
	fmt.Println("     - 100MB size limit")
	fmt.Println("     - 7 day rotation")
	fmt.Println("     - 10 backup files")
	fmt.Println("     - Compression enabled")
	fmt.Println("     - Async enabled")

	// Example 4: Daily rotation
	fmt.Println("\n4. Daily Rotation (NewDaily)")
	dailyLogger, err := lethe.NewDaily("logs/daily.log")
	if err != nil {
		log.Fatalf("Daily constructor failed: %v", err)
	}
	if _, err := dailyLogger.Write([]byte("Message from daily rotation logger\n")); err != nil {
		log.Printf("Warning: failed to write to daily logger: %v", err)
	}
	if err := dailyLogger.Close(); err != nil {
		log.Printf("Warning: failed to close daily logger: %v", err)
	}
	fmt.Println("   ✓ Created daily rotation logger (24h rotation)")

	// Example 5: Weekly rotation
	fmt.Println("\n5. Weekly Rotation (NewWeekly)")
	weeklyLogger, err := lethe.NewWeekly("logs/weekly.log")
	if err != nil {
		log.Fatalf("Weekly constructor failed: %v", err)
	}
	if _, err := weeklyLogger.Write([]byte("Message from weekly rotation logger\n")); err != nil {
		log.Printf("Warning: failed to write to weekly logger: %v", err)
	}
	if err := weeklyLogger.Close(); err != nil {
		log.Printf("Warning: failed to close weekly logger: %v", err)
	}
	fmt.Println("   ✓ Created weekly rotation logger (7d rotation)")

	// Example 6: Development logger
	fmt.Println("\n6. Development Logger (NewDevelopment)")
	devLogger, err := lethe.NewDevelopment("logs/debug.log")
	if err != nil {
		log.Fatalf("Development constructor failed: %v", err)
	}
	if _, err := devLogger.Write([]byte("Debug message from development logger\n")); err != nil {
		log.Printf("Warning: failed to write to dev logger: %v", err)
	}
	if err := devLogger.Close(); err != nil {
		log.Printf("Warning: failed to close dev logger: %v", err)
	}
	fmt.Println("   ✓ Created development logger:")
	fmt.Println("     - 10MB size limit")
	fmt.Println("     - 1 hour rotation")
	fmt.Println("     - No compression (easier debugging)")
	fmt.Println("     - Synchronous writes")

	// Example 7: Custom configuration
	fmt.Println("\n7. Custom Configuration (NewWithConfig)")
	config := &lethe.LoggerConfig{
		Filename:           "logs/custom.log",
		MaxSizeStr:         "500MB",
		MaxAgeStr:          "30d",
		MaxBackups:         20,
		MaxFileAge:         180 * 24 * time.Hour, // 6 months
		Compress:           true,
		Checksum:           true, // Enable integrity checks
		Async:              true,
		BackpressurePolicy: "adaptive",
		LocalTime:          true,
	}
	customLogger, err := lethe.NewWithConfig(config)
	if err != nil {
		log.Fatalf("Custom constructor failed: %v", err)
	}
	if _, err := customLogger.Write([]byte("Message from custom configured logger\n")); err != nil {
		log.Printf("Warning: failed to write to custom logger: %v", err)
	}
	if err := customLogger.Close(); err != nil {
		log.Printf("Warning: failed to close custom logger: %v", err)
	}
	fmt.Println("   ✓ Created custom logger with full configuration")

	// Example 8: Integration with standard library
	fmt.Println("\n8. Standard Library Integration")
	stdLogger, err := lethe.NewWithDefaults("logs/stdlib.log")
	if err != nil {
		log.Fatalf("Stdlib integration failed: %v", err)
	}

	// Replace default log output
	originalOutput := log.Writer()
	log.SetOutput(stdLogger)

	log.Println("This message goes through the standard library to Lethe")

	// Restore original output
	log.SetOutput(originalOutput)
	if err := stdLogger.Close(); err != nil {
		log.Printf("Warning: failed to close stdlib logger: %v", err)
	}
	fmt.Println("   ✓ Integrated with standard library log package")

	// Example 9: Error handling demonstration
	fmt.Println("\n9. Error Handling Examples")

	// This will fail - empty filename
	if _, err := lethe.NewWithDefaults(""); err != nil {
		fmt.Printf("   ✓ Proper error handling: %v\n", err)
	}

	// This will fail - invalid size format
	if _, err := lethe.NewSimple("test.log", "invalid_size", 5); err != nil {
		fmt.Printf("   ✓ Invalid size validation: %v\n", err)
	}

	fmt.Println("\n🎉 All constructor examples completed successfully!")
	fmt.Println("\nGenerated log files in ./logs/ directory:")

	// List generated files
	if files, err := os.ReadDir("logs"); err == nil {
		for _, file := range files {
			if !file.IsDir() {
				if info, err := file.Info(); err == nil {
					fmt.Printf("   - %s (%d bytes)\n", file.Name(), info.Size())
				}
			}
		}
	}

	fmt.Println("\n💡 Tips:")
	fmt.Println("   - Use NewWithDefaults() for most production applications")
	fmt.Println("   - Use NewDevelopment() during development and debugging")
	fmt.Println("   - Use NewDaily()/NewWeekly() for time-based rotation needs")
	fmt.Println("   - Use NewWithConfig() when you need full control")
	fmt.Println("   - Always defer logger.Close() for proper cleanup")
}
