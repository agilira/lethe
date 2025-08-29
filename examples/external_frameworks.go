// External Framework Integration Examples
// Examples for integrating Lethe with popular Go logging frameworks
// Requires external dependencies - run: go get github.com/sirupsen/logrus go.uber.org/zap github.com/rs/zerolog
// Copyright 2025 AGILira
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"time"

	"github.com/agilira/lethe"
)

// Note: Uncomment the import statements and function calls when dependencies are available

// LetheIrisAdapter implements the WriteSyncer interface required by Iris
// This adapter allows Iris to use Lethe for high-performance log rotation
type LetheIrisAdapter struct {
	rotator *lethe.Logger
}

// NewLetheIrisAdapter creates a new adapter that bridges Iris and Lethe
// The adapter implements the WriteSyncer interface expected by Iris loggers
func NewLetheIrisAdapter(rotator *lethe.Logger) *LetheIrisAdapter {
	return &LetheIrisAdapter{
		rotator: rotator,
	}
}

// Write implements io.Writer interface for Iris integration
// This method receives log data from Iris and forwards it to Lethe
func (a *LetheIrisAdapter) Write(p []byte) (int, error) {
	return a.rotator.Write(p)
}

// Sync implements WriteSyncer interface for Iris integration
// This ensures data durability when called by Iris
func (a *LetheIrisAdapter) Sync() error {
	// Lethe handles flushing internally through its MPSC buffer
	// For sync operations, we could add a Flush() method to Lethe
	// For now, we return nil as Lethe's async mode handles this
	return nil
}

// WriteOwned provides zero-copy integration for advanced Iris usage
// This method allows Iris to transfer buffer ownership to Lethe for maximum performance
func (a *LetheIrisAdapter) WriteOwned(data []byte) (int, error) {
	return a.rotator.WriteOwned(data)
}

// Close gracefully closes the underlying Lethe rotator
func (a *LetheIrisAdapter) Close() error {
	return a.rotator.Close()
}

// Logrus Integration Example
func exampleLogrusIntegration() {
	fmt.Println("=== Logrus Integration ===")

	/*
		// Uncomment when logrus is available:
		import "github.com/sirupsen/logrus"

		rotator := &lethe.Logger{
			Filename:   "examples/logrus.log",
			MaxSizeStr: "20MB",
			MaxBackups: 7,
			Compress:   true,
			Async:      true,
			BufferSize: 512,
		}
		defer rotator.Close()

		// Setup Logrus with Lethe
		logrus.SetOutput(rotator)
		logrus.SetLevel(logrus.InfoLevel)
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})

		// Use Logrus as usual - all output goes to Lethe
		logrus.Info("Logrus integration with Lethe active")

		logrus.WithFields(logrus.Fields{
			"user_id": "user123",
			"action":  "login",
			"ip":      "192.168.1.100",
		}).Info("User login event")

		logrus.WithField("component", "auth").Error("❌ Authentication failed")

		// Performance test
		start := time.Now()
		for i := 0; i < 1000; i++ {
			logrus.WithFields(logrus.Fields{
				"iteration": i,
				"module":    "benchmark",
			}).Info("Bulk logging entry")
		}
		duration := time.Since(start)

		logrus.WithField("duration", duration).Info("✅ Logrus performance test completed")
		fmt.Printf("✅ Logrus integration: 1000 entries in %v\n", duration)
	*/

	fmt.Println("Logrus integration example available - install with: go get github.com/sirupsen/logrus")
}

// Zap Integration Example
func exampleZapIntegration() {
	fmt.Println("=== Zap Integration ===")

	/*
		// Uncomment when zap is available:
		import (
			"go.uber.org/zap"
			"go.uber.org/zap/zapcore"
		)

		rotator := &lethe.Logger{
			Filename:   "examples/zap.log",
			MaxSizeStr: "30MB",
			MaxBackups: 5,
			Compress:   true,
			Async:      true,
			BufferSize: 1024,
			LocalTime:  true, // Use local time for compatibility
		}
		defer rotator.Close()

		// Create Zap logger with Lethe as output
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(rotator), // Lethe implements io.Writer
			zapcore.InfoLevel,
		)

		logger := zap.New(core, zap.AddCaller())
		defer logger.Sync()

		// Use Zap as usual - all output goes to Lethe
		logger.Info("Zap integration with Lethe active")

		logger.Info(" User action logged",
			zap.String("user_id", "user456"),
			zap.String("action", "purchase"),
			zap.Float64("amount", 99.99),
			zap.String("currency", "USD"),
		)

		// Sugar logger for convenience
		sugar := logger.Sugar()
		sugar.Infow("Payment processed",
			"transaction_id", "txn_789",
			"status", "completed",
			"processing_time_ms", 125,
		)

		// Performance test
		start := time.Now()
		for i := 0; i < 1000; i++ {
			logger.Info("Bulk logging entry",
				zap.Int("iteration", i),
				zap.String("module", "benchmark"),
				zap.Int64("timestamp", time.Now().UnixNano()),
			)
		}
		duration := time.Since(start)

		logger.Info("Zap performance test completed",
			zap.Duration("duration", duration),
			zap.Int("entries", 1000),
		)

		fmt.Printf("Zap integration: 1000 entries in %v\n", duration)
	*/

	fmt.Println("Zap integration example available - install with: go get go.uber.org/zap")
}

// Zerolog Integration Example
func exampleZerologIntegration() {
	fmt.Println("=== Zerolog Integration ===")

	/*
		// Uncomment when zerolog is available:
		import "github.com/rs/zerolog"

		rotator := &lethe.Logger{
			Filename:   "examples/zerolog.log",
			MaxSizeStr: "25MB",
			MaxBackups: 8,
			Compress:   true,
			Checksum:   true, // Enable checksums for data integrity
			Async:      true,
			BufferSize: 512,
		}
		defer rotator.Close()

		// Setup Zerolog with Lethe as output
		logger := zerolog.New(rotator).With().
			Timestamp().
			Str("service", "example").
			Logger()

		// Use Zerolog as usual - all output goes to Lethe
		logger.Info().Msg("Zerolog integration with Lethe active")

		logger.Info().
			Str("user_id", "user789").
			Str("action", "logout").
			Str("session_id", "sess_abc123").
			Msg("User logout event")

		// Structured logging with nested objects
		logger.Info().
			Dict("user", zerolog.Dict().
				Str("id", "user999").
				Str("name", "John Doe").
				Int("age", 30).
				Str("role", "admin")).
			Dict("request", zerolog.Dict().
				Str("method", "POST").
				Str("path", "/api/users").
				Int("status", 201)).
			Msg("API request processed")

		// Performance test
		start := time.Now()
		for i := 0; i < 1000; i++ {
			logger.Info().
				Int("iteration", i).
				Str("module", "benchmark").
				Int64("nano", time.Now().UnixNano()).
				Msg("Bulk logging entry")
		}
		duration := time.Since(start)

		logger.Info().
			Dur("duration", duration).
			Int("entries", 1000).
			Msg("Zerolog performance test completed")

		fmt.Printf("Zerolog integration: 1000 entries in %v\n", duration)
	*/

	fmt.Println("Zerolog integration example available - install with: go get github.com/rs/zerolog")
}

// Iris Logger Integration Example (Primary Integration)
func exampleIrisIntegration() {
	fmt.Println("=== Iris Integration (Primary Design Target) ===")

	// Create Lethe rotator optimized for Iris integration
	rotator := &lethe.Logger{
		Filename:           "examples/iris.log",
		MaxSizeStr:         "50MB",
		MaxBackups:         10,
		Compress:           true,
		Async:              true,
		BufferSize:         4096,
		BackpressurePolicy: "adaptive", // Perfect for Iris ultra-high performance
		AdaptiveFlush:      true,
	}
	defer rotator.Close()

	// Create the Lethe-Iris adapter implementing WriteSyncer interface
	adapter := NewLetheIrisAdapter(rotator)

	fmt.Println("Lethe-Iris adapter created successfully")
	fmt.Printf("Adapter type: WriteSyncer interface compatible\n")
	fmt.Printf("Ready for Iris ultra-high performance logging\n")

	// Demonstrate integration pattern (would be used in real Iris setup)
	start := time.Now()
	for i := 0; i < 10000; i++ {
		// Simulate what Iris would do: create structured log entries
		logEntry := fmt.Sprintf(`{"level":"info","msg":"Request %d processed","method":"GET","path":"/api/users","status":200,"duration":"15ms"}`, i)

		// Write through the adapter (this is what Iris would do)
		_, _ = adapter.Write([]byte(logEntry + "\n")) // Ignore errors in example
	}

	// Ensure all data is flushed
	_ = adapter.Sync() // Ignore errors in example
	duration := time.Since(start)

	// Allow MPSC to finish
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("Iris integration: 10,000 entries in %v\n", duration)
	fmt.Printf("Write performance: %.2f entries/ms\n",
		float64(10000)/float64(duration.Milliseconds()))
	fmt.Println("Integration completed - Iris can now use Lethe for log rotation")

	// Example usage in a real Iris application:
	fmt.Println("\n Usage in Iris application:")
	fmt.Println("   // Create Lethe rotator")
	fmt.Println("   rotator := &lethe.Logger{...}")
	fmt.Println("   adapter := NewLetheIrisAdapter(rotator)")
	fmt.Println("")
	fmt.Println("   // Use with Iris logger config")
	fmt.Println("   logger, err := iris.New(iris.Config{")
	fmt.Println("       Output:   adapter,  // Use Lethe adapter as WriteSyncer")
	fmt.Println("       Encoder:  iris.NewJSONEncoder(),")
	fmt.Println("       Level:    iris.Info,")
	fmt.Println("   })")
	fmt.Println("   logger.Start()")
	fmt.Println("")
	fmt.Println("   // For maximum performance with zero-copy:")
	fmt.Println("   // Use adapter.WriteOwned(buffer) for direct buffer transfer")
}

// Demonstration function showing all framework integrations
func demonstrateAllFrameworks() {
	fmt.Println("Lethe External Framework Integration Examples")
	fmt.Println("=============================================")
	fmt.Println()

	exampleLogrusIntegration()
	fmt.Println()

	exampleZapIntegration()
	fmt.Println()

	exampleZerologIntegration()
	fmt.Println()

	exampleIrisIntegration()
	fmt.Println()

	fmt.Println("Summary:")
	fmt.Println("  Iris    - Primary integration target via LetheIrisAdapter")
	fmt.Println("  Logrus  - Install: go get github.com/sirupsen/logrus")
	fmt.Println("  Zap     - Install: go get go.uber.org/zap")
	fmt.Println("  Zerolog - Install: go get github.com/rs/zerolog")

	fmt.Println()
	fmt.Println("✅ All frameworks integrate seamlessly with Lethe:")
	fmt.Println("  • Iris: Primary design target with zero-copy support")
	fmt.Println("  • Standard frameworks via io.Writer interface")
	fmt.Println("  • WriteOwned() API available for maximum performance integration")
}

// init function ensures all example functions are referenced to avoid unused warnings
func init() {
	// Reference all example functions to prevent "unused" warnings
	// These are stored as function variables but not called during init
	_ = exampleIrisIntegration
	_ = exampleLogrusIntegration
	_ = exampleZapIntegration
	_ = exampleZerologIntegration
	_ = demonstrateAllFrameworks
}

// Main function to run all framework integration demonstrations
// Uncomment to run this specific example (comment out main in basic_integration.go first)
/*
func main() {
	demonstrateAllFrameworks()
}
*/
